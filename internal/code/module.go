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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rogpeppe/go-internal/modfile"
	"github.com/rogpeppe/go-internal/module"
	"github.com/shurcooL/go/vfs/godocfs/vfsutil"
	"github.com/shurcooL/home/internal/mod"
	"github.com/shurcooL/httperror"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/git"
)

// ModuleHandler is a Go module server that implements the
// module proxy protocol. It serves each repository root
// available in Code as a Go module.
//
// At this time, it has various limitations compared to the
// general go mod download functionality that extracts module
// versions from a VCS repository:
//
// 	•	Versions served include only pseudo-versions from
// 		commits on master branch. Tags and other branches
// 		are not supported at this time.
// 	•	Multi-module repositories are not supported at this time.
//
// This may change over time as my needs evolve.
type ModuleHandler struct {
	Code *Service
}

// ServeModuleMaybe serves a module proxy protocol HTTP request, if it matches.
// It returns httperror.NotHandle if the HTTP request was explicitly not handled.
func (h ModuleHandler) ServeModuleMaybe(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return httperror.NotHandle
	}

	// Parse the module path, type, and version from the URL.
	r, ok := parseModuleProxyRequest("dmitri.shuralyov.com" + req.URL.Path)
	if !ok {
		return httperror.NotHandle
	}
	dec, ok := r.Decode() // Decode module path and version.
	if !ok {
		// Maybe it was an unencoded module path or version (e.g., from a human visitor).
		// Check if they can both be successfully encoded. If so, redirect to that URL.
		if enc, ok := r.Encode(); ok {
			// Preserve the current scheme and host.
			u, err := url.Parse("https://" + enc.URL())
			if err != nil {
				return fmt.Errorf("ModuleHandler.ServeModuleMaybe: failed to parse own redirect URL: %v", err)
			}
			return httperror.Redirect{URL: u.Path}
		}
		return httperror.NotHandle
	}
	modulePath, typ, version := dec.Module, dec.Type, dec.Version

	// Look up code directory by module path.
	d, ok := h.Code.Lookup(modulePath)
	if !ok || !d.IsRepoRoot() {
		return httperror.NotHandle
	}
	gitDir := filepath.Join(h.Code.reposDir, filepath.FromSlash(d.RepoRoot))

	// Handle "/@v/list" request.
	if typ == "list" {
		revs, err := listMasterCommits(req.Context(), gitDir)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for i := len(revs) - 1; i >= 0; i-- {
			fmt.Fprintln(w, revs[i].Version)
		}
		return nil
	}

	// Parse the time and revision from the pseudo-version.
	versionTime, versionRevision, err := mod.ParsePseudoVersion(version)
	if err != nil || len(versionRevision) != 12 || !mod.AllHex(versionRevision) {
		return os.ErrNotExist
	}

	// Open the git repository and get the commit that corresponds to the pseudo-version.
	repo, err := git.Open(gitDir)
	if err != nil {
		return err
	}
	defer func() {
		if err := repo.Close(); err != nil {
			log.Println("ModuleHandler.ServeModuleMaybe: repo.Close:", err)
		}
	}()
	commitID, err := repo.ResolveRevision(versionRevision)
	if err != nil {
		return os.ErrNotExist
	}
	commit, err := repo.GetCommit(commitID)
	if err != nil || commit.Committer == nil || !versionTime.Equal(time.Unix(commit.Committer.Date.Seconds, 0).UTC()) {
		return os.ErrNotExist
	}

	// Handle one of "/@v/<version>.<ext>" requests.
	switch typ {
	case "info":
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		err := enc.Encode(mod.RevInfo{
			Version: version,
			Time:    versionTime,
		})
		return err
	case "mod":
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
	case "zip":
		fs, err := repo.FileSystem(commitID)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/zip")
		z := zip.NewWriter(w)
		err = vfsutil.Walk(fs, "/", func(name string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				// We only care about files.
				return nil
			}
			b, err := vfs.ReadFile(fs, name)
			if err != nil {
				return err
			}
			f, err := z.Create(modulePath + "@" + version + name)
			if err != nil {
				return err
			}
			_, err = f.Write(b)
			return err
		})
		if err != nil {
			return err
		}
		err = z.Close()
		return err
	default:
		return os.ErrNotExist
	}
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
// The Module and Version fields may be encoded or unencoded.
type moduleProxyRequest struct {
	Module  string // Module path.
	Type    string // Type of request. One of "list", "info", "mod", or "zip".
	Version string // Module version. Applies only when Type is not "list".
}

// parseModuleProxyRequest parses the module proxy request
// from the given URL. It does not attempt to decode the
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

// Decode returns a copy of r with Module and Version fields decoded.
func (r moduleProxyRequest) Decode() (_ moduleProxyRequest, ok bool) {
	var err error
	r.Module, err = module.DecodePath(r.Module)
	if err != nil {
		return moduleProxyRequest{}, false
	}
	if r.Type == "list" {
		return r, true
	}
	r.Version, err = module.DecodeVersion(r.Version)
	if err != nil {
		return moduleProxyRequest{}, false
	}
	return r, true
}

// Encode returns a copy of r with Module and Version fields encoded.
func (r moduleProxyRequest) Encode() (_ moduleProxyRequest, ok bool) {
	var err error
	r.Module, err = module.EncodePath(r.Module)
	if err != nil {
		return moduleProxyRequest{}, false
	}
	if r.Type == "list" {
		return r, true
	}
	r.Version, err = module.EncodeVersion(r.Version)
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
