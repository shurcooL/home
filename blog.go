package main

import (
	"context"
	"net/http"
	"net/url"
	"os"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/issuesapp/common"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

// initBlog registers a blog handler with blog URI as blog content source.
func initBlog(issuesService issues.Service, blog issues.RepoSpec, notifications notifications.Service, users users.Service) error {
	onlyShurcoolCreatePosts := onlyShurcoolCreatePosts{
		Service: issuesService,
		users:   users,
	}

	opt := issuesapp.Options{
		Notifications: notifications,

		RepoSpec: func(req *http.Request) issues.RepoSpec {
			return req.Context().Value(issuesapp.RepoSpecContextKey).(issues.RepoSpec)
		},
		BaseURI: func(req *http.Request) string { return req.Context().Value(issuesapp.BaseURIContextKey).(string) },
		BaseState: func(req *http.Request) issuesapp.BaseState {
			reqPath := req.URL.Path
			if reqPath == "/" {
				reqPath = "" // This is needed so that absolute URL for root view, i.e., /blog, is "/blog" and not "/blog/" because of "/blog" + "/".
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
</style>`,
		BodyPre: `{{/* Override create issue button to only show up for shurcooL as New Blog Post button. */}}
{{define "create-issue"}}
	{{if and (eq .CurrentUser.ID 1924134) (eq .CurrentUser.Domain "github.com")}}
		<div style="text-align: right;"><button class="btn btn-success btn-small" onclick="window.location = '{{.BaseURI}}/new';">New Blog Post</button></div>
	{{end}}
{{end}}

<div style="max-width: 800px; margin: 0 auto 100px auto;">`,
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
			reqPath = "" // This is needed so that absolute URL for root view, i.e., /blog, is "/blog" and not "/blog/" because of "/blog" + "/".
		}
		returnURL := (&url.URL{Path: baseURI + reqPath, RawQuery: req.URL.RawQuery}).String()
		header := component.Header{
			CurrentUser:   authenticatedUser,
			ReturnURL:     returnURL,
			Notifications: notifications,
		}
		return []htmlg.ComponentContext{header}, nil
	}
	issuesApp := issuesapp.New(onlyShurcoolCreatePosts, users, opt)

	blogHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO: Factor this out?
		u, err := getUser(req)
		if err == errBadAccessToken {
			// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
			http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
		}
		req = withUser(req, u)

		req = req.WithContext(context.WithValue(req.Context(), issuesapp.RepoSpecContextKey, blog))
		req = req.WithContext(context.WithValue(req.Context(), issuesapp.BaseURIContextKey, "/blog"))

		prefixLen := len("/blog")
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
	http.Handle("/blog", blogHandler)
	http.Handle("/blog/", blogHandler)

	return nil
}

// onlyShurcoolCreatePosts limits an issues.Service's Create method to allow only shurcooL
// to create new blog posts.
type onlyShurcoolCreatePosts struct {
	issues.Service
	users users.Service
}

func (s onlyShurcoolCreatePosts) Create(ctx context.Context, repo issues.RepoSpec, issue issues.Issue) (issues.Issue, error) {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return issues.Issue{}, err
	}
	if currentUser != shurcool {
		return issues.Issue{}, os.ErrPermission
	}
	return s.Service.Create(ctx, repo, issue)
}

func (s onlyShurcoolCreatePosts) ThreadType() string {
	return s.Service.(interface {
		ThreadType() string
	}).ThreadType()
}
