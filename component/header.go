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

// Header is a header component that displays current user and notifications.
type Header struct {
	CurrentUser   users.User
	ReturnURL     string
	Notifications notifications.Service
}

// RenderContext implements htmlg.ComponentContext.
func (h Header) RenderContext(ctx context.Context) []*html.Node {
	// TODO: Make this much nicer.
	/*
		<style type="text/css">...</style>

		<header class="header">
			Logo{}

			<ul class="nav">
				<li class="nav"><a href="/blog">Blog</a></li>
				<li class="nav smaller"><a href="/idiomatic-go">Idiomatic Go</a></li>
				<li class="nav"><a href="/talks">Talks</a></li>
				<li class="nav"><a href="/projects">Projects</a></li>
				<li class="nav"><a href="/resume">Resume</a></li>
				<li class="nav"><a href="/about">About</a></li>
			</ul>

			{{if h.CurrentUser.ID}}
				Notifications{Unread: h.Notifications.Count() > 0}
				<a class="topbar-avatar" href="{{h.CurrentUser.HTMLURL}}" tabindex=-1>
					<img class="topbar-avatar" src="{{h.CurrentUser.AvatarURL}}" title="Signed in as {{h.CurrentUser.Login}}.">
				</a>
				PostButton{Action: "/logout", Text: "Sign out", ReturnURL: h.ReturnURL}
			{{else}}
				PostButton{Action: "/login/github", Text: "Sign in via GitHub", ReturnURL: h.ReturnURL}
			{{end}}
		</header>
	*/

	style := &html.Node{
		Type: html.ElementNode, Data: atom.Style.String(),
		Attr: []html.Attribute{{Key: atom.Type.String(), Val: "text/css"}},
	}
	style.AppendChild(htmlg.Text(`
header.header {
	font-family: sans-serif;
	font-size: 14px;
	margin-top: 30px;
	margin-bottom: 30px;
}

header.header a {
	color: black;
	text-decoration: none;
	font-weight: bold;
}
header.header a:hover {
	color: #4183c4;
}

header.header ul.nav {
	display: inline-block;
	margin-top: 0;
	margin-bottom: 0;
	padding-left: 0;
}
header.header li.nav {
	display: inline-block;
	margin-left: 20px;
}
header.header .nav.smaller {
	font-size: smaller;
}

header.header .user {
	float: right;
	padding-top: 8px;
}`))

	header := &html.Node{
		Type: html.ElementNode, Data: atom.Header.String(),
		Attr: []html.Attribute{{Key: atom.Class.String(), Val: "header"}},
	}

	for _, n := range (Logo{}).Render() {
		header.AppendChild(n)
	}

	header.AppendChild(htmlg.ULClass("nav",
		htmlg.LIClass("nav", htmlg.A("Blog", "/blog")),
		htmlg.LIClass("nav smaller", htmlg.A("Idiomatic Go", "/idiomatic-go")),
		htmlg.LIClass("nav", htmlg.A("Talks", "/talks")),
		htmlg.LIClass("nav", htmlg.A("Projects", "/projects")),
		htmlg.LIClass("nav", htmlg.A("Resume", "/resume")),
		htmlg.LIClass("nav", htmlg.A("About", "/about")),
	))

	userSpan := htmlg.SpanClass("user")
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
			userSpan.AppendChild(span)
		}

		{ // TODO: topbar-avatar component.
			a := &html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Href.String(), Val: string(h.CurrentUser.HTMLURL)},
					{Key: atom.Tabindex.String(), Val: "-1"},
					{Key: atom.Style.String(), Val: `margin-right: 6px;`},
				},
			}
			a.AppendChild(&html.Node{
				Type: html.ElementNode, Data: atom.Img.String(),
				Attr: []html.Attribute{
					{Key: atom.Src.String(), Val: string(h.CurrentUser.AvatarURL)},
					{Key: atom.Title.String(), Val: fmt.Sprintf("Signed in as %s.", h.CurrentUser.Login)},
					{Key: atom.Style.String(), Val: `border-radius: 2px;
width: 18px;
height: 18px;
vertical-align: top;`},
				},
			})
			userSpan.AppendChild(a)
		}

		signOut := PostButton{Action: "/logout", Text: "Sign out", ReturnURL: h.ReturnURL}
		for _, n := range signOut.Render() {
			userSpan.AppendChild(n)
		}
	} else {
		signInViaGitHub := PostButton{Action: "/login/github", Text: "Sign in via GitHub", ReturnURL: h.ReturnURL}
		for _, n := range signInViaGitHub.Render() {
			userSpan.AppendChild(n)
		}
	}
	header.AppendChild(userSpan)

	return []*html.Node{style, header}
}

// Notifications is an icon for displaying if user has unread notifications.
// It links to "/notifications".
type Notifications struct {
	// Unread is whether the user has unread notifications.
	Unread bool
}

// Render implements htmlg.Component.
func (n Notifications) Render() []*html.Node {
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Href.String(), Val: "/notifications"},
			{Key: atom.Style.String(), Val: `display: inline-block;
vertical-align: top;
position: relative;`},
		},
	}
	a.AppendChild(octiconssvg.Bell())
	if n.Unread {
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Title.String(), Val: "You have unread notifications."})
		notificationsUnread := &html.Node{
			Type: html.ElementNode, Data: atom.Span.String(),
			Attr: []html.Attribute{
				{Key: atom.Style.String(), Val: `display: inline-block;
width: 10px;
height: 10px;
background-color: #4183c4;
border: 2px solid white;
border-radius: 50%;
position: absolute;
right: -4px;
top: -6px;`},
			},
		}
		a.AppendChild(notificationsUnread)
	}
	return []*html.Node{a}
}

// Logo is a logo component. It links to "/".
type Logo struct{}

// Render implements htmlg.Component.
func (Logo) Render() []*html.Node {
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Href.String(), Val: "/"},
			{Key: atom.Style.String(), Val: "display: inline-block;"},
		},
	}
	svg := &html.Node{
		Type: html.ElementNode, Data: atom.Svg.String(),
		Attr: []html.Attribute{
			{Key: "xmlns", Val: "http://www.w3.org/2000/svg"},
			{Key: "viewBox", Val: "0 0 200 200"},
			{Key: atom.Width.String(), Val: "32"},
			{Key: atom.Height.String(), Val: "32"},
			{Key: atom.Style.String(), Val: `fill: currentColor;
stroke: currentColor;
vertical-align: middle;`}, // THINK: Is this right scope?
		},
	}
	svg.AppendChild(&html.Node{
		Type: html.ElementNode, Data: "circle",
		Attr: []html.Attribute{
			{Key: "cx", Val: "100"},
			{Key: "cy", Val: "100"},
			{Key: "r", Val: "90"},
			{Key: "stroke-width", Val: "20"},
			{Key: "fill", Val: "none"},
		},
	})
	svg.AppendChild(&html.Node{
		Type: html.ElementNode, Data: "circle",
		Attr: []html.Attribute{
			{Key: "cx", Val: "100"},
			{Key: "cy", Val: "100"},
			{Key: "r", Val: "60"},
		},
	})
	a.AppendChild(svg)
	return []*html.Node{a}
}
