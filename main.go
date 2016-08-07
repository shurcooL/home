// home is Dmitri Shuralyov's personal website.
package main

import (
	"flag"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/home/assets"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"golang.org/x/net/webdav"
)

var (
	httpFlag       = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
	productionFlag = flag.Bool("production", false, "Production mode.")
)

func run() error {
	flag.Parse()

	if err := mime.AddExtensionType(".md", "text/markdown"); err != nil {
		return err
	}

	users := newUsersService()
	reactions, err := newReactionsService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "reactions")),
		users,
	)
	if err != nil {
		return err
	}

	sessionsHandler := handler{handler: SessionsHandler{users}.Serve}
	http.Handle("/login/github", sessionsHandler)
	http.Handle("/callback/github", sessionsHandler)
	http.Handle("/logout", sessionsHandler)
	http.Handle("/api/user", sessionsHandler)
	http.Handle("/sessions", sessionsHandler)

	http.Handle("/react", reactHandler{reactions})

	notifications, err := initNotifications(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "notifications")),
		users,
	)
	if err != nil {
		return err
	}

	err = initBlog(
		filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "issues"),
		issues.RepoSpec{URI: "dmitri.shuralyov.com/blog"},
		notifications,
		users,
	)
	if err != nil {
		return err
	}

	fileServer := httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	//http.Handle("/assets/", http.StripPrefix("/assets", fileServer))
	initResume(fileServer)

	http.Handle("/", httpgzip.FileServer(
		http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri")),
		httpgzip.FileServerOptions{
			IndexHTML:  true,
			ServeError: httpgzip.Detailed,
		},
	))

	log.Println("Started.")

	return http.ListenAndServe(*httpFlag, nil)
}

func main() {
	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}
