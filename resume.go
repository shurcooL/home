package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/page/resume"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

var resumeHTML = template.Must(template.New("").Funcs(template.FuncMap{"noescape": func(s string) template.HTML { return template.HTML(s) }}).Parse(`<html>
	<head>
{{.AnalyticsHTML}}		<title>Dmitri Shuralyov - Resume</title>
		<link href="/icon.svg" rel="icon" type="image/svg+xml">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/assets/resume/style.css" rel="stylesheet" type="text/css">
		<script>var RedLogo = {{.RedLogo}};</script>

		{{noescape "<!-- Unminified source is at https://github.com/shurcooL/resume. -->"}}
		<script async src="/assets/resume/resume.js"></script>
	</head>
	<body>`))

func initResume(reactions reactions.Service, notification notification.Service, usersService users.Service) {
	http.Handle("/resume", cookieAuth{httputil.ErrorHandler(usersService, func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httperror.Method{Allowed: []string{"GET"}}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct {
			AnalyticsHTML template.HTML
			RedLogo       bool
		}{analyticsHTML, component.RedLogo}
		err := resumeHTML.Execute(w, data)
		if err != nil {
			return err
		}

		// Optional (still experimental) server-side rendering.
		prerender, _ := strconv.ParseBool(req.URL.Query().Get("prerender"))
		if prerender {
			authenticatedUser, err := usersService.GetAuthenticated(req.Context())
			if err != nil {
				log.Println(err)
				authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
			}
			returnURL := req.RequestURI
			err = resume.RenderBodyInnerHTML(req.Context(), w, reactions, notification, usersService, time.Now(), authenticatedUser, returnURL)
			if err != nil {
				return err
			}
		}

		_, err = io.WriteString(w, `</body></html>`)
		return err
	})})
}
