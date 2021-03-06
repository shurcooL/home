package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"time"

	statepkg "dmitri.shuralyov.com/state"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/exp/service/change"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
)

// repositoryHandler is a handler for a Go repository index page.
//
// It's very similar to the packages page, except it only applies to
// import path patterns like "example.com/repo/..." where example.com/repo
// is an existing repository root.
type repositoryHandler struct {
	Repo repoInfo

	code         *code.Service
	issues       issueCounter
	change       changeCounter
	notification notification.Service
	users        users.Service
}

var repositoryHTML = template.Must(template.New("").Parse(`<html>
	<head>
{{.AnalyticsHTML}}		<title>Repository {{.Name}} - Packages</title>
		<link href="/icon.svg" rel="icon" type="image/svg+xml">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/repository/style.css" rel="stylesheet" type="text/css">
	</head>
	<body>`))

func (h *repositoryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if err := httputil.AllowMethods(req, http.MethodGet, http.MethodHead); err != nil {
		return err
	}

	authenticatedUser, err := h.users.GetAuthenticated(req.Context())
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}
	var nc uint64
	if authenticatedUser.ID != 0 {
		nc, err = h.notification.CountNotifications(req.Context())
		if err != nil {
			return err
		}
	}

	t0 := time.Now()
	openIssues, err := h.issues.Count(req.Context(), issues.RepoSpec{URI: h.Repo.Spec}, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueOpen)})
	if err != nil {
		return err
	}
	openChanges, err := h.change.Count(req.Context(), h.Repo.Spec, change.ListOptions{Filter: change.FilterOpen})
	if err != nil {
		return err
	}
	fmt.Println("counting open issues & changes took:", time.Since(t0).Nanoseconds(), "for:", h.Repo.Spec)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if req.Method == http.MethodHead {
		return nil
	}
	err = repositoryHTML.Execute(w, struct {
		AnalyticsHTML template.HTML
		Name          string
	}{
		AnalyticsHTML: analyticsHTML,
		Name:          path.Base(h.Repo.Spec),
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	// Render the header.
	header := component.Header{
		CurrentUser:       authenticatedUser,
		NotificationCount: nc,
		ReturnURL:         req.RequestURI,
	}
	err = htmlg.RenderComponents(w, header)
	if err != nil {
		return err
	}

	err = html.Render(w, htmlg.H2(htmlg.Text(h.Repo.Spec+"/...")))
	if err != nil {
		return err
	}

	// Render the tabnav.
	err = htmlg.RenderComponents(w, component.RepositoryTabNav(component.PackagesTab, h.Repo.Path, h.Repo.Packages, openIssues, openChanges))
	if err != nil {
		return err
	}

	dirs, err := h.code.ListDirectories(req.Context())
	if err != nil {
		return err
	}
	err = renderPackages(w, expandPattern(dirs, nil, h.Repo.Spec+"/...")) // repositoryHandler is used only for self-hosted packages, so it's okay to leave out githubPackages when expanding pattern.
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `</div>`)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `</body></html>`)
	return err
}
