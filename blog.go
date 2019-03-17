package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	blogpkg "github.com/shurcooL/home/internal/page/blog"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

var blogHTML = template.Must(template.New("").Parse(`<html>
	<head>
{{.AnalyticsHTML}}		<title>Dmitri Shuralyov - Blog</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<link href="/blog/assets/gfm/gfm.css" rel="stylesheet" type="text/css">
		<style type="text/css">
			.markdown-body { font-family: Go; }
			tt, code, pre  { font-family: "Go Mono"; }
		</style>
		<link href="/assets/blog/style.css" rel="stylesheet" type="text/css">
		<script async src="/assets/blog/blog.js"></script>
	</head>
	<body>`))

// initBlog registers a blog handler with blog URI as blog content source.
func initBlog(mux *http.ServeMux, issuesService issues.Service, blog issues.RepoSpec, notifications notifications.Service, users users.Service) error {
	shurcoolBlogService := shurcoolBlogService{
		Service: issuesService,
		users:   users,
	}

	opt := issuesapp.Options{
		Notifications: notifications,

		HeadPre: analyticsHTML + `<title>Dmitri Shuralyov - Blog</title>
<link href="/icon.png" rel="icon" type="image/png">
<meta name="viewport" content="width=device-width">
<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
<style type="text/css">
	body {
		margin: 20px;
		font-family: Go;
		font-size: 14px;
		line-height: initial;
		color: rgb(35, 35, 35);
	}
	.btn {
		font-family: inherit;
		font-size: 11px;
		line-height: 11px;
		height: 18px;
		border-radius: 4px;
		border: solid #d2d2d2 1px;
		background-color: #fff;
		box-shadow: 0 1px 1px rgba(0, 0, 0, .05);
	}

	.post .markdown-body {
		font-size: 16px;
    	line-height: 1.6;
    }
    .post .black-link a, .black-link a:focus, .black-link a:hover {
		color: rgb(35, 35, 35);
	}
	.post ul.post-meta {
		padding-left: 0;
		list-style: none;
		margin-top: 10px;
		margin-bottom: 20px;

		font-family: inherit;
		font-size: 14px;
		line-height: 18px;
		color: #999;
	}
	.post li.post-meta {
		display: inline-block;
		margin-right: 30px;
	}
	.post div.reactable-container {
		display: inline-block;
		vertical-align: top;
		margin-left: 0;
	}
	.post .reaction-bar-appear:hover div.new-reaction {
		display: inline-block;
	}
	/* Make new-reaction button visible if there are no other reactions. */
	.post div.reactable-container a:first-child div.new-reaction {
		display: inline-block;
	}
</style>`,
		HeadPost: `<style type="text/css">
	.markdown-body { font-family: Go; }
	tt, code, pre  { font-family: "Go Mono"; }
</style>`,
		BodyPre: `{{/* Override create issue button to only show up for shurcooL as New Blog Post button. */}}
{{define "create-issue"}}
	{{if and (eq .CurrentUser.ID 1924134) (eq .CurrentUser.Domain "github.com")}}
		<div style="text-align: right;"><button class="btn btn-success btn-small" onclick="window.location = '{{.BaseURI}}/new';">New Blog Post</button></div>
	{{end}}
{{end}}

{{define "issue"}}
	{{if .ForceIssuesApp}}
		<h1>{{.Issue.Title}} <span class="gray">#{{.Issue.ID}}</span></h1>
		<div id="issue-state-badge" style="margin-bottom: 20px;">{{render (issueStateBadge .Issue)}}</div>
	{{else}}
		<h2 id="comments">Comments</h2>
	{{end}}
	{{range .Items}}
		{{template "issue-item" .}}
	{{end}}
	<div id="new-item-marker"></div>
	{{if (and (eq .CurrentUser.ID 0) (not .Items))}}
		{{/* HACK: Negative offset to make "Sign in via GitHub to comment." appear aligned. */}}
		<div style="margin-left: -58px;">{{template "new-comment" .}}</div>
	{{else}}
		{{template "new-comment" .}}
	{{end}}
{{end}}

<div style="max-width: 800px; margin: 0 auto 100px auto;">`,
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

		// Check if we're on an idividual blog post /{id:[0-9]+} page.
		// This is a copy of issueapp's router logic.
		_, forceIssuesApp := req.Context().Value(forceIssuesAppContextKey).(struct{})
		if issueID, err := strconv.ParseUint(req.URL.Path[1:], 10, 64); err == nil && !forceIssuesApp {
			issue, err := issuesService.Get(req.Context(), blog, issueID)
			if err != nil {
				return nil, err
			}
			comments, err := issuesService.ListComments(req.Context(), blog, issueID, &issues.ListOptions{Length: 1})
			if err != nil {
				return nil, err
			}
			if len(comments) == 0 {
				return nil, fmt.Errorf("blog post %d has no body", issueID)
			}
			issue.Comment = comments[0]
			post := blogpkg.Post{CurrentUser: authenticatedUser, Issue: issue}

			return []htmlg.Component{header, post}, nil
		}

		// If this is not an issue page, that's okay, only include the header.
		return []htmlg.Component{header}, nil
	}
	issuesApp := issuesapp.New(shurcoolBlogService, users, opt)

	blogHandler := cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		prefixLen := len("/blog")
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
		forceIssuesApp, _ := strconv.ParseBool(req.URL.Query().Get("issuesapp"))
		switch {
		case req.URL.Path == "/" && !forceIssuesApp:
			if req.Method != "GET" {
				return httperror.Method{Allowed: []string{"GET"}}
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			data := struct{ AnalyticsHTML template.HTML }{analyticsHTML}
			err := blogHTML.Execute(w, data)
			if err != nil {
				return err
			}

			authenticatedUser, err := users.GetAuthenticated(req.Context())
			if err != nil {
				return err // THINK: Should it be a fatal error or not? What about on frontend vs backend?
			}
			returnURL := req.RequestURI
			err = blogpkg.RenderBodyInnerHTML(req.Context(), w, issuesService, blog, notifications, authenticatedUser, returnURL)
			if err != nil {
				return err
			}

			_, err = io.WriteString(w, `</body></html>`)
			return err
		default:
			req = req.WithContext(context.WithValue(req.Context(), issuesapp.RepoSpecContextKey, blog))
			req = req.WithContext(context.WithValue(req.Context(), issuesapp.BaseURIContextKey, "/blog"))
			if forceIssuesApp {
				req = req.WithContext(context.WithValue(req.Context(), forceIssuesAppContextKey, struct{}{}))
			}
			issuesApp.ServeHTTP(w, req)
			return nil
		}
	})}
	mux.Handle("/blog", blogHandler)
	mux.Handle("/blog/", blogHandler)

	return nil
}

// shurcoolBlogService skips first comment (the issue body), because we're
// taking on responsibility to render it ourselves (unless forceIssuesApp
// is set). It also limits an issues.Service's Create method to allow only
// shurcooL to create new blog posts.
type shurcoolBlogService struct {
	issues.Service
	users users.Service
}

func (s shurcoolBlogService) ListComments(ctx context.Context, repo issues.RepoSpec, id uint64, opt *issues.ListOptions) ([]issues.Comment, error) {
	cs, listCommentsError := s.Service.ListComments(ctx, repo, id, opt)
	_, forceIssuesApp := ctx.Value(forceIssuesAppContextKey).(struct{})
	if len(cs) >= 1 && !forceIssuesApp {
		// Skip first comment (the issue body), we're taking on responsibility
		// to render it ourselves.
		cs = cs[1:]
	}
	return cs, listCommentsError
}

func (s shurcoolBlogService) Create(ctx context.Context, repo issues.RepoSpec, issue issues.Issue) (issues.Issue, error) {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return issues.Issue{}, err
	}
	if currentUser != shurcool {
		return issues.Issue{}, os.ErrPermission
	}
	return s.Service.Create(ctx, repo, issue)
}

func (s shurcoolBlogService) ThreadType(repo issues.RepoSpec) string {
	return s.Service.(interface {
		ThreadType(issues.RepoSpec) string
	}).ThreadType(repo)
}

// forceIssuesAppContextKey is a context key. It can be used to check whether
// issuesapp is being forced upon the blog. The associated value will be of type struct{}.
// Eventually, a better solution should be found, and this removed.
var forceIssuesAppContextKey = &contextKey{"ForceIssuesApp"}
