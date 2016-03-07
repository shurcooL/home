package main

import (
	"net/http"

	"github.com/shurcooL/fsissues"
	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/issuesapp/common"
	"github.com/shurcooL/users"
	"golang.org/x/net/context"
	"src.sourcegraph.com/apps/tracker/issues"
)

// initBlog registers a blog handler with blog URI as source, based in rootDir.
func initBlog(rootDir string, blog issues.RepoSpec) error {
	users := users.Static{}
	service, err := fs.NewService(rootDir, users)
	if err != nil {
		return err
	}

	opt := issuesapp.Options{
		Context:   func(req *http.Request) context.Context { return context.TODO() },
		RepoSpec:  func(req *http.Request) issues.RepoSpec { return blog },
		BaseURI:   func(req *http.Request) string { return "/blog" },
		CSRFToken: func(req *http.Request) string { return "" },
		Verbatim:  func(w http.ResponseWriter) {},
		BaseState: func(req *http.Request) issuesapp.BaseState {
			reqPath := req.URL.Path
			if reqPath == "/" {
				reqPath = ""
			}
			return issuesapp.BaseState{
				State: common.State{
					BaseURI:   "/blog",
					ReqPath:   reqPath,
					CSRFToken: "",
				},
			}
		},
		HeadPre: `<link href="//cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.0.0-alpha/css/bootstrap.css" media="all" rel="stylesheet" type="text/css" />
<style type="text/css">
	body {
		font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
		font-size: 14px;
		line-height: initial;
		margin: 20px;
	}
	.btn {
		font-size: 14px;
	}
</style>`,
	}
	issuesApp := issuesapp.New(service, opt)

	blogHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
