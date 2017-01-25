package main

import (
	"context"
	"net/http"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httphandler"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issues/fs"
	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func newIssuesService(root webdav.FileSystem, notifications notifications.ExternalService, users users.Service) (issues.Service, error) {
	return fs.NewService(root, notifications, users)
}

// initIssues registers handlers for the issues service HTTP API,
// and handlers for the issues app.
func initIssues(issuesService issues.Service, notifications notifications.Service, users users.Service) error {
	// Register HTTP API endpoint.
	issuesAPIHandler := httphandler.Issues{Issues: issuesService}
	http.Handle("/api/issues/list", userMiddleware{httputil.ErrorHandler(users, issuesAPIHandler.List)})
	http.Handle("/api/issues/count", userMiddleware{httputil.ErrorHandler(users, issuesAPIHandler.Count)})
	http.Handle("/api/issues/list-comments", userMiddleware{httputil.ErrorHandler(users, issuesAPIHandler.ListComments)})
	http.Handle("/api/issues/edit-comment", userMiddleware{httputil.ErrorHandler(users, issuesAPIHandler.EditComment)})

	opt := issuesapp.Options{
		Notifications: notifications,

		HeadPre: `<link href="/icon.png" rel="icon" type="image/png">
<style type="text/css">
	body {
		margin: 20px;
		font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
		font-size: 14px;
		line-height: initial;
		color: #373a3c;
	}
	a {
		color: #0275d8;
		text-decoration: none;
	}
	a:focus, a:hover {
		color: #014c8c;
		text-decoration: underline;
	}
	.btn {
		font-size: 11px;
		line-height: 11px;
		border-radius: 4px;
		border: solid #d2d2d2 1px;
		background-color: #fff;
		box-shadow: 0 1px 1px rgba(0, 0, 0, .05);
	}
</style>`,
		BodyPre: `<div style="max-width: 800px; margin: 0 auto 100px auto;">`,
	}
	if *productionFlag {
		opt.HeadPre += "\n\t\t" + googleAnalytics
	}
	opt.BodyTop = func(req *http.Request) ([]htmlg.ComponentContext, error) {
		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return nil, err
		}
		returnURL := req.RequestURI
		header := component.Header{
			CurrentUser:   authenticatedUser,
			ReturnURL:     returnURL,
			Notifications: notifications,
		}
		return []htmlg.ComponentContext{header}, nil
	}
	issuesApp := issuesapp.New(issuesService, users, opt)

	for _, repoSpec := range []issues.RepoSpec{
		{URI: "github.com/shurcooL/issuesapp"},
		{URI: "github.com/shurcooL/notificationsapp"},
		{URI: "dmitri.shuralyov.com/idiomatic-go"},
	} {
		repoSpec := repoSpec
		issuesHandler := userMiddleware{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
			prefixLen := len("/issues/") + len(repoSpec.URI)
			if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
				baseURL := prefix
				if req.URL.RawQuery != "" {
					baseURL += "?" + req.URL.RawQuery
				}
				return httputil.Redirect{URL: baseURL}
			}
			req.URL.Path = req.URL.Path[prefixLen:]
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req = req.WithContext(context.WithValue(req.Context(), issuesapp.RepoSpecContextKey, repoSpec))
			req = req.WithContext(context.WithValue(req.Context(), issuesapp.BaseURIContextKey, "/issues/"+repoSpec.URI))
			issuesApp.ServeHTTP(w, req)
			return nil
		})}
		http.Handle("/issues/"+repoSpec.URI, issuesHandler)
		http.Handle("/issues/"+repoSpec.URI+"/", issuesHandler)
	}

	return nil
}
