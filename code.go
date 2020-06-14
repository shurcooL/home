package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/users"
	"golang.org/x/tools/godoc/vfs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/git"
)

type codeHandler struct {
	code         *code.Service
	reposDir     string
	issuesApp    httperror.Handler
	changesApp   httperror.Handler
	issues       issueCounter
	change       changeCounter
	notification notification.Service
	users        users.Service
	gitUsers     map[string]users.User // Key is lower git author email.
}

func (h *codeHandler) ServeCodeMaybe(w http.ResponseWriter, req *http.Request) (ok bool) {
	// Parse the import path and wantRepoRoot from the URL.
	var (
		importPath   string
		wantRepoRoot bool
	)
	importPathPattern := "dmitri.shuralyov.com" + route.BeforeImportPathSeparator(req.URL.Path)
	if strings.HasSuffix(importPathPattern, "/...") && !strings.Contains(importPathPattern[:len(importPathPattern)-len("/...")], "...") {
		importPath = importPathPattern[:len(importPathPattern)-len("/...")]
		wantRepoRoot = true
	} else if strings.Contains(importPathPattern, "...") {
		// Trailing "/..." is the only supported import path pattern.
		return false
	} else {
		importPath = importPathPattern
	}

	// Look up code directory by import path.
	d, err := h.code.GetDirectory(req.Context(), importPath)
	if err != nil || !d.WithinRepo() || (wantRepoRoot && !d.IsRepoRoot()) {
		return false
	}

	repo := repoInfo{
		Spec:     d.RepoRoot,
		Path:     d.RepoRoot[len("dmitri.shuralyov.com"):],
		Dir:      filepath.Join(h.reposDir, filepath.FromSlash(d.RepoRoot)),
		Packages: d.RepoPackages,
	}
	pkgPath := d.ImportPath[len("dmitri.shuralyov.com"):]
	var licensePkgPath string
	if d.LicenseRoot != "" {
		licensePkgPath = d.LicenseRoot[len("dmitri.shuralyov.com"):]
	}
	switch {
	case req.URL.Path == route.PkgIndex(pkgPath):
		// Handle ?go-get=1 requests, serve a go-import meta tag page.
		if req.Method == http.MethodGet && req.URL.Query().Get("go-get") == "1" {
			metrics.IncGoGetRequestsTotal(d.ImportPath)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, `<meta name="go-import" content="%[1]s git https://%[1]s">
<meta name="go-import" content="%[1]s mod https://dmitri.shuralyov.com/api/module">
<meta name="go-source" content="%[1]s https://%[1]s https://gotools.org/%[2]s https://gotools.org/%[2]s#{file}-L{line}">`, d.RepoRoot, d.ImportPath)
			return true
		}

		// If there's no Go package in this directory, redirect to "{ImportPath}/..." package listing.
		if d.Package == nil {
			u := *req.URL
			u.Path += "/..."
			http.Redirect(w, req, u.String(), http.StatusSeeOther)
			return true
		}

		// Serve Go package index page.
		licenseURL := "/LICENSE" // Default license URL.
		if licensePkgPath != "" {
			// A more specific license override.
			licenseURL = route.PkgLicense(licensePkgPath)
		}
		h := cookieAuth{httputil.ErrorHandler(h.users, (&packageHandler{
			Repo: repo,
			Pkg: pkgInfo{
				Spec:       d.ImportPath,
				Name:       d.Package.Name,
				DocHTML:    d.Package.DocHTML,
				LicenseURL: licenseURL,
			},
			issues:       h.issues,
			change:       h.change,
			notification: h.notification,
			users:        h.users,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.PkgLicense(pkgPath):
		if !d.HasLicenseFile() {
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return true
		}
		license, err := readLicenseFile(repo.Dir, d)
		if err != nil {
			log.Println("readLicenseFile:", err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError) // TODO: Display full error to site admins.
			return true
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		httpgzip.ServeContent(w, req, "", time.Time{}, bytes.NewReader(license))
		return true
	case req.URL.Path == route.PkgHistory(pkgPath):
		h := cookieAuth{httputil.ErrorHandler(h.users, (&commitsHandlerPkg{
			Repo:         repo,
			PkgPath:      pkgPath,
			Dir:          d,
			notification: h.notification,
			users:        h.users,
			gitUsers:     h.gitUsers,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case strings.HasPrefix(req.URL.Path, route.PkgCommit(pkgPath)+"/"):
		req = stripPrefix(req, len(route.PkgCommit(pkgPath)))
		h := cookieAuth{httputil.ErrorHandler(h.users, (&commitHandlerPkg{
			Repo:         repo,
			PkgPath:      pkgPath,
			Dir:          d,
			notification: h.notification,
			users:        h.users,
			gitUsers:     h.gitUsers,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.RepoIndex(repo.Path):
		h := cookieAuth{httputil.ErrorHandler(h.users, (&repositoryHandler{
			Repo:         repo,
			code:         h.code,
			issues:       h.issues,
			change:       h.change,
			notification: h.notification,
			users:        h.users,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.RepoHistory(repo.Path):
		h := cookieAuth{httputil.ErrorHandler(h.users, (&commitsHandler{
			Repo:         repo,
			issues:       h.issues,
			change:       h.change,
			notification: h.notification,
			users:        h.users,
			gitUsers:     h.gitUsers,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case strings.HasPrefix(req.URL.Path, route.RepoCommit(repo.Path)+"/"):
		req = stripPrefix(req, len(route.RepoCommit(repo.Path)))
		h := cookieAuth{httputil.ErrorHandler(h.users, (&commitHandler{
			Repo:         repo,
			issues:       h.issues,
			change:       h.change,
			notification: h.notification,
			users:        h.users,
			gitUsers:     h.gitUsers,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.RepoIssues(repo.Path) ||
		strings.HasPrefix(req.URL.Path, route.RepoIssues(repo.Path)+"/"):

		h := cookieAuth{httputil.ErrorHandler(h.users, h.issuesApp.ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.RepoChanges(repo.Path) ||
		strings.HasPrefix(req.URL.Path, route.RepoChanges(repo.Path)+"/"):

		h := cookieAuth{httputil.ErrorHandler(h.users, h.changesApp.ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	default:
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return true
	}
}

type repoInfo struct {
	Spec     string // Repository spec. E.g., "example.com/repo".
	Path     string // Path corresponding to repository root, without domain. E.g., "/repo".
	Dir      string // Path to repository directory on disk.
	Packages int    // Number of packages contained by repository.
}

// repoInfoContextKey is a context key for the request's repo info.
// That value specifies which repo is being displayed.
// The associated value will be of type repoInfo.
var repoInfoContextKey = &contextKey{"RepoInfo"}

type pkgInfo struct {
	Spec       string // Package import path. E.g., "example.com/repo/package".
	Name       string // Package name. E.g., "pkg".
	DocHTML    string // Package documentation HTML. E.g., "<p>Package pkg provides some functionality.</p><p>More information about pkg.</p>".
	LicenseURL string // URL of license. E.g., "/repo/package$file/LICENSE".
}

// IsCommand reports whether the package is a command.
func (p pkgInfo) IsCommand() bool { return p.Name == "main" }

func readLicenseFile(gitDir string, d *code.Directory) ([]byte, error) {
	r, err := git.Open(gitDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := r.Close()
		if err != nil {
			log.Println("readLicenseFile: r.Close:", err)
		}
	}()
	master, err := r.ResolveBranch("master")
	if err != nil {
		return nil, err
	}
	fs, err := r.FileSystem(master)
	if err != nil {
		return nil, err
	}
	license, err := vfs.ReadFile(fs, path.Join("/", strings.TrimPrefix(d.ImportPath, d.RepoRoot), "LICENSE"))
	return license, err
}
