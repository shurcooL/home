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
	"github.com/shurcooL/home/httputil"
	gerritactivity "github.com/shurcooL/home/internal/exp/service/activity/gerrit"
	githubactivity "github.com/shurcooL/home/internal/exp/service/activity/github"
	"github.com/shurcooL/home/internal/exp/service/notification"
	notificationfs "github.com/shurcooL/home/internal/exp/service/notification/fs"
	"github.com/shurcooL/home/internal/exp/service/notification/httphandler"
	"github.com/shurcooL/home/internal/exp/service/notification/httproute"
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

func newNotificationServiceV2(
	ctx context.Context,
	wg *sync.WaitGroup,
	fs webdav.FileSystem,
	githubActivityDir string,
	gerritActivityDir string,
	users users.Service,
	router router,
) (notification.Service, *githubactivity.Service, *gerritactivity.Service, error) {
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
	githubActivity, err := githubactivity.NewService(
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
	gerritActivity, err := gerritactivity.NewService(
		ctx, wg, fs,
		http.Dir(gerritActivityDir), newGerritActivity,
		gerritChange,
		dmitshur, users, router,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	notifService := dmitshurSeesExternalNotificationsV2{
		local:                      notificationfs.NewService(fs, users),
		dmitshurGitHubNotification: githubActivity,
		dmitshurGerritNotification: gerritActivity,
		users:                      users,
	}

	return notifService, githubActivity, gerritActivity, nil
}

func initNotificationsV2(
	mux *http.ServeMux,
	notifService notification.Service,
	notifsApp httperror.Handler,
	githubActivity interface{ Status() string },
	gerritActivity interface{ Status() string },
	users users.Service,
) {
	// Register HTTP API endpoints.
	notificationAPIHandler := httphandler.Notification{Notification: notifService}
	mux.Handle(path.Join("/api/notificationv2", httproute.ListNotifications), headerAuth{httputil.ErrorHandler(users, notificationAPIHandler.ListNotifications)})
	mux.Handle(path.Join("/api/notificationv2", httproute.StreamNotifications), headerAuth{httputil.ErrorHandler(users, notificationAPIHandler.StreamNotifications)})
	mux.Handle(path.Join("/api/notificationv2", httproute.CountNotifications), headerAuth{httputil.ErrorHandler(users, notificationAPIHandler.CountNotifications)})
	mux.Handle(path.Join("/api/notificationv2", httproute.MarkThreadRead), headerAuth{httputil.ErrorHandler(users, notificationAPIHandler.MarkThreadRead)})

	notificationsHandler := cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		// TODO: Keep simplifying this.
		prefixLen := len("/notifications")
		if prefix := req.URL.Path[:prefixLen]; req.URL.Path == prefix+"/" {
			baseURL := prefix
			if req.URL.RawQuery != "" {
				baseURL += "?" + req.URL.RawQuery
			}
			return httperror.Redirect{URL: baseURL}
		}
		returnURL := req.URL.Path
		err := notifsApp.ServeHTTP(w, req)
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
	mux.Handle("/notifications", notificationsHandler)
	mux.Handle("/notifications/", notificationsHandler)

	statusHandler := cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		if err := httputil.AllowMethods(req, http.MethodGet, http.MethodHead); err != nil {
			return err
		}
		if user, err := users.GetAuthenticated(req.Context()); err != nil {
			return err
		} else if !user.SiteAdmin {
			return os.ErrPermission
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if req.Method == http.MethodHead {
			return nil
		}
		fmt.Fprintln(w, "GitHub Activity Service:", githubActivity.Status())
		fmt.Fprintln(w, "Gerrit Activity Service:", gerritActivity.Status())
		return nil
	})}
	mux.Handle("/notifications/status", statusHandler)
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

// dmitshurSeesExternalNotificationsV2 gives dmitshur access to notifications on GitHub and Gerrit,
// in addition to local ones.
type dmitshurSeesExternalNotificationsV2 struct {
	local                      notification.Service
	dmitshurGitHubNotification notification.Service
	dmitshurGerritNotification notification.Service
	users                      users.Service
}

func (s dmitshurSeesExternalNotificationsV2) ListNotifications(ctx context.Context, opt notification.ListOptions) ([]notification.Notification, error) {
	var nss []notification.Notification
	var errors []error
	ns, err := s.local.ListNotifications(ctx, opt)
	if err != nil {
		errors = append(errors, err)
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
				errors = append(errors, fmt.Errorf("dmitshurGitHubNotification.ListNotifications: %v", err))
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
				errors = append(errors, fmt.Errorf("dmitshurGerritNotification.ListNotifications: %v", err))
			}
			nss = append(nss, ns...)
		}
	}

	sort.SliceStable(nss, func(i, j int) bool { return nss[i].Time.After(nss[j].Time) })
	if len(nss) > 100 {
		nss = nss[:100]
	}

	if len(errors) > 0 {
		return nss, fmt.Errorf("%d errors, including: %v", len(errors), errors[0])
	}
	return nss, nil
}

func (s dmitshurSeesExternalNotificationsV2) StreamNotifications(ctx context.Context, ch chan<- []notification.Notification) error {
	err := s.local.StreamNotifications(ctx, ch)
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

func (s dmitshurSeesExternalNotificationsV2) CountNotifications(ctx context.Context) (uint64, error) {
	var count uint64
	var errors []error
	n, err := s.local.CountNotifications(ctx)
	if err != nil {
		errors = append(errors, err)
	}
	count += n

	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return 0, err
	}
	if currentUser == dmitshur {
		n, err := s.dmitshurGitHubNotification.CountNotifications(ctx)
		if err != nil {
			errors = append(errors, fmt.Errorf("dmitshurGitHubNotification.CountNotifications: %v", err))
		}
		count += n

		n, err = s.dmitshurGerritNotification.CountNotifications(ctx)
		if err != nil {
			errors = append(errors, fmt.Errorf("dmitshurGerritNotification.CountNotifications: %v", err))
		}
		count += n
	}

	if len(errors) > 0 {
		return count, fmt.Errorf("%d errors, including: %v", len(errors), errors[0])
	}
	return count, nil
}

func (s dmitshurSeesExternalNotificationsV2) MarkThreadRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	service, err := s.service(ctx, namespace)
	if err != nil {
		return err
	}
	return service.MarkThreadRead(ctx, namespace, threadType, threadID)
}

func (s dmitshurSeesExternalNotificationsV2) SubscribeThread(ctx context.Context, namespace, threadType string, threadID uint64, subscribers []users.UserSpec) error {
	service, err := s.service(ctx, namespace)
	if err != nil {
		return err
	}
	return service.SubscribeThread(ctx, namespace, threadType, threadID, subscribers)
}

func (s dmitshurSeesExternalNotificationsV2) NotifyThread(ctx context.Context, namespace, threadType string, threadID uint64, nr notification.NotificationRequest) error {
	service, err := s.service(ctx, namespace)
	if err != nil {
		return err
	}
	return service.NotifyThread(ctx, namespace, threadType, threadID, nr)
}

func (s dmitshurSeesExternalNotificationsV2) service(ctx context.Context, namespace string) (notification.Service, error) {
	switch {
	default:
		return s.local, nil
	case strings.HasPrefix(namespace, "github.com/") &&
		namespace != "github.com/shurcooL/issuesapp" && namespace != "github.com/shurcooL/notificationsapp":
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser != dmitshur {
			return nil, os.ErrPermission
		}
		return s.dmitshurGitHubNotification, nil
	case strings.HasPrefix(namespace, "go.googlesource.com/"):
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser != dmitshur {
			return nil, os.ErrPermission
		}
		return s.dmitshurGerritNotification, nil
	}
}
