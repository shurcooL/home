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
	"github.com/shurcooL/users"
)

// packageHandler is a handler for a Go package index page, as well as
// its ?go-get=1 go-import meta tag page.
//
// Currently, it is hardcoded for dmitri.shuralyov.com/kebabcase repo,
// and returns an error if Repo != "dmitri.shuralyov.com/kebabcase".
type packageHandler struct {
	Repo          string // Repo URI, e.g., "example.com/some/package".
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
	if h.Repo != "dmitri.shuralyov.com/kebabcase" {
		return fmt.Errorf("wrong repo")
	}
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}

	if req.URL.Query().Get("go-get") == "1" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := io.WriteString(w, `<meta name="go-import" content="dmitri.shuralyov.com/kebabcase git https://dmitri.shuralyov.com/kebabcase">
<meta name="go-source" content="dmitri.shuralyov.com/kebabcase https://dmitri.shuralyov.com/kebabcase https://gotools.org/dmitri.shuralyov.com/kebabcase{/dir} https://gotools.org/dmitri.shuralyov.com/kebabcase{/dir}#{file}-L{line}">`)
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := packageHTML.Execute(w, struct {
		Production bool
		Name       string
	}{
		Production: *productionFlag,
		Name:       "kebabcase",
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

	_, err = io.WriteString(w, `<h2>Package kebabcase</h2>
			<p>Package kebabcase provides a parser for identifier names using kebab-case naming convention.<br>
<br>
Reference: <a href="https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers">https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers</a>.</p>
			<h3>Installation</h3>
			<p><pre>go get -u dmitri.shuralyov.com/kebabcase</pre></p>
			<h3><a href="https://godoc.org/dmitri.shuralyov.com/kebabcase">Documentation</a></h3>
			<h3><a href="https://gotools.org/dmitri.shuralyov.com/kebabcase">Code</a></h3>
			<h3><a href="/kebabcase/issues">Issues</a></h3>
		</div>
	</body>
</html>`)
	return err
}
