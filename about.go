package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

var aboutHTML = template.Must(template.New("").Parse(`<html>
	<head>
		<meta charset="utf-8">
		<title>Dmitri Shuralyov</title>
		<link href="//maxcdn.bootstrapcdn.com/font-awesome/4.2.0/css/font-awesome.min.css" rel="stylesheet">
		<link href="/assets/about/style.css" rel="stylesheet" type="text/css">
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body>
		<div style="max-width: 800px; margin: 0 auto 20px auto;">`))

func initAbout(notifications notifications.Service, users users.Service) {
	http.Handle("/about", userMiddleware{httputil.ErrorHandler(func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httputil.MethodError{Allowed: []string{"GET"}}
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
		returnURL := req.RequestURI

		// Render the header.
		header := component.Header{
			CurrentUser:   authenticatedUser,
			ReturnURL:     returnURL,
			Notifications: notifications,
		}
		err = htmlg.RenderComponentsContext(req.Context(), w, header)
		if err != nil {
			return err
		}

		// Render the nav bar.
		_, err = io.WriteString(w, `<div class="nav">
				<ul class="nav">
					<li class="nav"><a href="/blog">Blog</a></li>
					<li class="nav smaller"><a href="/idiomatic-go">Idiomatic Go</a></li>
					<li class="nav"><a href="/talks">Talks</a></li>
					<li class="nav"><a href="/projects">Projects</a></li>
					<li class="nav"><a href="/resume">Resume</a></li>
					<li class="nav"><a href="/about">About</a></li>
				</ul>
			</div>`)
		if err != nil {
			return err
		}

		// Render content.
		_, err = io.WriteString(w, `<div style="float: left;"><img width="300" height="450" src="avatar-p.jpg"></div>
			<div style="margin-left: 340px; text-align: justify;">
				<p>I'm Dmitri Shuralyov, a software engineer and an avid <a title="Someone who uses Go." href="https://golang.org">gopher</a>. I strive to make software more delightful.</p>
				
				<p>Coming from a game development and graphics/UI background where C++ was used primarily, I discovered and made a full switch to Go <abbr title="2013.">three years ago</abbr>, which lead to increased developer happiness.</p>
				
				<p>In my spare time, I'm mostly interested in working on software development tools and exploring experimental ideas. I enjoy contributing to open source, fixing issues in existing tools and the <a href="https://github.com/golang/go/commits/master?author=shurcooL">Go project</a> itself.</p>
				
				<p>You can also find me on:</p>

				<div>
					<span class="entry"><a class="blue" title="GitHub" href="https://github.com/shurcooL" rel="me"><i class="fa fa-github fa-2x" style="vertical-align: middle;"></i></a></span>
					<span class="entry"><a class="blue" title="Twitter" href="https://twitter.com/shurcooL" rel="me"><span class="fa-stack"><i class="fa fa-circle fa-stack-2x"></i><i class="fa fa-twitter fa-stack-1x fa-inverse"></i></span></a></span>
					<span class="entry"><a class="blue" title="Gratipay" href="https://gratipay.com/~shurcooL"><i class="fa fa-gittip fa-2x" style="vertical-align: middle;"></i></a></span>
				</div>
			</div>
			<div style="clear: both;"></div>`)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, `</body></html>`)
		return err
	})})
}
