// Package github implements activity.Service for GitHub.
package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"dmitri.shuralyov.com/route/github"
	githubv3 "github.com/google/go-github/github"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

// NewService creates a GitHub-backed activity.Service using the
// given GitHub activity mail filesystem and GitHub API clients.
// It serves the specified user only,
// whose activity mail and authenticated GitHub API clients must be provided,
// and cannot be used to serve multiple users.
//
// This service uses Cache-Control: no-cache request header to disable caching.
//
// newActivityMail delivers a value when there is new mail,
// and must not be closed.
//
// If router is nil, github.DotCom router is used,
// which links to subjects on github.com.
func NewService(
	fs webdav.FileSystem,
	activityMail http.FileSystem, newActivityMail <-chan struct{},
	clientV3 *githubv3.Client, clientV4 *githubv4.Client,
	user users.User, users users.Service,
	router github.Router,
) (*Service, error) {
	if user.Domain != "github.com" {
		return nil, fmt.Errorf(`user.Domain is %q, it must be "github.com"`, user.Domain)
	}
	if router == nil {
		router = github.DotCom{}
	}
	s := &Service{
		fs:          fs,
		notifMail:   activityMail,
		notifEvents: newActivityMail,
		clV3:        clientV3,
		clV4:        clientV4,
		user:        user,
		users:       users,
		rtr:         router,
	}
	s.mail.chs = make(map[context.Context]chan<- []notification.Notification)
	go func() {
		err := s.loadAndPoll()
		if err != nil {
			log.Println("service/activity/github: loadAndPoll:", err)
			s.errorMu.Lock()
			s.error = err
			s.errorMu.Unlock()
		}
	}()
	go func() {
		err := s.pollList()
		if err != nil {
			log.Println("service/activity/github: pollList:", err)
			s.errorMu.Lock()
			s.error = err
			s.errorMu.Unlock()
		}
	}()
	go func() {
		err := s.pollNotifications()
		if err != nil {
			log.Println("service/activity/github: pollNotifications:", err)
			s.errorMu.Lock()
			s.error = err
			s.errorMu.Unlock()
		}
	}()
	return s, nil
}

type Service struct {
	fs          webdav.FileSystem // Persistent storage.
	notifMail   http.FileSystem   // GitHub notification mail messages, in monthly reclog-formatted files.
	notifEvents <-chan struct{}   // Never closed.
	clV3        *githubv3.Client  // GitHub REST API v3 client.
	clV4        *githubv4.Client  // GitHub GraphQL API v4 client.
	user        users.User
	users       users.Service
	rtr         github.Router

	// Entries derived from received notification mail messages
	// (with GitHub API calls to fetch details).
	mail struct {
		mu     sync.Mutex
		notifs []notifAndURL // Most recent notifications are at the front.
		events []eventAndURL // Most recent events are at the front.

		chsMu sync.Mutex
		chs   map[context.Context]chan<- []notification.Notification
	}

	// Entries derived from received GitHub Activity API list endpoint
	// (with GitHub API calls to fill in additional details).
	list struct {
		mu         sync.Mutex
		events     []*githubv3.Event
		repos      map[int64]repository       // Repo ID -> Module Path.
		commits    map[string]event.Commit    // SHA -> Commit.
		prs        map[string]bool            // PR API URL -> Pull Request merged.
		eventIDs   map[*githubv3.Event]uint64 // Event -> event ID.
		fetchError error
	}

	notifs struct {
		mu         sync.Mutex
		lastReadAt map[thread]time.Time
	}

	errorMu sync.Mutex
	error   error // Fatal service error, if any.
}

// ListNotifications implements notification.Service.
func (s *Service) ListNotifications(ctx context.Context, opt notification.ListOptions) ([]notification.Notification, error) {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return nil, err
	} else if u != s.user.UserSpec {
		return nil, os.ErrPermission
	}

	// Fetch all unread notifications now, in order to know
	// which notifications have been read or unread.
	// TODO, THINK: How to make this compatible with streaming notifications? Forced to poll, or is there a better way?
	// TODO: factor out into a 1-minute poll, etc.
	/*var lastReadAt = make(map[Thread]time.Time)
	ghOpt := &githubv3.NotificationListOptions{ListOptions: githubv3.ListOptions{PerPage: 100}}
	for {
		var ns []*githubv3.Notification
		var resp *githubv3.Response
		switch opt.Namespace {
		case "":
			var err error
			ns, resp, err = ghListNotifications(ctx, s.clV3, ghOpt, false)
			if err != nil {
				return nil, err
			}
		default:
			repo, err := ghRepoSpec(opt.Namespace)
			if err != nil {
				return nil, err
			}
			ns, resp, err = ghListRepositoryNotifications(ctx, s.clV3, repo.Owner, repo.Repo, ghOpt, false)
			if err != nil {
				return nil, err
			}
		}
		for _, n := range ns {
			var id uint64
			switch *n.Subject.Type {
			case "Issue":
				var err error
				_, id, err = parseIssueSpec(*n.Subject.URL)
				if err != nil {
					return nil, fmt.Errorf("failed to parseIssueSpec: %v", err)
				}
			case "PullRequest":
				var err error
				_, id, err = parsePullRequestSpec(*n.Subject.URL)
				if err != nil {
					return nil, fmt.Errorf("failed to parsePullRequestSpec: %v", err)
				}
			default:
				continue
			}
			th := Thread{
				Namespace: "github.com/" + *n.Repository.FullName,
				Type:      *n.Subject.Type,
				ID:        id,
			}
			if t, ok := lastReadAt[th]; !ok || n.GetLastReadAt().After(t) {
				lastReadAt[th] = n.GetLastReadAt()
			}
		}
		if resp.NextPage == 0 {
			break
		}
		ghOpt.Page = resp.NextPage
	}*/
	s.notifs.mu.Lock()
	lastReadAt := s.notifs.lastReadAt
	s.notifs.mu.Unlock()

	// TODO: Filter out notifs from other repos when
	//       opt.All == true and opt.Namespace != "".
	var notifs []notification.Notification
	switch opt.All {
	case true:
		s.mail.mu.Lock()
		notifs = make([]notification.Notification, len(s.mail.notifs))
		for i, notif := range s.mail.notifs {
			notifs[i] = notif.WithURL(ctx)
		}
		s.mail.mu.Unlock()

		for i, n := range notifs {
			lastReadAt, ok := lastReadAt[thread{n.Namespace, n.ThreadType, n.ThreadID}]
			notifs[i].Unread = ok && n.Time.After(lastReadAt)
		}
	case false:
		s.mail.mu.Lock()
		for _, n := range s.mail.notifs {
			lastReadAt, ok := lastReadAt[thread{n.Namespace, n.ThreadType, n.ThreadID}]
			unread := ok && n.Time.After(lastReadAt)
			if !unread {
				continue
			}
			n := n.WithURL(ctx)
			n.Unread = true
			notifs = append(notifs, n)
		}
		s.mail.mu.Unlock()
	}
	return notifs, nil
}

// StreamNotifications implements notification.Service.
func (s *Service) StreamNotifications(ctx context.Context, ch chan<- []notification.Notification) error {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return err
	} else if u != s.user.UserSpec {
		return os.ErrPermission
	}

	s.mail.chsMu.Lock()
	s.mail.chs[ctx] = ch
	s.mail.chsMu.Unlock()

	return nil
}

// CountNotifications implements notification.Service.
func (s *Service) CountNotifications(ctx context.Context) (uint64, error) {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return 0, err
	} else if u != s.user.UserSpec {
		return 0, os.ErrPermission
	}

	ghOpt := &githubv3.NotificationListOptions{ListOptions: githubv3.ListOptions{PerPage: 1}}
	ghNotifications, resp, err := ghListNotifications(ctx, s.clV3, ghOpt, false)
	if err != nil {
		return 0, err
	}
	if resp.LastPage != 0 {
		return uint64(resp.LastPage), nil
	} else {
		return uint64(len(ghNotifications)), nil
	}
}

// MarkNotificationRead implements notification.Service.
//
// Namespace must be of the form "github.com/{owner}/{repo}".
// E.g., "github.com/google/go-cmp".
func (s *Service) MarkNotificationRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return err
	} else if u != s.user.UserSpec {
		return os.ErrPermission
	}

	repo, err := ghRepoSpec(namespace)
	if err != nil {
		return err
	}
	if threadType != "Issue" && threadType != "PullRequest" {
		return fmt.Errorf("unrecognized threadType=%q", threadType)
	}

	var githubThreadID string

	s.mail.mu.Lock()
	for _, n := range s.mail.notifs {
		if n.Namespace != namespace || n.ThreadType != threadType || n.ThreadID != threadID {
			// Not a match.
			continue
		}
		githubThreadID = n.GitHubThreadID
		break
	}
	s.mail.mu.Unlock()

	// Figure out if the notification thread became read.
	// TODO: Use GitHub as source of truth to avoid local prediction false positives.
	var becameRead bool
	s.notifs.mu.Lock()
	if _, ok := s.notifs.lastReadAt[thread{namespace, threadType, threadID}]; ok {
		// Currently, lastReadAt map tracks only unread notifications.
		// So if this thread was there, it must've been unread.
		// It will get cleared by next iteration of pollNotifications loop.
		// TODO: Use GetThread before MarkThreadRead maybe?
		becameRead = true
	}
	s.notifs.mu.Unlock()

	if becameRead {
		// Notify streaming observers.
		s.mail.chsMu.Lock()
		for ctx, ch := range s.mail.chs {
			if ctx.Err() != nil {
				delete(s.mail.chs, ctx)
				continue
			}
			select {
			case ch <- []notification.Notification{{
				Namespace:  namespace,
				ThreadType: threadType,
				ThreadID:   threadID,
				Unread:     false,
			}}:
			default:
			}
		}
		s.mail.chsMu.Unlock()
	}

	if githubThreadID == "" {
		// Didn't find any matching notification to mark read.
		// Nothing to do.
		// TODO, HACK: Actually, need to do same thing as in v1 for now,
		//             because we're not yet tracking all notifications...
		//             Fix that and drop this temporary fallback.
		return s.markReadFromV1(ctx, repo, threadType, threadID)
	}

	// Found a matching notification, mark it read.
	_, err = s.clV3.Activity.MarkThreadRead(ctx, githubThreadID)
	if err != nil {
		return fmt.Errorf("failed to MarkThreadRead: %v", err)
	}
	return nil
}

// threadType must be "Issue" or "PullRequest".
func (s *Service) markReadFromV1(ctx context.Context, repo repoSpec, threadType string, threadID uint64) error {
	ghOpt := &githubv3.NotificationListOptions{ListOptions: githubv3.ListOptions{PerPage: 100}}
	for {
		uncached, resp, err := ghListRepositoryNotifications(ctx, s.clV3, repo.Owner, repo.Repo, ghOpt, false)
		if err != nil {
			return fmt.Errorf("failed to ListRepositoryNotifications: %v", err)
		}
		if notif, err := findNotification(uncached, threadType, threadID); err != nil {
			return err
		} else if notif != nil {
			// Found a matching notification, mark it read.
			log.Printf(`MarkRead: did not find notification %s/%s %s %d within v2 notifications, but did find within v1 ones`, repo.Owner, repo.Repo, threadType, threadID)
			_, err := s.clV3.Activity.MarkThreadRead(ctx, *notif.ID)
			if err != nil {
				return fmt.Errorf("failed to MarkThreadRead: %v", err)
			}
			return nil
		}
		if resp.NextPage == 0 {
			break
		}
		ghOpt.Page = resp.NextPage
	}

	// Didn't find any matching notification to mark read.
	// Nothing to do.
	return nil
}

// findNotification tries to find a notification that matches
// the provided threadType and threadID.
// threadType must be one of "Issue" or "PullRequest".
// It returns nil if no matching notification is found, and
// any error encountered.
func findNotification(ns []*githubv3.Notification, threadType string, threadID uint64) (*githubv3.Notification, error) {
	for _, n := range ns {
		if *n.Subject.Type != threadType {
			// Mismatched thread type.
			continue
		}

		var id uint64
		switch *n.Subject.Type {
		case "Issue":
			var err error
			_, id, err = parseIssueSpec(*n.Subject.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to parseIssueSpec: %v", err)
			}
		case "PullRequest":
			var err error
			_, id, err = parsePullRequestSpec(*n.Subject.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to parsePullRequestSpec: %v", err)
			}
		}
		if id != threadID {
			// Mismatched thread ID.
			continue
		}

		// Found a matching notification.
		return n, nil
	}
	return nil, nil
}

// List lists events.
func (s *Service) List(ctx context.Context) ([]event.Event, error) {
	var seen = make(map[githubEventID]struct{}) // Used to deduplicate same event from multiple sources.

	// Get events from mail.
	s.mail.mu.Lock()
	mailEvents := make([]event.Event, len(s.mail.events))
	for i, event := range s.mail.events {
		mailEvents[i] = event.WithURL(ctx)
		seen[event.ID] = struct{}{}
	}
	s.mail.mu.Unlock()

	// Get events from list.
	s.list.mu.Lock()
	events, repos, commits, prs, eventIDs, fetchError := s.list.events, s.list.repos, s.list.commits, s.list.prs, s.list.eventIDs, s.list.fetchError
	s.list.mu.Unlock()
	listEvents, listEventIDs := convert(ctx, events, repos, commits, prs, eventIDs, s.rtr)

	// Join both sources and sort.
	all := mailEvents
	for i, e := range listEvents {
		if _, ok := seen[listEventIDs[i]]; ok {
			// Already got the same event from another source, skip this duplicate.
			continue
		}
		all = append(all, e)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Time.After(all[j].Time) })
	if len(all) > 100 {
		all = all[:100]
	}

	return all, fetchError
}

// Log logs the event.
// event.Time time zone must be UTC.
func (*Service) Log(_ context.Context, event event.Event) error {
	if event.Time.Location() != time.UTC {
		return errors.New("event.Time time zone must be UTC")
	}
	// TODO, THINK: Where should a Log("dmitri.shuralyov.com/foo/bar") event get non-errored? Here, or in home.multiEvents?
	// Nothing to do. GitHub takes care of this on their end, even when performing actions via API.
	return nil
}

// Status reports the status of the service.
// The status is "ok" if everything is okay,
// or an error description otherwise.
func (s *Service) Status() string {
	s.errorMu.Lock()
	err := s.error
	s.errorMu.Unlock()
	if err != nil {
		return err.Error()
	}
	return "ok"
}
