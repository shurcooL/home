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
	i := sort.Search(len(*s), func(i int) bool { return (*s)[i].ImportPath >= dir.ImportPath })
	*s = append(*s, nil)
	copy((*s)[i+1:], (*s)[i:])
	(*s)[i] = dir
}

// Rediscover rediscovers all code in the repository store.
// It returns packages that have been added and removed.
func (s *Service) Rediscover() (added, removed []*Directory, err error) {
	// TODO: Can optimize this by rediscovering selectively (only the affected repo and its parent dirs).
	dirs, byImportPath, err := discover(s.reposDir)
	if err != nil {
		return nil, nil, err
	}

	s.mu.Lock()
	oldDirs := s.dirs
	oldByImportPath := s.byImportPath
	s.dirs = dirs
	s.byImportPath = byImportPath
	s.mu.Unlock()

	// Compute added, removed packages.
	for _, d := range dirs {
		if d.Package != nil && !dirExistsAndHasPackage(oldByImportPath[d.ImportPath]) {
			added = append(added, d)
		}
	}
	for _, d := range oldDirs {
		if d.Package != nil && !dirExistsAndHasPackage(byImportPath[d.ImportPath]) {
			removed = append(removed, d)
		}
	}

	return added, removed, nil
}

// dirExistsAndHasPackage reports whether dir exists and contains a Go package.
func dirExistsAndHasPackage(dir *Directory) bool { return dir != nil && dir.Package != nil }

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
