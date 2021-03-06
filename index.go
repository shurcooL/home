package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"dmitri.shuralyov.com/html/belt"
	statepkg "dmitri.shuralyov.com/state"
	"github.com/dustin/go-humanize"
	"github.com/shurcooL/component"
	"github.com/shurcooL/events"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/go/timeutil"
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var indexHTML = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html lang="en">
	<head>
{{.AnalyticsHTML}}		<title>Dmitri Shuralyov</title>
		<link href="/icon.svg" rel="icon" type="image/svg+xml">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/index/style.css" rel="stylesheet" type="text/css">
		<link href="https://github.com/{{.GitHubRelMe}}" rel="me">
	</head>
	<body>
		<div style="max-width: 800px; margin: 0 auto 100px auto;">`))

func initIndex(events events.Service, notification notification.Service, users users.Service) http.Handler {
	h := &indexHandler{
		AuthzEndpoint: indieauthMeFlag.Me != nil,
		events:        events,
		notification:  notification,
		users:         users,
	}
	return cookieAuth{httputil.ErrorHandler(users, h.ServeHTTP)}
}

type indexHandler struct {
	AuthzEndpoint bool // Whether to advertise the IndieAuth authorization endpoint.
	events        events.Service
	notification  notification.Service
	users         users.Service
}

func (h *indexHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if err := httputil.AllowMethods(req, http.MethodGet, http.MethodHead); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.AuthzEndpoint {
		// Advertise the IndieAuth authorization endpoint.
		w.Header().Set("Link", `</api/indieauth/authorization>; rel="authorization_endpoint"`)
	}
	if req.Method == http.MethodHead {
		return nil
	}

	data := struct {
		AnalyticsHTML template.HTML
		GitHubRelMe   string // GitHub username to advertise in a rel='me' link.
	}{analyticsHTML, *githubRelMeFlag}
	err := indexHTML.Execute(w, data)
	if err != nil {
		return err
	}

	authenticatedUser, err := h.users.GetAuthenticated(req.Context())
	if err != nil {
		return err
	}
	var nc uint64
	if authenticatedUser.ID != 0 {
		nc, err = h.notification.CountNotifications(req.Context())
		if err != nil {
			return err
		}
	}
	returnURL := req.RequestURI

	// Render the header.
	header := homecomponent.Header{
		CurrentUser:       authenticatedUser,
		NotificationCount: nc,
		ReturnURL:         returnURL,
	}
	err = htmlg.RenderComponents(w, header)
	if err != nil {
		return err
	}

	err = html.Render(w, htmlg.H3(htmlg.Text("Activity")))
	if err != nil {
		return err
	}

	events, eventsError := h.events.List(req.Context())
	var error string
	if eventsError != nil {
		error = "There was a problem getting latest activity."
		if authenticatedUser.SiteAdmin {
			error += "\n\n" + eventsError.Error()
		}
	}
	activity := activity{
		Events:  events,
		Error:   error,
		ShowWIP: req.URL.Query().Get("events") == "all" || authenticatedUser.UserSpec == dmitshur,
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
	Events []event.Event
	Error  string

	ShowWIP bool // Controls whether all events are displayed, including WIP ones.
	ShowRaw bool // Controls whether full raw payload are available as titles.
}

func (a activity) Render() []*html.Node {
	var nodes []*html.Node

	if a.Error != "" {
		nodes = append(nodes,
			&html.Node{
				Type: html.ElementNode, Data: atom.P.String(),
				Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "white-space: pre;"}},
				FirstChild: htmlg.Text(a.Error),
			},
		)
	}

	if len(a.Events) == 0 {
		nodes = append(nodes,
			htmlg.Text("No recent activity."),
		)

		return []*html.Node{htmlg.DivClass("activity", nodes...)}
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

	for _, e := range a.Events {
		// Heading.
		if len(headings) > 0 && headings[0].End.After(e.Time) {
			for len(headings) >= 2 && headings[1].End.After(e.Time) {
				headings = headings[1:]
			}
			nodes = append(nodes,
				htmlg.DivClass("events-heading", htmlg.Text(headings[0].Text)),
			)
			headings = headings[1:]
		}

		// Event.
		basicEvent := basicEvent{
			Time:  e.Time,
			Actor: e.Actor.Login,
		}

		if a.ShowRaw {
			// For debugging, include full raw payload as a title.
			raw, err := json.MarshalIndent(e.Payload, "", "\t")
			if err != nil {
				panic(err)
			}
			basicEvent.Raw = string(raw)
		}

		var displayEvent htmlg.Component
		switch p := e.Payload.(type) {
		case event.Issue:
			ae := activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.IssueOpened,
				Action:     component.Join(p.Action, " an issue ", issueFromAction(p, e.Container)),
			}
			if p.Action == "reopened" {
				ae.Icon = octicon.IssueReopened
			} else if p.Action == "opened" && p.IssueBody != "" {
				ae.Details = imageText{
					ImageURL: e.Actor.AvatarURL,
					Text:     shortBody(p.IssueBody),
				}
			}
			displayEvent = ae
		case event.Change:
			ae := activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.GitPullRequest,
				Action:     component.Join(p.Action, " a change ", changeFromAction(p, e.Container)),
			}
			if p.Action == "opened" && p.ChangeBody != "" {
				ae.Details = imageText{
					ImageURL: e.Actor.AvatarURL,
					Text:     shortBody(p.ChangeBody),
				}
			}
			displayEvent = ae

		case event.IssueComment:
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.CommentDiscussion,
				Action:     component.Join("commented on ", issueFromComment(p, e.Container)),
				Details: imageText{
					ImageURL: e.Actor.AvatarURL,
					Text:     shortBody(p.CommentBody),
				},
			}
		case event.ChangeComment:
			var verb string
			switch p.CommentReview {
			case 0:
				verb = "commented"
			default:
				verb = fmt.Sprintf("reviewed %+d", p.CommentReview)
			}
			var details htmlg.Component
			if p.CommentBody != "" {
				details = imageText{
					ImageURL: e.Actor.AvatarURL,
					Text:     shortBody(p.CommentBody),
				}
			}
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.CommentDiscussion,
				Action:     component.Join(verb, " on ", changeFromComment(p, e.Container)),
				Details:    details,
			}
		case event.CommitComment:
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.CommentDiscussion,
				Action:     component.Join("commented on ", commitName(p, e.Container)),
				Details: imageText{
					ImageURL: e.Actor.AvatarURL,
					Text:     shortBody(p.CommentBody),
				},
			}

		case event.Push:
			basicEvent.Container = e.Container
			ae := activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.GitCommit,
				Action:     component.Join("pushed to ", belt.Reference{Name: p.Branch}),
			}
			switch len(p.Commits) {
			default:
				ae.Details = commits{
					Commits: p.Commits,
				}
			case 0:
				before := belt.CommitID{SHA: p.Before, HTMLURL: p.BeforeHTMLURL}
				head := belt.CommitID{SHA: p.Head, HTMLURL: p.HeadHTMLURL}
				ae.Details = component.Join(before, " → ", head)
			}
			displayEvent = ae

		case event.Star:
			basicEvent.Container = e.Container
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.Star,
				Action:     component.Text("starred"),
			}

		case event.Create:
			basicEvent.Container = e.Container
			ae := activityEvent{
				basicEvent: &basicEvent,
			}
			switch p.Type {
			case "repository":
				ae.Icon = octicon.Repo
				ae.Action = component.Text("created repository")
				if p.Description != "" {
					ae.Details = plainText{Text: p.Description}
				}
			case "package":
				ae.Icon = octicon.Package
				ae.Action = component.Text("created package")
				if p.Description != "" {
					ae.Details = plainText{Text: p.Description}
				}
			case "branch":
				ae.Icon = octicon.GitBranch
				ae.Action = component.Text("created branch")
				ae.Details = belt.Reference{Name: p.Name}
			case "tag":
				ae.Icon = octicon.Tag
				ae.Action = component.Text("created tag")
				ae.Details = belt.Reference{Name: p.Name}

				//default:
				//basicEvent.WIP = true
				//e.Action = component.Text(fmt.Sprintf("created %v", *p.RefType))
				//e.Details = code{
				//	Text: p.Name,
				//}
			}
			displayEvent = ae
		case event.Fork:
			basicEvent.Container = e.Container
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.RepoForked,
				Action:     component.Text("forked"),
				Details: iconLink{
					Text:      p.Container,
					URL:       "https://" + p.Container,
					Black:     true,
					Icon:      octicon.Repo,
					IconColor: &RGB{R: 35, G: 35, B: 35}, // Black (not pure).
				},
			}
		case event.Delete:
			basicEvent.Container = e.Container
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.Trashcan,
				Action:     component.Text(fmt.Sprintf("deleted %v", p.Type)),
				Details: belt.Reference{
					Name:          p.Name,
					Strikethrough: true,
				},
			}

		case event.Wiki:
			basicEvent.Container = e.Container
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octicon.Book,
				Action:     component.Text("edited the wiki"),
				Details: pages{
					ActorAvatarURL: e.Actor.AvatarURL,
					Pages:          p.Pages,
				},
			}

		default:
			log.Printf("activity.Render: unexpected event type: %T\n", p)
			continue
		}
		if basicEvent.WIP && !a.ShowWIP {
			continue
		}

		nodes = append(nodes, displayEvent.Render()...)
	}

	return []*html.Node{htmlg.DivClass("activity", nodes...)}
}

func issueFromAction(p event.Issue, importPath string) htmlg.Component {
	var is statepkg.Issue
	switch p.Action {
	case "opened", "reopened":
		is = statepkg.IssueOpen
	case "closed":
		is = statepkg.IssueClosed
	default:
		log.Printf("issueFromAction: unsupported event.Issue action %q\n", p.Action)
		is = statepkg.IssueOpen
	}
	return belt.Issue{
		State:   is,
		Title:   importPath + ": " + p.IssueTitle,
		HTMLURL: p.IssueHTMLURL,
	}
}
func issueFromComment(p event.IssueComment, importPath string) htmlg.Component {
	return belt.Issue{
		State:   p.IssueState,
		Title:   importPath + ": " + p.IssueTitle,
		HTMLURL: p.CommentHTMLURL,
	}
}
func changeFromAction(p event.Change, importPath string) htmlg.Component {
	var cs statepkg.Change
	switch p.Action {
	case "opened", "reopened":
		cs = statepkg.ChangeOpen
	case "closed":
		cs = statepkg.ChangeClosed
	case "merged":
		cs = statepkg.ChangeMerged
	default:
		log.Printf("changeFromAction: unsupported event.Change action %q\n", p.Action)
		cs = statepkg.ChangeOpen
	}
	return belt.Change{
		State:   cs,
		Title:   importPath + ": " + p.ChangeTitle,
		HTMLURL: p.ChangeHTMLURL,
	}
}
func changeFromComment(p event.ChangeComment, importPath string) htmlg.Component {
	return belt.Change{
		State:   p.ChangeState,
		Title:   importPath + ": " + p.ChangeTitle,
		HTMLURL: p.CommentHTMLURL,
	}
}
func commitName(p event.CommitComment, importPath string) htmlg.Component {
	c := p.Commit
	if c.Message == "" {
		a := htmlg.A(importPath, "https://"+importPath)
		return component.Join("a commit in ", htmlg.NodeComponent(*a))
	}
	return belt.Commit{
		SHA:             c.SHA,
		Message:         importPath + ": " + c.Message,
		AuthorAvatarURL: c.AuthorAvatarURL,
		HTMLURL:         c.HTMLURL,
	}
}

type basicEvent struct {
	Time      time.Time
	Actor     string
	Container string // URL of container without schema. E.g., "github.com/user/repo". Optional.

	WIP bool   // Whether this event's presentation is a work in progress.
	Raw string // Raw event for debugging to display as title. Empty string excludes it.
}

// activityEvent is an event within the activity stream.
// Action must be not nil.
type activityEvent struct {
	*basicEvent
	Icon    func() *html.Node
	Action  htmlg.Component // Not nil.
	Details htmlg.Component
}

func (e activityEvent) Render() []*html.Node {
	divClass := "event"
	if e.WIP {
		divClass += " wip"
	}
	if e.Icon == nil {
		e.Icon = func() *html.Node { return &html.Node{Type: html.TextNode} }
	}
	action := &html.Node{Type: html.ElementNode, Data: atom.Span.String()}
	if e.Raw != "" {
		action.Attr = append(action.Attr, html.Attribute{Key: atom.Title.String(), Val: e.Raw})
	}
	htmlg.AppendChildren(action, e.Action.Render()...)
	middle := []*html.Node{
		htmlg.Text(e.Actor),
		htmlg.Text(" "),
		action,
	}
	if e.Container != "" {
		middle = append(middle, htmlg.Text(" in "))
		middle = append(middle, htmlg.A(e.Container, "https://"+e.Container))
	}
	div := htmlg.DivClass(divClass, htmlg.DivClass("overview",
		htmlg.SpanClass("icon", e.Icon()),
		htmlg.SpanClass("middle", middle...),
		&html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr: []html.Attribute{
				{Key: atom.Class.String(), Val: "time"},
				{Key: atom.Title.String(), Val: humanize.Time(e.Time) + " – " + e.Time.Local().Format(timeFormat)}, // TODO: Use local time of page viewer, not server.
			},
			FirstChild: htmlg.Text(compactTime(e.Time)),
		},
	))
	if e.Details != nil {
		div.AppendChild(htmlg.DivClass("details", e.Details.Render()...))
	}
	return []*html.Node{div}
}

const timeFormat = "Jan 2, 2006, 3:04 PM MST"

// compactTime formats time t into a relative string.
//
// For example, "5s" for 5 seconds ago, "47m" for 47 minutes ago,
// "3w" for 3 weeks ago, etc.
func compactTime(t time.Time) string {
	return humanize.CustomRelTime(t, time.Now(), "", "", compactMagnitudes)
}

var compactMagnitudes = []humanize.RelTimeMagnitude{
	{D: time.Minute, Format: "%ds", DivBy: time.Second},
	{D: time.Hour, Format: "%dm", DivBy: time.Minute},
	{D: humanize.Day, Format: "%dh", DivBy: time.Hour},
	{D: humanize.Week, Format: "%dd", DivBy: humanize.Day},
	{D: humanize.Month, Format: "%dw", DivBy: humanize.Week},
	{D: humanize.Year, Format: "%dmo", DivBy: humanize.Month},
	{D: math.MaxInt64, Format: "%dy", DivBy: humanize.Year},
}

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

// iconLink consists of an icon and a text link.
// Icon must be not nil.
type iconLink struct {
	Text      string
	Tooltip   string
	URL       string
	Black     bool              // Black link.
	Icon      func() *html.Node // Not nil.
	IconColor *RGB              // Optional icon color override.
}

func (d iconLink) Render() []*html.Node {
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{{Key: atom.Href.String(), Val: d.URL}},
	}
	if d.Tooltip != "" {
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Title.String(), Val: d.Tooltip})
	}
	if d.Black {
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Class.String(), Val: "black"})
	}
	iconSpanStyle := "margin-right: 4px;"
	if d.IconColor != nil {
		iconSpanStyle += fmt.Sprintf(" color: %s;", d.IconColor.HexString())
	}
	a.AppendChild(&html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr:       []html.Attribute{{Key: atom.Style.String(), Val: iconSpanStyle}},
		FirstChild: d.Icon(),
	})
	a.AppendChild(htmlg.Text(d.Text))
	return []*html.Node{a}
}

type plainText struct {
	Text string
}

func (d plainText) Render() []*html.Node {
	text := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "font-size: 13px; color: #666;"}},
		FirstChild: htmlg.Text(d.Text),
	}
	return []*html.Node{text}
}

type imageText struct {
	ImageURL string
	Text     string
}

func (d imageText) Render() []*html.Node {
	image := &html.Node{
		Type: html.ElementNode, Data: atom.Img.String(),
		Attr: []html.Attribute{
			{Key: atom.Src.String(), Val: d.ImageURL},
			{Key: atom.Style.String(), Val: "width: 28px; height: 28px; margin-right: 6px; flex-shrink: 0;"},
		},
	}
	text := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "font-size: 13px; color: #666; flex-grow: 1;"}},
		FirstChild: htmlg.Text(d.Text),
	}
	div := htmlg.Div(image, text)
	div.Attr = append(div.Attr, html.Attribute{Key: atom.Style.String(), Val: "display: flex;"})
	return []*html.Node{div}
}

func shortBody(s string) string {
	if len(s) <= 200 {
		return s
	}
	var i int
	for i = 1; i < utf8.UTFMax && !utf8.RuneStart(s[200-i]); i++ {
	}
	return s[:200-i] + "…"
}

type commits struct {
	Commits []event.Commit // Ordered from earliest to most recent (head).
}

func (d commits) Render() []*html.Node {
	var nodes []*html.Node

	// Display latest commits on top.
	for i := len(d.Commits) - 1; i >= 0; i-- {
		c := d.Commits[i]
		commit := belt.Commit{
			SHA:             c.SHA,
			Message:         c.Message,
			AuthorAvatarURL: c.AuthorAvatarURL,
			HTMLURL:         c.HTMLURL,
		}
		div := htmlg.Div(commit.Render()...)
		div.Attr = append(div.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 4px;"})
		nodes = append(nodes, div)
	}

	return nodes
}

type pages struct {
	ActorAvatarURL string       // Actor that acted on the pages.
	Pages          []event.Page // Wiki pages that are affected.
}

func (d pages) Render() []*html.Node {
	var nodes []*html.Node

	for _, p := range d.Pages {
		avatar := &html.Node{
			Type: html.ElementNode, Data: atom.Img.String(),
			Attr: []html.Attribute{
				{Key: atom.Src.String(), Val: d.ActorAvatarURL},
				{Key: atom.Style.String(), Val: "width: 16px; height: 16px; vertical-align: top; margin-right: 6px;"},
			},
		}
		commitID := belt.CommitID{SHA: p.SHA, HTMLURL: p.CompareHTMLURL}
		message := &html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr: []html.Attribute{{Key: atom.Style.String(), Val: "margin-left: 4px;"}},
		}
		switch p.Action {
		case "created":
			message.AppendChild(htmlg.Text("Create"))
		case "edited":
			message.AppendChild(htmlg.Text("Edit"))
		}
		message.AppendChild(htmlg.Text(" page "))
		message.AppendChild(&html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Href.String(), Val: p.HTMLURL},
			},
			FirstChild: htmlg.Text(p.Title),
		})
		message.AppendChild(htmlg.Text("."))

		var ns []*html.Node
		ns = append(ns, avatar)
		ns = append(ns, commitID.Render()...)
		ns = append(ns, message)
		div := htmlg.Div(ns...)
		div.Attr = append(div.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 4px;"})
		nodes = append(nodes, div)
	}

	return nodes
}
