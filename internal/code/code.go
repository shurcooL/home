// Package code implements a Go code service backed by a repository store.
package code

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/shurcooL/events"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

// Service is a Go code service implementation backed by a repository store.
type Service struct {
	reposDir string

	mu           sync.RWMutex
	dirs         []*Directory          // Sorted.
	byImportPath map[string]*Directory // Key is import path.

	notifications notifications.ExternalService
	events        events.ExternalService
	users         users.Service
}

// NewService discovers Go code inside the repository store at reposDir,
// and returns a code service that uses said repository store.
func NewService(reposDir string, notifications notifications.ExternalService, events events.ExternalService, users users.Service) (*Service, error) {
	dirs, byImportPath, err := discover(reposDir)
	if err != nil {
		return nil, err
	}
	return &Service{
		reposDir: reposDir,

		dirs:         dirs,
		byImportPath: byImportPath,

		notifications: notifications,
		events:        events,
		users:         users,
	}, nil
}

// List lists directories in sorted order.
func (s *Service) List() []*Directory {
	s.mu.RLock()
	dirs := s.dirs
	s.mu.RUnlock()
	return dirs
}

// Lookup looks up a directory by specified import path.
// Returned directory is nil if and only if ok is false.
func (s *Service) Lookup(importPath string) (_ *Directory, ok bool) {
	s.mu.RLock()
	dir, ok := s.byImportPath[importPath]
	s.mu.RUnlock()
	return dir, ok
}

// CreateRepo creates an empty repository with the specified repoSpec and description.
// If the directory already exists, os.ErrExist is returned.
func (s *Service) CreateRepo(ctx context.Context, repoSpec, description string) error {
	currentUser, err := s.users.GetAuthenticated(ctx)
	if err != nil {
		return err
	}

	// Authorization check.
	if !currentUser.SiteAdmin {
		return os.ErrPermission
	}

	s.mu.RLock()
	_, ok := s.byImportPath[repoSpec]
	s.mu.RUnlock()
	if ok {
		return os.ErrExist
	}

	// Create bare git repo.
	cmd := exec.Command("git", "init", "--bare", filepath.Join(s.reposDir, filepath.FromSlash(repoSpec)))
	err = cmd.Run()
	if err != nil {
		return err
	}

	// Empty repository.
	dir := &Directory{
		ImportPath: repoSpec,
		RepoRoot:   repoSpec,
	}
	s.mu.Lock()
	insertDir(&s.dirs, dir)
	s.byImportPath[repoSpec] = dir
	s.mu.Unlock()

	// Watch the newly created repository.
	err = s.notifications.Subscribe(ctx, notifications.RepoSpec{URI: repoSpec}, "", 0, []users.UserSpec{currentUser.UserSpec})
	if err != nil {
		return err
	}

	// Log a "created repository" event.
	err = s.events.Log(ctx, event.Event{
		Time:      time.Now().UTC(),
		Actor:     currentUser,
		Container: repoSpec,
		Payload: event.Create{
			Type:        "repository",
			Description: description,
		},
	})
	return err
}

// insertDir inserts directory dir into the sorted slice s,
// keeping the slice sorted. s must not already contain dir.
func insertDir(s *[]*Directory, dir *Directory) {
	// Use binary search to find index where dir should be inserted,
	// and insert it directly there.
	i := sort.Search(len(*s), func(i int) bool { return (*s)[i].RepoRoot >= dir.RepoRoot })
	*s = append(*s, nil)
	copy((*s)[i+1:], (*s)[i:])
	(*s)[i] = dir
}

// Rediscover rediscovers code in repoRoot of the repository store.
// It returns packages that have been added and removed.
func (s *Service) Rediscover(repoRoot string) (added, removed []*Directory, err error) {
	gitDir := filepath.Join(s.reposDir, filepath.FromSlash(repoRoot))
	newDirs, err := walkRepository(gitDir, repoRoot)
	if err != nil {
		return nil, nil, err
	}

	s.mu.Lock()
	oldDirs := replaceDirs(&s.dirs, repoRoot, newDirs)
	replaceDirsMap(s.byImportPath, oldDirs, newDirs)
	populateLicenseRoot(newDirs, s.byImportPath)
	s.mu.Unlock()

	// Compute added, removed packages.
	for _, d := range newDirs {
		if d.Package == nil || containsPackage(oldDirs, d.ImportPath) {
			continue
		}
		added = append(added, d)
	}
	for _, d := range oldDirs {
		if d.Package == nil || containsPackage(newDirs, d.ImportPath) {
			continue
		}
		removed = append(removed, d)
	}

	return added, removed, nil
}

// replaceDirs replaces directories with repoRoot in the sorted slice s
// with newDirs, keeping the slice sorted. It returns old directories that got replaced.
func replaceDirs(s *[]*Directory, repoRoot string, newDirs []*Directory) (oldDirs []*Directory) {
	// Use binary search to find index where directories should be replaced,
	// and replace them directly there.
	// i is the start index of old directories, and j is the end index.
	i := sort.Search(len(*s), func(i int) bool { return (*s)[i].RepoRoot >= repoRoot })
	old := 0 // Number of old directories to replace.
	for i+old < len(*s) && (*s)[i+old].RepoRoot == repoRoot {
		old++
	}
	j := i + old

	// Make a copy of old directories before they're overwritten.
	oldDirs = make([]*Directory, old)
	copy(oldDirs, (*s)[i:j])

	// Grow/shrink the slice by delta, and copy new directories into place.
	switch delta := len(newDirs) - len(oldDirs); {
	case delta > 0:
		// Grow s by delta.
		*s = append(*s, make([]*Directory, delta)...)
		copy((*s)[j+delta:], (*s)[j:])
	case delta < 0:
		// Shrink s by delta.
		copy((*s)[j+delta:], (*s)[j:])
		copy((*s)[len(*s)+delta:], make([]*Directory, -delta))
		*s = (*s)[:len(*s)+delta]
	}
	copy((*s)[i:], newDirs)

	return oldDirs
}

func replaceDirsMap(m map[string]*Directory, oldDirs, newDirs []*Directory) {
	for _, d := range oldDirs {
		delete(m, d.ImportPath)
	}
	for _, d := range newDirs {
		m[d.ImportPath] = d
	}
}

// containsPackage reports whether dirs contains a Go package with matching importPath.
func containsPackage(dirs []*Directory, importPath string) bool {
	for _, d := range dirs {
		if d.ImportPath != importPath {
			continue
		}
		return d.Package != nil
	}
	return false
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

// IsCommand reports whether the package is a command.
func (p Package) IsCommand() bool { return p.Name == "main" }
