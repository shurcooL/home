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
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"golang.org/x/net/html"
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
	notifications, err := initNotifications(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "notifications")),
		users,
	)
	if err != nil {
		return err
	}
	issuesService, err := newIssuesService(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "issues"),
		notifications, users)
	if err != nil {
		return err
	}

	sessionsHandler := handler{handler: SessionsHandler{users}.Serve}
	http.Handle("/login/github", sessionsHandler)
	http.Handle("/callback/github", sessionsHandler)
	http.Handle("/logout", sessionsHandler)
	http.Handle("/api/userspec", sessionsHandler)
	http.Handle("/api/user", sessionsHandler)
	http.Handle("/login", sessionsHandler)
	http.Handle("/sessions", sessionsHandler)

	http.Handle("/api/react", errorHandler{reactHandler{reactions}.ServeHTTP})

	userContent := userContent{
		store: webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "usercontent")),
		users: users,
	}
	http.Handle("/api/usercontent", errorHandler{userContent.UploadHandler})
	http.Handle("/usercontent/", http.StripPrefix("/usercontent", errorHandler{userContent.ServeHandler}))

	err = initBlog(issuesService, issues.RepoSpec{URI: "dmitri.shuralyov.com/blog"}, notifications, users)
	if err != nil {
		return err
	}

	err = initIssues(issuesService, notifications, users)
	if err != nil {
		return err
	}

	resumeJSCSS := httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	//http.Handle("/assets/", http.StripPrefix("/assets", fileServer)) // TODO.
	initResume(resumeJSCSS, reactions, notifications, users)

	indexPath := filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "index.html")
	indexHandler := errorHandler{func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return MethodError{Allowed: []string{"GET"}}
		}
		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return err
		}
		f, err := os.Open(indexPath)
		if err != nil {
			return err
		}
		defer f.Close()
		indexHTML, err := html.Parse(f)
		if err != nil {
			return err
		}
		{
			returnURL := req.URL.String()

			header := component.Header{
				MaxWidth:      800,
				CurrentUser:   authenticatedUser,
				ReturnURL:     returnURL,
				Notifications: notifications,
			}
			div := header.Render(req.Context())[0]

			indexHTML.FirstChild.LastChild.InsertBefore(div, indexHTML.FirstChild.LastChild.FirstChild)
		}

		return html.Render(w, indexHTML)
	}}
	staticFiles := httpgzip.FileServer(
		http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri")),
		httpgzip.FileServerOptions{
			IndexHTML:  true,
			ServeError: httpgzip.Detailed,
		},
	)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/":
			indexHandler.ServeHTTP(w, req)
		default:
			staticFiles.ServeHTTP(w, req)
		}
	})

	log.Println("Started.")

	return http.ListenAndServe(*httpFlag, nil)
}

func main() {
	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}
