package main

import (
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/idiomaticgo"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

var idiomaticGoHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Idiomatic Go</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<link href="/blog/assets/gfm/gfm.css" rel="stylesheet" type="text/css">
		<link href="/assets/idiomaticgo/style.css" rel="stylesheet" type="text/css">
		<script async src="/assets/idiomaticgo/idiomaticgo.js"></script>
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>`))

func initIdiomaticGo(issues issues.Service, notifications notifications.Service, usersService users.Service) {
	http.Handle("/idiomatic-go", userMiddleware{httputil.ErrorHandler(usersService, func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httperror.Method{Allowed: []string{"GET"}}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ Production bool }{*productionFlag}
		err := idiomaticGoHTML.Execute(w, data)
		if err != nil {
			return err
		}

		// Server-side rendering (for now).
		authenticatedUser, err := usersService.GetAuthenticated(req.Context())
		if err != nil {
			log.Println(err)
			authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
		}
		returnURL := req.RequestURI
		err = idiomaticgo.RenderBodyInnerHTML(req.Context(), w, issues, notifications, authenticatedUser, returnURL)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</body></html>`)
		return err
	})})
}
