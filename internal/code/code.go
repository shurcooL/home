// Package code implements a Go code service backed by a repository store.
package code

// Service is a Go code service implementation backed by a repository store.
type Service struct {
	dirs         []*Directory          // Sorted.
	byImportPath map[string]*Directory // Key is import path.
}

// NewService discovers Go code inside the repository store at reposDir,
// and returns a code service that uses said repository store.
func NewService(reposDir string) (*Service, error) {
	dirs, byImportPath, err := discover(reposDir)
	if err != nil {
		return nil, err
	}
	return &Service{
		dirs:         dirs,
		byImportPath: byImportPath,
	}, nil
}

// List lists directories in sorted order.
func (s *Service) List() []*Directory {
	return s.dirs
}

// Lookup looks up a directory by specified import path.
// Returned directory is nil if and only if ok is false.
func (s *Service) Lookup(importPath string) (_ *Directory, ok bool) {
	dir, ok := s.byImportPath[importPath]
	return dir, ok
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
