package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/issuesapp"
	"github.com/shurcooL/issuesapp/common"
	"github.com/shurcooL/play/173/wordpress"
	"golang.org/x/net/context"
	"src.sourcegraph.com/apps/tracker/issues"
)

// initBlog registers a blog handler with path to blog XML file.
func initBlog(path string) error {
	service, err := wordpress.NewService(path)
	if err != nil {
		log.Println("failed to init blog, going ahead without it:", err)
		return nil
	}

	opt := issuesapp.Options{
		Context:   func(req *http.Request) context.Context { return context.TODO() },
		RepoSpec:  func(req *http.Request) issues.RepoSpec { return issues.RepoSpec{} },
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
<link href="//cdnjs.cloudflare.com/ajax/libs/octicons/3.1.0/octicons.css" media="all" rel="stylesheet" type="text/css" />
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
