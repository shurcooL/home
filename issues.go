package main

import (
	"context"
	"net/http"
	"net/url"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httphandler"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issues/fs"
	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/issuesapp/common"
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
	http.Handle("/api/issues/list", userMiddleware{httputil.ErrorHandler(issuesAPIHandler.List)})
	http.Handle("/api/issues/list-comments", userMiddleware{httputil.ErrorHandler(issuesAPIHandler.ListComments)})
	http.Handle("/api/issues/edit-comment", userMiddleware{httputil.ErrorHandler(issuesAPIHandler.EditComment)})

	opt := issuesapp.Options{
		Notifications: notifications,

		RepoSpec: func(req *http.Request) issues.RepoSpec {
			return req.Context().Value(issuesapp.RepoSpecContextKey).(issues.RepoSpec)
		},
		BaseURI: func(req *http.Request) string { return req.Context().Value(issuesapp.BaseURIContextKey).(string) },
		BaseState: func(req *http.Request) issuesapp.BaseState {
			reqPath := req.URL.Path
			if reqPath == "/" {
				reqPath = "" // This is needed so that absolute URL for root view, i.e., /issues, is "/issues" and not "/issues/" because of "/issues" + "/".
			}
			return issuesapp.BaseState{
				State: common.State{
					BaseURI: req.Context().Value(issuesapp.BaseURIContextKey).(string),
					ReqPath: reqPath,
				},
			}
		},
		HeadPre: `<style type="text/css">
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

	/* TODO: Factor out, this is same as in index.html style. */
	.notifications {
		display: inline-block;
		vertical-align: top;
		position: relative;
	}
	.notifications:hover {
		color: #4183c4;
		fill: currentColor;
	}
</style>`,
	}
	if *productionFlag {
		opt.HeadPre += "\n\t\t" + googleAnalytics
	}
	opt.BodyTop = func(req *http.Request) ([]htmlg.ComponentContext, error) {
		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return nil, err
		}
		baseURI := req.Context().Value(issuesapp.BaseURIContextKey).(string)
		reqPath := req.URL.Path
		if reqPath == "/" {
			reqPath = "" // This is needed so that absolute URL for root view, i.e., /issues, is "/issues" and not "/issues/" because of "/issues" + "/".
		}
		returnURL := (&url.URL{Path: baseURI + reqPath, RawQuery: req.URL.RawQuery}).String()
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
		issuesHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// TODO: Factor this out?
			u, err := getUser(req)
			if err == errBadAccessToken {
				// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
				http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
			}
			req = withUser(req, u)

			req = req.WithContext(context.WithValue(req.Context(),
				issuesapp.RepoSpecContextKey, repoSpec))
			req = req.WithContext(context.WithValue(req.Context(),
				issuesapp.BaseURIContextKey, "/issues/"+repoSpec.URI))

			prefixLen := len("/issues/") + len(repoSpec.URI)
			if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
				baseURL := prefix
				if req.URL.RawQuery != "" {
					baseURL += "?" + req.URL.RawQuery
				}
				http.Redirect(w, req, baseURL, http.StatusMovedPermanently)
				return
			}
			req.URL.Path = req.URL.Path[prefixLen:]
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			issuesApp.ServeHTTP(w, req)
		})
		http.Handle("/issues/"+repoSpec.URI, issuesHandler)
		http.Handle("/issues/"+repoSpec.URI+"/", issuesHandler)
	}

	return nil
}
