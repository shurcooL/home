package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"dmitri.shuralyov.com/html/belt"
	"dmitri.shuralyov.com/service/change"
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/issues"
	issuescomponent "github.com/shurcooL/issuesapp/component"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
)

// commitsHandler is a handler for displaying a list of commits of a git repository.
type commitsHandler struct {
	Repo repoInfo

	issues        issueCounter
	change        changeCounter
	notifications notifications.Service
	users         users.Service
	gitUsers      map[string]users.User // Key is lower git author email.
}

var commitsHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>{{.FullName}} - History</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/commits/style.css" rel="stylesheet" type="text/css">
		<script async src="/assets/commits/commits.js"></script>
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>`))

func (h *commitsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
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

	// TODO: Pagination support.
	cs, err := listMasterCommits(h.Repo, "")
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = commitsHTML.Execute(w, struct {
		Production bool
		FullName   string
	}{
		Production: *productionFlag,
		FullName:   "Repository " + path.Base(h.Repo.Spec),
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	// Render the header.
	header := homecomponent.Header{
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
	err = htmlg.RenderComponents(w, repositoryTabnav(historyTab, h.Repo, openIssues, openChanges))
	if err != nil {
		return err
	}

	var commits []Commit
	for _, c := range cs {
		author, ok := h.gitUsers[strings.ToLower(c.Author.Email)]
		if !ok {
			author = users.User{
				Name:      c.Author.Name,
				Email:     c.Author.Email,
				AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96", // TODO: Use email.
			}
		}

		commits = append(commits, Commit{
			SHA:        string(c.ID),
			Message:    c.Message,
			Author:     author,
			AuthorTime: c.Author.Date.Time(),
		})
	}
	err = htmlg.RenderComponents(w, Commits{
		Commits:    commits,
		ImportPath: h.Repo.Spec,
		CommitURL:  func(sha string) string { return route.RepoCommit(h.Repo.Path) + "/" + sha },
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `</div>
	</body>
</html>`)
	return err
}

// commitsHandlerPkg is a handler for displaying a list of commits of a single package.
type commitsHandlerPkg struct {
	Repo    repoInfo
	PkgPath string
	Dir     *code.Directory

	notifications notifications.Service
	users         users.Service
	gitUsers      map[string]users.User // Key is lower git author email.
}

func (h *commitsHandlerPkg) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
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

	// TODO: Pagination support.
	cs, err := listMasterCommits(h.Repo, directoryGitPathspec(h.Dir))
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var fullName string
	if h.Dir.Package == nil {
		fullName = "Directory " + path.Base(h.Dir.ImportPath)
	} else if h.Dir.Package.IsCommand() {
		fullName = "Command " + path.Base(h.Dir.ImportPath)
	} else {
		fullName = "Package " + h.Dir.Package.Name
	}
	err = commitsHTML.Execute(w, struct {
		Production bool
		FullName   string
	}{
		Production: *productionFlag,
		FullName:   fullName,
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	// Render the header.
	header := homecomponent.Header{
		CurrentUser:       authenticatedUser,
		NotificationCount: nc,
		ReturnURL:         req.RequestURI,
	}
	err = htmlg.RenderComponents(w, header)
	if err != nil {
		return err
	}

	err = html.Render(w, htmlg.H2(htmlg.Text(h.Dir.ImportPath)))
	if err != nil {
		return err
	}

	// Render the tabnav.
	err = htmlg.RenderComponents(w, directoryTabnav(historyTab, h.PkgPath))
	if err != nil {
		return err
	}

	var commits []Commit
	for _, c := range cs {
		author, ok := h.gitUsers[strings.ToLower(c.Author.Email)]
		if !ok {
			author = users.User{
				Name:      c.Author.Name,
				Email:     c.Author.Email,
				AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96", // TODO: Use email.
			}
		}

		commits = append(commits, Commit{
			SHA:        string(c.ID),
			Message:    strings.TrimPrefix(c.Message, pathWithinRepo(h.Dir)+": "), // THINK: Trim package prefix from subject better?
			Author:     author,
			AuthorTime: c.Author.Date.Time(),
		})
	}
	err = htmlg.RenderComponents(w, Commits{
		Commits:    commits,
		ImportPath: h.Dir.ImportPath,
		CommitURL:  func(sha string) string { return route.PkgCommit(h.PkgPath) + "/" + sha },
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `</div>
	</body>
</html>`)
	return err
}

// listMasterCommits returns a list of commits in git repo on master branch,
// with an optionally specified pathspec.
// If master branch doesn't exist, an empty list is returned.
func listMasterCommits(repo repoInfo, pathspec string) ([]*vcs.Commit, error) {
	r := &gitcmd.Repository{Dir: repo.Dir}
	defer r.Close()
	master, err := r.ResolveBranch("master")
	if err == vcs.ErrBranchNotFound {
		// No master branch means there are no commits on it (e.g., an empty repository).
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	cs, _, err := r.Commits(vcs.CommitsOptions{
		Head:    master,
		Path:    pathspec,
		NoTotal: true,
	})
	return cs, err
}

type Commits struct {
	Commits    []Commit
	ImportPath string
	CommitURL  func(sha string) string
}

func (cs Commits) Render() []*html.Node {
	if len(cs.Commits) == 0 {
		// No commits. Let the user know via a blank slate.
		return homecomponent.BlankSlate{
			Content: htmlg.Nodes{htmlg.Text("There are no commits.")},
		}.Render()
	}

	var nodes []*html.Node
	for _, c := range cs.Commits {
		nodes = append(nodes, c.Render(cs.ImportPath, cs.CommitURL)...)
	}
	return []*html.Node{htmlg.DivClass("list-entry-border", nodes...)}
}

type Commit struct {
	SHA        string
	Message    string
	Author     users.User
	AuthorTime time.Time
}

func (c Commit) Render(importPath string, commitURL func(sha string) string) []*html.Node {
	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "display: flex;"}},
	}

	avatarDiv := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "margin-right: 6px;"}},
	}
	htmlg.AppendChildren(avatarDiv, issuescomponent.Avatar{User: c.Author, Size: 32}.Render()...)
	div.AppendChild(avatarDiv)

	titleAndByline := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "flex-grow: 1;"}},
	}
	{
		commitSubject, commitBody := splitCommitMessage(c.Message)

		title := htmlg.Div(
			&html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "black"},
					{Key: atom.Href.String(), Val: commitURL(c.SHA)},
				},
				FirstChild: htmlg.Strong(commitSubject),
			},
		)
		if commitBody != "" {
			htmlg.AppendChildren(title, homecomponent.EllipsisButton{OnClick: "ToggleDetails(this);"}.Render()...)
		}
		titleAndByline.AppendChild(title)

		byline := htmlg.DivClass("gray tiny")
		byline.Attr = append(byline.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 2px;"})
		htmlg.AppendChildren(byline, issuescomponent.User{User: c.Author}.Render()...)
		byline.AppendChild(htmlg.Text(" committed "))
		htmlg.AppendChildren(byline, issuescomponent.Time{Time: c.AuthorTime}.Render()...)
		titleAndByline.AppendChild(byline)

		if commitBody != "" {
			pre := &html.Node{
				Type: html.ElementNode, Data: atom.Pre.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "commit-details"},
					{Key: atom.Style.String(), Val: `font-size: 13px;
font-family: Go;
color: #444;
margin-top: 10px;
margin-bottom: 0;
display: none;`}},
				FirstChild: htmlg.Text(commitBody),
			}
			titleAndByline.AppendChild(pre)
		}
	}
	div.AppendChild(titleAndByline)

	commitID := belt.CommitID{SHA: c.SHA}
	htmlg.AppendChildren(div, commitID.Render()...)

	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "lightgray"},
			{Key: atom.Style.String(), Val: "height: 16px; margin-left: 12px;"},
			{Key: atom.Href.String(), Val: "https://gotools.org/" + importPath + "?rev=" + c.SHA},
			{Key: atom.Title.String(), Val: "View code at this revision."},
		},
		FirstChild: octicon.Code(),
	}
	div.AppendChild(a)

	listEntryDiv := htmlg.DivClass("list-entry-body multilist-entry commit-container", div)
	return []*html.Node{listEntryDiv}
}

// splitCommitMessage splits commit message s into subject and body, if any.
func splitCommitMessage(s string) (subject, body string) {
	i := strings.Index(s, "\n\n")
	if i == -1 {
		return s, ""
	}
	return s[:i], s[i+2:]
}
