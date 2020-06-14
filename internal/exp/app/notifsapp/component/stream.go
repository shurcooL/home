// Package component contains individual components that can render themselves as HTML.
package component

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"dmitri.shuralyov.com/html/belt"
	"dmitri.shuralyov.com/state"
	"github.com/dustin/go-humanize"
	"github.com/shurcooL/component"
	"github.com/shurcooL/go/timeutil"
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Stream component for display purposes.
type Stream struct {
	Notifications []notification.Notification
	Error         string
	GopherBot     bool // Controls whether to show all bot comments.
}

func (s Stream) Render() []*html.Node {
	var nodes []*html.Node

	if s.Error != "" {
		nodes = append(nodes,
			&html.Node{
				Type: html.ElementNode, Data: atom.P.String(),
				Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "white-space: pre;"}},
				FirstChild: htmlg.Text(s.Error),
			},
		)
	}

	if len(s.Notifications) == 0 {
		nodes = append(nodes, homecomponent.BlankSlate{
			Content: htmlg.Nodes{htmlg.Text("No new notifications.")},
		}.Render()...)

		return []*html.Node{htmlg.DivClass("notificationStream", nodes...)}
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

	for _, n := range s.Notifications {
		if !s.GopherBot && GopherBotNotification(n) {
			continue // Skip events posted via gopherbot comments.
		}

		// Heading.
		if len(headings) > 0 && headings[0].End.After(n.Time) {
			for len(headings) >= 2 && headings[1].End.After(n.Time) {
				headings = headings[1:]
			}
			nodes = append(nodes,
				htmlg.DivClass("heading", htmlg.Text(headings[0].Text)),
			)
			headings = headings[1:]
		}

		// Notification.
		notif, ok := RenderNotification(n)
		if !ok {
			continue
		}
		nodes = append(nodes, notif.Render()...)
	}

	return []*html.Node{htmlg.DivClass("notificationStream", nodes...)}
}

// RenderNotification renders notification n into an HTML component.
func RenderNotification(n notification.Notification) (htmlg.Component, bool) {
	basicNotification := basicNotification{
		Namespace:  n.Namespace,
		ThreadType: n.ThreadType,
		ThreadID:   n.ThreadID,
		Time:       n.Time,
		Actor:      n.Actor.Login,
	}
	if n.Unread {
		basicNotification.LeftBorderColor = &RGB{R: 65, G: 131, B: 196}
	}
	if n.Mentioned {
		basicNotification.BackgroundColor = &RGB{R: 255, G: 230, B: 230}
	}

	switch p := n.Payload.(type) {
	case notification.Issue:
		e := streamNotification{
			basicNotification: &basicNotification,
			Icon:              octicon.IssueOpened,
			Action:            component.Join(p.Action, " an issue ", issueFromAction(p, n.ImportPaths[0])),
		}
		if p.Action == "reopened" {
			e.Icon = octicon.IssueReopened
		} else if p.Action == "opened" && p.IssueBody != "" {
			e.Details = imageText{
				ImageURL: n.Actor.AvatarURL,
				Text:     shortBody(p.IssueBody),
			}
		}
		return e, true
	case notification.Change:
		e := streamNotification{
			basicNotification: &basicNotification,
			Icon:              octicon.GitPullRequest,
			Action:            component.Join(p.Action, " a change ", changeFromAction(p, n.ImportPaths[0])),
		}
		if p.Action == "opened" && p.ChangeBody != "" {
			e.Details = imageText{
				ImageURL: n.Actor.AvatarURL,
				Text:     shortBody(p.ChangeBody),
			}
		}
		return e, true
	case notification.IssueComment:
		return streamNotification{
			basicNotification: &basicNotification,
			Icon:              octicon.CommentDiscussion,
			Action:            component.Join("commented on ", issueFromComment(p, n.ImportPaths[0])),
			Details: imageText{
				ImageURL: n.Actor.AvatarURL,
				Text:     shortBody(p.CommentBody),
			},
		}, true
	case notification.ChangeComment:
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
				ImageURL: n.Actor.AvatarURL,
				Text:     shortBody(p.CommentBody),
			}
		}
		return streamNotification{
			basicNotification: &basicNotification,
			Icon:              octicon.CommentDiscussion,
			Action:            component.Join(verb, " on ", changeFromComment(p, n.ImportPaths[0])),
			Details:           details,
		}, true
	default:
		log.Printf("RenderNotification: unexpected notification type: %T\n", p)
		return streamNotification{}, false
	}
}

func issueFromAction(p notification.Issue, importPath string) htmlg.Component {
	var is state.Issue
	switch p.Action {
	case "opened", "reopened":
		is = state.IssueOpen
	case "closed":
		is = state.IssueClosed
	default:
		log.Printf("issueFromAction: unsupported notification.Issue action %q\n", p.Action)
		is = state.IssueOpen
	}
	return belt.Issue{
		State:   is,
		Title:   importPath + ": " + p.IssueTitle,
		HTMLURL: p.IssueHTMLURL,
		OnClick: "Open(event, this)",
	}
}
func issueFromComment(p notification.IssueComment, importPath string) htmlg.Component {
	return belt.Issue{
		State:   p.IssueState,
		Title:   importPath + ": " + p.IssueTitle,
		HTMLURL: p.CommentHTMLURL,
		OnClick: "Open(event, this)",
	}
}
func changeFromAction(p notification.Change, importPath string) htmlg.Component {
	var cs state.Change
	switch p.Action {
	case "opened", "reopened":
		cs = state.ChangeOpen
	case "closed":
		cs = state.ChangeClosed
	case "merged":
		cs = state.ChangeMerged
	default:
		log.Printf("changeFromAction: unsupported notification.Change action %q\n", p.Action)
		cs = state.ChangeOpen
	}
	return belt.Change{
		State:   cs,
		Title:   importPath + ": " + p.ChangeTitle,
		HTMLURL: p.ChangeHTMLURL,
		OnClick: "Open(event, this)",
	}
}
func changeFromComment(p notification.ChangeComment, importPath string) htmlg.Component {
	return belt.Change{
		State:   p.ChangeState,
		Title:   importPath + ": " + p.ChangeTitle,
		HTMLURL: p.CommentHTMLURL,
		OnClick: "Open(event, this)",
	}
}

// GopherBotNotification reports whether notification n
// is known to be authored by GopherBot.
func GopherBotNotification(n notification.Notification) bool {
	switch p := n.Payload.(type) {
	case notification.IssueComment:
		if n.Actor.UserSpec == gopherbot &&
			(strings.Contains(p.CommentBody, " mentions this issue: ") ||
				strings.HasPrefix(p.CommentBody, "Closed by merging ")) {
			return true
		}
	case notification.ChangeComment:
		if n.Actor.UserSpec == gobot {
			return true
		}
	}
	return false
}

type basicNotification struct {
	Namespace  string // TODO: Move into Thread struct.
	ThreadType string
	ThreadID   uint64

	Time  time.Time
	Actor string

	LeftBorderColor *RGB // Optional left border color override.
	BackgroundColor *RGB // Optional background color override.
}

// streamNotification is a notification within the notification stream.
// Action must be not nil.
type streamNotification struct {
	*basicNotification
	Icon    func() *html.Node
	Action  htmlg.Component // Not nil.
	Details htmlg.Component
}

func (e streamNotification) Render() []*html.Node {
	if e.Icon == nil {
		e.Icon = func() *html.Node { return &html.Node{Type: html.TextNode} }
	}
	action := &html.Node{Type: html.ElementNode, Data: atom.Span.String()}
	htmlg.AppendChildren(action, e.Action.Render()...)
	var markReadStyle string // HACK
	if e.LeftBorderColor == nil {
		markReadStyle = " display: none;"
	}
	div := htmlg.DivClass("notification", htmlg.DivClass("overview",
		htmlg.SpanClass("icon", e.Icon()),
		htmlg.SpanClass("middle",
			htmlg.Text(e.Actor),
			htmlg.Text(" "),
			action,
		),
		&html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr: []html.Attribute{
				{Key: atom.Class.String(), Val: "time"},
				{Key: atom.Title.String(), Val: humanize.Time(e.Time) + " – " + e.Time.Local().Format(timeFormat)}, // TODO: Use local time of page viewer, not server.
			},
			FirstChild: htmlg.Text(compactTime(e.Time)),
		},
		htmlg.SpanClass("right", &html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Class.String(), Val: "icon"},
				{Key: atom.Style.String(), Val: "cursor: pointer;" + markReadStyle},
				{Key: atom.Onclick.String(), Val: fmt.Sprintf("MarkRead(%q, %q, %d);", e.Namespace, e.ThreadType, e.ThreadID)},
				{Key: atom.Tabindex.String(), Val: "0"},
				{Key: atom.Title.String(), Val: "Mark as read"},
			},
			FirstChild: octicon.Check(),
		}),
	))
	if e.Details != nil {
		div.AppendChild(htmlg.DivClass("details", e.Details.Render()...))
	}
	var style string
	if e.LeftBorderColor != nil {
		style += fmt.Sprintf("box-shadow: 2px 0 0 %s inset;", e.LeftBorderColor.HexString())
	}
	if e.BackgroundColor != nil {
		style += fmt.Sprintf("background-color: %s;", e.BackgroundColor.HexString())
	}
	if style != "" {
		div.Attr = append(div.Attr, html.Attribute{Key: atom.Style.String(), Val: style})
	}
	div.Attr = append(div.Attr,
		html.Attribute{Key: "data-Namespace", Val: e.Namespace},
		html.Attribute{Key: "data-ThreadType", Val: e.ThreadType},
		html.Attribute{Key: "data-ThreadID", Val: strconv.FormatUint(e.ThreadID, 10)},
	)
	return []*html.Node{div}
}

var (
	gopherbot = users.UserSpec{ID: 8566911, Domain: "github.com"}
	gobot     = users.UserSpec{ID: 5976, Domain: "go-review.googlesource.com"} // TODO: get rid of "-review", etc.
)

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
