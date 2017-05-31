package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notifications/fs"
	"github.com/shurcooL/notifications/githubapi"
	"github.com/shurcooL/notificationsapp"
	"github.com/shurcooL/notificationsapp/httphandler"
	"github.com/shurcooL/notificationsapp/httproute"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
	"golang.org/x/oauth2"
)

// initNotifications creates and returns a notification service,
// registers handlers for its HTTP API,
// and handlers for the notifications app.
func initNotifications(mux *http.ServeMux, root webdav.FileSystem, users users.Service) (notifications.Service, error) {
	authTransport := &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("HOME_GH_SHURCOOL_NOTIFICATIONS")}),
	}
	cacheTransport := &httpcache.Transport{
		Transport:           authTransport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}
	shurcoolGitHubNotifications := githubapi.NewService(
		github.NewClient(&http.Client{Transport: authTransport, Timeout: 5 * time.Second}),
		github.NewClient(&http.Client{Transport: cacheTransport, Timeout: 5 * time.Second}),
	)

	notificationsService := shurcoolSeeHisGitHubNotifications{
		service:                     fs.NewService(root, users),
		shurcoolGitHubNotifications: shurcoolGitHubNotifications,
		users: users,
	}

	// Register HTTP API endpoints.
	notificationsAPIHandler := httphandler.Notifications{Notifications: notificationsService}
	mux.Handle(httproute.List, apiMiddleware{httputil.ErrorHandler(users, notificationsAPIHandler.List)})
	mux.Handle(httproute.Count, apiMiddleware{httputil.ErrorHandler(users, notificationsAPIHandler.Count)})
	mux.Handle(httproute.MarkRead, apiMiddleware{httputil.ErrorHandler(users, notificationsAPIHandler.MarkRead)})
	mux.Handle(httproute.MarkAllRead, apiMiddleware{httputil.ErrorHandler(users, notificationsAPIHandler.MarkAllRead)})

	// Register notifications app endpoints.
	opt := notificationsapp.Options{
		HeadPre: `<title>Notifications</title>
<link href="/icon.png" rel="icon" type="image/png">
<meta name="viewport" content="width=device-width">
<style type="text/css">
	body {
		margin: 20px;
	}
	body, table {
		font-family: sans-serif;
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
	opt.BodyTop = func(req *http.Request) ([]htmlg.Component, error) {
		authenticatedUser, err := users.GetAuthenticated(req.Context())
		if err != nil {
			return nil, err
		}
		var nc uint64
		if authenticatedUser.ID != 0 {
			nc, err = notificationsService.Count(req.Context(), nil)
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
	notificationsApp := notificationsapp.New(notificationsService, opt)

	notificationsHandler := userMiddleware{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		prefixLen := len("/notifications")
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
		rr.HeaderMap = w.Header()
		req = req.WithContext(context.WithValue(req.Context(), notificationsapp.BaseURIContextKey, "/notifications"))
		notificationsApp.ServeHTTP(rr, req)
		// TODO: Have notificationsApp.ServeHTTP return error, check if os.IsPermission(err) is true, etc.
		// TODO: Factor out this os.IsPermission(err) && u == nil check somewhere, if possible. (But this shouldn't apply for APIs.)
		if u := req.Context().Value(userContextKey).(*user); rr.Code == http.StatusForbidden && u == nil {
			loginURL := (&url.URL{
				Path:     "/login",
				RawQuery: url.Values{returnQueryName: {returnURL}}.Encode(),
			}).String()
			return httperror.Redirect{URL: loginURL}
		}
		w.WriteHeader(rr.Code)
		_, err := io.Copy(w, rr.Body)
		return err
	})}
	mux.Handle("/notifications", notificationsHandler)
	mux.Handle("/notifications/", notificationsHandler)

	return notificationsService, nil
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
