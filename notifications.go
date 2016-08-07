package main

import (
	"net/http"

	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notifications/fs"
	"github.com/shurcooL/notificationsapp"
	"github.com/shurcooL/notificationsapp/common"
	"github.com/shurcooL/users"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
)

// initNotifications creates and returns a notification service,
// and registers handlers for the notifications app.
func initNotifications(root webdav.FileSystem, users users.Service) (notifications.ExternalService, error) {
	service := fs.NewService(root, users)

	opt := notificationsapp.Options{
		Context: func(req *http.Request) context.Context {
			// TODO, THINK.
			return context.WithValue(context.Background(), requestContextKey, req)
		},
		BaseURI: func(req *http.Request) string {
			return "/notifications"
		},
		BaseState: func(req *http.Request) notificationsapp.BaseState {
			reqPath := req.URL.Path
			if reqPath == "/" {
				reqPath = ""
			}
			return notificationsapp.BaseState{
				State: common.State{
					BaseURI: "/notifications",
					ReqPath: reqPath,
				},
			}
		},
		// TODO: Update and unify octicons.css.
		//       But be mindful of https://github.com/shurcooL/notifications/blob/c38c34c46358723f7f329fa80f9a4ae105b60985/notifications.go#L39.
		HeadPre: `<link href="//cdnjs.cloudflare.com/ajax/libs/octicons/3.1.0/octicons.css" media="all" rel="stylesheet" type="text/css" />
<style type="text/css">
	body {
		margin: 20px;
	}
	body, table {
		font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
		font-size: 14px;
		line-height: initial;
		color: #373a3c;
	}
</style>`,
	}
	if *productionFlag {
		opt.HeadPre += "\n\t\t" + googleAnalytics
	}
	notificationsApp := notificationsapp.New(service, users, opt)

	notificationsHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO: Factor this out?
		_, err := getUser(req)
		if err == errBadAccessToken {
			// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
			http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
		}

		prefixLen := len("/notifications")
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
		notificationsApp.ServeHTTP(w, req)
	})
	http.Handle("/notifications", notificationsHandler)
	http.Handle("/notifications/", notificationsHandler)

	return service, nil
}
