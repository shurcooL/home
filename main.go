// home is my personal website.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/go/gzip_file_server"
	"src.sourcegraph.com/apps/tracker/issues"
)

var (
	httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
)

func main() {
	flag.Parse()

	http.Handle("/robots.txt", http.NotFoundHandler())
	http.Handle("/", http.FileServer(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri"))))
	err := initBlog(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "issues"), issues.RepoSpec{URI: "dmitri.shuralyov.com/blog"})
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: Currently assumes initBlog initializes usersService; make that better.
	sessionsHandler := handler{handler: SessionsHandler}
	http.Handle("/login/github", sessionsHandler)
	http.Handle("/callback/github", sessionsHandler)
	http.Handle("/logout", sessionsHandler)
	http.Handle("/user", sessionsHandler) // TODO: This is an API endpoint. Consider moving under /api/ or so?
	http.Handle("/sessions", sessionsHandler)

	fileServer := gzip_file_server.New(Assets)
	//http.Handle("/assets/", http.StripPrefix("/assets/", fileServer))
	// TODO: Currently assumes initBlog initializes usersService; make that better.
	err = initResume(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "reactions"), fileServer)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Started.")

	err = http.ListenAndServe(*httpFlag, nil)
	if err != nil {
		log.Fatalln(err)
	}
}
