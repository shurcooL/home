package main

import (
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/page/idiomaticgo"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

var idiomaticGoHTML = template.Must(template.New("").Parse(`<html>
	<head>
{{.AnalyticsHTML}}		<title>Idiomatic Go</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/blog/assets/gfm/gfm.css" rel="stylesheet" type="text/css">
		<style type="text/css">
			.markdown-body { font-family: Go; }
			tt, code, pre  { font-family: "Go Mono"; }
		</style>
		<link href="/assets/idiomaticgo/style.css" rel="stylesheet" type="text/css">
		<script async src="/assets/idiomaticgo/idiomaticgo.js"></script>
	</head>
	<body>`))

func initIdiomaticGo(issues issues.Service, notifications notifications.Service, usersService users.Service) {
	http.Handle("/idiomatic-go", cookieAuth{httputil.ErrorHandler(usersService, func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httperror.Method{Allowed: []string{"GET"}}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ AnalyticsHTML template.HTML }{analyticsHTML}
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
