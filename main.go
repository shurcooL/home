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
	"github.com/shurcooL/home/httphandler"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions/emojis"
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

	sessionsHandler := &SessionsHandler{users}
	http.Handle("/login/github", sessionsHandler)
	http.Handle("/callback/github", sessionsHandler)
	http.Handle("/logout", sessionsHandler)
	http.Handle("/login", sessionsHandler)
	http.Handle("/sessions", sessionsHandler)

	usersAPIHandler := httphandler.Users{Users: users}
	http.Handle("/api/userspec", userMiddleware{httputil.ErrorHandler(usersAPIHandler.GetAuthenticatedSpec)})
	http.Handle("/api/user", userMiddleware{httputil.ErrorHandler(usersAPIHandler.GetAuthenticated)})

	reactionsAPIHandler := httphandler.Reactions{Reactions: reactions}
	http.Handle("/api/react", userMiddleware{httputil.ErrorHandler(reactionsAPIHandler.GetOrToggle)})

	userContentHandler := userContentHandler{
		store: webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "usercontent")),
		users: users,
	}
	http.Handle("/api/usercontent", userMiddleware{httputil.ErrorHandler(userContentHandler.Upload)})
	http.Handle("/usercontent/", http.StripPrefix("/usercontent", userMiddleware{httputil.ErrorHandler(userContentHandler.Serve)}))

	indexHandler := initIndex(notifications, users)

	initAbout(notifications, users)

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
	http.Handle("/assets/", assetsHandler)

	initResume(assetsHandler, reactions, notifications, users)

	initIdiomaticGo(issuesService, notifications, users)

	initPackages(notifications, users)

	initTalks(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "talks")), notifications, users)

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
