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

	"dmitri.shuralyov.com/route/github"
	githubv3 "github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notifications/githubapi"
	"github.com/shurcooL/notificationsapp"
	"github.com/shurcooL/notificationsapp/httphandler"
	"github.com/shurcooL/notificationsapp/httproute"
	"github.com/shurcooL/users"
	"golang.org/x/oauth2"
)

// initNotifications creates and returns a notification service,
// registers handlers for its HTTP API,
// and handlers for the notifications app.
func initNotifications(
	mux *http.ServeMux,
	localNotifications notifications.Service,
	gerritNotifications notifications.Service,
	users users.Service,
	router github.Router,
) notifications.Service {
	authTransport := &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("HOME_GH_DMITSHUR_NOTIFICATIONS")}),
	}
	cacheTransport := &httpcache.Transport{
		Transport:           authTransport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}
	dmitshurGitHubNotifications := githubapi.NewService(
		githubv3.NewClient(&http.Client{Transport: cacheTransport, Timeout: 10 * time.Second}),
		githubv4.NewClient(&http.Client{Transport: authTransport, Timeout: 10 * time.Second}),
		router,
	)

	notificationsService := dmitshurSeesExternalNotifications{
		local:                       localNotifications,
		dmitshurGitHubNotifications: dmitshurGitHubNotifications,
		dmitshurGerritNotifications: gerritNotifications,
		users:                       users,
	}

	// Register HTTP API endpoints.
	notificationsAPIHandler := httphandler.Notifications{Notifications: notificationsService}
	mux.Handle(httproute.List, headerAuth{httputil.ErrorHandler(users, notificationsAPIHandler.List)})
	mux.Handle(httproute.Count, headerAuth{httputil.ErrorHandler(users, notificationsAPIHandler.Count)})
	mux.Handle(httproute.MarkRead, headerAuth{httputil.ErrorHandler(users, notificationsAPIHandler.MarkRead)})
	mux.Handle(httproute.MarkAllRead, headerAuth{httputil.ErrorHandler(users, notificationsAPIHandler.MarkAllRead)})

	// Register notifications app endpoints.
	opt := notificationsapp.Options{
		HeadPre: analyticsHTML + `<title>Notifications</title>
<link href="/icon.svg" rel="icon" type="image/svg+xml">
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
		color: rgb(35, 35, 35);
	}
</style>`,
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
		prefixLen := len("/notificationsv1")
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
		req = req.WithContext(context.WithValue(req.Context(), notificationsapp.BaseURIContextKey, "/notificationsv1"))
		notificationsApp.ServeHTTP(rr, req)
		// TODO: Have notificationsApp.ServeHTTP return error, check if os.IsPermission(err) is true, etc.
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
	})}
	mux.Handle("/notificationsv1", notificationsHandler)
	mux.Handle("/notificationsv1/", notificationsHandler)

	return notificationsService
}

// dmitshurSeesExternalNotifications gives dmitshur access to notifications on GitHub and Gerrit,
// in addition to local ones.
type dmitshurSeesExternalNotifications struct {
	local                       notifications.Service
	dmitshurGitHubNotifications notifications.Service
	dmitshurGerritNotifications notifications.Service
	users                       users.Service
}

func (s dmitshurSeesExternalNotifications) List(ctx context.Context, opt notifications.ListOptions) (notifications.Notifications, error) {
	var nss notifications.Notifications
	ns, err := s.local.List(ctx, opt)
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
		if currentUser == dmitshur {
			ns, err := s.dmitshurGitHubNotifications.List(ctx, opt)
			if err != nil {
				return nss, err
			}
			nss = append(nss, ns...)
		}
	}

	if opt.Repo == nil || strings.HasPrefix(opt.Repo.URI, "go.googlesource.com/") {
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser == dmitshur {
			ns, err := s.dmitshurGerritNotifications.List(ctx, opt)
			if err != nil {
				return nss, err
			}
			nss = append(nss, ns...)
		}
	}

	return nss, nil
}

func (s dmitshurSeesExternalNotifications) Count(ctx context.Context, opt interface{}) (uint64, error) {
	var count uint64
	n, err := s.local.Count(ctx, opt)
	if err != nil {
		return count, err
	}
	count += n

	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return 0, err
	}
	if currentUser == dmitshur {
		n, err := s.dmitshurGitHubNotifications.Count(ctx, opt)
		if err != nil {
			return count, err
		}
		count += n

		n, err = s.dmitshurGerritNotifications.Count(ctx, opt)
		if err != nil {
			return count, err
		}
		count += n
	}

	return count, nil
}

func (s dmitshurSeesExternalNotifications) MarkRead(ctx context.Context, repo notifications.RepoSpec, threadType string, threadID uint64) error {
	switch {
	default:
		return s.local.MarkRead(ctx, repo, threadType, threadID)
	case strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp":
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGitHubNotifications.MarkRead(ctx, repo, threadType, threadID)
	case strings.HasPrefix(repo.URI, "go.googlesource.com/"):
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGerritNotifications.MarkRead(ctx, repo, threadType, threadID)
	}
}

func (s dmitshurSeesExternalNotifications) MarkAllRead(ctx context.Context, repo notifications.RepoSpec) error {
	switch {
	default:
		return s.local.MarkAllRead(ctx, repo)
	case strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp":
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGitHubNotifications.MarkAllRead(ctx, repo)
	case strings.HasPrefix(repo.URI, "go.googlesource.com/"):
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGerritNotifications.MarkAllRead(ctx, repo)
	}
}

func (s dmitshurSeesExternalNotifications) Subscribe(ctx context.Context, repo notifications.RepoSpec, threadType string, threadID uint64, subscribers []users.UserSpec) error {
	switch {
	default:
		return s.local.Subscribe(ctx, repo, threadType, threadID, subscribers)
	case strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp":
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGitHubNotifications.Subscribe(ctx, repo, threadType, threadID, subscribers)
	case strings.HasPrefix(repo.URI, "go.googlesource.com/"):
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGerritNotifications.Subscribe(ctx, repo, threadType, threadID, subscribers)
	}
}

func (s dmitshurSeesExternalNotifications) Notify(ctx context.Context, repo notifications.RepoSpec, threadType string, threadID uint64, nr notifications.NotificationRequest) error {
	switch {
	default:
		return s.local.Notify(ctx, repo, threadType, threadID, nr)
	case strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" && repo.URI != "github.com/shurcooL/notificationsapp":
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGitHubNotifications.Notify(ctx, repo, threadType, threadID, nr)
	case strings.HasPrefix(repo.URI, "go.googlesource.com/"):
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGerritNotifications.Notify(ctx, repo, threadType, threadID, nr)
	}
}
