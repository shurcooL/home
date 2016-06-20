// home is Dmitri Shuralyov's personal website.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/go/gzip_file_server"
	"github.com/shurcooL/home/assets"
	"github.com/shurcooL/issues"
	"golang.org/x/net/webdav"
)

var (
	httpFlag       = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
	productionFlag = flag.Bool("production", false, "Production mode.")
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
	http.Handle("/api/user", sessionsHandler)
	http.Handle("/sessions", sessionsHandler)

	fileServer := gzip_file_server.New(assets.Assets)
	//http.Handle("/assets/", http.StripPrefix("/assets", fileServer))
	// TODO: Currently assumes initBlog initializes usersService; make that better.
	err = initResume(webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "reactions")), fileServer)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Started.")

	err = http.ListenAndServe(*httpFlag, nil)
	if err != nil {
		log.Fatalln(err)
	}
}
