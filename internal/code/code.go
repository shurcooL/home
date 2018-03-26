// Package code implements discovery of Go code within a repository store.
package code

import (
	"bytes"
	"go/build"
	"go/doc"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/shurcooL/go/vfs/godocfs/vfsutil"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/git"
)

// Code includes code that was discovered in a repository store.
type Code struct {
	Sorted       []*Directory
	ByImportPath map[string]*Directory // Key is import path.
}

// Directory represents a directory inside a repository store.
type Directory struct {
	ImportPath string
	RepoRoot   string // Empty string if directory is not in a repository.
	Package    *Package
}

// WithinRepo reports whether directory d is contained by a repository.
func (d Directory) WithinRepo() bool { return d.RepoRoot != "" }

// IsRepoRoot reports whether directory corresponds to a repository root.
func (d Directory) IsRepoRoot() bool { return d.RepoRoot == d.ImportPath }

// Package represents a Go package inside a repository store.
type Package struct {
	Name     string
	Synopsis string // Package documentation synopsis.
	DocHTML  string // Package documentation HTML.
}

func (p Package) IsCommand() bool { return p.Name == "main" }

// Discover discovers all Go code inside the repository store at reposDir.
func Discover(reposDir string) (Code, error) {
	dirs, err := walkRepositoryStore(reposDir)
	if err != nil {
		return Code{}, err
	}
	var byImportPath = make(map[string]*Directory)
	for _, d := range dirs {
		byImportPath[d.ImportPath] = d
	}
	return Code{
		Sorted:       dirs,
		ByImportPath: byImportPath,
	}, nil
}

// walkRepositoryStore walks the repository store at reposDir,
// and returns all Go packages discovered inside, sorted by import path.
func walkRepositoryStore(reposDir string) ([]*Directory, error) {
	var dirs []*Directory
	err := filepath.Walk(reposDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			// We only care about directories.
			return nil
		}
		if strings.HasPrefix(fi.Name(), ".") || strings.HasPrefix(fi.Name(), "_") || fi.Name() == "testdata" {
			return filepath.SkipDir
		}
		ok, err := isBareGitRepository(path)
		if err != nil {
			return err
		} else if !ok {
			// This directory isn't a repository, move on.
			return nil
		}
		ds, err := walkRepository(path, path[len(reposDir)+1:])
		if err != nil {
			return err
		}
		dirs = append(dirs, ds...)
		return filepath.SkipDir
	})
	return dirs, err
}

// isBareGitRepository reports whether there is a bare git repository at dir.
// dir is expected to point to an existing directory.
func isBareGitRepository(dir string) (bool, error) {
	head, err := os.Stat(filepath.Join(dir, "HEAD"))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return !head.IsDir(), nil
}

func walkRepository(gitDir, repoRoot string) ([]*Directory, error) {
	r, err := git.Open(gitDir)
	if err != nil {
		return nil, err
	}
	master, err := r.ResolveBranch("master")
	if err == vcs.ErrBranchNotFound {
		// Empty repository.
		return []*Directory{{
			ImportPath: repoRoot,
			RepoRoot:   repoRoot,
		}}, nil
	} else if err != nil {
		return nil, err
	}
	fs, err := r.FileSystem(master)
	if err != nil {
		return nil, err
	}
	var dirs []*Directory
	err = vfsutil.Walk(fs, "/", func(dir string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			// We only care about directories.
			return nil
		}
		if strings.HasPrefix(fi.Name(), ".") || strings.HasPrefix(fi.Name(), "_") || fi.Name() == "testdata" {
			return filepath.SkipDir
		}
		pkg, err := loadPackage(fs, dir, path.Join(repoRoot, dir))
		if err != nil {
			return err
		}
		dirs = append(dirs, &Directory{
			ImportPath: path.Join(repoRoot, dir),
			RepoRoot:   repoRoot,
			Package:    pkg,
		})
		return nil
	})
	return dirs, err
}

func loadPackage(fs vfs.FileSystem, dir, importPath string) (*Package, error) {
	bctx := build.Context{
		GOOS:        "linux",
		GOARCH:      "amd64",
		CgoEnabled:  true,
		Compiler:    build.Default.Compiler,
		ReleaseTags: build.Default.ReleaseTags,

		JoinPath:      path.Join,
		SplitPathList: splitPathList,
		IsAbsPath:     path.IsAbs,
		IsDir: func(path string) bool {
			fi, err := fs.Stat(path)
			return err == nil && fi.IsDir()
		},
		HasSubdir: hasSubdir,
		ReadDir:   func(dir string) ([]os.FileInfo, error) { return fs.ReadDir(dir) },
		OpenFile:  func(path string) (io.ReadCloser, error) { return fs.Open(path) },
	}
	p, err := bctx.ImportDir(dir, 0)
	if _, ok := err.(*build.NoGoError); ok {
		// This directory doesn't contain a package.
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	// TODO: Automate this.
	doc := p.Doc
	switch importPath {
	case "dmitri.shuralyov.com/text/kebabcase", "dmitri.shuralyov.com/kebabcase":
		doc += "\n\nReference: https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers."
	case "dmitri.shuralyov.com/scratch/image/jpeg":
		doc += "\n\nJPEG is defined in ITU-T T.81: http://www.w3.org/Graphics/JPEG/itu-t81.pdf."
	case "dmitri.shuralyov.com/scratch/image/png":
		doc += "\n\nThe PNG specification is at http://www.w3.org/TR/PNG/."
	case "dmitri.shuralyov.com/font/woff2":
		doc += "\n\nThe WOFF2 font packaging format is specified at https://www.w3.org/TR/WOFF2/."
	}
	return &Package{
		Name:     p.Name,
		Synopsis: p.Doc,
		DocHTML:  docHTML(doc),
	}, nil
}

func splitPathList(list string) []string { return strings.Split(list, ":") }

func hasSubdir(root, dir string) (rel string, ok bool) {
	root = path.Clean(root)
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	dir = path.Clean(dir)
	if !strings.HasPrefix(dir, root) {
		return "", false
	}
	return dir[len(root):], true
}

// docHTML returns documentation comment text converted to formatted HTML.
func docHTML(text string) string {
	var buf bytes.Buffer
	doc.ToHTML(&buf, text, nil)
	return buf.String()
}
