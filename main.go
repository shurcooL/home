// home is Dmitri Shuralyov's personal website.
package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/home/assets"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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
		{ // TODO: topbar component.
			returnURL := req.URL.String()

			div := &html.Node{
				Type: html.ElementNode, Data: atom.Div.String(),
				Attr: []html.Attribute{
					{Key: atom.Style.String(), Val: "max-width: 800px; margin: 0 auto; text-align: right; height: 18px; font-size: 12px;"},
				},
			}
			if authenticatedUser.ID != 0 {
				{ // Notifications icon.
					n, err := notifications.Count(req.Context(), nil)
					if err != nil {
						return err
					}
					span := &html.Node{
						Type: html.ElementNode, Data: atom.Span.String(),
						Attr: []html.Attribute{
							{Key: atom.Style.String(), Val: "margin-right: 10px;"},
						},
					}
					for _, n := range (Notifications{Unread: n > 0}).Render() {
						span.AppendChild(n)
					}
					div.AppendChild(span)
				}

				{ // TODO: topbar-avatar component.
					a := &html.Node{
						Type: html.ElementNode, Data: atom.A.String(),
						Attr: []html.Attribute{
							{Key: atom.Class.String(), Val: "topbar-avatar"},
							{Key: atom.Href.String(), Val: string(authenticatedUser.HTMLURL)},
							{Key: atom.Target.String(), Val: "_blank"},
							{Key: atom.Tabindex.String(), Val: "-1"},
						},
					}
					a.AppendChild(&html.Node{
						Type: html.ElementNode, Data: atom.Img.String(),
						Attr: []html.Attribute{
							{Key: atom.Class.String(), Val: "topbar-avatar"},
							{Key: atom.Src.String(), Val: string(authenticatedUser.AvatarURL)},
							{Key: atom.Title.String(), Val: fmt.Sprintf("Signed in as %s.", authenticatedUser.Login)},
						},
					})
					div.AppendChild(a)
				}

				signOut := PostButton{Action: "/logout", Text: "Sign out", ReturnURL: returnURL}
				for _, n := range signOut.Render() {
					div.AppendChild(n)
				}
			} else {
				signInViaGitHub := PostButton{Action: "/login/github", Text: "Sign in via GitHub", ReturnURL: returnURL}
				for _, n := range signInViaGitHub.Render() {
					div.AppendChild(n)
				}
			}
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
