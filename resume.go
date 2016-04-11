package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/reactions"
	"github.com/shurcooL/reactions/fs"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
)

var resumeHTML = template.Must(template.New("").Funcs(template.FuncMap{"noescape": func(s string) template.HTML { return template.HTML(s) }}).Parse(`<html>
	<head>
		<title>Dmitri Shuralyov - Resume</title>
		<link href="/blog/assets/octicons/octicons.css" rel="stylesheet" type="text/css">
		<link href="resume.css" rel="stylesheet" type="text/css">
		{{noescape "<!-- Unminified source is at https://github.com/shurcooL/resume. -->"}}
		<script src="resume.js"></script>
		{{if .Production}}` + googleAnalytics + `{{end}}
	</head>
	<body></body>
</html>
`))

const googleAnalytics = `<script>
		  (function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
		  (i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
		  m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
		  })(window,document,'script','//www.google-analytics.com/analytics.js','ga');

		  ga('create', 'UA-56541369-3', 'auto');
		  ga('send', 'pageview');

		</script>`

func initResume(root webdav.FileSystem, fileServer http.Handler) error {
	http.Handle("/resume", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct {
			Production bool
		}{*productionFlag}
		err := resumeHTML.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
	}))
	http.Handle("/resume.js", fileServer)
	http.Handle("/resume.css", fileServer)

	http.HandleFunc("/react", reactionHandler)

	var err error
	rs, err = fs.NewService(root, usersService)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Get rid of global.
var rs reactions.Service

func reactionHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method should be GET or POST", http.StatusMethodNotAllowed)
		return
	}

	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.WithValue(context.Background(), requestKey, req) // TODO, THINK: Is this the best place? Can it be generalized? Isn't it error prone otherwise?
	reactableURL := req.Form.Get("reactableURL")
	reactableID := req.Form.Get("reactableID")

	switch req.Method {
	case "GET":
		reactions, err := rs.Get(ctx, reactableURL, reactableID)
		if os.IsPermission(err) { // TODO: Move this to a higher level (and upate all other similar code too).
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("rs.Get:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(reactions)
		if err != nil {
			log.Println(err)
		}
	case "POST":
		tr := reactions.ToggleRequest{
			Reaction: reactions.EmojiID(req.PostForm.Get("reaction")),
		}
		reactions, err := rs.Toggle(ctx, reactableURL, reactableID, tr)
		if os.IsPermission(err) { // TODO: Move this to a higher level (and upate all other similar code too).
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("rs.Toggle:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(reactions)
		if err != nil {
			log.Println(err)
		}
	}
}
