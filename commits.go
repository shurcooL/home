package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/shurcooL/go/timeutil"
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	issuescomponent "github.com/shurcooL/issuesapp/component"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
)

// commitsHandler is a handler for displaying a list of commits of a git repository.
//
// Currently, it is hardcoded for dmitri.shuralyov.com/kebabcase repo,
// and returns an error if Repo != "dmitri.shuralyov.com/kebabcase".
type commitsHandler struct {
	Repo          string // Repo URI, e.g., "example.com/some/package".
	RepoDir       string // Path to repository directory on disk.
	notifications notifications.Service
	users         users.Service
	gitUsers      map[string]users.User // Key is lower git author email.
}

var commitsHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Package {{.Name}} - History</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/commits/style.css" rel="stylesheet" type="text/css">
		<script async src="/assets/commits/commits.js"></script>
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>`))

func (h *commitsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if h.Repo != "dmitri.shuralyov.com/kebabcase" {
		return fmt.Errorf("wrong repo")
	}
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
	r := &gitcmd.Repository{Dir: h.RepoDir}
	cs, _, err := r.Commits(vcs.CommitsOptions{
		Head:    vcs.CommitID("master"),
		NoTotal: true,
	})
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = commitsHTML.Execute(w, struct {
		Production bool
		Name       string
	}{
		Production: *productionFlag,
		Name:       "kebabcase",
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

	_, err = io.WriteString(w, `<h2>Package kebabcase</h2>`)
	if err != nil {
		return err
	}

	// Render the tabnav.
	err = htmlg.RenderComponents(w, tabnav{
		Tabs: []tab{
			{
				Content: iconText{Icon: octiconssvg.Book, Text: "Overview"},
				URL:     "/kebabcase",
			},
			{
				Content:  iconText{Icon: octiconssvg.History, Text: "History"},
				URL:      "/kebabcase/commits",
				Selected: true,
			},
			{
				Content: iconText{Icon: octiconssvg.IssueOpened, Text: "Issues"},
				URL:     "/kebabcase/issues",
			},
		},
	})
	if err != nil {
		return err
	}

	// TODO: Connect to real branches/tags data, add frontend logic, etc.
	_, err = fmt.Fprintf(w, `<div style="margin: 14px 0;" title="Branch"><span style="display: inline-block; vertical-align: middle; margin-right: 6px;">%s</span><select><option selected>master</option></select></div>`,
		htmlg.Render(octiconssvg.GitBranch()))
	if err != nil {
		return err
	}

	var commits []Commit
	for _, c := range cs {
		user, ok := h.gitUsers[strings.ToLower(c.Author.Email)]
		if !ok {
			user = users.User{
				Name:      c.Author.Name,
				Email:     c.Author.Email,
				AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96", // TODO: Use email.
			}
		}

		commits = append(commits, Commit{Commit: c, User: user})
	}
	err = htmlg.RenderComponents(w, Commits{Commits: commits})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `</div>
	</body>
</html>`)
	return err
}

type Commits struct {
	Commits []Commit
}

func (cs Commits) Render() []*html.Node {
	if len(cs.Commits) == 0 {
		// No commits. Let the user know via a blank slate.
		div := &html.Node{
			Type: html.ElementNode, Data: atom.Div.String(),
			Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "text-align: center; margin-top: 80px; margin-bottom: 80px;"}},
			FirstChild: htmlg.Text("There are no commits."),
		}
		return []*html.Node{htmlg.DivClass("list-entry-border", div)}
	}

	var (
		now      = time.Now()
		headings = []struct {
			Text string
			End  time.Time
		}{
			{Text: "Today", End: timeutil.StartOfDay(now).Add(24 * time.Hour)},
			{Text: "Yesterday", End: timeutil.StartOfDay(now)},
			{Text: "This Week", End: timeutil.StartOfDay(now).Add(-24 * time.Hour)},
			{Text: "Last Week", End: timeutil.StartOfWeek(now)},
			{Text: "Earlier", End: timeutil.StartOfWeek(now).Add(-7 * 24 * time.Hour)},
		}
	)

	var nodes []*html.Node
	var commits *html.Node
	for _, c := range cs.Commits {
		// Heading.
		if time := c.Commit.Committer.Date.Time(); len(headings) > 0 && headings[0].End.After(time) {
			for len(headings) >= 2 && headings[1].End.After(time) {
				headings = headings[1:]
			}
			commits = htmlg.DivClass("list-entry-border") // Create a new sequence of commits.
			nodes = append(nodes,
				htmlg.H4(htmlg.Text(headings[0].Text)),
				commits,
			)
			headings = headings[1:]
		}

		// Commit.
		htmlg.AppendChildren(commits, c.Render()...)
	}
	return nodes
}

type Commit struct {
	Commit *vcs.Commit
	User   users.User
}

func (c Commit) Render() []*html.Node {
	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "display: flex;"}},
	}

	avatarDiv := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "margin-right: 6px;"}},
	}
	htmlg.AppendChildren(avatarDiv, issuescomponent.Avatar{User: c.User, Size: 32}.Render()...)
	div.AppendChild(avatarDiv)

	titleAndByline := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "flex-grow: 1;"}},
	}
	{
		commitSubject, commitBody := splitCommitMessage(c.Commit.Message)

		title := htmlg.Div(
			&html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "black"},
					//{Key: atom.Href.String(), Val: "..."}, // TODO.
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
		htmlg.AppendChildren(byline, issuescomponent.User{User: c.User}.Render()...)
		byline.AppendChild(htmlg.Text(" committed "))
		htmlg.AppendChildren(byline, issuescomponent.Time{Time: c.Commit.Author.Date.Time()}.Render()...)
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

	commitID := commitID{SHA: string(c.Commit.ID)}
	htmlg.AppendChildren(div, commitID.Render()...)

	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "lightgray"},
			{Key: atom.Style.String(), Val: "height: 16px; margin-left: 12px;"},
			{Key: atom.Href.String(), Val: "https://gotools.org/dmitri.shuralyov.com/kebabcase?rev=" + string(c.Commit.ID)},
			{Key: atom.Title.String(), Val: "View code at this revision."},
		},
		FirstChild: octiconssvg.Code(),
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
