package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

type codeHandler struct {
	code          code.Code
	reposDir      string
	issuesApp     http.Handler
	changesApp    http.Handler
	notifications notifications.Service
	users         users.Service
	gitUsers      map[string]users.User // Key is lower git author email.
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
	d, ok := h.code.ByImportPath[importPath]
	if !ok || !d.WithinRepo() || (wantRepoRoot && !d.IsRepoRoot()) {
		return false
	}

	repo := repoInfo{
		Spec: d.RepoRoot,
		Path: d.RepoRoot[len("dmitri.shuralyov.com"):],
		Dir:  filepath.Join(h.reposDir, filepath.FromSlash(d.RepoRoot)),
	}
	pkgPath := d.ImportPath[len("dmitri.shuralyov.com"):]
	switch {
	case req.URL.Path == route.PkgIndex(pkgPath):
		// Handle ?go-get=1 requests, serve a go-import meta tag page.
		if req.Method == http.MethodGet && req.URL.Query().Get("go-get") == "1" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<meta name="go-import" content="%[1]s git https://%[1]s">
	<meta name="go-source" content="%[1]s https://%[1]s https://gotools.org/%[2]s https://gotools.org/%[2]s#{file}-L{line}">`, d.RepoRoot, d.ImportPath)
			return true
		}

		// If there's no Go package in this directory, redirect to "{ImportPath}/..." package listing.
		if d.Package == nil {
			u := *req.URL
			u.Path += "/..."
			if req.Method == http.MethodGet { // Workaround for https://groups.google.com/forum/#!topic/golang-nuts/9AVyMP9C8Ac.
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			}
			http.Redirect(w, req, u.String(), http.StatusSeeOther)
			return true
		}

		// Serve Go package index page.
		h := cookieAuth{httputil.ErrorHandler(h.users, (&packageHandler{
			Repo: repo,
			Pkg: pkgInfo{
				Spec:    d.ImportPath,
				Name:    d.Package.Name,
				DocHTML: d.Package.DocHTML,
			},
			notifications: h.notifications,
			users:         h.users,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.RepoIndex(repo.Path):
		h := cookieAuth{httputil.ErrorHandler(h.users, (&repositoryHandler{
			Repo:          repo,
			code:          h.code,
			notifications: h.notifications,
			users:         h.users,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.RepoHistory(repo.Path):
		h := cookieAuth{httputil.ErrorHandler(h.users, (&commitsHandler{
			Repo:          repo,
			notifications: h.notifications,
			users:         h.users,
			gitUsers:      h.gitUsers,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case strings.HasPrefix(req.URL.Path, route.RepoCommit(repo.Path)+"/"):
		req = stripPrefix(req, len(route.RepoCommit(repo.Path)))
		h := cookieAuth{httputil.ErrorHandler(h.users, (&commitHandler{
			Repo:          repo,
			notifications: h.notifications,
			users:         h.users,
			gitUsers:      h.gitUsers,
		}).ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.RepoIssues(repo.Path) ||
		strings.HasPrefix(req.URL.Path, route.RepoIssues(repo.Path)+"/"):

		h := cookieAuth{httputil.ErrorHandler(h.users, issuesHandler{
			SpecURL:   repo.Spec, // Issues trackers are mapped to repo roots at this time.
			BaseURL:   route.RepoIssues(repo.Path),
			issuesApp: h.issuesApp,
		}.ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	case req.URL.Path == route.RepoChanges(repo.Path) ||
		strings.HasPrefix(req.URL.Path, route.RepoChanges(repo.Path)+"/"):

		h := cookieAuth{httputil.ErrorHandler(h.users, changesHandler{
			SpecURL:    repo.Spec, // Change trackers are mapped to repo roots at this time.
			BaseURL:    route.RepoChanges(repo.Path),
			changesApp: h.changesApp,
		}.ServeHTTP)}
		h.ServeHTTP(w, req)
		return true
	default:
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return true
	}
}

type repoInfo struct {
	Spec string // Repository spec. E.g., "example.com/repo".
	Path string // Path corresponding to repository root, without domain. E.g., "/repo".
	Dir  string // Path to repository directory on disk.
}

type pkgInfo struct {
	Spec    string // Package import path. E.g., "example.com/repo/package".
	Name    string // Package name. E.g., "pkg".
	DocHTML string // Package documentation HTML. E.g., "<p>Package pkg provides some functionality.</p><p>More information about pkg.</p>".
}
