package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

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
		<link href="/assets/packages/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>`))

func initPackages(notifications notifications.Service, usersService users.Service) {
	http.Handle("/packages", userMiddleware{httputil.ErrorHandler(usersService, func(w http.ResponseWriter, req *http.Request) error {
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

		// Render the header.
		authenticatedUser, err := usersService.GetAuthenticated(req.Context())
		if err != nil {
			log.Println(err)
			authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
		}
		returnURL := req.RequestURI
		header := component.Header{
			CurrentUser:   authenticatedUser,
			ReturnURL:     returnURL,
			Notifications: notifications,
		}
		err = htmlg.RenderComponentsContext(req.Context(), w, header)
		if err != nil {
			return err
		}

		var commands bool
		switch req.URL.Query().Get("type") {
		default:
			commands = false
		case "command":
			commands = true
		}

		// Render the tabnav.
		err = htmlg.RenderComponents(w, tabnav{
			Tabs: []tab{
				{
					Content:  iconText{Icon: octiconssvg.Package, Text: fmt.Sprintf("%d Libraries", librariesCount)},
					URL:      "/packages?type=library",
					Selected: !commands,
				},
				{
					Content:  iconText{Icon: octiconssvg.Gist, Text: fmt.Sprintf("%d Commands", commandsCount)},
					URL:      "/packages?type=command",
					Selected: commands,
				},
			},
		})
		if err != nil {
			return err
		}

		// Render the table.
		io.WriteString(w, `<table class="table table-sm">
			<thead>
				<tr>
					<th>Path</th>
					<th>Synopsis</th>
				</tr>
			</thead>
			<tbody>`)
		for _, p := range packages {
			if p.Command != commands {
				continue
			}
			html.Render(w, htmlg.TR(
				htmlg.TD(htmlg.A(p.ImportPath, template.URL("https://godoc.org/"+p.ImportPath))),
				htmlg.TD(htmlg.Text(p.Doc)),
			))
		}
		io.WriteString(w, `</tbody></table>`)

		_, err = io.WriteString(w, `</div>`)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</body></html>`)
		return err
	})})
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
		for _, n := range t.Render() {
			nav.AppendChild(n)
		}
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
	for _, n := range t.Content.Render() {
		a.AppendChild(n)
	}
	return []*html.Node{a}
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

var librariesCount, commandsCount int

func init() {
	for _, p := range packages {
		switch p.Command {
		case false:
			librariesCount++
		case true:
			commandsCount++
		}
	}
}

var packages = []struct {
	ImportPath string
	Command    bool
	Doc        string
}{
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
	{
		ImportPath: "github.com/shurcooL/binstale",
		Command:    true,
		Doc:        "binstale tells you whether the binaries in your GOPATH/bin are stale or up to date.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dump_args",
	//	Command:    true,
	//	Doc:        "dump_args dumps the command-line arguments.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dump_glfw3_joysticks",
	//	Command:    true,
	//	Doc:        "dump_glfw3_joysticks dumps state of attached joysticks.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dump_httpreq",
	//	Command:    true,
	//	Doc:        "dump_httpreq dumps incoming HTTP requests with full detail.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/godoc_router",
	//	Command:    true,
	//	Doc:        "godoc_router is a reverse proxy that augments a private godoc server instance with global godoc.org instance.",
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
	//	ImportPath: "github.com/shurcooL/cmd/rune_stats",
	//	Command:    true,
	//	Doc:        "rune_stats prints counts of total and unique runes from stdin.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/table",
	//	Command:    true,
	//	Doc:        "table is a chef client command-line tool.",
	//},
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
		ImportPath: "github.com/shurcooL/frontend/checkbox",
		Command:    false,
		Doc:        "Package checkbox provides a checkbox connected to a query parameter.",
	},
	{
		ImportPath: "github.com/shurcooL/frontend/reactionsmenu",
		Command:    false,
		Doc:        "Package reactionsmenu provides a reactions menu component.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/frontend/select_menu",
	//	Command:    false,
	//	Doc:        "",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/frontend/table-of-contents/handler",
	//	Command:    false,
	//	Doc:        "",
	//},
	{
		ImportPath: "github.com/shurcooL/frontend/tabsupport",
		Command:    false,
		Doc:        "Package tabsupport offers functionality to add tab support to a textarea element.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/gostatus/status",
	//	Command:    false,
	//	Doc:        "Package status provides a func to check if two repo URLs are equal in the context of Go packages.",
	//},
	{
		ImportPath: "github.com/shurcooL/git-branches",
		Command:    true,
		Doc:        "git-branches is a go gettable command that displays branches with behind/ahead commit counts.",
	},
	{
		ImportPath: "github.com/shurcooL/github_flavored_markdown",
		Command:    false,
		Doc:        "Package github_flavored_markdown provides a GitHub Flavored Markdown renderer with fenced code block highlighting, clickable header anchor links.",
	},
	{
		ImportPath: "github.com/shurcooL/github_flavored_markdown/gfmstyle",
		Command:    false,
		Doc:        "Package gfmstyle contains CSS styles for rendering GitHub Flavored Markdown.",
	},
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
		ImportPath: "github.com/shurcooL/go/analysis",
		Command:    false,
		Doc:        "Package analysis provides a routine that determines if a file is generated or handcrafted.",
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
		ImportPath: "github.com/shurcooL/go/httpstoppable",
		Command:    false,
		Doc:        "Package httpstoppable provides ListenAndServe like http.ListenAndServe, but with ability to stop it.",
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
		ImportPath: "github.com/shurcooL/goexec",
		Command:    true,
		Doc:        "goexec is a command line tool to execute Go code.",
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
		ImportPath: "github.com/shurcooL/gtdo",
		Command:    true,
		Doc:        "gtdo is the source for gotools.org.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/gtdo/gtdo",
	//	Command:    false,
	//	Doc:        "Package gtdo contains common gtdo-specific consts for backend and frontend.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/gtdo/internal/sanitizedanchorname",
	//	Command:    false,
	//	Doc:        "Package sanitizedanchorname provides a func to create sanitized anchor names.",
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
	{
		ImportPath: "github.com/shurcooL/home/http",
		Command:    false,
		Doc:        "Package http contains service implementations over HTTP.",
	},
	//{
	//	ImportPath: "github.com/shurcooL/home/httphandler",
	//	Command:    false,
	//	Doc:        "Package httphandler contains API handlers used by home.",
	//},
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
		ImportPath: "github.com/shurcooL/issues/githubapi",
		Command:    false,
		Doc:        "Package githubapi implements issues.Service using GitHub API client.",
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
	//	Doc:        "",
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
		Doc:        "Package githubapi implements notifications.Service using GitHub API client.",
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
	//	ImportPath: "github.com/shurcooL/notificationsapp/common",
	//	Command:    false,
	//	Doc:        "",
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
		Doc:        "Package resume is Dmitri Shuralyov's résumé.",
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
