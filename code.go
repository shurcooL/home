package main

import (
	"bytes"
	"go/doc"
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
	notifications notifications.Service
	users         users.Service
	gitUsers      map[string]users.User // Key is lower git author email.
}

func (h *codeHandler) ServeCodeMaybe(w http.ResponseWriter, req *http.Request) (ok bool) {
	// Parse the import path and wantRepo from the URL.
	var (
		importPath string
		wantRepo   bool
	)
	importPathPattern := "dmitri.shuralyov.com" + route.BeforeImportPathSeparator(req.URL.Path)
	if strings.HasSuffix(importPathPattern, "/...") {
		importPath = importPathPattern[:len(importPathPattern)-len("/...")]
		wantRepo = true
	} else if strings.Contains(importPathPattern, "...") {
		// Trailing "/..." is the only supported import path pattern.
		return false
	} else {
		importPath = importPathPattern
	}

	// Look up code directory by import path.
	d, ok := h.code.ByImportPath[importPath]
	if !ok || (wantRepo && !d.IsRepository()) {
		return false
	}

	repo := repoInfo{
		Spec: d.RepoRoot,
		Path: d.RepoRoot[len("dmitri.shuralyov.com"):],
		Dir:  filepath.Join(h.reposDir, filepath.FromSlash(d.RepoRoot)),
	}
	pkgPath := d.ImportPath[len("dmitri.shuralyov.com"):]
	switch {
	case req.URL.Path == route.PkgIndex(pkgPath) && d.Package != nil:
		h := cookieAuth{httputil.ErrorHandler(h.users, (&packageHandler{
			Repo: repo,
			Pkg: pkgInfo{
				Spec:    d.ImportPath,
				Name:    d.Package.Name,
				DocHTML: docHTML(d.Package.Doc),
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
	Spec    string // Package import path. E.g., "example.com/repo/go-package".
	Name    string // Package name. E.g., "package".
	DocHTML string // Package documentation synopsis HTML. E.g., "<p>Package package provides some functionality.</p>".
}

// docHTML returns documentation comment text converted to formatted HTML.
func docHTML(text string) string {
	var buf bytes.Buffer
	doc.ToHTML(&buf, text, nil)
	return buf.String()
}
