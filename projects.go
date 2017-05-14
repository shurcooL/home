package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	pathpkg "path"
	"sort"
	"strings"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
)

var projectsHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Dmitri Shuralyov - Projects</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/projects/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>
		<div style="max-width: 800px; margin: 0 auto 100px auto;">`))

// initProjects registers a projects handler with root as projects content source.
func initProjects(mux *http.ServeMux, root http.FileSystem, notifications notifications.Service, users users.Service) {
	projectsHandler := http۰StripPrefix("/projects", userMiddleware{httputil.ErrorHandler(users, (&projectsHandler{
		fs: root,

		notifications: notifications,
		users:         users,
	}).ServeHTTP)})
	// Register "/projects/" but not "/projects", we need it to redirect to /projects/.
	mux.Handle("/projects/", projectsHandler)
}

type projectsHandler struct {
	fs http.FileSystem

	notifications notifications.Service
	users         users.Service
}

func (h *projectsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}

	path := pathpkg.Clean("/" + req.URL.Path)

	// Redirect .../index.html to .../.
	// Can't use Redirect() because that would make the path absolute,
	// which would be a problem running under StripPrefix.
	if strings.HasSuffix(path, "/index.html") {
		localRedirect(w, req, ".")
		return nil
	}

	f, err := h.fs.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	// Redirect to canonical path: / at end of directory url.
	url := req.URL.Path
	if fi.IsDir() {
		if !strings.HasSuffix(url, "/") && url != "" {
			localRedirect(w, req, pathpkg.Base(url)+"/")
			return nil
		}
	} else {
		if strings.HasSuffix(url, "/") && url != "/" {
			localRedirect(w, req, "../"+pathpkg.Base(url))
			return nil
		}
	}

	// Use contents of index.html for directory, if present.
	if fi.IsDir() {
		indexPath := pathpkg.Join(path, "index.html")
		f0, err := h.fs.Open(indexPath)
		if err == nil {
			defer f0.Close()
			fi0, err := f0.Stat()
			if err == nil {
				path = indexPath
				f = f0
				fi = fi0
			}
		}
	}

	switch fi.IsDir() {
	// Serve a directory listing.
	case true:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ Production bool }{*productionFlag}
		err := projectsHTML.Execute(w, data)
		if err != nil {
			return err
		}

		authenticatedUser, err := h.users.GetAuthenticated(req.Context())
		if err != nil {
			log.Println(err)
			authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
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
		header := component.Header{
			CurrentUser:       authenticatedUser,
			NotificationCount: nc,
			ReturnURL:         returnURL,
		}
		err = htmlg.RenderComponents(w, header)
		if err != nil {
			return err
		}

		err = html.Render(w, htmlg.H1(htmlg.Text("Projects")))
		if err != nil {
			return err
		}
		err = html.Render(w, htmlg.H2(htmlg.Text(path)))
		if err != nil {
			return err
		}

		// Render the directory listing.
		err = h.renderDir(w, f, path == "/")
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</div>`)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</body></html>`)
		return err

	// Serve regular files.
	case false:
		httpgzip.ServeContent(w, req, path, fi.ModTime(), f)
		return nil

	default:
		panic("unreachable")
	}
}

// renderDir renders to w the directory listing of d.
func (h *projectsHandler) renderDir(w io.Writer, d dirReader, root bool) error {
	fis, err := d.Readdir(0)
	if err != nil {
		return err
	}
	sort.Slice(fis, func(i, j int) bool { return fis[i].Name() < fis[j].Name() })

	fmt.Fprintln(w, "<pre>")
	switch root {
	case true:
		fmt.Fprintln(w, `<a href=".">.</a>`)
	case false:
		fmt.Fprintln(w, `<a href="..">..</a>`)
	}
	for _, fi := range fis {
		name := fi.Name()
		if fi.IsDir() {
			name += "/"
		}
		url := url.URL{Path: name}
		fmt.Fprintf(w, "<a href=\"%s\">%s</a>\n", url.String(), html.EscapeString(name))
	}
	fmt.Fprintln(w, "</pre>")
	return nil
}

// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like http.Redirect does.
func localRedirect(w http.ResponseWriter, req *http.Request, newPath string) {
	if req.URL.RawQuery != "" {
		newPath += "?" + req.URL.RawQuery
	}
	w.Header().Set("Location", newPath)
	w.WriteHeader(http.StatusMovedPermanently)
}
