// Package component contains individual components that can render themselves as HTML.
package component

import (
	"fmt"
	"image/color"
	"time"

	"dmitri.shuralyov.com/html/belt"
	"dmitri.shuralyov.com/state"
	"github.com/dustin/go-humanize"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Event is an event component.
type Event struct {
	Event issues.Event
}

func (e Event) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <div class="list-entry event event-{{.Type}}">
	// 	{{.Icon}}
	// 	<div class="event-header">
	// 		<img class="inline-avatar" width="16" height="16" src="{{.Actor.AvatarURL}}">
	// 		{{render (user .Actor)}} {{.Text}} {{render (time .CreatedAt)}}
	// 	</div>
	// </div>

	div := htmlg.DivClass("event-header")
	image := &html.Node{
		Type: html.ElementNode, Data: atom.Img.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "width: 16px; height: 16px; border-radius: 2px; vertical-align: middle; margin-right: 4px;"},
			{Key: atom.Src.String(), Val: e.Event.Actor.AvatarURL},
		},
	}
	div.AppendChild(image)
	htmlg.AppendChildren(div, User{e.Event.Actor}.Render()...)
	div.AppendChild(htmlg.Text(" "))
	htmlg.AppendChildren(div, e.text()...)
	div.AppendChild(htmlg.Text(" "))
	htmlg.AppendChildren(div, Time{e.Event.CreatedAt}.Render()...)

	outerDiv := htmlg.DivClass(fmt.Sprintf("list-entry event event-%s", e.Event.Type),
		e.icon(),
		div,
	)
	return []*html.Node{outerDiv}
}

func (e Event) icon() *html.Node {
	var (
		icon            *html.Node
		color           = "#767676"
		backgroundColor = "#f3f3f3"
	)
	switch e.Event.Type {
	case issues.Reopened:
		icon = octicon.PrimitiveDot()
		color, backgroundColor = "#fff", "#6cc644"
	case issues.Closed:
		icon = octicon.CircleSlash()
		color, backgroundColor = "#fff", "#bd2c00"
	case issues.Renamed:
		icon = octicon.Pencil()
	case issues.Labeled, issues.Unlabeled:
		icon = octicon.Tag()
	case issues.CommentDeleted:
		icon = octicon.X()
	default:
		icon = octicon.PrimitiveDot()
	}
	return &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "event-icon"},
			{Key: atom.Style.String(), Val: fmt.Sprintf("color: %s; background-color: %s;", color, backgroundColor)},
		},
		FirstChild: icon,
	}
}

func (e Event) text() []*html.Node {
	switch e.Event.Type {
	case issues.Reopened:
		return []*html.Node{htmlg.Text("reopened this")}
	case issues.Closed:
		ns := []*html.Node{htmlg.Text("closed this")}
		switch c := e.Event.Close.Closer.(type) {
		case issues.Change:
			ns = append(ns, htmlg.Text(" in "))
			ns = append(ns, belt.Change{
				State:   c.State,
				Title:   c.Title,
				HTMLURL: c.HTMLURL,
				Short:   true,
			}.Render()...)
		case issues.Commit:
			ns = append(ns, htmlg.Text(" in "))
			ns = append(ns, belt.Commit{
				SHA:             c.SHA,
				Message:         c.Message,
				AuthorAvatarURL: c.AuthorAvatarURL,
				HTMLURL:         c.HTMLURL,
				Short:           true,
			}.Render()...)
		}
		return ns
	case issues.Renamed:
		from := &html.Node{
			Type: html.ElementNode, Data: atom.Del.String(),
			Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "font-weight: bold;"}},
			FirstChild: htmlg.Text(e.Event.Rename.From),
		}
		to := htmlg.Strong(e.Event.Rename.To)
		return []*html.Node{htmlg.Text("changed the title "), from, htmlg.Text(" "), to}
	case issues.Labeled:
		var ns []*html.Node
		ns = append(ns, htmlg.Text("added the "))
		ns = append(ns, Label{Label: *e.Event.Label}.Render()...)
		ns = append(ns, htmlg.Text(" label"))
		return ns
	case issues.Unlabeled:
		var ns []*html.Node
		ns = append(ns, htmlg.Text("removed the "))
		ns = append(ns, Label{Label: *e.Event.Label}.Render()...)
		ns = append(ns, htmlg.Text(" label"))
		return ns
	case issues.CommentDeleted:
		return []*html.Node{htmlg.Text("deleted a comment")}
	default:
		return []*html.Node{htmlg.Text(string(e.Event.Type))}
	}
}

// IssueStateBadge is a component that displays the state of an issue
// with a badge, who opened it, and when it was opened.
type IssueStateBadge struct {
	Issue issues.Issue
}

func (i IssueStateBadge) Render() []*html.Node {
	// TODO: Make this much nicer.
	// {{render (issueBadge .State)}}
	// <span style="margin-left: 4px;">{{render (user .User)}} opened this issue {{render (time .CreatedAt)}}</span>
	var ns []*html.Node
	ns = append(ns, IssueBadge{State: i.Issue.State}.Render()...)
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "margin-left: 4px;"},
		},
	}
	htmlg.AppendChildren(span, User{i.Issue.User}.Render()...)
	span.AppendChild(htmlg.Text(" opened this issue "))
	htmlg.AppendChildren(span, Time{i.Issue.CreatedAt}.Render()...)
	ns = append(ns, span)
	return ns
}

// IssueBadge is an issue badge, displaying the issue's state.
type IssueBadge struct {
	State state.Issue
}

func (ib IssueBadge) Render() []*html.Node {
	// TODO: Make this much nicer.
	// {{if eq . "open"}}
	// 	<span style="display: inline-block; padding: 4px 6px 4px 6px; margin: 4px; color: #fff; background-color: #6cc644;"><span style="margin-right: 6px;" class="octicon octicon-issue-opened"></span>Open</span>
	// {{else if eq . "closed"}}
	// 	<span style="display: inline-block; padding: 4px 6px 4px 6px; margin: 4px; color: #fff; background-color: #bd2c00;"><span style="margin-right: 6px;" class="octicon octicon-issue-closed"></span>Closed</span>
	// {{else}}
	// 	{{.}}
	// {{end}}
	var (
		icon  *html.Node
		text  string
		color string
	)
	switch ib.State {
	case state.IssueOpen:
		icon = octicon.IssueOpened()
		text = "Open"
		color = "#6cc644"
	case state.IssueClosed:
		icon = octicon.IssueClosed()
		text = "Closed"
		color = "#bd2c00"
	default:
		return []*html.Node{htmlg.Text(string(ib.State))}
	}
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{{
			Key: atom.Style.String(),
			Val: `display: inline-block;
padding: 4px 6px 4px 6px;
margin: 4px;
color: #fff;
background-color: ` + color + `;`,
		}},
	}
	span.AppendChild(&html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "margin-right: 6px;"}},
		FirstChild: icon,
	})
	span.AppendChild(htmlg.Text(text))
	return []*html.Node{span}
}

// IssueIcon is an issue icon, displaying the issue's state.
type IssueIcon struct {
	State state.Issue
}

func (ii IssueIcon) Render() []*html.Node {
	// TODO: Make this much nicer.
	// {{if eq . "open"}}
	// 	<span style="margin-right: 6px; color: #6cc644;" class="octicon octicon-issue-opened"></span>
	// {{else if eq . "closed"}}
	// 	<span style="margin-right: 6px; color: #bd2c00;" class="octicon octicon-issue-closed"></span>
	// {{end}}
	var (
		icon  *html.Node
		color string
	)
	switch ii.State {
	case state.IssueOpen:
		icon = octicon.IssueOpened()
		color = "#6cc644"
	case state.IssueClosed:
		icon = octicon.IssueClosed()
		color = "#bd2c00"
	}
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{{
			Key: atom.Style.String(),
			Val: `margin-right: 6px;
color: ` + color + `;`,
		}},
		FirstChild: icon,
	}
	return []*html.Node{span}
}

// Label is a label component.
type Label struct {
	Label issues.Label
}

func (l Label) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <span style="...; color: {{.fontColor}}; background-color: {{.Color.HexString}};">{{.Name}}</span>
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{{
			Key: atom.Style.String(),
			Val: `display: inline-block;
font-size: 12px;
line-height: 1.2;
padding: 0px 3px 0px 3px;
border-radius: 2px;
color: ` + l.fontColor() + `;
background-color: ` + l.Label.Color.HexString() + `;`,
		}},
	}
	span.AppendChild(htmlg.Text(l.Label.Name))
	return []*html.Node{span}
}

// fontColor returns one of "#fff" or "#000", whichever is a better fit for
// the font color given the label color.
func (l Label) fontColor() string {
	// Convert label color to 8-bit grayscale, and make a decision based on that.
	switch y := color.GrayModel.Convert(l.Label.Color).(color.Gray).Y; {
	case y < 128:
		return "#fff"
	case y >= 128:
		return "#000"
	}
	panic("unreachable")
}

// User is a user component.
type User struct {
	User users.User
}

func (u User) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <a class="black" href="{{.HTMLURL}}"><strong>{{.Login}}</strong></a>
	if u.User.Login == "" {
		return []*html.Node{htmlg.Text(u.User.Name)}
	}
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "black"},
			{Key: atom.Href.String(), Val: u.User.HTMLURL},
		},
		FirstChild: htmlg.Strong(u.User.Login),
	}
	return []*html.Node{a}
}

// Avatar is an avatar component.
type Avatar struct {
	User users.User
	Size int // In pixels, e.g., 48.
}

func (a Avatar) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <a style="..." href="{{.User.HTMLURL}}" tabindex=-1>
	// 	<img style="..." width="{{.Size}}" height="{{.Size}}" src="{{.User.AvatarURL}}">
	// </a>
	return []*html.Node{{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "display: inline-block;"},
			{Key: atom.Href.String(), Val: a.User.HTMLURL},
			{Key: atom.Tabindex.String(), Val: "-1"},
		},
		FirstChild: &html.Node{
			Type: html.ElementNode, Data: atom.Img.String(),
			Attr: []html.Attribute{
				{Key: atom.Style.String(), Val: "border-radius: 3px;"},
				{Key: atom.Width.String(), Val: fmt.Sprint(a.Size)},
				{Key: atom.Height.String(), Val: fmt.Sprint(a.Size)},
				{Key: atom.Src.String(), Val: a.User.AvatarURL},
			},
		},
	}}
}

// Time component that displays human friendly relative time (e.g., "2 hours ago", "yesterday"),
// but also contains a tooltip with the full absolute time (e.g., "Jan 2, 2006, 3:04 PM MST").
//
// TODO: Factor out, it's the same as in notificationsapp.
type Time struct {
	Time time.Time
}

func (t Time) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <abbr title="{{.Format "Jan 2, 2006, 3:04 PM MST"}}">{{reltime .}}</abbr>
	abbr := &html.Node{
		Type: html.ElementNode, Data: atom.Abbr.String(),
		Attr:       []html.Attribute{{Key: atom.Title.String(), Val: t.Time.Format("Jan 2, 2006, 3:04 PM MST")}},
		FirstChild: htmlg.Text(humanize.Time(t.Time)),
	}
	return []*html.Node{abbr}
}
