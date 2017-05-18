package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/exp/vec"
	"github.com/shurcooL/home/exp/vec/attr"
	"github.com/shurcooL/home/exp/vec/elem"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

var aboutHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<title>Dmitri Shuralyov - About</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="//maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css" rel="stylesheet">
		<link href="/assets/about/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>
		<div style="max-width: 800px; margin: 0 auto 100px auto;">`))

func initAbout(notifications notifications.Service, users users.Service) {
	http.Handle("/about", userMiddleware{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httperror.Method{Allowed: []string{"GET"}}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ Production bool }{*productionFlag}
		err := aboutHTML.Execute(w, data)
		if err != nil {
			return err
		}

		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return err
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

		// Render content.
		err = vec.RenderHTML(w,
			elem.Span(
				attr.Style("display: table; margin-left: auto; margin-right: auto;"),
				elem.Img(
					attr.Style("width: 240px; height: 240px; border-radius: 8px; margin-bottom: 8px;"),
					attr.Src("avatar-s.jpg"),
				),
				elem.Div(
					attr.Style("font-size: 26px; font-weight: 600;"),
					"Dmitri Shuralyov",
				),
				elem.Div(
					attr.Style("font-size: 20px; font-weight: 300; color: #666;"),
					"shurcooL",
				),
			),
			elem.Div(
				attr.Style("margin-top: 24px;"),
				elem.P(
					"Dmitri Shuralyov is a software engineer and an avid ",
					elem.A(attr.Title("Someone who uses Go."), attr.Href("https://golang.org"),
						"gopher",
					),
					". He strives to make software more delightful.",
				),
				elem.P(
					"Coming from a game development and graphics/UI background where C++ was used predominantly, he discovered and made a full switch to Go ",
					elem.Abbr(attr.Title("2013."), attr.Href("https://golang.org"),
						"four years ago",
					),
					", which lead to increased developer happiness.",
				),
				elem.P(
					"In his spare time, he's mostly working on software development tools and exploring experimental ideas. He enjoys contributing to open source, fixing issues in existing tools and the ",
					elem.A(attr.Href("https://github.com/golang/go/commits/master?author=shurcooL"),
						"Go project",
					),
					" itself.",
				),
			),
			elem.Div(
				attr.Style("border-top: 1px solid #f0f0f0;"),
				elem.Ul(
					attr.Style("padding-left: 0;"),
					elem.Li(
						attr.Style("display: block; margin-bottom: 8px;"),
						elem.Span(attr.Style("color: #666; margin-right: 6px;"),
							elem.I(attr.Class("fa fa-github"))),
						elem.A(attr.Href("https://github.com/shurcooL"), "github.com/shurcooL"),
					),
					elem.Li(
						attr.Style("display: block;"),
						elem.Span(attr.Style("color: #666; margin-right: 6px;"),
							elem.I(attr.Class("fa fa-twitter"))),
						elem.A(attr.Href("https://twitter.com/shurcooL"), "twitter.com/shurcooL"),
					),
				),
			),
		)
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
