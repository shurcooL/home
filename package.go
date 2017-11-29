package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
)

// packageHandler is a handler for a Go package index page, as well as
// its ?go-get=1 go-import meta tag page.
type packageHandler struct {
	Repo    string // Repo URI. E.g., "example.com/some/package".
	Path    string // Path corresponding to repo root, without domain. E.g., "/some/package".
	Name    string // Package name. E.g., "package".
	DocHTML string // Package doc HTML. E.g., "<p>Package package provides some functionality.</p>".

	notifications notifications.Service
	users         users.Service
}

var packageHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Package {{.Name}}</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/package/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>`))

func (h *packageHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}

	if req.URL.Query().Get("go-get") == "1" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := fmt.Fprintf(w, `<meta name="go-import" content="%[1]s git https://%[1]s">
<meta name="go-source" content="%[1]s https://%[1]s https://gotools.org/%[1]s{/dir} https://gotools.org/%[1]s{/dir}#{file}-L{line}">`, h.Repo)
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := packageHTML.Execute(w, struct {
		Production bool
		Name       string
	}{
		Production: *productionFlag,
		Name:       h.Name,
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
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

	// Render the header.
	header := component.Header{
		CurrentUser:       authenticatedUser,
		NotificationCount: nc,
		ReturnURL:         req.RequestURI,
	}
	err = htmlg.RenderComponents(w, header)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, `<h2>Package %s</h2>`, h.Name)
	if err != nil {
		return err
	}

	// Render the tabnav.
	err = htmlg.RenderComponents(w, tabnav{
		Tabs: []tab{
			{
				Content:  iconText{Icon: octiconssvg.Book, Text: "Overview"},
				URL:      h.Path,
				Selected: true,
			},
			{
				Content: iconText{Icon: octiconssvg.History, Text: "History"},
				URL:     h.Path + "/commits",
			},
			{
				Content: iconText{Icon: octiconssvg.IssueOpened, Text: "Issues"},
				URL:     h.Path + "/issues",
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, h.DocHTML)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, `<h3>Installation</h3>
			<p><pre>go get -u %[1]s</pre></p>
			<h3><a href="https://godoc.org/%[1]s">Documentation</a></h3>
			<h3><a href="https://gotools.org/%[1]s">Code</a></h3>
		</div>
	</body>
</html>`, h.Repo)
	return err
}
