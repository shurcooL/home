package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/go-github/github"
	"github.com/shurcooL/go/timeutil"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var indexHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Dmitri Shuralyov</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<link href="/blog/assets/octicons/octicons.min.css" rel="stylesheet" type="text/css">
		<link href="/blog/assets/gfm/gfm.css" rel="stylesheet" type="text/css">
		<link href="/assets/index/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>
		<div style="max-width: 800px; margin: 0 auto 100px auto;">`))

func initIndex(notifications notifications.Service, users users.Service) http.Handler {
	h := &indexHandler{
		notifications: notifications,
		users:         users,
	}
	go func() {
		for {
			events, commits, err := fetchActivity()
			h.mu.Lock()
			h.events, h.commits, h.activityError = events, commits, err
			h.mu.Unlock()

			time.Sleep(time.Minute)
		}
	}()
	return userMiddleware{httputil.ErrorHandler(h.ServeHTTP)}
}

func fetchActivity() ([]*github.Event, map[string]*github.RepositoryCommit, error) {
	events, _, err := unauthenticatedGitHubClient.Activity.ListEventsPerformedByUser("shurcooL", true, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, nil, err
	}
	commits := make(map[string]*github.RepositoryCommit)
	for _, e := range events {
		switch p := e.Payload().(type) {
		case *github.PushEvent:
			for _, c := range p.Commits {
				rc, err := fetchCommit(*c.URL)
				if err, ok := err.(*github.ErrorResponse); ok && err.Response.StatusCode == http.StatusNotFound {
					continue
				}
				if err != nil {
					return nil, nil, fmt.Errorf("fetchCommit: %v", err)
				}
				commits[*c.SHA] = rc
			}
		}
	}
	return events, commits, nil
}

func fetchCommit(commitAPIURL string) (*github.RepositoryCommit, error) {
	req, err := unauthenticatedGitHubClient.NewRequest("GET", commitAPIURL, nil)
	if err != nil {
		return nil, err
	}
	commit := new(github.RepositoryCommit)
	_, err = unauthenticatedGitHubClient.Do(req, commit)
	if err != nil {
		return nil, err
	}
	return commit, nil
}

type indexHandler struct {
	notifications notifications.Service
	users         users.Service

	mu            sync.Mutex
	events        []*github.Event
	commits       map[string]*github.RepositoryCommit // SHA -> Commit.
	activityError error
}

func (h *indexHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct{ Production bool }{*productionFlag}
	err := indexHTML.Execute(w, data)
	if err != nil {
		return err
	}

	authenticatedUser, err := h.users.GetAuthenticated(req.Context())
	if err != nil {
		return err
	}
	returnURL := req.RequestURI

	// Render the header.
	header := component.Header{
		CurrentUser:   authenticatedUser,
		ReturnURL:     returnURL,
		Notifications: h.notifications,
	}
	err = htmlg.RenderComponentsContext(req.Context(), w, header)
	if err != nil {
		return err
	}

	h.mu.Lock()
	events, commits, activityError := h.events, h.commits, h.activityError
	h.mu.Unlock()
	activity := activity{
		Events:  events,
		Commits: commits,
		Error:   activityError,
		ShowWIP: req.URL.Query().Get("events") == "all" || authenticatedUser.UserSpec == shurcool,
	}
	activity.ShowRaw, _ = strconv.ParseBool(req.URL.Query().Get("raw"))
	err = htmlg.RenderComponents(w, activity)
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

type activity struct {
	Events  []*github.Event
	Commits map[string]*github.RepositoryCommit // SHA -> Commit.
	Error   error

	ShowWIP bool // Controls whether all events are displayed, including WIP ones.
	ShowRaw bool // Controls whether full raw payload are available as titles.
}

func (a activity) Render() []*html.Node {
	var nodes []*html.Node

	if a.Error != nil {
		nodes = append(nodes,
			htmlg.H3(htmlg.Text("Activity Error")),
			htmlg.P(htmlg.Text(a.Error.Error())),
		)

		return []*html.Node{htmlg.DivClass("activity", nodes...)}
	}

	if len(a.Events) == 0 {
		nodes = append(nodes,
			htmlg.Text("No recent activity."),
		)

		return []*html.Node{htmlg.DivClass("activity", nodes...)}
	}

	var (
		now     = time.Now()
		headers = []struct {
			Text string
			End  time.Time
		}{
			{Text: "Today", End: timeutil.StartOfDay(now).Add(24 * time.Hour)},
			{Text: "Yesterday", End: timeutil.StartOfDay(now)},
			{Text: "This week", End: timeutil.StartOfDay(now).Add(-24 * time.Hour)},
			{Text: "Last week", End: timeutil.StartOfWeek(now)},
			{Text: "Earlier", End: timeutil.StartOfWeek(now).Add(-7 * 24 * time.Hour)},
		}
	)

	for _, e := range a.Events {
		// Header.
		if len(headers) > 0 && headers[0].End.After(*e.CreatedAt) {
			for len(headers) >= 2 && headers[1].End.After(*e.CreatedAt) {
				headers = headers[1:]
			}
			nodes = append(nodes,
				htmlg.DivClass("events-header", htmlg.Text(headers[0].Text)),
			)
			headers = headers[1:]
		}

		// Event.
		basicEvent := basicEvent{
			Time:      *e.CreatedAt,
			Actor:     *e.Actor.Login,
			Container: "github.com/" + *e.Repo.Name,
		}

		if a.ShowRaw {
			// For debugging, include full raw payload as a title.
			var raw bytes.Buffer
			err := json.Indent(&raw, (*e.RawPayload), "", "\t")
			if err != nil {
				panic(err)
			}
			basicEvent.Raw = raw.String()
		}

		var displayEvent htmlg.Component
		switch p := e.Payload().(type) {
		case *github.IssuesEvent:
			e := event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.IssueOpened,
				Action:     fmt.Sprintf("%v an issue in", *p.Action),
			}
			details := iconLinkDetails{
				Text:  *p.Issue.Title,
				URL:   *p.Issue.HTMLURL,
				Black: true,
			}
			switch *p.Action {
			case "opened":
				details.Icon = octiconssvg.IssueOpened
				details.Color = RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
			case "closed":
				details.Icon = octiconssvg.IssueClosed
				details.Color = RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
			case "reopened":
				details.Icon = octiconssvg.IssueReopened
				details.Color = RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
			default:
				log.Println("activity.Render: unsupported *github.IssuesEvent action:", *p.Action)
				details.Icon = octiconssvg.IssueOpened
			}
			e.Details = details
			displayEvent = e
		case *github.PullRequestEvent:
			e := event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.GitPullRequest,
			}
			details := iconLinkDetails{
				Text:  *p.PullRequest.Title,
				URL:   *p.PullRequest.HTMLURL,
				Black: true,
			}
			switch {
			case !*p.PullRequest.Merged && *p.PullRequest.State == "open":
				e.Action = "opened a pull request in"
				details.Icon = octiconssvg.GitPullRequest
				details.Color = RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
			case !*p.PullRequest.Merged && *p.PullRequest.State == "closed":
				e.Action = "closed a pull request in"
				details.Icon = octiconssvg.GitPullRequest
				details.Color = RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
			case *p.PullRequest.Merged:
				e.Action = "merged a pull request in"
				details.Icon = octiconssvg.GitMerge
				details.Color = RGB{R: 0x6e, G: 0x54, B: 0x94} // Purple.
			default:
				log.Println("activity.Render: unsupported *github.PullRequestEvent action:", *p.Action)
				details.Icon = octiconssvg.GitPullRequest
			}
			e.Details = details
			displayEvent = e

		case *github.IssueCommentEvent:
			e := event{
				basicEvent: &basicEvent,
			}
			switch p.Issue.PullRequestLinks {
			case nil: // Issue.
				switch *p.Action {
				case "created":
					e.Action = "commented on an issue in"
				default:
					basicEvent.WIP = true
					e.Action = fmt.Sprintf("%v on an issue in", *p.Action)
				}
			default: // Pull Request.
				switch *p.Action {
				case "created":
					e.Action = "commented on a pull request in"
				default:
					basicEvent.WIP = true
					e.Action = fmt.Sprintf("%v on a pull request in", *p.Action)
				}
			}
			displayEvent = e
		case *github.PullRequestReviewCommentEvent:
			e := event{
				basicEvent: &basicEvent,
			}
			switch *p.Action {
			case "created":
				e.Action = "commented on a pull request in"
			default:
				basicEvent.WIP = true
				e.Action = fmt.Sprintf("%v on a pull request in", *p.Action)
			}
			displayEvent = e
		case *github.CommitCommentEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Action:     "commented on a commit in",
			}

		case *github.PushEvent:
			var commits []*github.RepositoryCommit
			for _, c := range p.Commits {
				commit := a.Commits[*c.SHA]
				if commit == nil {
					avatarURL := "https://secure.gravatar.com/avatar?d=mm&f=y&s=96"
					if *c.Author.Email == "shurcooL@gmail.com" {
						// TODO: Can we de-dup this in a good way? It's in users service.
						avatarURL = "https://dmitri.shuralyov.com/avatar-s.jpg"
					}
					commit = &github.RepositoryCommit{
						SHA:    c.SHA,
						Commit: &github.Commit{Message: c.Message},
						Author: &github.User{AvatarURL: &avatarURL},
					}
				}
				commits = append(commits, commit)
			}

			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.GitCommit,
				Action:     "pushed to",
				Details: &commitsDetails{
					Commits: commits,
				},
			}

		case *github.ForkEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.RepoForked,
				Action:     "forked",
				Details: iconLinkDetails{
					Text: "github.com/" + *p.Forkee.FullName,
					URL:  *p.Forkee.HTMLURL,
					Icon: octiconssvg.Repo,
				},
			}

		case *github.WatchEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.Star,
				Action:     "starred",
			}

		case *github.CreateEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.GitBranch,
				Action:     fmt.Sprintf("created %v in", *p.RefType),
				Details: codeDetails{
					Text: *p.Ref,
				},
			}
		case *github.DeleteEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.Trashcan,
				Action:     fmt.Sprintf("deleted %v in", *p.RefType),
				Details: codeDetails{
					Text:          *p.Ref,
					Strikethrough: true,
				},
			}

		case *github.GollumEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.Book,
				Action:     "edited the wiki in",
				Details: &pagesDetails{
					Actor: e.Actor,
					Pages: p.Pages,
				},
			}

		default:
			basicEvent.WIP = true
			displayEvent = event{
				basicEvent: &basicEvent,
				Action:     *e.Type,
			}
		}
		if displayEvent == nil {
			continue
		}
		if basicEvent.WIP && !a.ShowWIP {
			continue
		}

		nodes = append(nodes, displayEvent.Render()...)
	}

	return []*html.Node{htmlg.DivClass("activity", nodes...)}
}

type basicEvent struct {
	Time      time.Time
	Actor     string
	Container string // URL of container without schema. E.g., "github.com/user/repo".

	WIP bool   // Whether this event's presentation is a work in progress.
	Raw string // Raw event for debugging to display as title. Empty string excludes it.
}

type event struct {
	*basicEvent
	Icon    func() *html.Node
	Action  string
	Details htmlg.Component
}

func (e event) Render() []*html.Node {
	divClass := "event"
	if e.WIP {
		divClass += " wip"
	}
	if e.Icon == nil {
		e.Icon = func() *html.Node { return &html.Node{Type: html.TextNode} }
	}
	var actionAttr []html.Attribute
	if e.Raw != "" {
		actionAttr = []html.Attribute{{Key: atom.Title.String(), Val: e.Raw}}
	}
	div := htmlg.DivClass(divClass,
		htmlg.SpanClass("icon", e.Icon()),
		htmlg.Text(e.Actor),
		htmlg.Text(" "),
		&html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr:       actionAttr,
			FirstChild: htmlg.Text(e.Action),
		},
		htmlg.Text(" "),
		htmlg.A(e.Container, template.URL("https://"+e.Container)),
		&html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr: []html.Attribute{
				{Key: atom.Class.String(), Val: "time"},
				{Key: atom.Title.String(), Val: e.Time.Local().Format(timeFormat)}, // TODO: Use local time of page viewer, not server.
			},
			FirstChild: htmlg.Text(humanize.Time(e.Time)),
		},
	)
	if e.Details != nil {
		for _, n := range e.Details.Render() {
			div.AppendChild(n)
		}
	}
	return []*html.Node{div}
}

const timeFormat = "Jan _2, 2006, 3:04 PM MST"

// TODO: Dedup.
//
// RGB represents a 24-bit color without alpha channel.
type RGB struct {
	R, G, B uint8
}

// HexString returns a hexadecimal color string. For example, "#ff0000" for red.
func (c RGB) HexString() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// iconLinkDetails are details consisting of an icon and a text link.
// Icon must be not nil.
type iconLinkDetails struct {
	Text  string
	URL   string
	Black bool              // Black link.
	Icon  func() *html.Node // Must be not nil.
	Color RGB
}

func (d iconLinkDetails) Render() []*html.Node {
	icon := htmlg.Span(d.Icon())
	icon.Attr = append(icon.Attr, html.Attribute{
		Key: atom.Style.String(), Val: fmt.Sprintf("color: %s; margin-right: 4px;", d.Color.HexString()),
	})
	link := htmlg.A(d.Text, template.URL(d.URL))
	if d.Black {
		link.Attr = append(link.Attr, html.Attribute{Key: atom.Class.String(), Val: "black"})
	}
	div := htmlg.DivClass("details",
		icon,
		link,
	)
	return []*html.Node{div}
}

type codeDetails struct {
	Text          string
	Strikethrough bool
}

func (d codeDetails) Render() []*html.Node {
	codeStyle := `padding: 2px 6px;
background-color: rgb(232, 241, 246);
border-radius: 3px;`
	if d.Strikethrough {
		codeStyle += `text-decoration: line-through; color: gray;`
	}
	code := &html.Node{
		Type: html.ElementNode, Data: atom.Code.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: codeStyle}},
	}
	code.AppendChild(htmlg.Text(d.Text))
	div := htmlg.DivClass("details",
		code,
	)
	return []*html.Node{div}
}

type commitsDetails struct {
	Commits []*github.RepositoryCommit
}

func (d commitsDetails) Render() []*html.Node {
	var nodes []*html.Node

	for _, c := range d.Commits {
		avatar := &html.Node{
			Type: html.ElementNode, Data: atom.Img.String(),
			Attr: []html.Attribute{
				{Key: atom.Src.String(), Val: *c.Author.AvatarURL},
				{Key: atom.Style.String(), Val: "width: 16px; height: 16px; vertical-align: top; margin-right: 6px;"},
			},
		}
		sha := &html.Node{
			Type: html.ElementNode, Data: atom.Code.String(),
			FirstChild: htmlg.Text(shortSHA(*c.SHA)),
		}
		if c.HTMLURL != nil {
			sha = &html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Href.String(), Val: *c.HTMLURL},
				},
				FirstChild: sha,
			}
		}
		message := &html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr: []html.Attribute{
				{Key: atom.Style.String(), Val: "margin-left: 6px;"},
			},
			FirstChild: htmlg.Text(firstParagraph(*c.Commit.Message)),
		}

		div := htmlg.Div(avatar, sha, message)
		div.Attr = append(div.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 4px;"})
		nodes = append(nodes, div)
	}

	div := htmlg.DivClass("details", nodes...)
	return []*html.Node{div}
}

func shortSHA(sha string) string {
	return sha[:8]
}

// firstParagraph returns the first paragraph of text s.
func firstParagraph(s string) string {
	i := strings.Index(s, "\n\n")
	if i == -1 {
		return s
	}
	return s[:i]
}

type pagesDetails struct {
	Actor *github.User   // Actor that acted on the pages.
	Pages []*github.Page // Wiki pages that are affected.
}

func (d pagesDetails) Render() []*html.Node {
	var nodes []*html.Node

	for _, p := range d.Pages {
		avatar := &html.Node{
			Type: html.ElementNode, Data: atom.Img.String(),
			Attr: []html.Attribute{
				{Key: atom.Src.String(), Val: *d.Actor.AvatarURL},
				{Key: atom.Style.String(), Val: "width: 16px; height: 16px; vertical-align: top; margin-right: 6px;"},
			},
		}
		action := &html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			FirstChild: htmlg.Text(*p.Action),
		}
		switch *p.Action {
		case "edited":
			action = &html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Href.String(), Val: "https://github.com" + *p.HTMLURL + "/_compare/" + *p.SHA + "^..." + *p.SHA},
				},
				FirstChild: action,
			}
		}
		title := &html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Href.String(), Val: "https://github.com" + *p.HTMLURL},
			},
			FirstChild: &html.Node{
				Type: html.ElementNode, Data: atom.Span.String(),
				FirstChild: htmlg.Text(*p.Title),
			},
		}

		div := htmlg.Div(avatar, action, htmlg.Text(" page "), title)
		div.Attr = append(div.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 4px;"})
		nodes = append(nodes, div)
	}

	div := htmlg.DivClass("details", nodes...)
	return []*html.Node{div}
}
