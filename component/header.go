package component

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Header is a header component that displays current user and notifications.
type Header struct {
	CurrentUser       users.User
	NotificationCount uint64 // Only needed if CurrentUser.ID != 0.
	ReturnURL         string
}

// RedLogo controls whether the logo is displayed in red,
// rather than its normal color.
var RedLogo bool

// Render implements htmlg.Component.
func (h Header) Render() []*html.Node {
	// TODO: Make this much nicer.
	/*
		<style type="text/css">...</style>

		<header class="header">
			Logo{}

			<ul class="nav">
				<li class="nav"><a href="/packages">Packages</a></li>
				<li class="nav"><a href="/blog">Blog</a></li>
				<li class="nav smaller"><a href="/idiomatic-go">Idiomatic Go</a></li>
				<li class="nav"><a href="/talks">Talks</a></li>
				<li class="nav"><a href="/projects">Projects</a></li>
				<li class="nav"><a href="/resume">Resume</a></li>
				<li class="nav"><a href="/about">About</a></li>
			</ul>

			{{if h.CurrentUser.ID}}
				Notifications{Count: h.NotificationCount}
				<a class="topbar-avatar" href="{{h.CurrentUser.HTMLURL}}">
					<img class="topbar-avatar" src="{{h.CurrentUser.AvatarURL}}" title="Signed in as {{h.CurrentUser.Login}}.">
				</a>
				PostButton{Action: "/logout", Text: "Sign out", ReturnURL: h.ReturnURL}
			{{else}}
				<a class="Login" href="/login?return={{.ReturnURL}}">Sign in via URL</a>
			{{end}}
		</header>
	*/

	style := &html.Node{
		Type: html.ElementNode, Data: atom.Style.String(),
		Attr: []html.Attribute{{Key: atom.Type.String(), Val: "text/css"}},
	}
	style.AppendChild(htmlg.Text(`
header.header {
	font-family: inherit;
	font-size: 14px;
	margin-top: 30px;
	margin-bottom: 30px;
}

header.header a {
	color: rgb(35, 35, 35);
	text-decoration: none;
}
header.header a:hover {
	color: #4183c4;
}
header.header a.Login {
	color: #4183c4;
	text-decoration: none;
}
header.header a.Login:hover {
	text-decoration: underline;
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
	font-weight: bold;
}
header.header .smaller {
	font-size: 12px;
}

header.header .user {
	float: right;
	padding-top: 8px;
}`))
	if RedLogo {
		style.AppendChild(htmlg.Text(`
header.header a.Logo {
	color: red;
}
header.header a.Logo:hover {
	color: darkred;
}`))
	}

	header := &html.Node{
		Type: html.ElementNode, Data: atom.Header.String(),
		Attr: []html.Attribute{{Key: atom.Class.String(), Val: "header"}},
	}

	htmlg.AppendChildren(header, Logo{}.Render()...)

	header.AppendChild(htmlg.ULClass("nav",
		htmlg.LIClass("nav", htmlg.A("Packages", "/packages")),
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
			span := &html.Node{
				Type: html.ElementNode, Data: atom.Span.String(),
				Attr: []html.Attribute{
					{Key: atom.Style.String(), Val: "margin-right: 10px;"},
				},
			}
			htmlg.AppendChildren(span, Notifications{Count: h.NotificationCount}.Render()...)
			userSpan.AppendChild(span)
		}

		{ // TODO: topbar-avatar component.
			a := &html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Href.String(), Val: h.CurrentUser.HTMLURL},
					{Key: atom.Style.String(), Val: `margin-right: 6px;`},
				},
			}
			a.AppendChild(&html.Node{
				Type: html.ElementNode, Data: atom.Img.String(),
				Attr: []html.Attribute{
					{Key: atom.Src.String(), Val: h.CurrentUser.AvatarURL},
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
		htmlg.AppendChildren(userSpan, signOut.Render()...)
	} else {
		u := url.URL{Path: "/login"}
		if h.ReturnURL != "/" {
			u.RawQuery = url.Values{"return": {h.ReturnURL}}.Encode()
		}
		signInViaURL := &html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Class.String(), Val: "Login"},
				{Key: atom.Href.String(), Val: u.String()},
			},
			FirstChild: htmlg.Text("Sign in via URL"),
		}
		userSpan.AppendChild(signInViaURL)
	}
	header.AppendChild(userSpan)

	return []*html.Node{style, header}
}

// Notifications is an icon for displaying if user has unread notifications.
// It links to "/notifications".
type Notifications struct {
	// Count is the number of unread notifications the user has.
	Count uint64
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
	a.AppendChild(octicon.Bell())
	switch n.Count {
	case 0:
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Title.String(), Val: "You have no unread notifications."})
	case 1:
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Title.String(), Val: "You have 1 unread notification."})
	default:
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Title.String(), Val: fmt.Sprintf("You have %d unread notifications.", n.Count)})
	}
	if n.Count > 0 {
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
			{Key: atom.Class.String(), Val: "Logo"},
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

// PackageSelector ...
type PackageSelector struct {
	ImportPath string
}

// Render implements htmlg.Component.
func (h PackageSelector) Render() []*html.Node {
	// TODO: Make this much nicer.
	/*
		{{if pattern}}
			<h2>pattern</h2>
		}} else {{
			<style type="text/css">...</style>
			<h2 class="PackageSelector">
				<span>
					<a href="/...">dmitri.shuralyov.com/</a>
					<a href="/service/...">service/</a>
					<a href="/service/change/...">change/</a>
					<a href="/service/change/gerritapi/...">gerritapi</a>
				</span>
			</h2>
		{{end}}
	*/

	/*if strings.Contains(h.ImportPath, "...") {
		return []*html.Node{htmlg.H2(htmlg.Text(h.ImportPath))}
	}*/
	//if strings.HasSuffix(h.ImportPath, "/...") {
	//	h.ImportPath = h.ImportPath[:len(h.ImportPath)-len("/...")]
	//}

	switch strings.HasSuffix(h.ImportPath, "...") {
	case false:
		style := &html.Node{
			Type: html.ElementNode, Data: atom.Style.String(),
			Attr: []html.Attribute{{Key: atom.Type.String(), Val: "text/css"}},
		}
		style.AppendChild(htmlg.Text(`
.PackageSelector a {
	color: black;
	text-decoration: none;
}
.PackageSelector span:hover a {
	color: #4183c4;
}
.PackageSelector a:hover ~ a {
	color: black;
}
.PackageSelector a:hover::after {
	content: '...';
	position: absolute;
	pointer-events: none;
	width: 40px;
	background-image: linear-gradient(to right, rgba(255, 255, 255, 0.95) 40%, transparent);
}
.PackageSelector a:last-child:hover::after {
	content: '/...';
}`))

		var ns []*html.Node
		els := strings.SplitAfter(h.ImportPath, "/")
		for i, el := range els {
			if i < len(els)-1 {
				ns = append(ns, htmlg.A(el, strings.TrimPrefix(strings.Join(els[:i+1], ""), "dmitri.shuralyov.com")+"..."))
			} else {
				ns = append(ns, htmlg.A(el, strings.TrimPrefix(strings.Join(els[:i+1], ""), "dmitri.shuralyov.com")+"/..."))
			}
		}
		h2 := &html.Node{
			Type: html.ElementNode, Data: atom.H2.String(),
			Attr:       []html.Attribute{{Key: atom.Class.String(), Val: "PackageSelector"}},
			FirstChild: htmlg.Span(ns...),
		}

		return []*html.Node{style, h2}
	case true:
		style := &html.Node{
			Type: html.ElementNode, Data: atom.Style.String(),
			Attr: []html.Attribute{{Key: atom.Type.String(), Val: "text/css"}},
		}
		style.AppendChild(htmlg.Text(`
.PackageSelector a {
	color: black;
	text-decoration: none;
}
.PackageSelector span:hover a {
	color: #4183c4;
}
.PackageSelector a:hover ~ a {
	color: black;
}
.PackageSelector a:hover::after {
	content: '...';
	position: absolute;
	pointer-events: none;
	width: 40px;
	background-image: linear-gradient(to right, rgba(255, 255, 255, 0.95) 40%, transparent);
}`))

		var ns []*html.Node
		els := strings.SplitAfter(h.ImportPath, "/")
		for i, el := range els {
			if i < len(els)-1 {
				ns = append(ns, htmlg.A(el, strings.TrimPrefix(strings.Join(els[:i+1], ""), "dmitri.shuralyov.com")+"..."))
			}
		}
		h2 := &html.Node{
			Type: html.ElementNode, Data: atom.H2.String(),
			Attr: []html.Attribute{{Key: atom.Class.String(), Val: "PackageSelector"}},
		}
		h2.AppendChild(htmlg.Span(ns...))
		h2.AppendChild(htmlg.Span(htmlg.Text(els[len(els)-1])))

		return []*html.Node{style, h2}
	default:
		panic("unreachable")
	}
}
