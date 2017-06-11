package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/shurcooL/component"
	"github.com/shurcooL/events"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/go/timeutil"
	homecomponent "github.com/shurcooL/home/component"
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
		<meta name="viewport" content="width=device-width">
		<link href="/assets/index/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>
		<div style="max-width: 800px; margin: 0 auto 100px auto;">`))

func initIndex(events events.Service, notifications notifications.Service, users users.Service) http.Handler {
	h := &indexHandler{
		events:        events,
		notifications: notifications,
		users:         users,
	}
	return cookieAuth{httputil.ErrorHandler(users, h.ServeHTTP)}
}

type indexHandler struct {
	events        events.Service
	notifications notifications.Service
	users         users.Service
}

func (h *indexHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if err := httputil.AllowMethods(req, http.MethodGet, http.MethodHead); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if req.Method == http.MethodHead {
		return nil
	}

	data := struct{ Production bool }{*productionFlag}
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
		nc, err = h.notifications.Count(req.Context(), nil)
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
			Time:      e.Time,
			Actor:     e.Actor.Login,
			Container: e.Container,
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
			e := activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.IssueOpened,
				Action:     component.Text(fmt.Sprintf("%v an issue in", p.Action)),
			}
			details := iconLink{
				Text:  p.IssueTitle,
				URL:   p.IssueHTMLURL,
				Black: true,
			}
			switch p.Action {
			case "opened":
				details.Icon = octiconssvg.IssueOpened
				details.Color = RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
			case "closed":
				details.Icon = octiconssvg.IssueClosed
				details.Color = RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
			case "reopened":
				details.Icon = octiconssvg.IssueReopened
				details.Color = RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.

				//default:
				//log.Println("activity.Render: unsupported event.Issue action:", p.Action)
				//details.Icon = octiconssvg.IssueOpened
			}
			e.Details = details
			displayEvent = e
		case event.PullRequest:
			e := activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.GitPullRequest,
			}
			details := iconLink{
				Text:  p.PullRequestTitle,
				URL:   p.PullRequestHTMLURL,
				Black: true,
			}
			switch p.Action {
			case "opened":
				e.Action = component.Text("opened a pull request in")
				details.Icon = octiconssvg.GitPullRequest
				details.Color = RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
			case "closed":
				e.Action = component.Text("closed a pull request in")
				details.Icon = octiconssvg.GitPullRequest
				details.Color = RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
			case "merged":
				e.Action = component.Text("merged a pull request in")
				details.Icon = octiconssvg.GitMerge
				details.Color = RGB{R: 0x6e, G: 0x54, B: 0x94} // Purple.

				//default:
				//log.Println("activity.Render: unsupported event.PullRequest action:", p.Action)
				//details.Icon = octiconssvg.GitPullRequest
			}
			e.Details = details
			displayEvent = e

		case event.IssueComment:
			e := activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.CommentDiscussion,
			}
			e.Action = component.Join("commented on ", issueName(p), " in")
			e.Details = imageText{
				ImageURL: p.CommentUserAvatarURL,
				Text:     shortBody(p.CommentBody),
			}
			displayEvent = e
		case event.PullRequestComment:
			e := activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.CommentDiscussion,
			}
			e.Action = component.Join("commented on ", prName(p), " in")
			e.Details = imageText{
				ImageURL: p.CommentUserAvatarURL,
				Text:     shortBody(p.CommentBody),
			}
			displayEvent = e
		case event.CommitComment:
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.CommentDiscussion,
				Action:     component.Join("commented on ", commitName(p), " in"),
				Details: imageText{
					ImageURL: p.CommentUserAvatarURL,
					Text:     shortBody(p.CommentBody),
				},
			}

		case event.Push:
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.GitCommit,
				Action:     component.Text("pushed to"),
				Details: commits{
					Commits: p.Commits,
				},
			}

		case event.Star:
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.Star,
				Action:     component.Text("starred"),
			}

		case event.Create:
			e := activityEvent{
				basicEvent: &basicEvent,
			}
			switch p.Type {
			case "repository":
				e.Icon = octiconssvg.Repo
				e.Action = component.Text("created repository")
				e.Details = text{
					Text: p.Description,
				}
			case "branch":
				e.Icon = octiconssvg.GitBranch
				e.Action = component.Text("created branch in")
				e.Details = code{
					Text: p.Name,
				}
			case "tag":
				e.Icon = octiconssvg.Tag
				e.Action = component.Text("created tag in")
				e.Details = code{
					Text: p.Name,
				}

				//default:
				//basicEvent.WIP = true
				//e.Action = component.Text(fmt.Sprintf("created %v in", *p.RefType))
				//e.Details = code{
				//	Text: p.Name,
				//}
			}
			displayEvent = e
		case event.Fork:
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.RepoForked,
				Action:     component.Text("forked"),
				Details: iconLink{
					Text: p.Container,
					URL:  "https://" + p.Container,
					Icon: octiconssvg.Repo,
				},
			}
		case event.Delete:
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.Trashcan,
				Action:     component.Text(fmt.Sprintf("deleted %v in", p.Type)),
				Details: code{
					Text:          p.Name,
					Strikethrough: true,
				},
			}

		case event.Gollum:
			displayEvent = activityEvent{
				basicEvent: &basicEvent,
				Icon:       octiconssvg.Book,
				Action:     component.Text("edited the wiki in"),
				Details: &pages{
					ActorAvatarURL: p.ActorAvatarURL,
					Pages:          p.Pages,
				},
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

func issueName(p event.IssueComment) htmlg.Component {
	n := iconLink{
		Text:    shortTitle(p.IssueTitle),
		Tooltip: p.IssueTitle,
		URL:     p.CommentHTMLURL,
		Black:   true,
	}
	switch p.IssueState {
	case "open":
		n.Icon = octiconssvg.IssueOpened
		n.Color = RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
	case "closed":
		n.Icon = octiconssvg.IssueClosed
		n.Color = RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.

		//default:
		//log.Println("issueName: unsupported event.IssueComment State:", p.State)
		//n.Icon = octiconssvg.IssueOpened
	}
	return n
}
func prName(p event.PullRequestComment) htmlg.Component {
	n := iconLink{
		Text:    shortTitle(p.PullRequestTitle),
		Tooltip: p.PullRequestTitle,
		URL:     p.CommentHTMLURL,
		Black:   true,
	}
	switch p.PullRequestState {
	case "open":
		n.Icon = octiconssvg.GitPullRequest
		n.Color = RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
	case "closed":
		n.Icon = octiconssvg.GitPullRequest
		n.Color = RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
	case "merged":
		n.Icon = octiconssvg.GitMerge
		n.Color = RGB{R: 0x6e, G: 0x54, B: 0x94} // Purple.

		//default:
		//log.Println("prName: unsupported event.PullRequestComment State:", p.State)
		//n.Icon = octiconssvg.GitPullRequest
	}
	return n
}
func commitName(p event.CommitComment) htmlg.Component {
	c := p.Commit
	if c.CommitMessage == "" {
		return component.Text("a commit")
	}
	return commit{Commit: c, Short: true}
}

type basicEvent struct {
	Time      time.Time
	Actor     string
	Container string // URL of container without schema. E.g., "github.com/user/repo".

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
	for _, n := range e.Action.Render() {
		action.AppendChild(n)
	}
	div := htmlg.DivClass(divClass,
		htmlg.SpanClass("icon", e.Icon()),
		htmlg.Text(e.Actor),
		htmlg.Text(" "),
		action,
		htmlg.Text(" "),
		htmlg.A(e.Container, "https://"+e.Container),
		&html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr: []html.Attribute{
				{Key: atom.Class.String(), Val: "time"},
				{Key: atom.Title.String(), Val: humanize.Time(e.Time) + " – " + e.Time.Local().Format(timeFormat)}, // TODO: Use local time of page viewer, not server.
			},
			FirstChild: htmlg.Text(compactTime(e.Time)),
		},
	)
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
	{D: humanize.Year, Format: "%dm", DivBy: humanize.Month},
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
	Text    string
	Tooltip string
	URL     string
	Black   bool              // Black link.
	Icon    func() *html.Node // Not nil.
	Color   RGB               // Icon color.
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
	a.AppendChild(&html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: fmt.Sprintf("color: %s; margin-right: 4px;", d.Color.HexString())},
		},
		FirstChild: d.Icon(),
	})
	a.AppendChild(htmlg.Text(d.Text))
	return []*html.Node{a}
}

type text struct {
	Text string
}

func (d text) Render() []*html.Node {
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
	return s[:199] + "…"
}

func shortTitle(s string) string {
	if len(s) <= 36 {
		return s
	}
	return s[:35] + "…"
}

type code struct {
	Text          string
	Strikethrough bool
}

func (d code) Render() []*html.Node {
	codeStyle := `padding: 2px 6px;
background-color: rgb(232, 241, 246);
border-radius: 3px;`
	if d.Strikethrough {
		codeStyle += `text-decoration: line-through; color: gray;`
	}
	code := &html.Node{
		Type: html.ElementNode, Data: atom.Code.String(),
		Attr:       []html.Attribute{{Key: atom.Style.String(), Val: codeStyle}},
		FirstChild: htmlg.Text(d.Text),
	}
	return []*html.Node{code}
}

type commits struct {
	Commits []event.Commit
}

func (d commits) Render() []*html.Node {
	var nodes []*html.Node

	for _, c := range d.Commits {
		div := htmlg.Div(commit{Commit: c}.Render()...)
		div.Attr = append(div.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 4px;"})
		nodes = append(nodes, div)
	}

	return nodes
}

type commit struct {
	event.Commit
	Short bool
}

func (c commit) Render() []*html.Node {
	avatar := &html.Node{
		Type: html.ElementNode, Data: atom.Img.String(),
		Attr: []html.Attribute{
			{Key: atom.Src.String(), Val: c.AuthorAvatarURL},
			{Key: atom.Style.String(), Val: "width: 16px; height: 16px; vertical-align: top; margin-right: 4px;"},
		},
	}
	sha := &html.Node{
		Type: html.ElementNode, Data: atom.Code.String(),
		FirstChild: htmlg.Text(shortSHA(c.SHA)),
	}
	if c.HTMLURL != "" {
		sha = &html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Href.String(), Val: c.HTMLURL},
			},
			FirstChild: sha,
		}
	}
	message := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "margin-left: 4px;"},
			{Key: atom.Title.String(), Val: c.CommitMessage},
		},
	}
	switch c.Short {
	case false:
		message.AppendChild(htmlg.Text(firstParagraph(c.CommitMessage)))
	case true:
		message.AppendChild(htmlg.Text(shortCommit(firstParagraph(c.CommitMessage))))
	}
	return []*html.Node{avatar, sha, message}
}

func shortSHA(sha string) string {
	return sha[:8]
}

func shortCommit(s string) string {
	if len(s) <= 24 {
		return s
	}
	return s[:23] + "…"
}

// firstParagraph returns the first paragraph of text s.
func firstParagraph(s string) string {
	i := strings.Index(s, "\n\n")
	if i == -1 {
		return s
	}
	return s[:i]
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
		action := &html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			FirstChild: htmlg.Text(p.Action),
		}
		switch p.Action {
		case "edited":
			action = &html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Href.String(), Val: p.CompareHTMLURL},
				},
				FirstChild: action,
			}
		}
		title := &html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Href.String(), Val: p.PageHTMLURL},
			},
			FirstChild: &html.Node{
				Type: html.ElementNode, Data: atom.Span.String(),
				FirstChild: htmlg.Text(p.Title),
			},
		}

		div := htmlg.Div(avatar, action, htmlg.Text(" page "), title)
		div.Attr = append(div.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 4px;"})
		nodes = append(nodes, div)
	}

	return nodes
}
