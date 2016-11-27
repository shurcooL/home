package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
)

func initAbout(notifications notifications.Service, users users.Service) {
	aboutPath := filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri", "about.html")
	http.Handle("/about", userMiddleware{httputil.ErrorHandler(func(w http.ResponseWriter, req *http.Request) error {
		if req.Method != "GET" {
			return httputil.MethodError{Allowed: []string{"GET"}}
		}
		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return err
		}
		f, err := os.Open(aboutPath)
		if err != nil {
			return err
		}
		defer f.Close()
		aboutHTML, err := html.Parse(f)
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

			aboutHTML.FirstChild.LastChild.InsertBefore(div, aboutHTML.FirstChild.LastChild.FirstChild)
		}

		return html.Render(w, aboutHTML)
	})})
}
