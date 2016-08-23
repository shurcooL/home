package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notifications/fs"
	"github.com/shurcooL/notifications/githubapi"
	"github.com/shurcooL/notificationsapp"
	"github.com/shurcooL/notificationsapp/common"
	"github.com/shurcooL/users"
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
	shurcoolGitHubNotifications := githubapi.NewService(
		github.NewClient(&http.Client{Transport: authTransport}),
		github.NewClient(&http.Client{Transport: cacheTransport}),
	)

	shurcoolSeeHisGitHubNotificationsService := shurcoolSeeHisGitHubNotifications{
		service:                     service,
		shurcoolGitHubNotifications: shurcoolGitHubNotifications,
		users: users,
	}

	opt := notificationsapp.Options{
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
		u, err := getUser(req)
		if err == errBadAccessToken {
			// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
			http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
		}
		req = req.WithContext(context.WithValue(req.Context(), userContextKey, u))

		prefixLen := len("/notifications")
		if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
			baseURL := prefix
			if req.URL.RawQuery != "" {
				baseURL += "?" + req.URL.RawQuery
			}
			http.Redirect(w, req, baseURL, http.StatusMovedPermanently)
			return
		}
		returnURL := req.URL.String()
		req.URL.Path = req.URL.Path[prefixLen:]
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		rr := httptest.NewRecorder()
		rr.HeaderMap = w.Header()
		notificationsApp.ServeHTTP(rr, req)
		if rr.Code == http.StatusUnauthorized && u == nil {
			loginURL := (&url.URL{
				Path:     "/login",
				RawQuery: url.Values{returnQueryName: {returnURL}}.Encode(),
			}).String()
			http.Redirect(w, req, loginURL, http.StatusSeeOther)
			return
		}
		w.WriteHeader(rr.Code)
		io.Copy(w, rr.Body)
	})
	http.Handle("/notifications", notificationsHandler)
	http.Handle("/notifications/", notificationsHandler)

	return service, nil
}

// shurcoolSeeHisGitHubNotifications lets shurcooL also see his GitHub notifications,
// in addition to local ones.
type shurcoolSeeHisGitHubNotifications struct {
	service                     notifications.Service
	shurcoolGitHubNotifications notifications.Service
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

func (s shurcoolSeeHisGitHubNotifications) MarkRead(ctx context.Context, appID string, repo notifications.RepoSpec, threadID uint64) error {
	if currentUser, err := s.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == shurcool &&
		strings.HasPrefix(repo.URI, "github.com/") {

		return s.shurcoolGitHubNotifications.MarkRead(ctx, appID, repo, threadID)
	}

	return s.service.MarkRead(ctx, appID, repo, threadID)
}

func (s shurcoolSeeHisGitHubNotifications) MarkAllRead(ctx context.Context, repo notifications.RepoSpec) error {
	if currentUser, err := s.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == shurcool &&
		strings.HasPrefix(repo.URI, "github.com/") {

		return s.shurcoolGitHubNotifications.MarkAllRead(ctx, repo)
	}

	return s.service.MarkAllRead(ctx, repo)
}

func (s shurcoolSeeHisGitHubNotifications) Subscribe(ctx context.Context, appID string, repo notifications.RepoSpec, threadID uint64, subscribers []users.UserSpec) error {
	return fmt.Errorf("shurcoolSeeHisGitHubNotifications.Subscribe not implemented")
}
func (s shurcoolSeeHisGitHubNotifications) Notify(ctx context.Context, appID string, repo notifications.RepoSpec, threadID uint64, nr notifications.NotificationRequest) error {
	return fmt.Errorf("shurcoolSeeHisGitHubNotifications.Notify not implemented")
}
