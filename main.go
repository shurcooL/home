// home is Dmitri Shuralyov's personal website.
package main

import (
	"flag"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/shurcooL/go/httpstoppable"
	"github.com/shurcooL/home/assets"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions/emojis"
	"golang.org/x/net/html"
	"golang.org/x/net/webdav"
)

var (
	httpFlag       = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
	productionFlag = flag.Bool("production", false, "Production mode.")
	statefileFlag  = flag.String("statefile", "", "File to save/load state (file is deleted after loading).")
)

func run() error {
	flag.Parse()

	if err := mime.AddExtensionType(".md", "text/markdown"); err != nil {
		return err
	}

	users := newUsersService()
	reactions, err := newReactionsService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "reactions")),
		users)
	if err != nil {
		return err
	}
	notifications, err := initNotifications(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "notifications")),
		users)
	if err != nil {
		return err
	}
	issuesService, err := newIssuesService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "issues")),
		notifications, users)
	if err != nil {
		return err
	}

	sessionsHandler := handler{handler: SessionsHandler{users}.Serve}
	http.Handle("/login/github", sessionsHandler)
	http.Handle("/callback/github", sessionsHandler)
	http.Handle("/logout", sessionsHandler)
	http.Handle("/login", sessionsHandler)
	http.Handle("/sessions", sessionsHandler)

	usersAPIHandler := usersAPIHandler{users: users}
	http.Handle("/api/userspec", userMiddleware{httputil.ErrorHandler{H: usersAPIHandler.GetAuthenticatedSpec}})
	http.Handle("/api/user", userMiddleware{httputil.ErrorHandler{H: usersAPIHandler.GetAuthenticated}})

	http.Handle("/api/react", userMiddleware{httputil.ErrorHandler{H: reactionsAPIHandler{reactions}.ServeHTTP}})

	userContentHandler := userContentHandler{
		store: webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "usercontent")),
		users: users,
	}
	http.Handle("/api/usercontent", userMiddleware{httputil.ErrorHandler{H: userContentHandler.Upload}})
	http.Handle("/usercontent/", http.StripPrefix("/usercontent", userMiddleware{httputil.ErrorHandler{H: userContentHandler.Serve}}))

	err = initBlog(issuesService, issues.RepoSpec{URI: "dmitri.shuralyov.com/blog"}, notifications, users)
	if err != nil {
		return err
	}

	err = initIssues(issuesService, notifications, users)
	if err != nil {
		return err
	}

	emojisHandler := httpgzip.FileServer(emojis.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	http.Handle("/emojis/", http.StripPrefix("/emojis", emojisHandler))

	assetsHandler := httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	//http.Handle("/assets/", http.StripPrefix("/assets", fileServer)) // TODO.
	initResume(assetsHandler, reactions, notifications, users)

	initIdiomaticGo(assetsHandler, issuesService, notifications, users)

	initTalks(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "talks")), notifications, users)

	indexPath := filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "index.html")
	indexHandler := userMiddleware{httputil.ErrorHandler{H: func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httputil.MethodError{Allowed: []string{"GET"}}
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
			returnURL := req.RequestURI

			header := component.Header{
				MaxWidth:      800,
				CurrentUser:   authenticatedUser,
				ReturnURL:     returnURL,
				Notifications: notifications,
			}
			div := header.RenderContext(req.Context())[0]

			indexHTML.FirstChild.LastChild.InsertBefore(div, indexHTML.FirstChild.LastChild.FirstChild)
		}

		return html.Render(w, indexHTML)
	}}}
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

	if *statefileFlag != "" {
		err := sessions.LoadAndRemove(*statefileFlag)
		log.Println("sessions.LoadAndRemove:", err)
	}

	log.Println("Started.")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	stop := make(chan struct{})
	go func() {
		<-interrupt
		close(stop)
	}()
	err = httpstoppable.ListenAndServe(*httpFlag, nil, stop)
	if err != nil {
		log.Println("httpstoppable.ListenAndServe:", err)
	}

	if *statefileFlag != "" {
		err := sessions.Save(*statefileFlag)
		log.Println("sessions.Save:", err)
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}
