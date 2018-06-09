// Package code implements discovery of Go code within a repository store.
package code

import (
	"bytes"
	"go/build"
	"go/doc"
	"io"
	"log"
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
	ImportPath   string
	RepoRoot     string // Empty string if directory is not in a repository.
	RepoPackages int    // Number of packages contained by repository (if any, otherwise 0).

	// LicenseRoot is the import path corresponding to this or nearest parent directory
	// that contains a LICENSE file, or empty string if there isn't such a directory.
	LicenseRoot string

	Package *Package
}

// WithinRepo reports whether directory d is contained by a repository.
func (d Directory) WithinRepo() bool { return d.RepoRoot != "" }

// IsRepoRoot reports whether directory d corresponds to a repository root.
func (d Directory) IsRepoRoot() bool { return d.RepoRoot == d.ImportPath }

// HasLicenseFile reports whether directory d contains a LICENSE file.
func (d Directory) HasLicenseFile() bool { return d.LicenseRoot == d.ImportPath }

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

	// Populate LicenseRoot values for all remaining directories
	// that don't directly contain a LICENSE file.
	for _, dir := range dirs {
		if dir.HasLicenseFile() {
			continue
		}
		elems := strings.Split(dir.ImportPath, "/")
		for i := len(elems) - 1; i >= 1; i-- { // Start from parent directory and traverse up.
			p, ok := byImportPath[path.Join(elems[:i]...)]
			if ok && p.HasLicenseFile() {
				dir.LicenseRoot = p.ImportPath
				break
			}
		}
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
	defer func() {
		err := r.Close()
		if err != nil {
			log.Println("walkRepository: r.Close:", err)
		}
	}()
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
	var (
		dirs         []*Directory
		repoPackages int
	)
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
		importPath := path.Join(repoRoot, dir)
		var licenseRoot string
		if ok, err := hasLicenseFile(fs, dir); err == nil && ok {
			licenseRoot = importPath
		} else if err != nil {
			return err
		}
		pkg, err := loadPackage(fs, dir, importPath)
		if err != nil {
			return err
		}
		if pkg != nil {
			repoPackages++
		}
		dirs = append(dirs, &Directory{
			ImportPath:  importPath,
			RepoRoot:    repoRoot,
			LicenseRoot: licenseRoot,
			Package:     pkg,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		d.RepoPackages = repoPackages
	}
	return dirs, nil
}

// loadPackage loads a Go package with import path importPath
// from filesystem fs in directory dir.
// It returns a nil Package if the directory doesn't contain a Go package.
func loadPackage(fs vfs.FileSystem, dir, importPath string) (*Package, error) {
	for _, env := range [...]struct{ GOOS, GOARCH string }{
		{"linux", "amd64"},
		{"darwin", "amd64"},
	} {
		bctx := build.Context{
			GOOS:        env.GOOS,
			GOARCH:      env.GOARCH,
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
			if buildConstraintsExcludeAll(p) {
				// Try again with a different environment.
				continue
			}
			// This directory doesn't contain a package.
			break
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
		case "dmitri.shuralyov.com/gpu/mtl":
			doc += "\n\nThis package is in very early stages of development. The API will change when opportunities for improvement are discovered; it is not yet frozen. Less than 20% of the Metal API surface is implemented. Current functionality is sufficient to render very basic geometry."
		case "dmitri.shuralyov.com/gpu/mtl/example/hellotriangle":
			doc += " It writes the frame to a triangle.png file in current working directory."
		}
		return &Package{
			Name:     p.Name,
			Synopsis: p.Doc,
			DocHTML:  docHTML(doc),
		}, nil
	}
	// This directory doesn't contain a package.
	return nil, nil
}

// buildConstraintsExcludeAll reports whether Go files exist in p,
// but they were ignored due to build constraints.
func buildConstraintsExcludeAll(p *build.Package) bool {
	// Count files beginning with _ and ., which we will pretend don't exist at all.
	dummy := 0
	for _, name := range p.IgnoredGoFiles {
		if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
			dummy++
		}
	}
	return len(p.IgnoredGoFiles) > dummy
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

func hasLicenseFile(fs vfs.FileSystem, dir string) (bool, error) {
	fi, err := fs.Stat(path.Join(dir, "LICENSE"))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return !fi.IsDir(), nil
}
