package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/reactions"
	"github.com/shurcooL/reactions/fsreactions"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
)

const resumeHTML = `<html>
	<head>
		<title>Dmitri Shuralyov - Resume</title>
		<link href="/blog/assets/octicons/octicons.css" rel="stylesheet" type="text/css">
		<link href="resume.css" rel="stylesheet" type="text/css">
		<script src="resume.js"></script>
	</head>
	<body></body>
</html>
`

func initResume(root webdav.FileSystem, fileServer http.Handler) error {
	http.Handle("/resume", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, resumeHTML)
	}))
	http.Handle("/resume.js", fileServer)
	http.Handle("/resume.css", fileServer)

	http.HandleFunc("/react", reactionHandler)

	var err error
	rs, err = fsreactions.NewService(root, usersService)
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

	ctx := context.TODO()
	reactableURL := req.Form.Get("reactableURL")

	switch req.Method {
	case "GET":
		reactions, err := rs.Get(ctx, reactableURL)
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
		reactions, err := rs.Toggle(ctx, reactableURL, tr)
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
