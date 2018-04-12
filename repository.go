package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"path"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
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

	code          code.Code
	notifications notifications.Service
	users         users.Service
}

var repositoryHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Repository {{.Name}} - Packages</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/repository/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>`))

func (h *repositoryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := repositoryHTML.Execute(w, struct {
		Production bool
		Name       string
	}{
		Production: *productionFlag,
		Name:       path.Base(h.Repo.Spec),
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	authenticatedUser, err := h.users.GetAuthenticated(req.Context())
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}
	var nc uint64
	if authenticatedUser.ID != 0 {
		nc, err = h.notifications.Count(req.Context(), nil)
		if err != nil {
			return err
		}
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
	err = htmlg.RenderComponents(w, tabnav{
		Tabs: []tab{
			{
				Content:  iconText{Icon: octiconssvg.Package, Text: "Packages"},
				URL:      route.RepoIndex(h.Repo.Path),
				Selected: true,
			},
			{
				Content: iconText{Icon: octiconssvg.History, Text: "History"},
				URL:     route.RepoHistory(h.Repo.Path),
			},
			{
				Content: iconText{Icon: octiconssvg.IssueOpened, Text: "Issues"},
				URL:     route.RepoIssues(h.Repo.Path),
			},
			{
				Content: iconText{Icon: octiconssvg.GitPullRequest, Text: "Changes"},
				URL:     route.RepoChanges(h.Repo.Path),
			},
		},
	})
	if err != nil {
		return err
	}

	err = renderPackages(w, expandPattern(h.code.Sorted, nil, h.Repo.Spec+"/...")) // repositoryHandler is used only for self-hosted packages, so it's okay to leave out githubPackages when expanding pattern.
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
