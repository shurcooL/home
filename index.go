package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
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
			events, _, err := unauthenticatedGitHubClient.Activity.ListEventsPerformedByUser("shurcooL", true, &github.ListOptions{PerPage: 100})
			h.mu.Lock()
			h.events, h.eventsError = events, err
			h.mu.Unlock()

			time.Sleep(time.Minute)
		}
	}()
	return userMiddleware{httputil.ErrorHandler(h.ServeHTTP)}
}

type indexHandler struct {
	notifications notifications.Service
	users         users.Service

	mu          sync.Mutex
	events      []*github.Event
	eventsError error
}

func (h *indexHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}

	h.mu.Lock()
	events, err := h.events, h.eventsError
	h.mu.Unlock()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct{ Production bool }{*productionFlag}
	err = indexHTML.Execute(w, data)
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

	activity := activity{Events: events, ShowWIP: req.URL.Query().Get("events") == "all" || authenticatedUser.UserSpec == shurcool}
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
	ShowWIP bool // ShowWIP controls whether all events are displayed, including WIP ones, and full raw payload as titles.
}

func (a activity) Render() []*html.Node {
	var nodes []*html.Node

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
			{Text: "Today", End: now.Truncate(24 * time.Hour).Add(24 * time.Hour)},
			{Text: "Yesterday", End: now.Truncate(24 * time.Hour)},
			{Text: "This week", End: now.Truncate(24 * time.Hour).Add(-24 * time.Hour)},
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

		if a.ShowWIP {
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
				Action:     fmt.Sprintf("%v an issue in", *p.Action),
			}
			switch *p.Action {
			case "opened":
				e.Icon = octiconssvg.IssueOpened
			case "closed":
				e.Icon = octiconssvg.IssueClosed
			}
			displayEvent = e
		case *github.PullRequestEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.GitPullRequest,
				Action:     fmt.Sprintf("%v a pull request in", *p.Action),
			}

		case *github.IssueCommentEvent:
			e := event{
				basicEvent: &basicEvent,
			}
			switch *p.Action {
			case "created":
				e.Action = "commented on an issue in"
			default:
				basicEvent.WIP = true
				e.Action = fmt.Sprintf("%v on an issue in", *p.Action)
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
				e.Action = fmt.Sprintf("%v comment on an issue in", *p.Action)
			}
			displayEvent = e
		case *github.CommitCommentEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Action:     "commented on a commit in",
			}

		case *github.PushEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.GitCommit,
				Action:     "pushed to",
			}

		case *github.ForkEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.RepoForked,
				Action:     "forked",
				//p.Forkee.FullName
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
				//*p.Ref
			}
		case *github.DeleteEvent:
			displayEvent = event{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.Trashcan,
				Action:     fmt.Sprintf("deleted %v in", *p.RefType),
				//*p.Ref
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

	Raw string // Raw event for debugging.
	WIP bool   // Whether this event's presentation is a work in progress.
}

type event struct {
	*basicEvent
	Icon   func() *html.Node
	Action string
}

func (e event) Render() []*html.Node {
	divClass := "event"
	if e.WIP {
		divClass += " wip"
	}
	if e.Icon == nil {
		e.Icon = func() *html.Node { return &html.Node{Type: html.TextNode} }
	}
	div := htmlg.DivClass(divClass,
		htmlg.SpanClass("icon", e.Icon()),
		htmlg.Text(e.Actor),
		htmlg.Text(" "),
		&html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr:       []html.Attribute{{Key: atom.Title.String(), Val: e.Raw}},
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
	return []*html.Node{div}
}

const timeFormat = "Jan _2, 2006, 3:04 PM MST"
