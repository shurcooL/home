// home is Dmitri Shuralyov's personal website.
package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/shurcooL/home/assets"
	"github.com/shurcooL/home/httphandler"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/httpfs/filter"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions/emojis"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

var (
	httpFlag       = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
	productionFlag = flag.Bool("production", false, "Production mode.")
	statefileFlag  = flag.String("statefile", "", "File to save/load state (file is deleted after loading).")
)

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	if err := mime.AddExtensionType(".md", "text/markdown"); err != nil {
		return err
	}

	users, userStore, err := newUsersService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "users")),
	)
	if err != nil {
		return err
	}
	reactions, err := newReactionsService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "reactions")),
		users)
	if err != nil {
		return err
	}
	notifications, err := initNotifications(
		http.DefaultServeMux,
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "notifications")),
		users)
	if err != nil {
		return err
	}
	events, err := newEventsService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "events")),
		users,
	)
	if err != nil {
		return err
	}
	issuesService, err := newIssuesService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "issues")),
		notifications, events, users)
	if err != nil {
		return err
	}

	sessionsHandler := &sessionsHandler{users, userStore}
	http.Handle("/login/github", sessionsHandler)
	http.Handle("/callback/github", sessionsHandler)
	http.Handle("/logout", sessionsHandler)
	http.Handle("/login", sessionsHandler)
	http.Handle("/sessions", sessionsHandler)

	usersAPIHandler := httphandler.Users{Users: users}
	http.Handle("/api/userspec", cookieAuth{httputil.ErrorHandler(users, usersAPIHandler.GetAuthenticatedSpec)})
	http.Handle("/api/user", cookieAuth{httputil.ErrorHandler(users, usersAPIHandler.GetAuthenticated)})
	http.Handle("/api/user/", cookieAuth{httputil.ErrorHandler(users, usersAPIHandler.Get)})

	reactionsAPIHandler := httphandler.Reactions{Reactions: reactions}
	http.Handle("/api/react", cookieAuth{httputil.ErrorHandler(users, reactionsAPIHandler.GetOrToggle)})
	http.Handle("/api/react/list", cookieAuth{httputil.ErrorHandler(users, reactionsAPIHandler.List)})

	userContentHandler := userContentHandler{
		store: webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "usercontent")),
		users: users,
	}
	http.Handle("/api/usercontent", cookieAuth{httputil.ErrorHandler(users, userContentHandler.Upload)})
	http.Handle("/usercontent/", http۰StripPrefix("/usercontent", cookieAuth{httputil.ErrorHandler(users, userContentHandler.Serve)}))

	indexHandler := initIndex(events, notifications, users)

	initAbout(notifications, users)

	err = initBlog(issuesService, issues.RepoSpec{URI: "dmitri.shuralyov.com/blog"}, notifications, users)
	if err != nil {
		return err
	}

	err = initIssues(issuesService, notifications, users)
	if err != nil {
		return err
	}

	emojisHandler := cookieAuth{httpgzip.FileServer(emojis.Assets, httpgzip.FileServerOptions{ServeError: detailedForAdmin{Users: users}.ServeError})}
	http.Handle("/emojis/", http۰StripPrefix("/emojis", emojisHandler))

	assetsHandler := cookieAuth{httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: detailedForAdmin{Users: users}.ServeError})}
	http.Handle("/assets/", assetsHandler)

	initResume(assetsHandler, reactions, notifications, users)

	initIdiomaticGo(issuesService, notifications, users)

	initPackages(notifications, users)

	initTalks(
		skipDot(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "talks"))),
		notifications, users)

	initProjects(
		http.DefaultServeMux,
		skipDot(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "projects"))),
		notifications, users)

	staticFiles := cookieAuth{httpgzip.FileServer(
		skipDot(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri"))),
		httpgzip.FileServerOptions{
			IndexHTML:  true,
			ServeError: detailedForAdmin{Users: users}.ServeError,
		},
	)}
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/":
			indexHandler.ServeHTTP(w, req)
		default:
			staticFiles.ServeHTTP(w, req)
		}
	})

	if *statefileFlag != "" {
		err := global.LoadAndRemove(*statefileFlag)
		global.mu.Lock()
		n := len(global.sessions)
		global.mu.Unlock()
		log.Println("sessions.LoadAndRemove:", n, err)
	}

	server := &http.Server{Addr: *httpFlag, Handler: topMux{}}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		err := server.Close()
		if err != nil {
			log.Println("server.Close:", err)
		}
	}()

	log.Println("Started.")

	err = server.ListenAndServe()
	if err != nil {
		log.Println("server.ListenAndServe:", err)
	}

	if *statefileFlag != "" {
		err := global.Save(*statefileFlag)
		log.Println("sessions.Save:", err)
	}

	return nil
}

// topMux adds some instrumentation on top of http.DefaultServeMux.
type topMux struct{}

func (topMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	started := time.Now()
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Printf("TIMING: %s: %v\n", path, time.Since(started))
	if path != req.URL.Path {
		log.Printf("warning: req.URL.Path was modified from %v to %v\n", path, req.URL.Path)
	}
	if _, haveType := w.Header()["Content-Type"]; !haveType {
		if path == "/login/github" || path == "/callback/github" { // TODO: A better way to skip these. Ideally, they shouldn't be detected because they don't write any response, and/or the status code is redirect. But that'd require some interspection of written response.
			return
		}
		// BUG: There are false positives for redirect responses, as well as 304 responses, etc.
		log.Printf("warning: Content-Type header not set for %q\n", path)
	}
}

// skipDot returns src without dot files.
func skipDot(src http.FileSystem) http.FileSystem {
	skip := func(path string, fi os.FileInfo) bool {
		for _, e := range strings.Split(path[1:], "/") {
			if strings.HasPrefix(e, ".") {
				return true
			}
		}
		return false
	}
	return filter.Skip(src, skip)
}

// detailedForAdmin serves detailed errors for admin users,
// but non-specific errors for others.
type detailedForAdmin struct {
	Users users.Service
}

func (d detailedForAdmin) ServeError(w http.ResponseWriter, req *http.Request, err error) {
	switch user, e := d.Users.GetAuthenticated(req.Context()); {
	case e == nil && user.SiteAdmin:
		httpgzip.Detailed(w, req, err)
	default:
		httpgzip.NonSpecific(w, req, err)
	}
}
