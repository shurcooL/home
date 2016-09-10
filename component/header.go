package component

import (
	"context"
	"fmt"
	"log"

	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Header struct {
	MaxWidth      int
	CurrentUser   users.User
	ReturnURL     string
	Notifications notifications.Service
}

func (h Header) containerStyle() string {
	switch h.MaxWidth {
	case 0:
		return "margin-bottom: 20px; text-align: right; height: 18px; font-size: 12px;"
	default:
		return fmt.Sprintf("max-width: %dpx; margin: 0 auto 20px auto; text-align: right; height: 18px; font-size: 12px;", h.MaxWidth)
	}
}

func (h Header) Render(ctx context.Context) []*html.Node {
	// TODO: Make this much nicer.
	/*
		<div style="text-align: right; margin-bottom: 20px; height: 18px; font-size: 12px;">
			{{if h.CurrentUser.ID}}
				Notifications{Unread: h.Notifications.Count() > 0}
				<a class="topbar-avatar" href="{{h.CurrentUser.HTMLURL}}" target="_blank" tabindex=-1>
					<img class="topbar-avatar" src="{{h.CurrentUser.AvatarURL}}" title="Signed in as {{h.CurrentUser.Login}}.">
				</a>
				PostButton{Action: "/logout", Text: "Sign out", ReturnURL: h.ReturnURL}
			{{else}}
				PostButton{Action: "/login/github", Text: "Sign in via GitHub", ReturnURL: h.ReturnURL}
			{{end}}
		</div>
	*/

	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: h.containerStyle()}},
	}

	if h.CurrentUser.ID != 0 {
		{ // Notifications icon.
			n, err := h.Notifications.Count(ctx, nil)
			if err != nil {
				log.Println(err)
				n = 0
			}
			span := &html.Node{
				Type: html.ElementNode, Data: atom.Span.String(),
				Attr: []html.Attribute{
					{Key: atom.Style.String(), Val: "margin-right: 10px;"},
				},
			}
			for _, n := range (Notifications{Unread: n > 0}).Render() {
				span.AppendChild(n)
			}
			div.AppendChild(span)
		}

		{ // TODO: topbar-avatar component.
			a := &html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "topbar-avatar"},
					{Key: atom.Href.String(), Val: string(h.CurrentUser.HTMLURL)},
					{Key: atom.Target.String(), Val: "_blank"},
					{Key: atom.Tabindex.String(), Val: "-1"},
				},
			}
			a.AppendChild(&html.Node{
				Type: html.ElementNode, Data: atom.Img.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "topbar-avatar"},
					{Key: atom.Src.String(), Val: string(h.CurrentUser.AvatarURL)},
					{Key: atom.Title.String(), Val: fmt.Sprintf("Signed in as %s.", h.CurrentUser.Login)},
				},
			})
			div.AppendChild(a)
		}

		signOut := PostButton{Action: "/logout", Text: "Sign out", ReturnURL: h.ReturnURL}
		for _, n := range signOut.Render() {
			div.AppendChild(n)
		}
	} else {
		signInViaGitHub := PostButton{Action: "/login/github", Text: "Sign in via GitHub", ReturnURL: h.ReturnURL}
		for _, n := range signInViaGitHub.Render() {
			div.AppendChild(n)
		}
	}

	return []*html.Node{div}
}

// Notifications is an icon for displaying if user has unread notifications.
type Notifications struct {
	// Unread is whether the user has unread notifications.
	Unread bool
}

func (n Notifications) Render() []*html.Node {
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "notifications"}, // TODO: Factor in that CSS class's declaration block, and :hover selector.
			{Key: atom.Href.String(), Val: "/notifications"},
		},
	}
	a.AppendChild(octiconssvg.Bell())
	if n.Unread {
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Title.String(), Val: "You have unread notifications."})
		a.AppendChild(htmlg.SpanClass("notifications-unread")) // TODO: Factor in that CSS class's declaration block.
	}
	return []*html.Node{a}
}
