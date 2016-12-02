package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
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
	// TODO: Unify this unauthenticated GitHub client with that of newUsersService.
	var transport http.RoundTripper
	transport = &github.UnauthenticatedRateLimitedTransport{
		ClientID:     githubConfig.ClientID,
		ClientSecret: githubConfig.ClientSecret,
	}
	transport = &httpcache.Transport{
		Transport:           transport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}
	gh := github.NewClient(&http.Client{Transport: transport})

	h := &indexHandler{
		notifications: notifications,
		users:         users,
	}
	go func() {
		for {
			events, _, err := gh.Activity.ListEventsPerformedByUser("shurcooL", true, &github.ListOptions{PerPage: 30})
			//fmt.Println("called gh.Activity.ListEventsPerformedByUser:", len(events), err, resp.Rate.Remaining)
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

	activity := activity{events: events}
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
	events []*github.Event
}

func (a activity) Render() []*html.Node {
	var nodes []*html.Node

	nodes = append(nodes,
		htmlg.DivClass("events-header", htmlg.Text(fmt.Sprintf("Today"))),
	)

	for _, e := range a.events {
		switch e.Payload().(type) {
		case *github.WatchEvent:
			e := event{
				Actor:  *e.Actor.Login,
				Verb:   "starred",
				Target: "github.com/" + *e.Repo.Name,
				Time:   *e.CreatedAt,
			}
			for _, n := range e.Render() {
				nodes = append(nodes, n)
			}
		}
	}

	return []*html.Node{htmlg.DivClass("activity", nodes...)}
}

type event struct {
	Actor  string
	Verb   string
	Target string // URL of the target without schema. E.g., "github.com/user/repo".
	Time   time.Time
}

func (e event) Render() []*html.Node {
	var nodes []*html.Node
	nodes = append(nodes,
		htmlg.DivClass("event",
			htmlg.Text(e.Actor),
			htmlg.Text(" "),
			htmlg.Text(e.Verb),
			htmlg.Text(" "),
			htmlg.A(e.Target, template.URL("https://"+e.Target)),
			htmlg.SpanClass("time", htmlg.Text(humanize.Time(e.Time))),
		),
	)
	return nodes
}
