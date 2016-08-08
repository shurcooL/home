package main

import (
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notifications/fs"
	"github.com/shurcooL/notifications/githubapi"
	"github.com/shurcooL/notificationsapp"
	"github.com/shurcooL/notificationsapp/common"
	"github.com/shurcooL/users"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
	"golang.org/x/oauth2"
)

// initNotifications creates and returns a notification service,
// and registers handlers for the notifications app.
func initNotifications(root webdav.FileSystem, users users.Service) (notifications.ExternalService, error) {
	service := fs.NewService(root, users)

	authTransport := &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("HOME_GH_SHURCOOL_NOTIFICATIONS")}),
	}
	cacheTransport := &httpcache.Transport{
		Transport:           authTransport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}
	var shurcoolGitHubNotifications notifications.InternalService = githubapi.NewService(
		github.NewClient(&http.Client{Transport: authTransport}),
		github.NewClient(&http.Client{Transport: cacheTransport}),
	)

	shurcoolSeeHisGitHubNotificationsService := shurcoolSeeHisGitHubNotifications{
		service:                     service,
		shurcoolGitHubNotifications: shurcoolGitHubNotifications,
		users: users,
	}

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
	notificationsApp := notificationsapp.New(shurcoolSeeHisGitHubNotificationsService, users, opt)

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

// shurcoolSeeHisGitHubNotifications lets shurcooL also see his GitHub notifications,
// in addition to local ones.
type shurcoolSeeHisGitHubNotifications struct {
	service                     notifications.InternalService
	shurcoolGitHubNotifications notifications.InternalService
	users                       users.Service
}

func (s shurcoolSeeHisGitHubNotifications) List(ctx context.Context, opt interface{}) (notifications.Notifications, error) {
	var nss notifications.Notifications
	ns, err := s.service.List(ctx, opt)
	if err != nil {
		return nss, err
	}
	nss = append(nss, ns...)

	if currentUser, err := s.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == shurcool {
		ns, err := s.shurcoolGitHubNotifications.List(ctx, opt)
		if err != nil {
			return nss, err
		}
		nss = append(nss, ns...)
	}

	return nss, nil
}

func (s shurcoolSeeHisGitHubNotifications) Count(ctx context.Context, opt interface{}) (uint64, error) {
	var count uint64
	n, err := s.service.Count(ctx, opt)
	if err != nil {
		return count, err
	}
	count += n

	if currentUser, err := s.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == shurcool {
		n, err := s.shurcoolGitHubNotifications.Count(ctx, opt)
		if err != nil {
			return count, err
		}
		count += n
	}

	return count, nil
}
