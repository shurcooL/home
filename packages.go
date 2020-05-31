package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
)

var packagesHTML = template.Must(template.New("").Parse(`<html>
	<head>
{{.AnalyticsHTML}}		<title>Packages</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/packages/style.css" rel="stylesheet" type="text/css">
	</head>
	<body>`))

func initPackages(code *code.Service, notification notification.Service, usersService users.Service) func(w http.ResponseWriter, req *http.Request) bool {
	packagesHandler := cookieAuth{httputil.ErrorHandler(usersService, func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httperror.Method{Allowed: []string{"GET"}}
		}

		if route.HasImportPathSeparator(req.URL.Path) {
			return os.ErrNotExist
		}
		importPathPattern := "dmitri.shuralyov.com" + req.URL.Path
		if req.URL.Path == "/packages" {
			switch pattern := req.URL.Query().Get("pattern"); pattern {
			default:
				importPathPattern = pattern
			case "":
				importPathPattern = "..."
			}
		}

		authenticatedUser, err := usersService.GetAuthenticated(req.Context())
		if err != nil {
			log.Println(err)
			authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
		}
		var nc uint64
		if authenticatedUser.ID != 0 {
			nc, err = notification.CountNotifications(req.Context())
			if err != nil {
				return err
			}
		}
		returnURL := req.RequestURI

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ AnalyticsHTML template.HTML }{analyticsHTML}
		err = packagesHTML.Execute(w, data)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
		if err != nil {
			return err
		}

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

		if importPathPattern != "..." {
			err = html.Render(w, htmlg.H2(htmlg.Text(importPathPattern)))
			if err != nil {
				return err
			}
		}

		err = html.Render(w, htmlg.H3(htmlg.Text("Packages")))
		if err != nil {
			return err
		}

		dsDirs, err := code.ListDirectories(req.Context())
		if err != nil {
			return err
		}
		err = renderPackages(w, expandPattern(dsDirs, githubPackages, importPathPattern)) // We know that "dmitri.shuralyov.com/..." comes before "github.com/...", that's why dsDirs, githubPackages are guaranteed to be in alphabetical order.
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</div>`)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</body></html>`)
		return err
	})}
	http.Handle("/packages", packagesHandler)

	servePackagesMaybe := func(w http.ResponseWriter, req *http.Request) (ok bool) {
		if !strings.Contains(req.URL.Path, "...") {
			return false
		}
		packagesHandler.ServeHTTP(w, req)
		return true
	}
	return servePackagesMaybe
}

func renderPackages(w io.Writer, packages []*code.Directory) error {
	if len(packages) == 0 {
		// No packages. Let the user know via a blank slate.
		err := htmlg.RenderComponents(w, component.BlankSlate{
			Content: htmlg.Nodes{htmlg.Text("There are no packages.")},
		})
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
		err := html.Render(w, htmlg.TR(
			htmlg.TD(htmlg.A(p.ImportPath, packageHomeURL(p.ImportPath))),
			htmlg.TD(htmlg.Text(p.Package.Synopsis)),
		))
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, `</tbody></table>`)
	return err
}

// packageHomeURL returns the home URL for package with specified import path.
func packageHomeURL(importPath string) string {
	switch strings.HasPrefix(importPath, "dmitri.shuralyov.com/") {
	case true:
		return importPath[len("dmitri.shuralyov.com"):]
	case false:
		return "https://godoc.org/" + importPath
	default:
		panic("unreachable")
	}
}

// githubPackages is a hardcoded list Go packages on github.com,
// specifically, a subset of packages made by dmitshur, excluding less noteworthy ones.
// It's sorted by import path.
var githubPackages = []*code.Directory{
	//{
	//	ImportPath: "github.com/goxjs/example/motionblur",
	//	Command:    true,
	//	Synopsis:   "Render a square with and without motion blur.",
	//},
	//{
	//	ImportPath: "github.com/goxjs/example/triangle",
	//	Command:    true,
	//	Synopsis:   "Render a basic triangle.",
	//},
	{
		ImportPath: "github.com/goxjs/gl",
		RepoRoot:   "github.com/goxjs/gl",
		Package: &code.Package{
			Name:     "gl",
			Synopsis: "Package gl is a Go cross-platform binding for OpenGL, with an OpenGL ES 2-like API.",
		},
	},
	{
		ImportPath: "github.com/goxjs/gl/glutil",
		RepoRoot:   "github.com/goxjs/gl",
		Package: &code.Package{
			Name:     "glutil",
			Synopsis: "Package glutil implements OpenGL utility functions.",
		},
	},
	//{
	//	ImportPath: "github.com/goxjs/gl/test",
	//	Command:    false,
	//	Synopsis:   "Package test contains tests for goxjs/gl.",
	//},
	{
		ImportPath: "github.com/goxjs/glfw",
		RepoRoot:   "github.com/goxjs/glfw",
		Package: &code.Package{
			Name:     "glfw",
			Synopsis: "Package glfw experimentally provides a glfw-like API with desktop (via glfw) and browser (via HTML5 canvas) backends.",
		},
	},
	//{
	//	ImportPath: "github.com/goxjs/glfw/test/events",
	//	Command:    true,
	//	Synopsis:   "events hooks every available callback and outputs their arguments.",
	//},
	{
		ImportPath: "github.com/goxjs/websocket",
		RepoRoot:   "github.com/goxjs/websocket",
		Package: &code.Package{
			Name:     "websocket",
			Synopsis: "Package websocket is a Go cross-platform implementation of a client for the WebSocket protocol.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store",
	//	Command:    false,
	//	Synopsis:   "Package gps defines domain types for Go Package Store.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/assets",
	//	Command:    false,
	//	Synopsis:   "Package assets contains assets for Go Package Store.",
	//},
	{
		ImportPath: "github.com/shurcooL/Go-Package-Store/cmd/Go-Package-Store",
		RepoRoot:   "github.com/shurcooL/Go-Package-Store",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "Go Package Store displays updates for the Go packages in your GOPATH.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/component",
	//	Command:    false,
	//	Synopsis:   "Package component contains Vecty HTML components used by Go Package Store.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/frontend",
	//	Command:    true,
	//	Synopsis:   "Command frontend runs on frontend of Go Package Store.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/frontend/action",
	//	Command:    false,
	//	Synopsis:   "Package action defines actions that can be applied to the data model in store.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/frontend/model",
	//	Command:    false,
	//	Synopsis:   "Package model is a frontend data model for updates.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/frontend/store",
	//	Command:    false,
	//	Synopsis:   "Package store is a store for updates.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/presenter",
	//	Command:    false,
	//	Synopsis:   "Package presenter defines domain types for Go Package Store presenters.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/presenter/github",
	//	Command:    false,
	//	Synopsis:   "Package github provides a GitHub API-powered presenter.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/presenter/gitiles",
	//	Command:    false,
	//	Synopsis:   "Package gitiles provides a Gitiles API-powered presenter.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/updater",
	//	Command:    false,
	//	Synopsis:   "Package updater contains gps.Updater implementations.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/Go-Package-Store/workspace",
	//	Command:    false,
	//	Synopsis:   "Package workspace contains a pipeline for processing a Go workspace.",
	//},
	{
		ImportPath: "github.com/shurcooL/Hover",
		RepoRoot:   "github.com/shurcooL/Hover",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "Hover is a work-in-progress port of Hover, a game originally created by Eric Undersander in 2000.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/Hover/track",
	//	Command:    false,
	//	Synopsis:   "Package track defines Hover track data structure and provides loading functionality.",
	//},
	{
		ImportPath: "github.com/shurcooL/binstale",
		RepoRoot:   "github.com/shurcooL/binstale",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "binstale tells you whether the binaries in your GOPATH/bin are stale or up to date.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dumpargs",
	//	Command:    true,
	//	Synopsis:   "dumpargs dumps the command-line arguments.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dumpglfw3joysticks",
	//	Command:    true,
	//	Synopsis:   "dumpglfw3joysticks dumps state of attached joysticks.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/dumphttpreq",
	//	Command:    true,
	//	Synopsis:   "dumphttpreq dumps incoming HTTP requests with full detail.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/godocrouter",
	//	Command:    true,
	//	Synopsis:   "godocrouter is a reverse proxy that augments a private godoc server instance with global godoc.org instance.",
	//},
	{
		ImportPath: "github.com/shurcooL/cmd/goimporters",
		RepoRoot:   "github.com/shurcooL/cmd",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "goimporters displays an import graph of Go packages that import the specified Go package in your GOPATH workspace.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/cmd/goimportgraph",
		RepoRoot:   "github.com/shurcooL/cmd",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "goimportgraph displays an import graph within specified Go packages.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/cmd/gopathshadow",
		RepoRoot:   "github.com/shurcooL/cmd",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "gopathshadow reports if you have any shadowed Go packages in your GOPATH workspaces.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/cmd/gorepogen",
		RepoRoot:   "github.com/shurcooL/cmd",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "gorepogen generates boilerplate files for Go repositories hosted on GitHub.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/cmd/jsonfmt",
		RepoRoot:   "github.com/shurcooL/cmd",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "jsonfmt pretty-prints JSON from stdin.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/cmd/runestats",
	//	Command:    true,
	//	Synopsis:   "runestats prints counts of total and unique runes from stdin.",
	//},
	{
		ImportPath: "github.com/shurcooL/component",
		RepoRoot:   "github.com/shurcooL/component",
		Package: &code.Package{
			Name:     "component",
			Synopsis: "Package component is a collection of basic HTML components.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/eX0/eX0-go",
		RepoRoot:   "github.com/shurcooL/eX0",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "eX0-go is a work in progress Go implementation of eX0.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/eX0/eX0-go/gpc",
	//	Command:    false,
	//	Synopsis:   "Package gpc parses GPC format files.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/eX0/eX0-go/packet",
	//	Command:    false,
	//	Synopsis:   "Package packet is for TCP and UDP packets used in eX0 networking protocol.",
	//},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/events",
		RepoRoot:   "github.com/shurcooL/events",
		Package: &code.Package{
			Name:     "events",
			Synopsis: "Package events provides an events service definition.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/events/event",
		RepoRoot:   "github.com/shurcooL/events",
		Package: &code.Package{
			Name:     "event",
			Synopsis: "Package event defines event types.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/events/fs",
		RepoRoot:   "github.com/shurcooL/events",
		Package: &code.Package{
			Name:     "fs",
			Synopsis: "Package fs implements events.Service using a virtual filesystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/events/githubapi",
		RepoRoot:   "github.com/shurcooL/events",
		Package: &code.Package{
			Name:     "githubapi",
			Synopsis: "Package githubapi implements events.Service using GitHub API client.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/frontend/checkbox",
		RepoRoot:   "github.com/shurcooL/frontend",
		Package: &code.Package{
			Name:     "checkbox",
			Synopsis: "Package checkbox provides a checkbox connected to a query parameter.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/frontend/reactionsmenu",
		RepoRoot:   "github.com/shurcooL/frontend",
		Package: &code.Package{
			Name:     "reactionsmenu",
			Synopsis: "Package reactionsmenu provides a reactions menu component.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/frontend/select_menu",
		RepoRoot:   "github.com/shurcooL/frontend",
		Package: &code.Package{
			Name:     "select_menu",
			Synopsis: "Package select_menu provides a select menu component.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/frontend/table-of-contents/handler",
	//	Command:    false,
	//	Synopsis:   "Package handler registers \"/table-of-contents.{js,css}\" routes on http.DefaultServeMux on init.",
	//},
	{
		ImportPath: "github.com/shurcooL/frontend/tabsupport",
		RepoRoot:   "github.com/shurcooL/frontend",
		Package: &code.Package{
			Name:     "tabsupport",
			Synopsis: "Package tabsupport offers functionality to add tab support to a textarea element.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/git-branches",
		RepoRoot:   "github.com/shurcooL/git-branches",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "git-branches is a go gettable command that displays branches with behind/ahead commit counts.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/github_flavored_markdown",
		RepoRoot:   "github.com/shurcooL/github_flavored_markdown",
		Package: &code.Package{
			Name:     "github_flavored_markdown",
			Synopsis: "Package github_flavored_markdown provides a GitHub Flavored Markdown renderer with fenced code block highlighting, clickable heading anchor links.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/github_flavored_markdown/gfmstyle",
		RepoRoot:   "github.com/shurcooL/github_flavored_markdown",
		Package: &code.Package{
			Name:     "gfmstyle",
			Synopsis: "Package gfmstyle contains CSS styles for rendering GitHub Flavored Markdown.",
		},
	},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/githubv4",
		RepoRoot:   "github.com/shurcooL/githubv4",
		Package: &code.Package{
			Name:     "githubv4",
			Synopsis: "Package githubv4 is a client library for accessing GitHub GraphQL API v4 (https://developer.github.com/v4/).",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/githubv4/example/githubv4dev",
	//	Command:    true,
	//	Synopsis:   "githubv4dev is a test program currently being used for developing githubv4 package.",
	//},
	{
		ImportPath: "github.com/shurcooL/go-goon",
		RepoRoot:   "github.com/shurcooL/go-goon",
		Package: &code.Package{
			Name:     "goon",
			Synopsis: "Package goon is a deep pretty printer with Go-like notation.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go-goon/bypass",
		RepoRoot:   "github.com/shurcooL/go-goon",
		Package: &code.Package{
			Name:     "bypass",
			Synopsis: "Package bypass allows bypassing reflect restrictions on accessing unexported struct fields.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/browser",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "browser",
			Synopsis: "Package browser provides utilities for interacting with users' browsers.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/gddo",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "gddo",
			Synopsis: "Package gddo is a simple client library for accessing the godoc.org API.",
		},
	},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/go/generated",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "generated",
			Synopsis: "Package generated provides a function that parses a Go file and reports whether it contains a \"// Code generated … DO NOT EDIT.\" line comment.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/gfmutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "gfmutil",
			Synopsis: "Package gfmutil offers functionality to render GitHub Flavored Markdown to io.Writer.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/gopherjs_http",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "gopherjs_http",
			Synopsis: "Package gopherjs_http provides helpers for compiling Go using GopherJS and serving it over HTTP.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/gopherjs_http/jsutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "jsutil",
			Synopsis: "Package jsutil provides utility functions for interacting with native JavaScript APIs.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/importgraphutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "importgraphutil",
			Synopsis: "Package importgraphutil augments \"golang.org/x/tools/refactor/importgraph\" with a way to build graphs ignoring tests.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/indentwriter",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "indentwriter",
			Synopsis: "Package indentwriter implements an io.Writer wrapper that indents every non-empty line with specified number of tabs.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/open",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "open",
			Synopsis: "Package open offers ability to open files or URLs as if user double-clicked it in their OS.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/openutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "openutil",
			Synopsis: "Package openutil displays Markdown or HTML in a new browser tab.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/ospath",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "ospath",
			Synopsis: "Package ospath provides utilities to get OS-specific directories.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/osutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "osutil",
			Synopsis: "Package osutil offers a utility for manipulating a set of environment variables.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/parserutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "parserutil",
			Synopsis: "Package parserutil offers convenience functions for parsing Go code to AST.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/pipeutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "pipeutil",
			Synopsis: "Package pipeutil provides additional functionality for gopkg.in/pipe.v2 package.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/printerutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "printerutil",
			Synopsis: "Package printerutil provides formatted printing of AST nodes.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/reflectfind",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "reflectfind",
			Synopsis: "Package reflectfind offers funcs to perform deep-search via reflect to find instances that satisfy given query.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/reflectsource",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "reflectsource",
			Synopsis: "Package sourcereflect implements run-time source reflection, allowing a program to look up string representation of objects from the underlying .go source files.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/timeutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "timeutil",
			Synopsis: "Package timeutil provides a func for getting start of week of given time.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/trash",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "trash",
			Synopsis: "Package trash implements functionality to move files into trash.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/trim",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "trim",
			Synopsis: "Package trim contains helpers for trimming strings.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/vfs/godocfs/godocfs",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "godocfs",
			Synopsis: "Package godocfs implements vfs.FileSystem using a http.FileSystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/vfs/godocfs/html/vfstemplate",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "vfstemplate",
			Synopsis: "Package vfstemplate offers html/template helpers that use vfs.FileSystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/vfs/godocfs/path/vfspath",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "vfspath",
			Synopsis: "Package vfspath implements utility routines for manipulating virtual file system paths.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/go/vfs/godocfs/vfsutil",
		RepoRoot:   "github.com/shurcooL/go",
		Package: &code.Package{
			Name:     "vfsutil",
			Synopsis: "Package vfsutil implements some I/O utility functions for vfs.FileSystem.",
		},
	},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/godecl",
		RepoRoot:   "github.com/shurcooL/godecl",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "A godecl experiment.",
		},
	},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/godecl/decl",
		RepoRoot:   "github.com/shurcooL/godecl",
		Package: &code.Package{
			Name:     "decl",
			Synopsis: "Package decl implements functionality to convert fragments of Go code to an English representation.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/goexec",
		RepoRoot:   "github.com/shurcooL/goexec",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "goexec is a command line tool to execute Go code.",
		},
	},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/gofontwoff",
		RepoRoot:   "github.com/shurcooL/gofontwoff",
		Package: &code.Package{
			Name:     "gofontwoff",
			Synopsis: "Package gofontwoff provides the Go font family in Web Open Font Format.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/gopherjslib",
		RepoRoot:   "github.com/shurcooL/gopherjslib",
		Package: &code.Package{
			Name:     "gopherjslib",
			Synopsis: "Package gopherjslib provides helpers for in-process GopherJS compilation.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/gostatus",
		RepoRoot:   "github.com/shurcooL/gostatus",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "gostatus is a command line tool that shows the status of Go repositories.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/gostatus/status",
	//	Command:    false,
	//	Synopsis:   "Package status provides a func to check if two repo URLs are equal in the context of Go packages.",
	//},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/graphql",
		RepoRoot:   "github.com/shurcooL/graphql",
		Package: &code.Package{
			Name:     "graphql",
			Synopsis: "Package graphql provides a GraphQL client implementation.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/graphql/example/graphqldev",
	//	Command:    true,
	//	Synopsis:   "graphqldev is a test program currently being used for developing graphql package.",
	//},
	{
		ImportPath: "github.com/shurcooL/graphql/ident",
		RepoRoot:   "github.com/shurcooL/graphql",
		Package: &code.Package{
			Name:     "ident",
			Synopsis: "Package ident provides functions for parsing and converting identifier names between various naming convention.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/gtdo",
		RepoRoot:   "github.com/shurcooL/gtdo",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "gtdo is the source for gotools.org.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/gtdo/assets",
	//	Command:    false,
	//	Synopsis:   "Package assets contains assets for gtdo.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/gtdo/gtdo",
	//	Command:    false,
	//	Synopsis:   "Package gtdo contains common gtdo-specific consts for backend and frontend.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/gtdo/page",
	//	Command:    false,
	//	Synopsis:   "Package page contains code to render pages that can be used from backend and frontend.",
	//},
	{
		ImportPath: "github.com/shurcooL/highlight_diff",
		RepoRoot:   "github.com/shurcooL/highlight_diff",
		Package: &code.Package{
			Name:     "highlight_diff",
			Synopsis: "Package highlight_diff provides syntaxhighlight.Printer and syntaxhighlight.Annotator implementations for diff format.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/highlight_go",
		RepoRoot:   "github.com/shurcooL/highlight_go",
		Package: &code.Package{
			Name:     "highlight_go",
			Synopsis: "Package highlight_go provides a syntax highlighter for Go, using go/scanner.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/home",
		RepoRoot:   "github.com/shurcooL/home",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "home is Dmitri Shuralyov's personal website.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/home/assets",
	//	Command:    false,
	//	Synopsis:   "Package assets contains assets for home.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/component",
	//	Command:    false,
	//	Synopsis:   "Package component contains individual components that can render themselves as HTML.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/exp/vec",
	//	Command:    false,
	//	Synopsis:   "Package vec provides a vecty-like API for backend HTML rendering.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/exp/vec/attr",
	//	Command:    false,
	//	Synopsis:   "Package attr defines functions to set attributes of an HTML node.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/home/exp/vec/elem",
	//	Command:    false,
	//	Synopsis:   "Package elem defines functions to create HTML elements.",
	//},
	{
		ImportPath: "github.com/shurcooL/home/http",
		RepoRoot:   "github.com/shurcooL/home",
		Package: &code.Package{
			Name:     "http",
			Synopsis: "Package http contains service implementations over HTTP.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/home/httphandler",
		RepoRoot:   "github.com/shurcooL/home",
		Package: &code.Package{
			Name:     "httphandler",
			Synopsis: "Package httphandler contains API handlers used by home.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/home/httputil",
	//	Command:    false,
	//	Synopsis:   "Package httputil is a custom HTTP framework created specifically for home.",
	//},
	{
		ImportPath: "github.com/shurcooL/home/presentdata",
		RepoRoot:   "github.com/shurcooL/home",
		Package: &code.Package{
			Name:     "presentdata",
			Synopsis: "Package presentdata contains static data for present format.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/htmlg",
		RepoRoot:   "github.com/shurcooL/htmlg",
		Package: &code.Package{
			Name:     "htmlg",
			Synopsis: "Package htmlg contains helper funcs for generating HTML nodes and rendering them.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/httperror",
		RepoRoot:   "github.com/shurcooL/httperror",
		Package: &code.Package{
			Name:     "httperror",
			Synopsis: "Package httperror provides common basic building blocks for custom HTTP frameworks.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/filter",
		RepoRoot:   "github.com/shurcooL/httpfs",
		Package: &code.Package{
			Name:     "filter",
			Synopsis: "Package filter offers an http.FileSystem wrapper with the ability to keep or skip files.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/html/vfstemplate",
		RepoRoot:   "github.com/shurcooL/httpfs",
		Package: &code.Package{
			Name:     "vfstemplate",
			Synopsis: "Package vfstemplate offers html/template helpers that use http.FileSystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/httputil",
		RepoRoot:   "github.com/shurcooL/httpfs",
		Package: &code.Package{
			Name:     "httputil",
			Synopsis: "Package httputil implements HTTP utility functions for http.FileSystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/path/vfspath",
		RepoRoot:   "github.com/shurcooL/httpfs",
		Package: &code.Package{
			Name:     "vfspath",
			Synopsis: "Package vfspath implements utility routines for manipulating virtual file system paths.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/union",
		RepoRoot:   "github.com/shurcooL/httpfs",
		Package: &code.Package{
			Name:     "union",
			Synopsis: "Package union offers a simple http.FileSystem that can unify multiple filesystems at various mount points.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/httpfs/vfsutil",
		RepoRoot:   "github.com/shurcooL/httpfs",
		Package: &code.Package{
			Name:     "vfsutil",
			Synopsis: "Package vfsutil implements some I/O utility functions for http.FileSystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/httpgzip",
		RepoRoot:   "github.com/shurcooL/httpgzip",
		Package: &code.Package{
			Name:     "httpgzip",
			Synopsis: "Package httpgzip provides net/http-like primitives that use gzip compression when serving HTTP requests.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/issues",
		RepoRoot:   "github.com/shurcooL/issues",
		Package: &code.Package{
			Name:     "issues",
			Synopsis: "Package issues provides an issues service definition.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/issues/asanaapi",
		RepoRoot:   "github.com/shurcooL/issues",
		Package: &code.Package{
			Name:     "asanaapi",
			Synopsis: "Package asanaapi implements issues.Service using Asana API client.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/issues/fs",
		RepoRoot:   "github.com/shurcooL/issues",
		Package: &code.Package{
			Name:     "fs",
			Synopsis: "Package fs implements issues.Service using a filesystem.",
		},
	},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/issues/githubapi",
		RepoRoot:   "github.com/shurcooL/issues",
		Package: &code.Package{
			Name:     "githubapi",
			Synopsis: "Package githubapi implements issues.Service using GitHub API clients.",
		},
	},
	{
		//New:        true,
		ImportPath: "github.com/shurcooL/issues/maintner",
		RepoRoot:   "github.com/shurcooL/issues",
		Package: &code.Package{
			Name:     "maintner",
			Synopsis: "Package maintner implements a read-only issues.Service using a x/build/maintner corpus.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/issuesapp",
		RepoRoot:   "github.com/shurcooL/issuesapp",
		Package: &code.Package{
			Name:     "issuesapp",
			Synopsis: "Package issuesapp is a web frontend for an issues service.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/assets",
	//	Command:    false,
	//	Synopsis:   "Package assets contains assets for issuesapp.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/common",
	//	Command:    false,
	//	Synopsis:   "Package common contains common code for backend and frontend.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/component",
	//	Command:    false,
	//	Synopsis:   "Package component contains individual components that can render themselves as HTML.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/frontend",
	//	Command:    true,
	//	Synopsis:   "frontend script for issuesapp.",
	//},
	{
		ImportPath: "github.com/shurcooL/issuesapp/httpclient",
		RepoRoot:   "github.com/shurcooL/issuesapp",
		Package: &code.Package{
			Name:     "httpclient",
			Synopsis: "Package httpclient contains issues.Service implementation over HTTP.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/issuesapp/httphandler",
		RepoRoot:   "github.com/shurcooL/issuesapp",
		Package: &code.Package{
			Name:     "httphandler",
			Synopsis: "Package httphandler contains an API handler for issues.Service.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/issuesapp/httproute",
	//	Command:    false,
	//	Synopsis:   "Package httproute contains route paths for httpclient, httphandler.",
	//},
	{
		ImportPath: "github.com/shurcooL/ivybrowser",
		RepoRoot:   "github.com/shurcooL/ivybrowser",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "ivy in the browser.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/markdownfmt",
		RepoRoot:   "github.com/shurcooL/markdownfmt",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "markdownfmt formats Markdown.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/markdownfmt/markdown",
		RepoRoot:   "github.com/shurcooL/markdownfmt",
		Package: &code.Package{
			Name:     "markdown",
			Synopsis: "Package markdown provides a Markdown renderer.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/notifications",
		RepoRoot:   "github.com/shurcooL/notifications",
		Package: &code.Package{
			Name:     "notifications",
			Synopsis: "Package notifications provides a notifications service definition.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/notifications/fs",
		RepoRoot:   "github.com/shurcooL/notifications",
		Package: &code.Package{
			Name:     "fs",
			Synopsis: "Package fs implements notifications.Service using a virtual filesystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/notifications/githubapi",
		RepoRoot:   "github.com/shurcooL/notifications",
		Package: &code.Package{
			Name:     "githubapi",
			Synopsis: "Package githubapi implements notifications.Service using GitHub API clients.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/notificationsapp",
		RepoRoot:   "github.com/shurcooL/notificationsapp",
		Package: &code.Package{
			Name:     "notificationsapp",
			Synopsis: "Package notificationsapp is a web frontend for a notifications service.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/notificationsapp/assets",
	//	Command:    false,
	//	Synopsis:   "Package assets contains assets for notificationsapp.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/notificationsapp/component",
	//	Command:    false,
	//	Synopsis:   "Package component contains individual components that can render themselves as HTML.",
	//},
	//{
	//	ImportPath: "github.com/shurcooL/notificationsapp/frontend",
	//	Command:    true,
	//	Synopsis:   "frontend script for notificationsapp.",
	//},
	{
		ImportPath: "github.com/shurcooL/notificationsapp/httpclient",
		RepoRoot:   "github.com/shurcooL/notificationsapp",
		Package: &code.Package{
			Name:     "httpclient",
			Synopsis: "Package httpclient contains notifications.Service implementation over HTTP.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/notificationsapp/httphandler",
		RepoRoot:   "github.com/shurcooL/notificationsapp",
		Package: &code.Package{
			Name:     "httphandler",
			Synopsis: "Package httphandler contains an API handler for notifications.Service.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/notificationsapp/httproute",
	//	Command:    false,
	//	Synopsis:   "Package httproute contains route paths for httpclient, httphandler.",
	//},
	{
		ImportPath: "github.com/shurcooL/octicon",
		RepoRoot:   "github.com/shurcooL/octicon",
		Package: &code.Package{
			Name:     "octicon",
			Synopsis: "Package octicon provides GitHub Octicons.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/reactions",
		RepoRoot:   "github.com/shurcooL/reactions",
		Package: &code.Package{
			Name:     "reactions",
			Synopsis: "Package reactions provides a reactions service definition.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/reactions/component",
		RepoRoot:   "github.com/shurcooL/reactions",
		Package: &code.Package{
			Name:     "component",
			Synopsis: "Package component contains individual components that can render themselves as HTML.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/reactions/emojis",
		RepoRoot:   "github.com/shurcooL/reactions",
		Package: &code.Package{
			Name:     "emojis",
			Synopsis: "Package emojis contains emojis image data.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/reactions/fs",
		RepoRoot:   "github.com/shurcooL/reactions",
		Package: &code.Package{
			Name:     "fs",
			Synopsis: "Package fs implements reactions.Service using a virtual filesystem.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/reactions/mousemoveclick",
	//	Command:    true,
	//	Synopsis:   "mousemoveclick is a script to demonstrate a peculiar browser behavior on iOS.",
	//},
	{
		ImportPath: "github.com/shurcooL/resume",
		RepoRoot:   "github.com/shurcooL/resume",
		Package: &code.Package{
			Name:     "resume",
			Synopsis: "Package resume contains Dmitri Shuralyov's résumé.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/resume/component",
	//	Command:    false,
	//	Synopsis:   "Package component contains individual components that can render themselves as HTML.",
	//},
	{
		ImportPath: "github.com/shurcooL/sanitized_anchor_name",
		RepoRoot:   "github.com/shurcooL/sanitized_anchor_name",
		Package: &code.Package{
			Name:     "sanitized_anchor_name",
			Synopsis: "Package sanitized_anchor_name provides a func to create sanitized anchor names.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/tictactoe",
		RepoRoot:   "github.com/shurcooL/tictactoe",
		Package: &code.Package{
			Name:     "tictactoe",
			Synopsis: "Package tictactoe defines the game of tic-tac-toe.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/tictactoe/cmd/tictactoe",
		RepoRoot:   "github.com/shurcooL/tictactoe",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "tictactoe plays a game of tic-tac-toe with two players.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/tictactoe/player/bad",
		RepoRoot:   "github.com/shurcooL/tictactoe",
		Package: &code.Package{
			Name:     "bad",
			Synopsis: "Package bad contains a bad tic-tac-toe player.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/tictactoe/player/random",
		RepoRoot:   "github.com/shurcooL/tictactoe",
		Package: &code.Package{
			Name:     "random",
			Synopsis: "Package random implements a random player of tic-tac-toe.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/trayhost",
		RepoRoot:   "github.com/shurcooL/trayhost",
		Package: &code.Package{
			Name:     "trayhost",
			Synopsis: "Package trayhost is a cross-platform Go library to place an icon in the host operating system's taskbar.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/users",
		RepoRoot:   "github.com/shurcooL/users",
		Package: &code.Package{
			Name:     "users",
			Synopsis: "Package users provides a users service definition.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/users/asanaapi",
		RepoRoot:   "github.com/shurcooL/users",
		Package: &code.Package{
			Name:     "asanaapi",
			Synopsis: "Package asanaapi implements users.Service using Asana API client.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/users/fs",
		RepoRoot:   "github.com/shurcooL/users",
		Package: &code.Package{
			Name:     "fs",
			Synopsis: "Package fs implements an in-memory users.Store backed by a virtual filesystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/users/githubapi",
		RepoRoot:   "github.com/shurcooL/users",
		Package: &code.Package{
			Name:     "githubapi",
			Synopsis: "Package githubapi implements users.Service using GitHub API client.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/vcsstate",
		RepoRoot:   "github.com/shurcooL/vcsstate",
		Package: &code.Package{
			Name:     "vcsstate",
			Synopsis: "Package vcsstate allows getting the state of version control system repositories.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/vfsgen",
		RepoRoot:   "github.com/shurcooL/vfsgen",
		Package: &code.Package{
			Name:     "vfsgen",
			Synopsis: "Package vfsgen takes an http.FileSystem (likely at `go generate` time) and generates Go code that statically implements the provided http.FileSystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/vfsgen/cmd/vfsgendev",
		RepoRoot:   "github.com/shurcooL/vfsgen",
		Package: &code.Package{
			Name:     "main",
			Synopsis: "vfsgendev is a convenience tool for using vfsgen in a common development configuration.",
		},
	},
	//{
	//	ImportPath: "github.com/shurcooL/vfsgen/test",
	//	Command:    false,
	//	Synopsis:   "Package test contains tests for virtual filesystem implementation generated by vfsgen.",
	//},
	{
		ImportPath: "github.com/shurcooL/webdavfs/vfsutil",
		RepoRoot:   "github.com/shurcooL/webdavfs",
		Package: &code.Package{
			Name:     "vfsutil",
			Synopsis: "Package vfsutil implements some I/O utility functions for webdav.FileSystem.",
		},
	},
	{
		ImportPath: "github.com/shurcooL/webdavfs/webdavfs",
		RepoRoot:   "github.com/shurcooL/webdavfs",
		Package: &code.Package{
			Name:     "webdavfs",
			Synopsis: "Package webdavfs implements webdav.FileSystem using an http.FileSystem.",
		},
	},
}

// expandPattern returns a list of Go packages matched by specified
// import path pattern, which may have the following forms:
//
// 	example.org/single/package     # a single package
// 	example.org/dir/...            # all packages beneath dir
// 	example.org/.../tools/...      # all matching packages
// 	...                            # the entire workspace
//
// A trailing slash in a pattern is ignored.
func expandPattern(part1, part2 []*code.Directory, pattern string) []*code.Directory {
	var dirs []*code.Directory
	match := matchPattern(pattern)
	for _, part := range [...][]*code.Directory{part1, part2} {
		for _, dir := range part {
			if dir.Package == nil {
				continue
			}
			if !match(dir.ImportPath) {
				continue
			}
			dirs = append(dirs, dir)
		}
	}
	return dirs
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
