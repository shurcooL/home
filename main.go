// home is Dmitri Shuralyov's personal website.
package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shurcooL/home/assets"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httphandler"
	"github.com/shurcooL/home/httputil"
	codepkg "github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/httpfs/filter"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

var (
	httpFlag          = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
	metricsHTTPFlag   = flag.String("metrics-http", "", "Listen for metrics HTTP connections on this address, if any.")
	secureCookieFlag  = flag.Bool("secure-cookie", false, "Value of cookie attribute Secure.")
	storeDirFlag      = flag.String("store-dir", filepath.Join(os.TempDir(), "home-store"), "Directory of home store (required).")
	stateFileFlag     = flag.String("state-file", "", "Optional path to file to save/load state (file is deleted after loading).")
	analyticsFileFlag = flag.String("analytics-file", "", "Optional path to file containing analytics HTML to insert at the beginning of <head>.")
	noRobotsFlag      = flag.Bool("no-robots", false, "Disallow all robots on all pages.")
	siteNameFlag      = flag.String("site-name", "home (local devel)", "Name of site, displayed on sign in page.")
	redLogoFlag       = flag.Bool("red-logo", false, "Display the logo in red.")
)

var (
	analyticsHTML template.HTML // Set early in run.
)

func main() {
	flag.Parse()

	int := make(chan os.Signal, 1)
	signal.Notify(int, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { <-int; cancel() }()

	err := run(ctx, cancel, *storeDirFlag, *stateFileFlag, *analyticsFileFlag, *noRobotsFlag, *redLogoFlag)
	if err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context, cancel context.CancelFunc, storeDir, stateFile, analyticsFile string, noRobots, redLogo bool) error {
	if err := mime.AddExtensionType(".md", "text/markdown"); err != nil {
		return err
	}
	if err := mime.AddExtensionType(".woff2", "font/woff2"); err != nil {
		return err
	}

	if analyticsFile != "" {
		b, err := ioutil.ReadFile(analyticsFile)
		if err != nil {
			return err
		}
		analyticsHTML = template.HTML(b)
	}
	if noRobots {
		http.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, "User-agent: *\nDisallow: /\n")
		})
	}
	if redLogo {
		component.RedLogo = true
		http.HandleFunc("/icon.png", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Header()["Content-Encoding"] = nil // Disable automatic gzip compression, png is already compressed.
			err := serveFile(w, req, filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "icon-red.png"))
			if err != nil {
				log.Println(`serveFile("icon-red.png"):`, err)
			}
		})
	}

	initStores := func(storeDir string) error {
		// Make sure storeDir exists and is a directory.
		fi, err := os.Stat(storeDir)
		if os.IsNotExist(err) {
			return fmt.Errorf("store directory %q does not exist: %v", storeDir, err)
		} else if err != nil {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("store directory %q is not a directory", storeDir)
		}

		// Create store directories if they're missing.
		for _, storeName := range []string{
			"users",
			"reactions",
			"notifications",
			"notificationv2",
			"events",
			"issues",
			"usercontent",
			"repositories",
		} {
			err := os.MkdirAll(filepath.Join(storeDir, storeName), 0700)
			if err != nil {
				return err
			}
		}

		return nil
	}
	err := initStores(storeDir)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	users, userStore, err := newUsersService(
		webdav.Dir(filepath.Join(storeDir, "users")),
	)
	if err != nil {
		return fmt.Errorf("newUsersService: %v", err)
	}
	reactions, err := newReactionsService(
		webdav.Dir(filepath.Join(storeDir, "reactions")),
		users,
	)
	if err != nil {
		return fmt.Errorf("newReactionsService: %v", err)
	}
	githubRouter := dmitshurSeesHomeRouter{users: users}
	githubActivity, gerritActivity, err := initNotificationsV2(
		ctx, &wg, http.DefaultServeMux,
		webdav.Dir(filepath.Join(storeDir, "notificationv2")),
		filepath.Join(storeDir, "mail", "githubnotif"),
		filepath.Join(storeDir, "mail", "gerritnotif"),
		users,
		githubRouter,
	)
	if err != nil {
		return fmt.Errorf("initNotificationsV2: %v", err)
	}
	notifications := initNotifications(
		http.DefaultServeMux,
		webdav.Dir(filepath.Join(storeDir, "notifications")),
		gerritActivity,
		users,
		githubRouter,
	)
	events, err := newEventsService(
		webdav.Dir(filepath.Join(storeDir, "events")),
		githubActivity,
		gerritActivity,
		users,
	)
	if err != nil {
		return fmt.Errorf("newEventsService: %v", err)
	}
	issuesService, err := newIssuesService(
		webdav.Dir(filepath.Join(storeDir, "issues")),
		notifications, events, users, githubRouter,
	)
	if err != nil {
		return fmt.Errorf("newIssuesService: %v", err)
	}
	changeService := newChangeService(reactions, users, githubRouter)

	initAuth(users, userStore)

	usersAPIHandler := httphandler.Users{Users: users}
	http.Handle("/api/userspec", cookieAuth{httputil.ErrorHandler(users, usersAPIHandler.GetAuthenticatedSpec)})
	http.Handle("/api/user", cookieAuth{httputil.ErrorHandler(users, usersAPIHandler.GetAuthenticated)})
	http.Handle("/api/user/", cookieAuth{httputil.ErrorHandler(users, usersAPIHandler.Get)})

	reactionsAPIHandler := httphandler.Reactions{Reactions: reactions}
	http.Handle("/api/react", cookieAuth{httputil.ErrorHandler(users, reactionsAPIHandler.GetOrToggle)})
	http.Handle("/api/react/list", cookieAuth{httputil.ErrorHandler(users, reactionsAPIHandler.List)})

	eventsAPIHandler := httphandler.Events{Events: events}
	http.Handle("/api/events/list", headerAuth{httputil.ErrorHandler(users, eventsAPIHandler.List)})

	userContentHandler := userContentHandler{
		store: webdav.Dir(filepath.Join(storeDir, "usercontent")),
		users: users,
	}
	http.Handle("/api/usercontent", cookieAuth{httputil.ErrorHandler(users, userContentHandler.Upload)})
	http.Handle("/usercontent/", http.StripPrefix("/usercontent", cookieAuth{httputil.ErrorHandler(users, userContentHandler.Serve)}))

	indexHandler := initIndex(events, notifications, users)

	initAbout(notifications, users)

	err = initBlog(http.DefaultServeMux, issuesService, issues.RepoSpec{URI: "dmitri.shuralyov.com/blog"}, notifications, users)
	if err != nil {
		return fmt.Errorf("initBlog: %v", err)
	}

	issuesApp := initIssues(http.DefaultServeMux, issuesService, changeService, notifications, users)
	changesApp := initChanges(http.DefaultServeMux, changeService, issuesService, notifications, users)

	emojisHandler := cookieAuth{httpgzip.FileServer(assets.Emojis, httpgzip.FileServerOptions{ServeError: detailedForAdmin{Users: users}.ServeError})}
	http.Handle("/emojis/", http.StripPrefix("/emojis", emojisHandler))

	assetsHandler := cookieAuth{httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: detailedForAdmin{Users: users}.ServeError})}
	http.Handle("/assets/", assetsHandler)

	fontsHandler := cookieAuth{httpgzip.FileServer(assets.Fonts, httpgzip.FileServerOptions{ServeError: detailedForAdmin{Users: users}.ServeError})}
	http.Handle("/assets/fonts/", http.StripPrefix("/assets/fonts", fontsHandler))

	initResume(reactions, notifications, users)

	initIdiomaticGo(issuesService, notifications, users)

	// Code repositories.
	reposDir := filepath.Join(storeDir, "repositories")
	code, err := codepkg.NewService(reposDir, notifications, events, users)
	if err != nil {
		return fmt.Errorf("code.NewService: %v", err)
	}
	moduleHandler := codepkg.ModuleHandler{Code: code}
	http.Handle("/api/module/", http.StripPrefix("/api/module/", httputil.ErrorHandler(nil, moduleHandler.ServeModule)))
	gitUsers, err := initGitUsers(users)
	if err != nil {
		return fmt.Errorf("initGitUsers: %v", err)
	}
	gitHandler, err := codepkg.NewGitHandler(code, reposDir, events, users, gitUsers, func(req *http.Request) *http.Request {
		session, _ := lookUpSessionViaBasicAuth(req, users)
		return withSession(req, session)
	})
	if err != nil {
		return fmt.Errorf("code.NewGitHandler: %v", err)
	}
	codeHandler := codeHandler{code, reposDir, issuesApp, changesApp, issuesService, changeService, notifications, users, gitUsers}
	servePackagesMaybe := initPackages(code, notifications, users)

	initAction(code, users)

	initTalks(
		skipDot(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "talks"))),
		notifications, users)

	initProjects(
		http.DefaultServeMux,
		skipDot(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "projects"))),
		notifications, users)

	if *metricsHTTPFlag != "" {
		initMetrics(cancel, *metricsHTTPFlag)
		go measureGitHubV3RateLimit()
		go measureGitHubV4RateLimit()
	}

	staticFiles := cookieAuth{httpgzip.FileServer(
		skipDot(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri"))),
		httpgzip.FileServerOptions{
			IndexHTML:  true,
			ServeError: detailedForAdmin{Users: users}.ServeError,
		},
	)}
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// Serve index page.
		if req.URL.Path == "/" {
			indexHandler.ServeHTTP(w, req)
			return
		}
		// Serve git protocol requests for existing repos, if the request matches.
		if ok := gitHandler.ServeGitMaybe(w, req); ok {
			return
		}
		// Serve code pages for existing repos/packages, if the request matches.
		if ok := codeHandler.ServeCodeMaybe(w, req); ok {
			return
		}
		// Serve remaining import path pattern queries, if the request matches.
		if ok := servePackagesMaybe(w, req); ok {
			return
		}
		// Serve static files last.
		staticFiles.ServeHTTP(w, req)
	})

	if stateFile != "" {
		err := global.LoadAndRemove(stateFile)
		global.mu.Lock()
		n := len(global.sessions)
		global.mu.Unlock()
		log.Println("sessions.LoadAndRemove:", n, err)
	}

	server := &http.Server{Addr: *httpFlag, Handler: top{httputil.GzipHandler(http.DefaultServeMux)}}

	go func() {
		<-ctx.Done()
		err := server.Close()
		if err != nil {
			log.Println("server.Close:", err)
		}
	}()

	log.Println("Starting HTTP server.")

	err = server.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Println("server.ListenAndServe:", err)
		cancel()
	}

	log.Println("Ended HTTP server.")

	wg.Wait()

	if stateFile != "" {
		err := global.Save(stateFile)
		log.Println("sessions.Save:", err)
	}

	return nil
}

// top adds some instrumentation on top of Handler.
type top struct{ Handler http.Handler }

func (t top) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	started := time.Now()
	rw := &responseWriter{ResponseWriter: w, Flusher: w.(http.Flusher)}
	t.Handler.ServeHTTP(rw, req)
	fmt.Printf("TIMING: %s: %v\n", req.URL, time.Since(started))
	if path != req.URL.Path {
		log.Printf("warning: req.URL.Path was modified from %v to %v\n", path, req.URL.Path)
	}
	if rw.WroteBytes && !haveType(w) {
		log.Printf("warning: Content-Type header not set for %v %q\n", req.Method, req.URL)
	}
}

// haveType reports whether w has the Content-Type header set.
func haveType(w http.ResponseWriter) bool {
	_, ok := w.Header()["Content-Type"]
	return ok
}

// responseWriter wraps a real http.ResponseWriter and captures
// whether any bytes were written.
type responseWriter struct {
	http.ResponseWriter
	http.Flusher

	WroteBytes bool // Whether non-zero bytes have been written.
}

func (rw *responseWriter) Write(p []byte) (n int, err error) {
	if len(p) > 0 {
		rw.WroteBytes = true
	}
	return rw.ResponseWriter.Write(p)
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
