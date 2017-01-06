package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httphandler"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
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
// registers handlers for its HTTP API,
// and handlers for the notifications app.
func initNotifications(root webdav.FileSystem, users users.Service) (notifications.Service, error) {
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

	service := shurcoolSeeHisGitHubNotifications{
		service:                     fs.NewService(root, users),
		shurcoolGitHubNotifications: shurcoolGitHubNotifications,
		users: users,
	}

	// Register HTTP API endpoint.
	notificationsAPIHandler := httphandler.Notifications{Notifications: service}
	http.Handle("/api/notifications/count", userMiddleware{httputil.ErrorHandler(notificationsAPIHandler.Count)})

	// Register notifications app endpoints.
	opt := notificationsapp.Options{
		BaseURI: func(req *http.Request) string {
			return "/notifications"
		},
		BaseState: func(req *http.Request) notificationsapp.BaseState {
			reqPath := req.URL.Path
			if reqPath == "/" {
				reqPath = "" // This is needed so that absolute URL for root view, i.e., /notifications, is "/notifications" and not "/notifications/" because of "/notifications" + "/".
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
		HeadPre: `<title>Notifications</title>
<link href="/icon.png" rel="icon" type="image/png">
<link href="//cdnjs.cloudflare.com/ajax/libs/octicons/3.1.0/octicons.css" media="all" rel="stylesheet" type="text/css" />
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
	opt.BodyPre = `<div style="max-width: 800px; margin: 0 auto 100px auto;">`
	opt.BodyTop = func(req *http.Request) ([]htmlg.ComponentContext, error) {
		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return nil, err
		}
		returnURL := req.RequestURI
		header := component.Header{
			CurrentUser:   authenticatedUser,
			ReturnURL:     returnURL,
			Notifications: service,
		}
		return []htmlg.ComponentContext{header}, nil
	}
	notificationsApp := notificationsapp.New(service, users, opt)

	notificationsHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO: Factor this out?
		u, err := getUser(req)
		if err == errBadAccessToken {
			// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
			http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
		}
		req = withUser(req, u)

		prefixLen := len("/notifications")
		if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
			baseURL := prefix
			if req.URL.RawQuery != "" {
				baseURL += "?" + req.URL.RawQuery
			}
			http.Redirect(w, req, baseURL, http.StatusMovedPermanently)
			return
		}
		returnURL := req.RequestURI
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

func (s shurcoolSeeHisGitHubNotifications) List(ctx context.Context, opt notifications.ListOptions) (notifications.Notifications, error) {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return nil, err
	}

	var nss notifications.Notifications
	ns, err := s.service.List(ctx, opt)
	if err != nil {
		return nss, err
	}
	nss = append(nss, ns...)

	if currentUser == shurcool && (opt.Repo == nil || strings.HasPrefix(opt.Repo.URI, "github.com/") &&
		opt.Repo.URI != "github.com/shurcooL/issuesapp" && opt.Repo.URI != "github.com/shurcooL/notificationsapp") {

		ns, err := s.shurcoolGitHubNotifications.List(ctx, opt)
		if err != nil {
			return nss, err
		}
		nss = append(nss, ns...)
	}

	return nss, nil
}

func (s shurcoolSeeHisGitHubNotifications) Count(ctx context.Context, opt interface{}) (uint64, error) {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return 0, err
	}

	var count uint64
	n, err := s.service.Count(ctx, opt)
	if err != nil {
		return count, err
	}
	count += n

	if currentUser == shurcool {
		n, err := s.shurcoolGitHubNotifications.Count(ctx, opt)
		if err != nil {
			return count, err
		}
		count += n
	}

	return count, nil
}

func (s shurcoolSeeHisGitHubNotifications) MarkRead(ctx context.Context, appID string, repo notifications.RepoSpec, threadID uint64) error {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return err
	}

	if currentUser == shurcool && strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp" {

		return s.shurcoolGitHubNotifications.MarkRead(ctx, appID, repo, threadID)
	}

	return s.service.MarkRead(ctx, appID, repo, threadID)
}

func (s shurcoolSeeHisGitHubNotifications) MarkAllRead(ctx context.Context, repo notifications.RepoSpec) error {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return err
	}

	if currentUser == shurcool && strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp" {

		return s.shurcoolGitHubNotifications.MarkAllRead(ctx, repo)
	}

	return s.service.MarkAllRead(ctx, repo)
}

func (s shurcoolSeeHisGitHubNotifications) Subscribe(ctx context.Context, appID string, repo notifications.RepoSpec, threadID uint64, subscribers []users.UserSpec) error {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return err
	}

	if currentUser == shurcool && strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp" {

		return s.shurcoolGitHubNotifications.Subscribe(ctx, appID, repo, threadID, subscribers)
	}

	return s.service.Subscribe(ctx, appID, repo, threadID, subscribers)
}

func (s shurcoolSeeHisGitHubNotifications) Notify(ctx context.Context, appID string, repo notifications.RepoSpec, threadID uint64, nr notifications.NotificationRequest) error {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return err
	}

	if currentUser == shurcool && strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp" {

		return s.shurcoolGitHubNotifications.Notify(ctx, appID, repo, threadID, nr)
	}

	return s.service.Notify(ctx, appID, repo, threadID, nr)
}
