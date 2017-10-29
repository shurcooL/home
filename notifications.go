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
	"github.com/shurcooL/githubql"
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
	httpClient := &http.Client{Transport: cacheTransport, Timeout: 5 * time.Second}
	shurcoolGitHubNotifications := githubapi.NewService(
		github.NewClient(httpClient),
		githubql.NewClient(httpClient),
		notificationsRouter{},
	)

	notificationsService := shurcoolSeesGitHubNotifications{
		service:                     fs.NewService(root, users),
		shurcoolGitHubNotifications: shurcoolGitHubNotifications,
		users: users,
	}

	// Register HTTP API endpoints.
	notificationsAPIHandler := httphandler.Notifications{Notifications: notificationsService}
	mux.Handle(httproute.List, headerAuth{httputil.ErrorHandler(users, notificationsAPIHandler.List)})
	mux.Handle(httproute.Count, headerAuth{httputil.ErrorHandler(users, notificationsAPIHandler.Count)})
	mux.Handle(httproute.MarkRead, headerAuth{httputil.ErrorHandler(users, notificationsAPIHandler.MarkRead)})
	mux.Handle(httproute.MarkAllRead, headerAuth{httputil.ErrorHandler(users, notificationsAPIHandler.MarkAllRead)})

	// Register notifications app endpoints.
	opt := notificationsapp.Options{
		HeadPre: `<title>Notifications</title>
<link href="/icon.png" rel="icon" type="image/png">
<meta name="viewport" content="width=device-width">
<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
<style type="text/css">
	body {
		margin: 20px;
	}
	body, table {
		font-family: Go;
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
	notificationsApp := notificationsapp.New(notificationsService, users, opt)

	notificationsHandler := cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
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
		if s := req.Context().Value(sessionContextKey).(*session); rr.Code == http.StatusForbidden && s == nil {
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

// shurcoolSeesGitHubNotifications lets shurcooL also see notifications on GitHub,
// in addition to local ones.
type shurcoolSeesGitHubNotifications struct {
	service                     notifications.Service
	shurcoolGitHubNotifications notifications.Service
	users                       users.Service
}

func (s shurcoolSeesGitHubNotifications) List(ctx context.Context, opt notifications.ListOptions) (notifications.Notifications, error) {
	var nss notifications.Notifications
	ns, err := s.service.List(ctx, opt)
	if err != nil {
		return nss, err
	}
	nss = append(nss, ns...)

	if opt.Repo == nil || strings.HasPrefix(opt.Repo.URI, "github.com/") &&
		opt.Repo.URI != "github.com/shurcooL/issuesapp" && opt.Repo.URI != "github.com/shurcooL/notificationsapp" {

		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser == shurcool {
			ns, err := s.shurcoolGitHubNotifications.List(ctx, opt)
			if err != nil {
				return nss, err
			}
			nss = append(nss, ns...)
		}
	}

	return nss, nil
}

func (s shurcoolSeesGitHubNotifications) Count(ctx context.Context, opt interface{}) (uint64, error) {
	var count uint64
	n, err := s.service.Count(ctx, opt)
	if err != nil {
		return count, err
	}
	count += n

	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return 0, err
	}
	if currentUser == shurcool {
		n, err := s.shurcoolGitHubNotifications.Count(ctx, opt)
		if err != nil {
			return count, err
		}
		count += n
	}

	return count, nil
}

func (s shurcoolSeesGitHubNotifications) MarkRead(ctx context.Context, repo notifications.RepoSpec, threadType string, threadID uint64) error {
	if strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp" {
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != shurcool {
			return os.ErrPermission
		}
		return s.shurcoolGitHubNotifications.MarkRead(ctx, repo, threadType, threadID)
	}

	return s.service.MarkRead(ctx, repo, threadType, threadID)
}

func (s shurcoolSeesGitHubNotifications) MarkAllRead(ctx context.Context, repo notifications.RepoSpec) error {
	if strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp" {
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != shurcool {
			return os.ErrPermission
		}
		return s.shurcoolGitHubNotifications.MarkAllRead(ctx, repo)
	}

	return s.service.MarkAllRead(ctx, repo)
}

func (s shurcoolSeesGitHubNotifications) Subscribe(ctx context.Context, repo notifications.RepoSpec, threadType string, threadID uint64, subscribers []users.UserSpec) error {
	if strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp" {
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != shurcool {
			return os.ErrPermission
		}
		return s.shurcoolGitHubNotifications.Subscribe(ctx, repo, threadType, threadID, subscribers)
	}

	return s.service.Subscribe(ctx, repo, threadType, threadID, subscribers)
}

func (s shurcoolSeesGitHubNotifications) Notify(ctx context.Context, repo notifications.RepoSpec, threadType string, threadID uint64, nr notifications.NotificationRequest) error {
	if strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp" {
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != shurcool {
			return os.ErrPermission
		}
		return s.shurcoolGitHubNotifications.Notify(ctx, repo, threadType, threadID, nr)
	}

	return s.service.Notify(ctx, repo, threadType, threadID, nr)
}
