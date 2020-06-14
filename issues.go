package main

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

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
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/webdav"
)

func newIssuesServiceV1(root webdav.FileSystem, notifications notifications.ExternalService, events events.ExternalService, users users.Service) (issues.Service, error) {
	return fs.NewService(root, notifications, events, users)
}

// initIssuesV1 registers handlers for the issues service HTTP API,
// and handlers for the issues app.
func initIssuesV1(mux *http.ServeMux, issuesService issues.Service, notifications notifications.Service, users users.Service) {
	// Register HTTP API endpoints.
	issuesAPIHandler := httphandler.Issues{Issues: issuesService}
	mux.Handle(httproute.List, headerAuth{httputil.ErrorHandler(users, issuesAPIHandler.List)})
	mux.Handle(httproute.Count, headerAuth{httputil.ErrorHandler(users, issuesAPIHandler.Count)})
	mux.Handle(httproute.ListComments, headerAuth{httputil.ErrorHandler(users, issuesAPIHandler.ListComments)})
	mux.Handle(httproute.ListEvents, headerAuth{httputil.ErrorHandler(users, issuesAPIHandler.ListEvents)})
	mux.Handle(httproute.EditComment, headerAuth{httputil.ErrorHandler(users, issuesAPIHandler.EditComment)})

	opt := issuesapp.Options{
		Notifications: notifications,

		HeadPre: analyticsHTML + `<link href="/icon.png" rel="icon" type="image/png">
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
	a {
		color: #4183c4;
		text-decoration: none;
	}
	a:hover {
		text-decoration: underline;
	}
	a.gray {
		color: #bbb;
	}
	a.gray:hover {
		color: black;
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

	/* https://github.com/primer/primer-navigation */
	.counter{display:inline-block;padding:2px 5px;font-size:12px;font-weight:600;line-height:1;color:#666;background-color:#eee;border-radius:20px}.menu{margin-bottom:15px;list-style:none;background-color:#fff;border:1px solid #d8d8d8;border-radius:3px}.menu-item{position:relative;display:block;padding:8px 10px;border-bottom:1px solid #eee}.menu-item:first-child{border-top:0;border-top-left-radius:2px;border-top-right-radius:2px}.menu-item:first-child::before{border-top-left-radius:2px}.menu-item:last-child{border-bottom:0;border-bottom-right-radius:2px;border-bottom-left-radius:2px}.menu-item:last-child::before{border-bottom-left-radius:2px}.menu-item:hover{text-decoration:none;background-color:#f9f9f9}.menu-item.selected{font-weight:bold;color:#222;cursor:default;background-color:#fff}.menu-item.selected::before{position:absolute;top:0;bottom:0;left:0;width:2px;content:"";background-color:#d26911}.menu-item .octicon{width:16px;margin-right:5px;color:#333;text-align:center}.menu-item .counter{float:right;margin-left:5px}.menu-item .menu-warning{float:right;color:#d26911}.menu-item .avatar{float:left;margin-right:5px}.menu-item.alert .counter{color:#bd2c00}.menu-heading{display:block;padding:8px 10px;margin-top:0;margin-bottom:0;font-size:13px;font-weight:bold;line-height:20px;color:#555;background-color:#f7f7f7;border-bottom:1px solid #eee}.menu-heading:hover{text-decoration:none}.menu-heading:first-child{border-top-left-radius:2px;border-top-right-radius:2px}.menu-heading:last-child{border-bottom:0;border-bottom-right-radius:2px;border-bottom-left-radius:2px}.tabnav{margin-top:0;margin-bottom:15px;border-bottom:1px solid #ddd}.tabnav .counter{margin-left:5px}.tabnav-tabs{margin-bottom:-1px}.tabnav-tab{display:inline-block;padding:8px 12px;font-size:14px;line-height:20px;color:#666;text-decoration:none;background-color:transparent;border:1px solid transparent;border-bottom:0}.tabnav-tab.selected{color:#333;background-color:#fff;border-color:#ddd;border-radius:3px 3px 0 0}.tabnav-tab:hover,.tabnav-tab:focus{text-decoration:none}.tabnav-extra{display:inline-block;padding-top:10px;margin-left:10px;font-size:12px;color:#666}.tabnav-extra>.octicon{margin-right:2px}a.tabnav-extra:hover{color:#4078c0;text-decoration:none}.tabnav-btn{margin-left:10px}.filter-list{list-style-type:none}.filter-list.small .filter-item{padding:4px 10px;margin:0 0 2px;font-size:12px}.filter-list.pjax-active .filter-item{color:#767676;background-color:transparent}.filter-list.pjax-active .filter-item.pjax-active{color:#fff;background-color:#4078c0}.filter-item{position:relative;display:block;padding:8px 10px;margin-bottom:5px;overflow:hidden;font-size:14px;color:#767676;text-decoration:none;text-overflow:ellipsis;white-space:nowrap;cursor:pointer;border-radius:3px}.filter-item:hover{text-decoration:none;background-color:#eee}.filter-item.selected{color:#fff;background-color:#4078c0}.filter-item .count{float:right;font-weight:bold}.filter-item .bar{position:absolute;top:2px;right:0;bottom:2px;z-index:-1;display:inline-block;background-color:#f1f1f1}.subnav{margin-bottom:20px}.subnav::before{display:table;content:""}.subnav::after{display:table;clear:both;content:""}.subnav-bordered{padding-bottom:20px;border-bottom:1px solid #eee}.subnav-flush{margin-bottom:0}.subnav-item{position:relative;float:left;padding:6px 14px;font-weight:600;line-height:20px;color:#666;border:1px solid #e5e5e5}.subnav-item+.subnav-item{margin-left:-1px}.subnav-item:hover,.subnav-item:focus{text-decoration:none;background-color:#f5f5f5}.subnav-item.selected,.subnav-item.selected:hover,.subnav-item.selected:focus{z-index:2;color:#fff;background-color:#4078c0;border-color:#4078c0}.subnav-item:first-child{border-top-left-radius:3px;border-bottom-left-radius:3px}.subnav-item:last-child{border-top-right-radius:3px;border-bottom-right-radius:3px}.subnav-search{position:relative;margin-left:10px}.subnav-search-input{width:320px;padding-left:30px;color:#767676;border-color:#d5d5d5}.subnav-search-input-wide{width:500px}.subnav-search-icon{position:absolute;top:9px;left:8px;display:block;color:#ccc;text-align:center;pointer-events:none}.subnav-search-context .btn{color:#555;border-top-right-radius:0;border-bottom-right-radius:0}.subnav-search-context .btn:hover,.subnav-search-context .btn:focus,.subnav-search-context .btn:active,.subnav-search-context .btn.selected{z-index:2}.subnav-search-context+.subnav-search{margin-left:-1px}.subnav-search-context+.subnav-search .subnav-search-input{border-top-left-radius:0;border-bottom-left-radius:0}.subnav-search-context .select-menu-modal-holder{z-index:30}.subnav-search-context .select-menu-modal{width:220px}.subnav-search-context .select-menu-item-icon{color:inherit}.subnav-spacer-right{padding-right:10px}
</style>`,
		HeadPost: `<style type="text/css">
	.markdown-body { font-family: Go; }
	tt, code, pre  { font-family: "Go Mono"; }
</style>`,
		BodyPre: `<div style="max-width: 800px; margin: 0 auto 100px auto;">`,
		SignIn:  signInViaURL,
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

	for _, repo := range [...]struct{ SpecURL, BaseURL string }{
		{SpecURL: "dmitri.shuralyov.com/idiomatic-go", BaseURL: "/idiomatic-go/entries"},
	} {
		issuesHandler := cookieAuth{httputil.ErrorHandler(users, issuesHandler{
			SpecURL:   repo.SpecURL,
			BaseURL:   repo.BaseURL,
			issuesApp: issuesApp,
		}.ServeHTTP)}
		mux.Handle(repo.BaseURL, issuesHandler)
		mux.Handle(repo.BaseURL+"/", issuesHandler)
	}
}

func signInViaURL(returnURL string) template.HTML {
	const style = `a.Login { color: #4183c4; text-decoration: none; } a.Login:hover { text-decoration: underline; }`
	u := url.URL{Path: "/login", RawQuery: url.Values{"return": {returnURL}}.Encode()}
	signInViaURL := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "Login"},
			{Key: atom.Href.String(), Val: u.String()},
		},
		FirstChild: htmlg.Text("Sign in via URL"),
	}
	return template.HTML("<style>" + style + "</style>" + htmlg.Render(signInViaURL))
}

type issuesHandler struct {
	SpecURL   string
	BaseURL   string
	Repo      repoInfo // Set for self-hosted Go package issue trackers only.
	issuesApp http.Handler
}

func (h issuesHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	prefixLen := len(h.BaseURL)
	if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
		baseURL := prefix
		if req.URL.RawQuery != "" {
			baseURL += "?" + req.URL.RawQuery
		}
		return httperror.Redirect{URL: baseURL}
	}
	returnURL := req.RequestURI
	req = copyRequestAndURL(req)
	req.URL.Path = req.URL.Path[prefixLen:]
	if req.URL.Path == "" {
		req.URL.Path = "/"
	}
	rr := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), repoInfoContextKey, h.Repo)) // For BodyTop.
	req = req.WithContext(context.WithValue(req.Context(), issuesapp.RepoSpecContextKey, issues.RepoSpec{URI: h.SpecURL}))
	req = req.WithContext(context.WithValue(req.Context(), issuesapp.BaseURIContextKey, h.BaseURL))
	h.issuesApp.ServeHTTP(rr, req)
	// TODO: Have issuesApp.ServeHTTP return error, check if os.IsPermission(err) is true, etc.
	// TODO: Factor out this os.IsPermission(err) && u == nil check somewhere, if possible. (But this shouldn't apply for APIs.)
	if s := req.Context().Value(sessionContextKey).(*session); rr.Code == http.StatusForbidden && s == nil {
		loginURL := (&url.URL{
			Path:     "/login",
			RawQuery: url.Values{returnParameterName: {returnURL}}.Encode(),
		}).String()
		return httperror.Redirect{URL: loginURL}
	}
	for k, vs := range rr.Header() {
		w.Header()[k] = vs
	}
	w.WriteHeader(rr.Code)
	_, err := io.Copy(w, rr.Body)
	return err
}
