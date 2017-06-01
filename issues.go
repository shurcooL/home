package main

import (
	"context"
	"net/http"

	"github.com/shurcooL/events"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issues/fs"
	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/issuesapp/httphandler"
	"github.com/shurcooL/issuesapp/httproute"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func newIssuesService(root webdav.FileSystem, notifications notifications.ExternalService, events events.ExternalService, users users.Service) (issues.Service, error) {
	return fs.NewService(root, notifications, events, users)
}

// initIssues registers handlers for the issues service HTTP API,
// and handlers for the issues app.
func initIssues(issuesService issues.Service, notifications notifications.Service, users users.Service) error {
	// Register HTTP API endpoints.
	issuesAPIHandler := httphandler.Issues{Issues: issuesService}
	http.Handle(httproute.List, apiMiddleware{httputil.ErrorHandler(users, issuesAPIHandler.List)})
	http.Handle(httproute.Count, apiMiddleware{httputil.ErrorHandler(users, issuesAPIHandler.Count)})
	http.Handle(httproute.ListComments, apiMiddleware{httputil.ErrorHandler(users, issuesAPIHandler.ListComments)})
	http.Handle(httproute.EditComment, apiMiddleware{httputil.ErrorHandler(users, issuesAPIHandler.EditComment)})

	opt := issuesapp.Options{
		Notifications: notifications,

		HeadPre: `<link href="/icon.png" rel="icon" type="image/png">
<meta name="viewport" content="width=device-width">
<style type="text/css">
	body {
		margin: 20px;
		font-family: sans-serif;
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
	opt.BodyTop = func(req *http.Request) ([]htmlg.Component, error) {
		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return nil, err
		}
		var nc uint64
		if authenticatedUser.ID != 0 {
			nc, err = notifications.Count(req.Context(), nil)
			if err != nil {
				return nil, err
			}
		}
		returnURL := req.RequestURI

		header := component.Header{
			CurrentUser:       authenticatedUser,
			NotificationCount: nc,
			ReturnURL:         returnURL,
		}
		return []htmlg.Component{header}, nil
	}
	issuesApp := issuesapp.New(issuesService, users, opt)

	for _, repoSpec := range []issues.RepoSpec{
		{URI: "github.com/shurcooL/issuesapp"},
		{URI: "github.com/shurcooL/notificationsapp"},
		{URI: "dmitri.shuralyov.com/idiomatic-go"},
		{URI: "dmitri.shuralyov.com/temp/go-get-issue-unicode/испытание"}, // TODO: Delete after https://github.com/golang/go/issues/18660 and https://github.com/golang/gddo/issues/468 are resolved.
	} {
		repoSpec := repoSpec
		issuesHandler := userMiddleware{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
			prefixLen := len("/issues/") + len(repoSpec.URI)
			if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
				baseURL := prefix
				if req.URL.RawQuery != "" {
					baseURL += "?" + req.URL.RawQuery
				}
				return httperror.Redirect{URL: baseURL}
			}
			req = copyRequestAndURL(req)
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
