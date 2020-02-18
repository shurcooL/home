// pre-receive is a pre-receive git hook
// for use with home's git server.
//
// It verifies commits pushed to master branch
// to ensure they produce good module versions.
//
// An environment variable HOME_MODULE_PATH must be set to
// the module path corresponding to the git repository root.
package main

import (
	"archive/tar"
	archivezip "archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/mod"
	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/dirhash"
	modzip "golang.org/x/mod/zip"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func main() {
	ctx := Context{
		ModulePath: os.Getenv("HOME_MODULE_PATH"),
	}
	err := ctx.Verify(os.Stdin)
	if err != nil {
		fmt.Printf("something went wrong: %v\n", err)
		os.Exit(1)
	}
	ok := ctx.Report(os.Stdout)
	if !ok {
		os.Exit(1)
	}
}

// Context holds the working context for a pre-receive hook run.
type Context struct {
	// Input.
	ModulePath string

	// Output.
	Commits []Commit
	Bad     int // Number of commits that have errors.
}

// Commit represents a commit that corresponds to a module version.
type Commit struct {
	ID      string
	Subject string
	Version module.Version
	Errors  []string
}

// Verify runs all the checks for the given pre-receive hook input.
func (ctx *Context) Verify(stdin io.Reader) error {
	err := foreachRef(stdin, func(shaOld, shaNew, refName string) error {
		if refName != "refs/heads/master" {
			// We are only verifying commits
			// to master branch at this time.
			return nil
		}
		err := foreachCommit(shaOld, shaNew, func(r vcs.Repository, commit *vcs.Commit) error {
			c, err := VerifyCommit(ctx.ModulePath, r, commit)
			if err != nil {
				return err
			}
			ctx.Commits = append(ctx.Commits, c)
			if len(c.Errors) > 0 {
				ctx.Bad++
			}
			return nil
		})
		return err
	})
	return err
}

// Report reports the results of verify.
func (ctx *Context) Report(w io.Writer) (ok bool) {
	fmt.Fprintf(w, "publishing %d pseudo-versions\n", len(ctx.Commits))
	if ctx.Bad > 0 {
		fmt.Fprintf(w, "error: rejecting push due to %d bad module versions\n", ctx.Bad)
	}
	fmt.Fprintln(w)
	for _, c := range ctx.Commits {
		fmt.Fprintln(w, " commit:", c.ID)
		fmt.Fprintln(w, "subject:", c.Subject)
		fmt.Fprintln(w, "version:", c.Version)
		for _, e := range c.Errors {
			fmt.Fprintln(w, "  error:", e)
		}
		fmt.Fprintln(w)
	}
	if ctx.Bad > 0 {
		fmt.Fprintln(w, "there were problems, stopping")
		return false
	}
	fmt.Fprintln(w, "done")
	return true
}

// VerifyCommit verifies the given commit.
func VerifyCommit(modulePath string, r vcs.Repository, c *vcs.Commit) (Commit, error) {
	// Get commit and module version information.
	commit := Commit{
		ID:      string(c.ID),
		Subject: subject(c.Message),
		Version: module.Version{
			Path:    modulePath,
			Version: mod.PseudoVersion("", "", time.Unix(c.Committer.Date.Seconds, 0).UTC(), string(c.ID[:12])),
		},
	}

	// Verify pseudo-version time.
	err := verifyPseudoVersionTime(r, c)
	if e := (BadVersionError{}); errors.As(err, &e) {
		commit.Errors = append(commit.Errors, e.Text)
	} else if err != nil {
		return Commit{}, err
	}

	// Verify module zip contents.
	err = verifyModuleZip(commit.Version, r, c.ID)
	if e := (BadVersionError{}); errors.As(err, &e) {
		commit.Errors = append(commit.Errors, e.Text)
	} else if err != nil {
		return Commit{}, err
	}

	// Verify there is a LICENSE file.
	err = verifyHasLICENSE(r, c.ID)
	if e := (BadVersionError{}); errors.As(err, &e) {
		commit.Errors = append(commit.Errors, e.Text)
	} else if err != nil {
		return Commit{}, err
	}

	return commit, nil
}

// BadVersionError represents an error where a module version is bad.
type BadVersionError struct {
	Text string
}

func (e BadVersionError) Error() string { return "bad module version: " + e.Text }

// verifyPseudoVersionTime verifies that pseudo-version
// time is strictly after its parent's time.
func verifyPseudoVersionTime(r vcs.Repository, c *vcs.Commit) error {
	if len(c.Parents) == 0 {
		// Initial commit. Nothing to check.
		return nil
	} else if len(c.Parents) > 1 {
		return BadVersionError{fmt.Sprintf("commit %q has %d parents, want no more than 1", c.ID, len(c.Parents[0]))}
	}
	p, err := getCommit(r, c.Parents[0])
	if err != nil {
		return err
	}
	strictlyAfter := c.Committer.Date.Seconds > p.Committer.Date.Seconds
	if !strictlyAfter {
		return BadVersionError{fmt.Sprintf("commit %q time (%v) is not strictly after its parent's time (%v)", c.ID,
			time.Unix(c.Committer.Date.Seconds, 0).UTC(),
			time.Unix(p.Committer.Date.Seconds, 0).UTC())}
	}
	return nil
}

// verifyModuleZip verifies that the module zip created for
// the given commit using the simplified code.WriteModuleZip
// algorithm would have an identical hash as that of a module
// zip created by the official "golang.org/x/mod/zip".Create
// algorithm.
func verifyModuleZip(m module.Version, r vcs.Repository, commitID vcs.CommitID) error {
	var got, want struct {
		Zip []byte
		Sum string
	}

	// Compute reference module zip.
	// Do this first, because it's more likely to detect a problem earlier.
	{
		var files []modzip.File

		// Get a git archive.
		cmd := exec.Command("git", "-c", "core.autocrlf=input", "archive", "--format=tar", string(commitID))
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		err = cmd.Start()
		if err != nil {
			return err
		}
		t := tar.NewReader(stdout)
		for {
			hdr, err := t.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			if hdr.Typeflag == tar.TypeXGlobalHeader || hdr.Typeflag == tar.TypeDir {
				continue
			}
			b, err := ioutil.ReadAll(t)
			if err != nil {
				return err
			}
			files = append(files, tarFile{path: hdr.Name, fi: hdr.FileInfo(), b: b})
		}
		err = cmd.Wait()
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		err = modzip.Create(&buf, m, files)
		if err != nil {
			return BadVersionError{errors.Unwrap(err).Error()}
		}
		want.Zip = buf.Bytes()
	}

	// Compute our own module zip.
	{
		var buf bytes.Buffer
		err := code.WriteModuleZip(&buf, m, r, commitID)
		if err != nil {
			return err
		}
		got.Zip = buf.Bytes()
	}

	var err error
	got.Sum, err = mod.HashZip(got.Zip, dirhash.DefaultHash)
	if err != nil {
		return err
	}
	want.Sum, err = mod.HashZip(want.Zip, dirhash.DefaultHash)
	if err != nil {
		return err
	}
	if got.Sum != want.Sum {
		gotFiles, err := zipFiles(got.Zip, m)
		if err != nil {
			return fmt.Errorf("error reading zip 1: %v", err)
		}
		wantFiles, err := zipFiles(want.Zip, m)
		if err != nil {
			return fmt.Errorf("error reading zip 2: %v", err)
		}
		diff := cmp.Diff(wantFiles, gotFiles)
		if diff == "" {
			return BadVersionError{fmt.Sprintf("zip hashes don't match: got %s, want %s", got.Sum, want.Sum)}
		}
		return BadVersionError{fmt.Sprintf("commit has files that can't be included in module (-want +got):\n\n%s", strings.TrimSuffix(diff, "\n"))}
	}

	return nil
}

// zipFiles returns the list of all files in the zip file b.
// It trims the mandatory "{m.Path}@{m.Version}/" prefix from file names.
func zipFiles(b []byte, m module.Version) ([]string, error) {
	z, err := archivezip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, err
	}
	var files []string
	for _, f := range z.File {
		files = append(files, strings.TrimPrefix(f.Name, m.Path+"@"+m.Version+"/"))
	}
	return files, nil
}

// tarFile implements "golang.org/x/mod/zip".File using a tar file.
type tarFile struct {
	path string // Clean '/'-separated relative path.
	fi   os.FileInfo
	b    []byte
}

func (f tarFile) Path() string                 { return f.path }
func (f tarFile) Lstat() (os.FileInfo, error)  { return f.fi, nil }
func (f tarFile) Open() (io.ReadCloser, error) { return ioutil.NopCloser(bytes.NewReader(f.b)), nil }

// verifyHasLICENSE verifies that the commit has a LICENSE file.
func verifyHasLICENSE(r vcs.Repository, commitID vcs.CommitID) error {
	fs, err := r.FileSystem(commitID)
	if err != nil {
		return err
	}
	fi, err := fs.Stat("/LICENSE")
	if os.IsNotExist(err) {
		return BadVersionError{"commit does not have a LICENSE file"}
	} else if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return BadVersionError{"commit has a LICENSE but it's not a regular file"}
	}
	return nil
}

// subject returns the subject of the commit message s.
// The subject is separated from the body by a blank line.
func subject(s string) string {
	i := strings.Index(s, "\n\n")
	if i == -1 {
		return strings.ReplaceAll(s, "\n", " ")
	}
	return strings.ReplaceAll(s[:i], "\n", " ")
}
