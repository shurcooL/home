package main

import (
	"net/http"
	"strings"
	"time"
)

// TODO: Delete after https://github.com/golang/go/issues/18660 and https://github.com/golang/gddo/issues/468 are resolved.
func init() {
	http.HandleFunc("/temp/go-get-issue-unicode", func(w http.ResponseWriter, req *http.Request) {
		http.ServeContent(w, req, ".html", time.Time{}, strings.NewReader(`<html>
	<head>
		<meta name="go-import" content="dmitri.shuralyov.com/temp/go-get-issue-unicode git https://github.com/shurcooL-test/go-get-issue-unicode">
	</head>
	<body>
		<div><a href="https://github.com/shurcooL-test/go-get-issue-unicode">Source</a></div>
	</body>
</html>`))
	})

	http.HandleFunc("/temp/go-get-issue-unicode/испытание", func(w http.ResponseWriter, req *http.Request) {
		http.ServeContent(w, req, ".html", time.Time{}, strings.NewReader(`<html>
	<head>
		<meta name="go-import" content="dmitri.shuralyov.com/temp/go-get-issue-unicode git https://github.com/shurcooL-test/go-get-issue-unicode">
	</head>
	<body>
		<div>Install: <code>go get -u dmitri.shuralyov.com/temp/go-get-issue-unicode/испытание</code></div>
		<div><a href="https://godoc.org/dmitri.shuralyov.com/temp/go-get-issue-unicode/испытание">Documentation</a></div>
		<div><a href="https://github.com/shurcooL-test/go-get-issue-unicode/tree/master/испытание">Source</a></div>
		<div><a href="/issues/dmitri.shuralyov.com/temp/go-get-issue-unicode/испытание">Issues</a></div>
	</body>
</html>`))
	})
}
