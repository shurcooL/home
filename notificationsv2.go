package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"dmitri.shuralyov.com/route/gerrit"
	"dmitri.shuralyov.com/route/github"
	gerritapichange "dmitri.shuralyov.com/service/change/gerritapi"
	gerritapi "github.com/andygrunwald/go-gerrit"
	"github.com/fsnotify/fsnotify"
	githubv3 "github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	notificationsv2 "github.com/shurcooL/home/internal/exp/app/notifications"
	gerritactivity "github.com/shurcooL/home/internal/exp/service/activity/gerrit"
	githubactivity "github.com/shurcooL/home/internal/exp/service/activity/github"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/exp/service/notification/httphandler"
	"github.com/shurcooL/home/internal/exp/service/notification/httproute"
	notificationmem "github.com/shurcooL/home/internal/exp/service/notification/mem"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
	"golang.org/x/oauth2"
)

// For now, the notification v2 service and app are additive.
// They co-exist with the v1 service and app side-by-side.
// Need to do more work to make it feature complete and stable,
// migrate other apps to be able to use notification service v2,
// and then remove the v1 service and app.

type router interface {
	github.Router
	gerrit.Router
}

func initNotificationsV2Disabled(
	ctx context.Context,
	wg *sync.WaitGroup,
	mux *http.ServeMux,
	fs webdav.FileSystem,
	githubActivityDir string,
	gerritActivityDir string,
	users users.Service,
	router router,
) (
	githubActivity *githubactivity.Service,
	gerritActivity *gerritactivity.Service,
	fullNotif notification.FullService,
	_ error,
) {
	dmitshur, err := users.Get(context.Background(), dmitshur)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("users.Get(dmitshur): %v", err)
	}
	return nil, nil, notificationmem.NewService(dmitshur, users), nil
}

func initNotificationsV2(
	ctx context.Context,
	wg *sync.WaitGroup,
	mux *http.ServeMux,
	fs webdav.FileSystem,
	githubActivityDir string,
	gerritActivityDir string,
	users users.Service,
	router router,
) (
	githubActivity *githubactivity.Service,
	gerritActivity *gerritactivity.Service,
	fullNotif notification.FullService,
	_ error,
) {
	dmitshur, err := users.Get(context.Background(), dmitshur)
	if err != nil {
		return nil, nil, nil, err
	}

	newGitHubActivity, err := newDirWatcher(ctx, githubActivityDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("newDirWatcher: %v", err)
	}
	authTransport := &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("HOME_GH_DMITSHUR_NOTIFICATIONS")}),
	}
	cacheTransport := &httpcache.Transport{
		Transport:           authTransport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}
	githubActivity, err = githubactivity.NewService(
		fs,
		http.Dir(githubActivityDir), newGitHubActivity,
		githubv3.NewClient(&http.Client{Transport: cacheTransport, Timeout: 10 * time.Second}),
		githubv4.NewClient(&http.Client{Transport: authTransport, Timeout: 10 * time.Second}),
		dmitshur, users, router,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	newGerritActivity, err := newDirWatcher(ctx, gerritActivityDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("newDirWatcher: %v", err)
	}
	// TODO, THINK: reuse client from newChangeService?
	gerritClient, err := gerritapi.NewClient( // TODO: Auth.
		"https://go-review.googlesource.com/",
		&http.Client{Transport: httpcache.NewMemoryCacheTransport()},
	)
	if err != nil {
		panic(fmt.Errorf("internal error: gerrit.NewClient returned non-nil error: %v", err))
	}
	gerritChange := gerritapichange.NewService(gerritClient)
	gerritActivity, err = gerritactivity.NewService(
		ctx, wg, fs,
		http.Dir(gerritActivityDir), newGerritActivity,
		gerritChange,
		dmitshur, users, router,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	fullNotif = notificationmem.NewService(dmitshur, users)

	notificationService := dmitshurSeesOwnNotificationsV2{
		service:                    fullNotif,
		dmitshurGitHubNotification: githubActivity,
		dmitshurGerritNotification: gerritActivity,
		users:                      users,
	}

	// Register HTTP API endpoints.
	notificationAPIHandler := httphandler.Notification{Notification: notificationService}
	mux.Handle(path.Join("/api/notificationv2", httproute.ListNotifications), headerAuth{httputil.ErrorHandler(users, notificationAPIHandler.ListNotifications)})
	mux.Handle(path.Join("/api/notificationv2", httproute.StreamNotifications), headerAuth{httputil.ErrorHandler(users, notificationAPIHandler.StreamNotifications)})
	mux.Handle(path.Join("/api/notificationv2", httproute.CountNotifications), headerAuth{httputil.ErrorHandler(users, notificationAPIHandler.CountNotifications)})
	mux.Handle(path.Join("/api/notificationv2", httproute.MarkNotificationRead), headerAuth{httputil.ErrorHandler(users, notificationAPIHandler.MarkNotificationRead)})

	// Register notifications app endpoints.
	opt := notificationsv2.Options{
		BaseURL: "/notificationsv2",
		RedLogo: component.RedLogo,
		HeadPre: analyticsHTML + `<title>Notifications v2</title>
<link href="/icon.png" rel="icon" type="image/png">
<meta name="viewport" content="width=device-width">
<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
<style type="text/css">
	body {
		margin: 20px;
		font-family: Go;
		font-size: 87.5%;
		line-height: initial;
		color: rgb(35, 35, 35);
	}
</style>`,
	}
	notificationsApp := notificationsv2.New(
		notificationService,
		githubActivity, gerritActivity,
		users,
		opt,
	)

	notificationsHandler := cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		// TODO: Keep simplifying this.
		prefixLen := len("/notificationsv2")
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
		err := notificationsApp.ServeHTTP(w, req)
		// TODO: Factor out this os.IsPermission(err) && u == nil check somewhere, if possible. (But this shouldn't apply for APIs.)
		if s := req.Context().Value(sessionContextKey).(*session); os.IsPermission(err) && s == nil {
			loginURL := (&url.URL{
				Path:     "/login",
				RawQuery: url.Values{returnParameterName: {returnURL}}.Encode(),
			}).String()
			return httperror.Redirect{URL: loginURL}
		}
		return err
	})}
	mux.Handle("/notificationsv2", notificationsHandler)
	mux.Handle("/notificationsv2/", notificationsHandler)

	return githubActivity, gerritActivity, fullNotif, nil
}

func newDirWatcher(ctx context.Context, dir string) (<-chan struct{}, error) {
	var ch = make(chan struct{}, 1)
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify.NewWatcher: %v", err)
	}
	go func() {
		defer w.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-w.Events:
				if !ok {
					return
				}
				select {
				case ch <- struct{}{}:
				default:
				}
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	err = w.Add(dir)
	if err != nil {
		return nil, fmt.Errorf("watcher.Add(%q): %v", dir, err)
	}
	return ch, nil
}

// dmitshurSeesOwnNotificationsV2 lets dmitshur see own notifications on GitHub and Gerrit,
// in addition to local ones.
type dmitshurSeesOwnNotificationsV2 struct {
	service                    notification.Service
	dmitshurGitHubNotification notification.Service
	dmitshurGerritNotification notification.Service
	users                      users.Service
}

func (s dmitshurSeesOwnNotificationsV2) ListNotifications(ctx context.Context, opt notification.ListOptions) ([]notification.Notification, error) {
	var nss []notification.Notification
	ns, err := s.service.ListNotifications(ctx, opt)
	if err != nil {
		return nss, err
	}
	nss = append(nss, ns...)

	if opt.Namespace == "" || strings.HasPrefix(opt.Namespace, "github.com/") &&
		opt.Namespace != "github.com/shurcooL/issuesapp" && opt.Namespace != "github.com/shurcooL/notificationsapp" {

		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser == dmitshur {
			ns, err := s.dmitshurGitHubNotification.ListNotifications(ctx, opt)
			if err != nil {
				return nss, fmt.Errorf("dmitshurGitHubNotification.ListNotifications: %v", err)
			}
			nss = append(nss, ns...)
		}
	}

	if opt.Namespace == "" || strings.HasPrefix(opt.Namespace, "go.googlesource.com/") {
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser == dmitshur {
			ns, err := s.dmitshurGerritNotification.ListNotifications(ctx, opt)
			if err != nil {
				return nss, fmt.Errorf("dmitshurGerritNotification.ListNotifications: %v", err)
			}
			nss = append(nss, ns...)
		}
	}

	sort.SliceStable(nss, func(i, j int) bool { return nss[i].Time.After(nss[j].Time) })
	if len(nss) > 100 {
		nss = nss[:100]
	}

	return nss, nil
}

func (s dmitshurSeesOwnNotificationsV2) StreamNotifications(ctx context.Context, ch chan<- []notification.Notification) error {
	err := s.service.StreamNotifications(ctx, ch)
	if err != nil {
		return err
	}

	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return err
	}
	if currentUser == dmitshur {
		err := s.dmitshurGitHubNotification.StreamNotifications(ctx, ch)
		if err != nil {
			return err
		}

		err = s.dmitshurGerritNotification.StreamNotifications(ctx, ch)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s dmitshurSeesOwnNotificationsV2) CountNotifications(ctx context.Context) (uint64, error) {
	var count uint64
	n, err := s.service.CountNotifications(ctx)
	if err != nil {
		return count, err
	}
	count += n

	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return 0, err
	}
	if currentUser == dmitshur {
		n, err := s.dmitshurGitHubNotification.CountNotifications(ctx)
		if err != nil {
			return count, err
		}
		count += n

		n, err = s.dmitshurGerritNotification.CountNotifications(ctx)
		if err != nil {
			return count, err
		}
		count += n
	}

	return count, nil
}

func (s dmitshurSeesOwnNotificationsV2) MarkNotificationRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	switch {
	case strings.HasPrefix(namespace, "github.com/") &&
		namespace != "github.com/shurcooL/issuesapp" && namespace != "github.com/shurcooL/notificationsapp":
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGitHubNotification.MarkNotificationRead(ctx, namespace, threadType, threadID)
	case strings.HasPrefix(namespace, "go.googlesource.com/"):
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return err
		}
		if currentUser != dmitshur {
			return os.ErrPermission
		}
		return s.dmitshurGerritNotification.MarkNotificationRead(ctx, namespace, threadType, threadID)
	}

	return s.service.MarkNotificationRead(ctx, namespace, threadType, threadID)
}
