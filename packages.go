package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var packagesHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Packages</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/packages/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>`))

func initPackages(notifications notifications.Service, usersService users.Service) {
	http.Handle("/packages", cookieAuth{httputil.ErrorHandler(usersService, func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httperror.Method{Allowed: []string{"GET"}}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ Production bool }{*productionFlag}
		err := packagesHTML.Execute(w, data)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
		if err != nil {
			return err
		}

		authenticatedUser, err := usersService.GetAuthenticated(req.Context())
		if err != nil {
			log.Println(err)
			authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
		}
		var nc uint64
		if authenticatedUser.ID != 0 {
			nc, err = notifications.Count(req.Context(), nil)
			if err != nil {
				return err
			}
		}
		returnURL := req.RequestURI

		// Render the header.
		header := component.Header{
			CurrentUser:       authenticatedUser,
			NotificationCount: nc,
			ReturnURL:         returnURL,
		}
		err = htmlg.RenderComponents(w, header)
		if err != nil {
			return err
		}

		packages := packages
		if patterns, ok := req.URL.Query()["pattern"]; ok {
			packages = expandPatterns(packages, patterns)

			err := htmlg.RenderComponents(w, patternFilter{Patterns: patterns, ClearURL: urlWith(*req.URL, "pattern", "")})
			if err != nil {
				return err
			}
		}

		var commands bool
		switch req.URL.Query().Get("type") {
		default:
			commands = false
		case "command":
			commands = true
		}

		var count struct{ Libraries, Commands int }
		for _, p := range packages {
			switch p.Command {
			case false:
				count.Libraries++
			case true:
				count.Commands++
			}
		}

		// Render the tabnav.
		err = htmlg.RenderComponents(w, tabnav{
			Tabs: []tab{
				{
					Content:  iconText{Icon: octiconssvg.Package, Text: fmt.Sprintf("%d Libraries", count.Libraries)},
					URL:      urlWith(*req.URL, "type", ""),
					Selected: !commands,
				},
				{
					Content:  iconText{Icon: octiconssvg.Gist, Text: fmt.Sprintf("%d Commands", count.Commands)},
					URL:      urlWith(*req.URL, "type", "command"),
					Selected: commands,
				},
			},
		})
		if err != nil {
			return err
		}

		err = renderPackages(w, packages, commands, count)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</div>`)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</body></html>`)
		return err
	})})
}

// patternFilter is an HTML component that displays currently applied filter,
// with a link to clear it.
type patternFilter struct {
	Patterns []string
	ClearURL string
}

func (f patternFilter) Render() []*html.Node {
	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "margin-bottom: 20px;"}},
	}
	div.AppendChild(htmlg.Strong("Filter"))
	for _, pattern := range f.Patterns {
		div.AppendChild(&html.Node{
			Type: html.ElementNode, Data: atom.Code.String(),
			Attr: []html.Attribute{{Key: atom.Style.String(), Val: `color: white;
background-color: #4183c4;
padding: 4px 8px;
border-radius: 3px;
margin: 0 4px 0 4px;`}},
			FirstChild: htmlg.Text(pattern),
		})
	}
	htmlg.AppendChildren(div, iconLink{Icon: octiconssvg.X, Text: "Clear", URL: f.ClearURL, Black: true}.Render()...)
	return []*html.Node{div}
}

func renderPackages(w io.Writer, packages []goPackage, commands bool, count struct{ Libraries, Commands int }) error {
	if !commands && count.Libraries == 0 ||
		commands && count.Commands == 0 {
		_, err := io.WriteString(w, `<div>No matching packages.</div>`)
		return err
	}

	// Render the table.
	_, err := io.WriteString(w, `<table class="table table-sm">
		<thead>
			<tr>
				<th>Path</th>
				<th>Synopsis</th>
			</tr>
		</thead>
		<tbody>`)
	if err != nil {
		return err
	}
	for _, p := range packages {
		if p.Command != commands {
			continue
		}
		path := []*html.Node{
			htmlg.A(p.ImportPath, p.HomeURL()),
		}
		if p.New {
			new := &html.Node{
				Type: html.ElementNode, Data: atom.Span.String(),
				Attr: []html.Attribute{{Key: atom.Style.String(), Val: `font-size: 10px;
vertical-align: middle;
color: #e85d00;
padding: 1px 4px;
border: 1px solid #e85d00;
border-radius: 3px;
margin-left: 6px;`}},
				FirstChild: htmlg.Text("New"),
			}
			path = append(path, new)
		}
		err := html.Render(w, htmlg.TR(
			htmlg.TD(path...),
			htmlg.TD(htmlg.Text(p.Doc)),
		))
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, `</tbody></table>`)
	return err
}

// tabnav is a left-aligned horizontal row of tabs Primer CSS component.
//
// http://primercss.io/nav/#tabnav
type tabnav struct {
	Tabs []tab
}

func (t tabnav) Render() []*html.Node {
	nav := &html.Node{
		Type: html.ElementNode, Data: atom.Nav.String(),
		Attr: []html.Attribute{{Key: atom.Class.String(), Val: "tabnav-tabs"}},
	}
	for _, t := range t.Tabs {
		htmlg.AppendChildren(nav, t.Render()...)
	}
	return []*html.Node{htmlg.DivClass("tabnav", nav)}
}

// tab is a single tab entry within a tabnav.
type tab struct {
	Content  htmlg.Component
	URL      string
	Selected bool
}

func (t tab) Render() []*html.Node {
	aClass := "tabnav-tab"
	if t.Selected {
		aClass += " selected"
	}
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Href.String(), Val: t.URL},
			{Key: atom.Class.String(), Val: aClass},
		},
	}
	htmlg.AppendChildren(a, t.Content.Render()...)
	return []*html.Node{a}
}

// urlWith returns url with query key set to value.
// If value is the empty string, query key is deleted.
func urlWith(url url.URL, key, value string) string {
	q := url.Query()
	switch value {
	default:
		q.Set(key, value)
	case "":
		q.Del(key)
	}
	url.RawQuery = q.Encode()
	return url.String()
}

// iconText is an icon with text on the right.
// Icon must be not nil.
type iconText struct {
	Icon func() *html.Node // Must be not nil.
	Text string
}

func (it iconText) Render() []*html.Node {
	icon := htmlg.Span(it.Icon())
	icon.Attr = append(icon.Attr, html.Attribute{
		Key: atom.Style.String(), Val: "margin-right: 4px;",
	})
	text := htmlg.Text(it.Text)
	return []*html.Node{icon, text}
}

type goPackage struct {
	ImportPath string
	Command    bool
	Doc        string
	New        bool // New badge.
}

func (p goPackage) HomeURL() string {
	switch strings.HasPrefix(p.ImportPath, "dmitri.shuralyov.com/") {
	case true:
		return p.ImportPath[len("dmitri.shuralyov.com"):]
	case false:
		return "https://godoc.org/" + p.ImportPath
	default:
		panic("unreachable")
	}
}

var packages = []goPackage{
	{
		New:        true,
		ImportPath: "dmitri.shuralyov.com/kebabcase",
		Command:    false,
		Doc:        "Package kebabcase provides a parser for identifier names using kebab-case naming convention.",
	},
	{
		New:        true,
		ImportPath: "dmitri.shuralyov.com/scratch",
		Command:    false,
		Doc:        "Package scratch is used for testing.",
	},
	//{
	//	ImportPath: "github.com/goxjs/example/motionblur",
	//	Command:    true,
	//	Doc:        "Render a square with and without motion blur.",
	//},
	//{
	//	ImportPath: "github.com/goxjs/example/triangle",
	//	Command:    true,
	//	Doc:        "Render a basic triangle.",
	//},
	{
		ImportPath: "github.com/goxjs/gl",
		Command:    false,
		Doc:        "Package gl is a Go cross-platform binding for OpenGL, with an OpenGL ES 2-like API.",
	},
	{
		ImportPath: "github.com/goxjs/gl/glutil",
		Command:    false,
		Doc:        "Package glutil implements OpenGL utility functions.",
	},
	//{
	//	ImportPath: "github.com/goxjs/gl/test",
	//	Command:    false,
	//	Doc:        "Package test contains tests for goxjs/gl.",
	//},
	{
		ImportPath: "github.com/goxjs/glfw",
		Command:    false,
		Doc:        "Package glfw experimentally provides a glfw-like API with desktop (via glfw) and browser (via HTML5 canvas) backends.",
	},
	//{
	//	ImportPath: "github.com/goxjs/glfw/test/events",
	//	Command:    true,
	//	Doc:        "events hooks every available callback and outputs their arguments.",
	//},
	{
		ImportPath: "github.com/goxjs/websocket",
		Command:    false,
		Doc:        "Package websocket is a Go cross-platform implementation of a client for the WebSocket protocol.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store",
	//	Command:    false,
	//	Doc:        "Package gps defines domain types for Go Package Store.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/assets",
	//	Command:    false,
	//	Doc:        "Package assets contains assets for Go Package Store.",
	//},
	{
		ImportPath: "github.com/shurcooL/Go-Package-Store/cmd/Go-Package-Store",
		Command:    true,
		Doc:        "Go Package Store displays updates for the Go packages in your GOPATH.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/component",
	//	Command:    false,
	//	Doc:        "Package component contains Vecty HTML components used by Go Package Store.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/frontend",
	//	Command:    true,
	//	Doc:        "Command frontend runs on frontend of Go Package Store.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/frontend/action",
	//	Command:    false,
	//	Doc:        "Package action defines actions that can be applied to the data model in store.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/frontend/model",
	//	Command:    false,
	//	Doc:        "Package model is a frontend data model for updates.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/frontend/store",
	//	Command:    false,
	//	Doc:        "Package store is a store for updates.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/presenter",
	//	Command:    false,
	//	Doc:        "Package presenter defines domain types for Go Package Store presenters.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/presenter/github",
	//	Command:    false,
	//	Doc:        "Package github provides a GitHub API-powered presenter.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/presenter/gitiles",
	//	Command:    false,
	//	Doc:        "Package gitiles provides a Gitiles API-powered presenter.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/updater",
	//	Command:    false,
	//	Doc:        "Package updater contains gps.Updater implementations.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/workspace",
	//	Command:    false,
	//	Doc:        "Package workspace contains a pipeline for processing a Go workspace.",
	//},
	{
		ImportPath: "github.com/shurcooL/Hover",
		Command:    true,
		Doc:        "Hover is a work-in-progress port of Hover, a game originally created by Eric Undersander in 2000.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/Hover/track",
	//	Command:    false,
	//	Doc:        "Package track defines Hover track data structure and provides loading functionality.",
	//},
	{
		ImportPath: "github.com/shurcooL/binstale",
		Command:    true,
		Doc:        "binstale tells you whether the binaries in your GOPATH/bin are stale or up to date.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dumpargs",
	//	Command:    true,
	//	Doc:        "dumpargs dumps the command-line arguments.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dumpglfw3joysticks",
	//	Command:    true,
	//	Doc:        "dumpglfw3joysticks dumps state of attached joysticks.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dumphttpreq",
	//	Command:    true,
	//	Doc:        "dumphttpreq dumps incoming HTTP requests with full detail.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/godocrouter",
	//	Command:    true,
	//	Doc:        "godocrouter is a reverse proxy that augments a private godoc server instance with global godoc.org instance.",
	//},
	{
		ImportPath: "github.com/shurcooL/cmd/goimporters",
		Command:    true,
		Doc:        "goimporters displays an import graph of Go packages that import the specified Go package in your GOPATH workspace.",
	},
	{
		ImportPath: "github.com/shurcooL/cmd/goimportgraph",
		Command:    true,
		Doc:        "goimportgraph displays an import graph within specified Go packages.",
	},
	{
		ImportPath: "github.com/shurcooL/cmd/gopathshadow",
		Command:    true,
		Doc:        "gopathshadow reports if you have any shadowed Go packages in your GOPATH workspaces.",
	},
	{
		ImportPath: "github.com/shurcooL/cmd/gorepogen",
		Command:    true,
		Doc:        "gorepogen generates boilerplate files for Go repositories hosted on GitHub.",
	},
	{
		ImportPath: "github.com/shurcooL/cmd/jsonfmt",
		Command:    true,
		Doc:        "jsonfmt pretty-prints JSON from stdin.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/runestats",
	//	Command:    true,
	//	Doc:        "runestats prints counts of total and unique runes from stdin.",
	//},
	{
		ImportPath: "github.com/shurcooL/component",
		Command:    false,
		Doc:        "Package component is a collection of basic HTML components.",
	},
	{
		ImportPath: "github.com/shurcooL/eX0/eX0-go",
		Command:    true,
		Doc:        "eX0-go is a work in progress Go implementation of eX0.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/eX0/eX0-go/gpc",
	//	Command:    false,
	//	Doc:        "Package gpc parses GPC format files.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/eX0/eX0-go/packet",
	//	Command:    false,
	//	Doc:        "Package packet is for TCP and UDP packets used in eX0 networking protocol.",
	//},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/events",
		Command:    false,
		Doc:        "Package events provides an events service definition.",
	},
	{
		ImportPath: "github.com/shurcooL/events/event",
		Command:    false,
		Doc:        "Package event defines event types.",
	},
	{
		ImportPath: "github.com/shurcooL/events/fs",
		Command:    false,
		Doc:        "Package fs implements events.Service using a virtual filesystem.",
	},
	{
		ImportPath: "github.com/shurcooL/events/githubapi",
		Command:    false,
		Doc:        "Package githubapi implements events.Service using GitHub API client.",
	},
	{
		ImportPath: "github.com/shurcooL/frontend/checkbox",
		Command:    false,
		Doc:        "Package checkbox provides a checkbox connected to a query parameter.",
	},
	{
		ImportPath: "github.com/shurcooL/frontend/reactionsmenu",
		Command:    false,
		Doc:        "Package reactionsmenu provides a reactions menu component.",
	},
	{
		ImportPath: "github.com/shurcooL/frontend/select_menu",
		Command:    false,
		Doc:        "Package select_menu provides a select menu component.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/frontend/table-of-contents/handler",
	//	Command:    false,
	//	Doc:        "Package handler registers \"/table-of-contents.{js,css}\" routes on http.DefaultServeMux on init.",
	//},
	{
		ImportPath: "github.com/shurcooL/frontend/tabsupport",
		Command:    false,
		Doc:        "Package tabsupport offers functionality to add tab support to a textarea element.",
	},
	{
		ImportPath: "github.com/shurcooL/git-branches",
		Command:    true,
		Doc:        "git-branches is a go gettable command that displays branches with behind/ahead commit counts.",
	},
	{
		ImportPath: "github.com/shurcooL/github_flavored_markdown",
		Command:    false,
		Doc:        "Package github_flavored_markdown provides a GitHub Flavored Markdown renderer with fenced code block highlighting, clickable heading anchor links.",
	},
	{
		ImportPath: "github.com/shurcooL/github_flavored_markdown/gfmstyle",
		Command:    false,
		Doc:        "Package gfmstyle contains CSS styles for rendering GitHub Flavored Markdown.",
	},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/githubql",
		Command:    false,
		Doc:        "Package githubql is a client library for accessing GitHub GraphQL API v4 (https://developer.github.com/v4/).",
	},
	//{
	//	ImportPath: "github.com/shurcooL/githubql/example/githubqldev",
	//	Command:    true,
	//	Doc:        "githubqldev is a test program currently being used for developing githubql package.",
	//},
	{
		ImportPath: "github.com/shurcooL/go-goon",
		Command:    false,
		Doc:        "Package goon is a deep pretty printer with Go-like notation.",
	},
	{
		ImportPath: "github.com/shurcooL/go-goon/bypass",
		Command:    false,
		Doc:        "Package bypass allows bypassing reflect restrictions on accessing unexported struct fields.",
	},
	{
		ImportPath: "github.com/shurcooL/go/browser",
		Command:    false,
		Doc:        "Package browser provides utilities for interacting with users' browsers.",
	},
	{
		ImportPath: "github.com/shurcooL/go/ctxhttp",
		Command:    false,
		Doc:        "Package ctxhttp provides helper functions for performing context-aware HTTP requests.",
	},
	{
		ImportPath: "github.com/shurcooL/go/gddo",
		Command:    false,
		Doc:        "Package gddo is a simple client library for accessing the godoc.org API.",
	},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/go/generated",
		Command:    false,
		Doc:        "Package generated provides a function that parses a Go file and reports whether it contains a \"// Code generated … DO NOT EDIT.\" line comment.",
	},
	{
		ImportPath: "github.com/shurcooL/go/gfmutil",
		Command:    false,
		Doc:        "Package gfmutil offers functionality to render GitHub Flavored Markdown to io.Writer.",
	},
	{
		ImportPath: "github.com/shurcooL/go/gopathutil",
		Command:    false,
		Doc:        "Package gopathutil provides tools to operate on GOPATH workspace.",
	},
	{
		ImportPath: "github.com/shurcooL/go/gopherjs_http",
		Command:    false,
		Doc:        "Package gopherjs_http provides helpers for compiling Go using GopherJS and serving it over HTTP.",
	},
	{
		ImportPath: "github.com/shurcooL/go/gopherjs_http/jsutil",
		Command:    false,
		Doc:        "Package jsutil provides utility functions for interacting with native JavaScript APIs.",
	},
	{
		ImportPath: "github.com/shurcooL/go/importgraphutil",
		Command:    false,
		Doc:        "Package importgraphutil augments \"golang.org/x/tools/refactor/importgraph\" with a way to build graphs ignoring tests.",
	},
	{
		ImportPath: "github.com/shurcooL/go/indentwriter",
		Command:    false,
		Doc:        "Package indentwriter implements an io.Writer wrapper that indents every non-empty line with specified number of tabs.",
	},
	{
		ImportPath: "github.com/shurcooL/go/ioutil",
		Command:    false,
		Doc:        "Package ioutil provides a WriteFile func with an io.Reader as input.",
	},
	{
		ImportPath: "github.com/shurcooL/go/open",
		Command:    false,
		Doc:        "Package open offers ability to open files or URLs as if user double-clicked it in their OS.",
	},
	{
		ImportPath: "github.com/shurcooL/go/openutil",
		Command:    false,
		Doc:        "Package openutil displays Markdown or HTML in a new browser tab.",
	},
	{
		ImportPath: "github.com/shurcooL/go/ospath",
		Command:    false,
		Doc:        "Package ospath provides utilities to get OS-specific directories.",
	},
	{
		ImportPath: "github.com/shurcooL/go/osutil",
		Command:    false,
		Doc:        "Package osutil offers a utility for manipulating a set of environment variables.",
	},
	{
		ImportPath: "github.com/shurcooL/go/parserutil",
		Command:    false,
		Doc:        "Package parserutil offers convenience functions for parsing Go code to AST.",
	},
	{
		ImportPath: "github.com/shurcooL/go/pipeutil",
		Command:    false,
		Doc:        "Package pipeutil provides additional functionality for gopkg.in/pipe.v2 package.",
	},
	{
		ImportPath: "github.com/shurcooL/go/printerutil",
		Command:    false,
		Doc:        "Package printerutil provides formatted printing of AST nodes.",
	},
	{
		ImportPath: "github.com/shurcooL/go/reflectfind",
		Command:    false,
		Doc:        "Package reflectfind offers funcs to perform deep-search via reflect to find instances that satisfy given query.",
	},
	{
		ImportPath: "github.com/shurcooL/go/reflectsource",
		Command:    false,
		Doc:        "Package sourcereflect implements run-time source reflection, allowing a program to look up string representation of objects from the underlying .go source files.",
	},
	{
		ImportPath: "github.com/shurcooL/go/timeutil",
		Command:    false,
		Doc:        "Package timeutil provides a func for getting start of week of given time.",
	},
	{
		ImportPath: "github.com/shurcooL/go/trash",
		Command:    false,
		Doc:        "Package trash implements functionality to move files into trash.",
	},
	{
		ImportPath: "github.com/shurcooL/go/trim",
		Command:    false,
		Doc:        "Package trim contains helpers for trimming strings.",
	},
	{
		ImportPath: "github.com/shurcooL/go/vfs/godocfs/godocfs",
		Command:    false,
		Doc:        "Package godocfs implements vfs.FileSystem using a http.FileSystem.",
	},
	{
		ImportPath: "github.com/shurcooL/go/vfs/godocfs/html/vfstemplate",
		Command:    false,
		Doc:        "Package vfstemplate offers html/template helpers that use vfs.FileSystem.",
	},
	{
		ImportPath: "github.com/shurcooL/go/vfs/godocfs/path/vfspath",
		Command:    false,
		Doc:        "Package vfspath implements utility routines for manipulating virtual file system paths.",
	},
	{
		ImportPath: "github.com/shurcooL/go/vfs/godocfs/vfsutil",
		Command:    false,
		Doc:        "Package vfsutil implements some I/O utility functions for vfs.FileSystem.",
	},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/godecl",
		Command:    true,
		Doc:        "A godecl experiment.",
	},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/godecl/decl",
		Command:    false,
		Doc:        "Package decl implements functionality to convert fragments of Go code to an English representation.",
	},
	{
		ImportPath: "github.com/shurcooL/goexec",
		Command:    true,
		Doc:        "goexec is a command line tool to execute Go code.",
	},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/gofontwoff",
		Command:    false,
		Doc:        "Package gofontwoff provides the Go font family in Web Open Font Format.",
	},
	{
		ImportPath: "github.com/shurcooL/gopherjslib",
		Command:    false,
		Doc:        "Package gopherjslib provides helpers for in-process GopherJS compilation.",
	},
	{
		ImportPath: "github.com/shurcooL/gostatus",
		Command:    true,
		Doc:        "gostatus is a command line tool that shows the status of Go repositories.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/gostatus/status",
	//	Command:    false,
	//	Doc:        "Package status provides a func to check if two repo URLs are equal in the context of Go packages.",
	//},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/graphql",
		Command:    false,
		Doc:        "Package graphql provides a GraphQL client implementation.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/graphql/example/graphqldev",
	//	Command:    true,
	//	Doc:        "graphqldev is a test program currently being used for developing graphql package.",
	//},
	{
		ImportPath: "github.com/shurcooL/graphql/ident",
		Command:    false,
		Doc:        "Package ident provides functions for parsing and converting identifier names between various naming convention.",
	},
	{
		ImportPath: "github.com/shurcooL/gtdo",
		Command:    true,
		Doc:        "gtdo is the source for gotools.org.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/gtdo/assets",
	//	Command:    false,
	//	Doc:        "Package assets contains assets for gtdo.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/gtdo/gtdo",
	//	Command:    false,
	//	Doc:        "Package gtdo contains common gtdo-specific consts for backend and frontend.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/gtdo/page",
	//	Command:    false,
	//	Doc:        "Package page contains code to render pages that can be used from backend and frontend.",
	//},
	{
		ImportPath: "github.com/shurcooL/highlight_diff",
		Command:    false,
		Doc:        "Package highlight_diff provides syntaxhighlight.Printer and syntaxhighlight.Annotator implementations for diff format.",
	},
	{
		ImportPath: "github.com/shurcooL/highlight_go",
		Command:    false,
		Doc:        "Package highlight_go provides a syntax highlighter for Go, using go/scanner.",
	},
	{
		ImportPath: "github.com/shurcooL/home",
		Command:    true,
		Doc:        "home is Dmitri Shuralyov's personal website.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/home/assets",
	//	Command:    false,
	//	Doc:        "Package assets contains assets for home.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/blog",
	//	Command:    false,
	//	Doc:        "Package blog contains functionality for rendering /blog page.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/component",
	//	Command:    false,
	//	Doc:        "Package component contains individual components that can render themselves as HTML.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/exp/vec",
	//	Command:    false,
	//	Doc:        "Package vec provides a vecty-like API for backend HTML rendering.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/exp/vec/attr",
	//	Command:    false,
	//	Doc:        "Package attr defines functions to set attributes of an HTML node.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/exp/vec/elem",
	//	Command:    false,
	//	Doc:        "Package elem defines functions to create HTML elements.",
	//},
	{
		ImportPath: "github.com/shurcooL/home/http",
		Command:    false,
		Doc:        "Package http contains service implementations over HTTP.",
	},
	{
		ImportPath: "github.com/shurcooL/home/httphandler",
		Command:    false,
		Doc:        "Package httphandler contains API handlers used by home.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/home/httputil",
	//	Command:    false,
	//	Doc:        "Package httputil is a custom HTTP framework created specifically for home.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/idiomaticgo",
	//	Command:    false,
	//	Doc:        "Package idiomaticgo contains functionality for rendering /idiomatic-go page.",
	//},
	{
		ImportPath: "github.com/shurcooL/home/presentdata",
		Command:    false,
		Doc:        "Package presentdata contains static data for present format.",
	},
	{
		ImportPath: "github.com/shurcooL/htmlg",
		Command:    false,
		Doc:        "Package htmlg contains helper funcs for generating HTML nodes and rendering them.",
	},
	{
		ImportPath: "github.com/shurcooL/httperror",
		Command:    false,
		Doc:        "Package httperror provides common basic building blocks for custom HTTP frameworks.",
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/filter",
		Command:    false,
		Doc:        "Package filter offers an http.FileSystem wrapper with the ability to keep or skip files.",
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/html/vfstemplate",
		Command:    false,
		Doc:        "Package vfstemplate offers html/template helpers that use http.FileSystem.",
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/httputil",
		Command:    false,
		Doc:        "Package httputil implements HTTP utility functions for http.FileSystem.",
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/path/vfspath",
		Command:    false,
		Doc:        "Package vfspath implements utility routines for manipulating virtual file system paths.",
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/union",
		Command:    false,
		Doc:        "Package union offers a simple http.FileSystem that can unify multiple filesystems at various mount points.",
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/vfsutil",
		Command:    false,
		Doc:        "Package vfsutil implements some I/O utility functions for http.FileSystem.",
	},
	{
		ImportPath: "github.com/shurcooL/httpgzip",
		Command:    false,
		Doc:        "Package httpgzip provides net/http-like primitives that use gzip compression when serving HTTP requests.",
	},
	{
		ImportPath: "github.com/shurcooL/issues",
		Command:    false,
		Doc:        "Package issues provides an issues service definition.",
	},
	{
		ImportPath: "github.com/shurcooL/issues/asanaapi",
		Command:    false,
		Doc:        "Package asanaapi implements issues.Service using Asana API client.",
	},
	{
		ImportPath: "github.com/shurcooL/issues/fs",
		Command:    false,
		Doc:        "Package fs implements issues.Service using a filesystem.",
	},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/issues/githubapi",
		Command:    false,
		Doc:        "Package githubapi implements issues.Service using GitHub API clients.",
	},
	{
		New:        true,
		ImportPath: "github.com/shurcooL/issues/maintner",
		Command:    false,
		Doc:        "Package maintner implements a read-only issues.Service using a x/build/maintner corpus.",
	},
	{
		ImportPath: "github.com/shurcooL/issuesapp",
		Command:    false,
		Doc:        "Package issuesapp is a web frontend for an issues service.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/assets",
	//	Command:    false,
	//	Doc:        "Package assets contains assets for issuesapp.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/common",
	//	Command:    false,
	//	Doc:        "Package common contains common code for backend and frontend.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/component",
	//	Command:    false,
	//	Doc:        "Package component contains individual components that can render themselves as HTML.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/frontend",
	//	Command:    true,
	//	Doc:        "frontend script for issuesapp.",
	//},
	{
		ImportPath: "github.com/shurcooL/issuesapp/httpclient",
		Command:    false,
		Doc:        "Package httpclient contains issues.Service implementation over HTTP.",
	},
	{
		ImportPath: "github.com/shurcooL/issuesapp/httphandler",
		Command:    false,
		Doc:        "Package httphandler contains an API handler for issues.Service.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/httproute",
	//	Command:    false,
	//	Doc:        "Package httproute contains route paths for httpclient, httphandler.",
	//},
	{
		ImportPath: "github.com/shurcooL/ivybrowser",
		Command:    true,
		Doc:        "ivy in the browser.",
	},
	{
		ImportPath: "github.com/shurcooL/markdownfmt",
		Command:    true,
		Doc:        "markdownfmt formats Markdown.",
	},
	{
		ImportPath: "github.com/shurcooL/markdownfmt/markdown",
		Command:    false,
		Doc:        "Package markdown provides a Markdown renderer.",
	},
	{
		ImportPath: "github.com/shurcooL/notifications",
		Command:    false,
		Doc:        "Package notifications provides a notifications service definition.",
	},
	{
		ImportPath: "github.com/shurcooL/notifications/fs",
		Command:    false,
		Doc:        "Package fs implements notifications.Service using a virtual filesystem.",
	},
	{
		ImportPath: "github.com/shurcooL/notifications/githubapi",
		Command:    false,
		Doc:        "Package githubapi implements notifications.Service using GitHub API clients.",
	},
	{
		ImportPath: "github.com/shurcooL/notificationsapp",
		Command:    false,
		Doc:        "Package notificationsapp is a web frontend for a notifications service.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/notificationsapp/assets",
	//	Command:    false,
	//	Doc:        "Package assets contains assets for notificationsapp.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/notificationsapp/component",
	//	Command:    false,
	//	Doc:        "Package component contains individual components that can render themselves as HTML.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/notificationsapp/frontend",
	//	Command:    true,
	//	Doc:        "frontend script for notificationsapp.",
	//},
	{
		ImportPath: "github.com/shurcooL/notificationsapp/httpclient",
		Command:    false,
		Doc:        "Package httpclient contains notifications.Service implementation over HTTP.",
	},
	{
		ImportPath: "github.com/shurcooL/notificationsapp/httphandler",
		Command:    false,
		Doc:        "Package httphandler contains an API handler for notifications.Service.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/notificationsapp/httproute",
	//	Command:    false,
	//	Doc:        "Package httproute contains route paths for httpclient, httphandler.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/octicons",
	//	Command:    false,
	//	Doc:        "Package octicons provides GitHub Octicons.",
	//},
	{
		ImportPath: "github.com/shurcooL/octiconssvg",
		Command:    false,
		Doc:        "Package octiconssvg provides GitHub Octicons in SVG format.",
	},
	{
		ImportPath: "github.com/shurcooL/reactions",
		Command:    false,
		Doc:        "Package reactions provides a reactions service definition.",
	},
	{
		ImportPath: "github.com/shurcooL/reactions/component",
		Command:    false,
		Doc:        "Package component contains individual components that can render themselves as HTML.",
	},
	{
		ImportPath: "github.com/shurcooL/reactions/emojis",
		Command:    false,
		Doc:        "Package emojis contains emojis image data.",
	},
	{
		ImportPath: "github.com/shurcooL/reactions/fs",
		Command:    false,
		Doc:        "Package fs implements reactions.Service using a virtual filesystem.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/reactions/mousemoveclick",
	//	Command:    true,
	//	Doc:        "mousemoveclick is a script to demonstrate a peculiar browser behavior on iOS.",
	//},
	{
		ImportPath: "github.com/shurcooL/resume",
		Command:    false,
		Doc:        "Package resume contains Dmitri Shuralyov's résumé.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/resume/component",
	//	Command:    false,
	//	Doc:        "Package component contains individual components that can render themselves as HTML.",
	//},
	{
		ImportPath: "github.com/shurcooL/resume/frontend",
		Command:    true,
		Doc:        "frontend renders the resume entirely on the frontend.",
	},
	{
		ImportPath: "github.com/shurcooL/sanitized_anchor_name",
		Command:    false,
		Doc:        "Package sanitized_anchor_name provides a func to create sanitized anchor names.",
	},
	{
		ImportPath: "github.com/shurcooL/tictactoe",
		Command:    false,
		Doc:        "Package tictactoe defines the game of tic-tac-toe.",
	},
	{
		ImportPath: "github.com/shurcooL/tictactoe/cmd/tictactoe",
		Command:    true,
		Doc:        "tictactoe plays a game of tic-tac-toe with two players.",
	},
	{
		ImportPath: "github.com/shurcooL/tictactoe/player/bad",
		Command:    false,
		Doc:        "Package bad contains a bad tic-tac-toe player.",
	},
	{
		ImportPath: "github.com/shurcooL/tictactoe/player/random",
		Command:    false,
		Doc:        "Package random implements a random player of tic-tac-toe.",
	},
	{
		ImportPath: "github.com/shurcooL/trayhost",
		Command:    false,
		Doc:        "Package trayhost is a cross-platform Go library to place an icon in the host operating system's taskbar.",
	},
	{
		ImportPath: "github.com/shurcooL/users",
		Command:    false,
		Doc:        "Package users provides a users service definition.",
	},
	{
		ImportPath: "github.com/shurcooL/users/asanaapi",
		Command:    false,
		Doc:        "Package asanaapi implements users.Service using Asana API client.",
	},
	{
		ImportPath: "github.com/shurcooL/users/fs",
		Command:    false,
		Doc:        "Package fs implements an in-memory users.Store backed by a virtual filesystem.",
	},
	{
		ImportPath: "github.com/shurcooL/users/githubapi",
		Command:    false,
		Doc:        "Package githubapi implements users.Service using GitHub API client.",
	},
	{
		ImportPath: "github.com/shurcooL/vcsstate",
		Command:    false,
		Doc:        "Package vcsstate allows getting the state of version control system repositories.",
	},
	{
		ImportPath: "github.com/shurcooL/vfsgen",
		Command:    false,
		Doc:        "Package vfsgen takes an http.FileSystem (likely at `go generate` time) and generates Go code that statically implements the provided http.FileSystem.",
	},
	{
		ImportPath: "github.com/shurcooL/vfsgen/cmd/vfsgendev",
		Command:    true,
		Doc:        "vfsgendev is a convenience tool for using vfsgen in a common development configuration.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/vfsgen/test",
	//	Command:    false,
	//	Doc:        "Package test contains tests for virtual filesystem implementation generated by vfsgen.",
	//},
	{
		ImportPath: "github.com/shurcooL/webdavfs/vfsutil",
		Command:    false,
		Doc:        "Package vfsutil implements some I/O utility functions for webdav.FileSystem.",
	},
	{
		ImportPath: "github.com/shurcooL/webdavfs/webdavfs",
		Command:    false,
		Doc:        "Package webdavfs implements webdav.FileSystem using an http.FileSystem.",
	},
}

// expandPatterns returns the set of Go packages matched by specified
// import path patterns, which may have the following forms:
//
//		example.org/single/package     # a single package
//		example.org/dir/...            # all packages beneath dir
//		exam.../tools/...              # all matching packages
//		...                            # the entire workspace
//
// A trailing slash in a pattern is ignored.
func expandPatterns(all []goPackage, patterns []string) []goPackage {
	pkgs := make(map[string]struct{})
	for _, pattern := range patterns {
		if pattern == "..." {
			// ... matches all packages.
			return all
		} else if strings.Contains(pattern, "...") {
			match := matchPattern(pattern)
			for _, p := range all {
				if match(p.ImportPath) {
					pkgs[p.ImportPath] = struct{}{}
				}
			}
		} else {
			// Single package.
			pkgs[strings.TrimSuffix(pattern, "/")] = struct{}{}
		}
	}
	var packages []goPackage
	for _, p := range all {
		if _, ok := pkgs[p.ImportPath]; !ok {
			continue
		}
		packages = append(packages, p)
	}
	return packages
}

// matchPattern(pattern)(name) reports whether name matches pattern.
// Pattern is a limited glob pattern in which '...' means 'any string',
// foo/... matches foo too, and there is no other special syntax.
func matchPattern(pattern string) func(name string) bool {
	re := regexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	return regexp.MustCompile(`^` + re + `$`).MatchString
}
