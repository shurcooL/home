package code

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shurcooL/go/vfs/godocfs/vfsutil"
	"github.com/shurcooL/home/internal/mod"
	"github.com/shurcooL/httperror"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/git"
)

// ModuleHandler is a Go module server that implements the
// module proxy protocol, as specified at
// https://golang.org/cmd/go/#hdr-Module_proxy_protocol.
//
// At this time, it has various restrictions compared to the
// general go mod download functionality that extracts module
// versions from a VCS repository:
//
// • It serves only pseudo-versions derived from commits
// on master branch. No other versions or module queries
// are supported at this time.
//
// • It serves a single module corresponding to the root
// of each repository. Multi-module repositories are not
// supported at this time.
//
// • It serves only the v0 major version. Major versions
// other than v0 are not supported at this time.
//
// This may change over time as my needs evolve.
type ModuleHandler struct {
	// Code is the underlying source of Go code.
	// Each repository root available in it is served as a Go module.
	Code *Service
}

// ServeModule serves a module proxy protocol HTTP request.
//
// The "$GOPROXY/" prefix must be stripped from req.URL.Path, so that
// the given req.URL.Path is like "<module>/@v/<version>.info" (no leading slash).
func (h ModuleHandler) ServeModule(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return httperror.Method{Allowed: []string{http.MethodGet}}
	}

	// Parse the module path, type, and version from the URL.
	r, ok := parseModuleProxyRequest(req.URL.Path)
	if !ok {
		return os.ErrNotExist
	}
	unesc, ok := r.Unescape() // Unescape module path and version.
	if !ok {
		return httperror.BadRequest{Err: fmt.Errorf("failed to unescape module path=%q and/or version=%q", r.Module, r.Version)}
	}
	modulePath, typ, version := unesc.Module, unesc.Type, unesc.Version

	// Look up code directory by module path.
	d, err := h.Code.GetDirectory(req.Context(), modulePath)
	if err != nil || !d.IsRepoRoot() {
		return os.ErrNotExist
	}
	gitDir := filepath.Join(h.Code.reposDir, filepath.FromSlash(d.RepoRoot))

	// Handle "/@v/list" request.
	if typ == "list" {
		return h.serveList(req.Context(), w, gitDir)
	}

	// Parse the time and revision from the v0.0.0 pseudo-version.
	versionTime, versionRevision, err := mod.ParseV000PseudoVersion(version)
	if err != nil {
		return os.ErrNotExist
	}

	// Open the git repository and get the commit that corresponds to the pseudo-version.
	repo, err := git.Open(gitDir)
	if err != nil {
		return err
	}
	defer func() {
		if err := repo.Close(); err != nil {
			log.Println("ModuleHandler.ServeModule: repo.Close:", err)
		}
	}()
	commitID, err := repo.ResolveRevision(versionRevision)
	if err != nil {
		return os.ErrNotExist
	}
	commit, err := repo.GetCommit(commitID)
	if err != nil || commit.Committer == nil || !versionTime.Equal(time.Unix(commit.Committer.Date.Seconds, 0).UTC()) {
		return os.ErrNotExist
	} else if !isCommitOnMaster(req.Context(), gitDir, commit) {
		return os.ErrNotExist
	}

	// Handle one of "/@v/<version>.<ext>" requests.
	switch typ {
	case "info":
		return h.serveInfo(w, version, versionTime)
	case "mod":
		return h.serveMod(w, modulePath, repo, commitID)
	case "zip":
		return h.serveZip(w, modulePath, version, repo, commitID)
	default:
		panic("unreachable")
	}
}

func (ModuleHandler) serveList(ctx context.Context, w http.ResponseWriter, gitDir string) error {
	revs, err := listMasterCommits(ctx, gitDir)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for i := len(revs) - 1; i >= 0; i-- {
		fmt.Fprintln(w, revs[i].Version)
	}
	return nil
}

func (ModuleHandler) serveInfo(w http.ResponseWriter, version string, time time.Time) error {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	err := enc.Encode(mod.RevInfo{
		Version: version,
		Time:    time,
	})
	return err
}

func (ModuleHandler) serveMod(w http.ResponseWriter, modulePath string, repo *git.Repository, commitID vcs.CommitID) error {
	fs, err := repo.FileSystem(commitID)
	if err != nil {
		return err
	}
	f, err := fs.Open("/go.mod")
	if os.IsNotExist(err) {
		// go.mod file doesn't exist in this commit.
		f = nil
	} else if err != nil {
		return err
	}
	if f != nil {
		defer f.Close()
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if f != nil {
		// Copy the existing go.mod file.
		_, err := io.Copy(w, f)
		return err
	} else {
		// Synthesize a go.mod file with just the module path.
		_, err := fmt.Fprintf(w, "module %s\n", modfile.AutoQuote(modulePath))
		return err
	}
}

func (ModuleHandler) serveZip(w http.ResponseWriter, modulePath, version string, repo *git.Repository, commitID vcs.CommitID) error {
	w.Header().Set("Content-Type", "application/zip")
	return WriteModuleZip(w, module.Version{Path: modulePath, Version: version}, repo, commitID)
}

// WriteModuleZip builds a zip archive for module version m
// by including all files from repository r at commit id,
// and writes the result to w.
//
// WriteModuleZip does not support multi-module repositories.
// A go.mod file may be in root, but not in any other directory.
//
// Unlike "golang.org/x/mod/zip".Create, it does not verify
// any module zip restrictions. It will produce an invalid
// module zip if given a commit containing invalid files.
// It should be used on commits that are known to have files
// that are all acceptable to include in a module zip.
//
func WriteModuleZip(w io.Writer, m module.Version, r vcs.Repository, id vcs.CommitID) error {
	fs, err := r.FileSystem(id)
	if err != nil {
		return err
	}
	z := zip.NewWriter(w)
	err = vfsutil.Walk(fs, "/", func(name string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			// We need to include only files, not directories.
			return nil
		}
		dst, err := z.Create(m.Path + "@" + m.Version + name)
		if err != nil {
			return err
		}
		src, err := fs.Open(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		src.Close()
		return err
	})
	if err != nil {
		return err
	}
	err = z.Close()
	return err
}

// isCommitOnMaster reports whether commit c is a part of master branch
// of git repo at gitDir, and no errors occurred while determining that.
func isCommitOnMaster(ctx context.Context, gitDir string, c *vcs.Commit) bool {
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", "--", string(c.ID), "master")
	cmd.Dir = gitDir
	err := cmd.Run()
	return err == nil
}

// listMasterCommits returns a list of commits in git repo on master branch.
// If master branch doesn't exist, an empty list is returned.
func listMasterCommits(ctx context.Context, gitDir string) ([]mod.RevInfo, error) {
	cmd := exec.CommandContext(ctx, "git", "log",
		"--format=tformat:%H%x00%ct",
		"-z",
		"master")
	cmd.Dir = gitDir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("could not start command: %v", err)
	}
	err = cmd.Wait()
	if ee, _ := err.(*exec.ExitError); ee != nil && ee.Sys().(syscall.WaitStatus).ExitStatus() == 128 {
		return nil, nil // Master branch doesn't exist.
	} else if err != nil {
		return nil, fmt.Errorf("%v: %v", cmd.Args, err)
	}

	var revs []mod.RevInfo
	for b := buf.Bytes(); len(b) != 0; {
		var (
			// Calls to readLine match exactly what is specified in --format.
			commitHash    = readLine(&b)
			committerDate = readLine(&b)
		)
		timestamp, err := strconv.ParseInt(committerDate, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid time from git log: %v", err)
		}
		t := time.Unix(timestamp, 0).UTC()
		revs = append(revs, mod.RevInfo{
			Version: mod.PseudoVersion("", "", t, commitHash[:12]),
			Time:    t,
		})
	}
	return revs, nil
}

// readLine reads a line until zero byte, then updates b to the byte that immediately follows.
// A zero byte must exist in b, otherwise readLine panics.
func readLine(b *[]byte) string {
	i := bytes.IndexByte(*b, 0)
	s := string((*b)[:i])
	*b = (*b)[i+1:]
	return s
}

// moduleProxyRequest represents a module proxy request.
// The Module and Version fields may be escaped or unescaped.
type moduleProxyRequest struct {
	Module  string // Module path.
	Type    string // Type of request. One of "list", "info", "mod", or "zip".
	Version string // Module version. Applies only when Type is not "list".
}

// parseModuleProxyRequest parses the module proxy request
// from the given URL. It does not attempt to unescape the
// module path and version, the caller is responsible for that.
func parseModuleProxyRequest(url string) (_ moduleProxyRequest, ok bool) {
	// Split "<module>/@v/<file>" into module and file.
	i := strings.Index(url, "/@v/")
	if i == -1 {
		return moduleProxyRequest{}, false
	}
	module, file := url[:i], url[i+len("/@v/"):]

	// Return early for "/@v/list" request. It has no Version.
	if file == "list" {
		return moduleProxyRequest{Module: module, Type: "list"}, true
	}

	// Split "/@v/<version>.<ext>" into version and ext.
	i = strings.LastIndexByte(file, '.')
	if i == -1 {
		return moduleProxyRequest{}, false
	}
	version, ext := file[:i], file[i+1:]

	// Check that ext is valid.
	switch ext {
	case "info", "mod", "zip":
		return moduleProxyRequest{Module: module, Type: ext, Version: version}, true
	default:
		return moduleProxyRequest{}, false
	}
}

// Unescape returns a copy of r with Module and Version fields unescaped.
func (r moduleProxyRequest) Unescape() (_ moduleProxyRequest, ok bool) {
	var err error
	r.Module, err = module.UnescapePath(r.Module)
	if err != nil {
		return moduleProxyRequest{}, false
	}
	if r.Type == "list" {
		return r, true
	}
	r.Version, err = module.UnescapeVersion(r.Version)
	if err != nil {
		return moduleProxyRequest{}, false
	}
	return r, true
}

// Escape returns a copy of r with Module and Version fields escaped.
func (r moduleProxyRequest) Escape() (_ moduleProxyRequest, ok bool) {
	var err error
	r.Module, err = module.EscapePath(r.Module)
	if err != nil {
		return moduleProxyRequest{}, false
	}
	if r.Type == "list" {
		return r, true
	}
	r.Version, err = module.EscapeVersion(r.Version)
	if err != nil {
		return moduleProxyRequest{}, false
	}
	return r, true
}

// URL returns the URL of the module proxy request.
func (r moduleProxyRequest) URL() string {
	switch r.Type {
	case "list":
		return r.Module + "/@v/list"
	default:
		return r.Module + "/@v/" + r.Version + "." + r.Type
	}
}
