package main

import (
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issues/fs"
	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/issuesapp/common"
	"github.com/shurcooL/users"
	"golang.org/x/net/context"
)

var requestKey struct{}

// initBlog registers a blog handler with blog URI as source, based in rootDir.
func initBlog(rootDir string, blog issues.RepoSpec) error {
	var othersCantCreateBlogPostsService issues.Service
	{
		var transport http.RoundTripper
		transport = &github.UnauthenticatedRateLimitedTransport{
			ClientID:     gitHubConfig.ClientID,
			ClientSecret: gitHubConfig.ClientSecret,
		}
		transport = &httpcache.Transport{
			Transport:           transport,
			Cache:               httpcache.NewMemoryCache(),
			MarkCachedResponses: true,
		}
		usersService = Users{gh: github.NewClient(&http.Client{Transport: transport})}
		service, err := fs.NewService(rootDir, usersService)
		if err != nil {
			return err
		}
		othersCantCreateBlogPostsService = othersCantCreateBlogPosts{Service: service, users: usersService}
	}

	opt := issuesapp.Options{
		Context: func(req *http.Request) context.Context {
			// TODO, THINK.
			return context.WithValue(context.Background(), requestKey, req)
		},
		RepoSpec: func(req *http.Request) issues.RepoSpec { return blog },
		BaseURI:  func(req *http.Request) string { return "/blog" },
		BaseState: func(req *http.Request) issuesapp.BaseState {
			reqPath := req.URL.Path
			if reqPath == "/" {
				reqPath = ""
			}
			return issuesapp.BaseState{
				State: common.State{
					BaseURI: "/blog",
					ReqPath: reqPath,
				},
			}
		},
		HeadPre: `<!--link href="//cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.0.0-alpha/css/bootstrap.css" media="all" rel="stylesheet" type="text/css" /-->
<style type="text/css">
	body {
		font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
		font-size: 14px;
		line-height: initial;
		margin: 20px;
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
		BodyPre: `<div style="text-align: right; margin-bottom: 20px; height: 18px; font-size: 12px;">
	{{if .CurrentUser}}
		<a class="topbar-avatar" href="{{.CurrentUser.HTMLURL}}" target="_blank" tabindex=-1
			><img class="topbar-avatar" src="{{.CurrentUser.AvatarURL}}" title="Signed in as {{.CurrentUser.Login}}."
		></a>
		<form method="post" action="/logout" style="display: inline-block; margin-bottom: 0;"><input class="btn" type="submit" value="Sign out"><input type="hidden" name="return" value="{{.BaseURI}}{{.ReqPath}}"></form>
	{{else}}
		<form method="post" action="/login/github" style="display: inline-block; margin-bottom: 0;"><input class="btn" type="submit" value="Sign in via GitHub"><input type="hidden" name="return" value="{{.BaseURI}}{{.ReqPath}}"></form>
	{{end}}
</div>`,
	}
	if *productionFlag {
		opt.HeadPre += "\n\t\t" + googleAnalytics
	}
	issuesApp := issuesapp.New(othersCantCreateBlogPostsService, opt)

	blogHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO: Factor this out?
		_, err := getUser(req)
		if err == errBadAccessToken {
			// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
			http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
		}

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

// othersCantCreateBlogPosts limits an issues.Service's Create method to allow only shurcooL
// to create new blog posts.
type othersCantCreateBlogPosts struct {
	issues.Service
	users users.Service
}

func (s othersCantCreateBlogPosts) Create(ctx context.Context, repo issues.RepoSpec, issue issues.Issue) (issues.Issue, error) {
	currentUser, err := s.users.GetAuthenticated(ctx)
	if err != nil {
		return issues.Issue{}, err
	}
	shurcooL := users.UserSpec{ID: 1924134, Domain: "github.com"}
	if currentUser == nil || *currentUser != shurcooL {
		return issues.Issue{}, os.ErrPermission
	}
	return s.Service.Create(ctx, repo, issue)
}
