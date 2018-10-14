package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"time"

	"dmitri.shuralyov.com/service/change"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octicon"
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

	code          *code.Service
	issues        issueCounter
	change        changeCounter
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

	t0 := time.Now()
	openIssues, err := h.issues.Count(req.Context(), issues.RepoSpec{URI: h.Repo.Spec}, issues.IssueListOptions{State: issues.StateFilter(issues.OpenState)})
	if err != nil {
		return err
	}
	openChanges, err := h.change.Count(req.Context(), h.Repo.Spec, change.ListOptions{Filter: change.FilterOpen})
	if err != nil {
		return err
	}
	fmt.Println("counting open issues & changes took:", time.Since(t0).Nanoseconds(), "for:", h.Repo.Spec)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = repositoryHTML.Execute(w, struct {
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
	err = htmlg.RenderComponents(w, repositoryTabnav(packagesTab, h.Repo, openIssues, openChanges))
	if err != nil {
		return err
	}

	err = renderPackages(w, expandPattern(h.code.List(), nil, h.Repo.Spec+"/...")) // repositoryHandler is used only for self-hosted packages, so it's okay to leave out githubPackages when expanding pattern.
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

func repositoryTabnav(selected repositoryTab, repo repoInfo, openIssues, openChanges uint64) htmlg.Component {
	return tabnav{
		Tabs: []tab{
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.Package, Text: "Packages"},
					Count:   repo.Packages,
				},
				URL:      route.RepoIndex(repo.Path),
				Selected: selected == packagesTab,
			},
			{
				Content:  iconText{Icon: octicon.History, Text: "History"},
				URL:      route.RepoHistory(repo.Path),
				Selected: selected == historyTab,
			},
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.IssueOpened, Text: "Issues"},
					Count:   int(openIssues),
				},
				URL:      route.RepoIssues(repo.Path),
				Selected: selected == issuesTab,
			},
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.GitPullRequest, Text: "Changes"},
					Count:   int(openChanges),
				},
				URL:      route.RepoChanges(repo.Path),
				Selected: selected == changesTab,
			},
		},
	}
}

type repositoryTab uint8

const (
	noTab repositoryTab = iota
	packagesTab
	historyTab
	issuesTab
	changesTab
)
