// Package v2tov1 provides a notifv1.Service wrapper
// on top of a notifv2.Service implementation.
package v2tov1

import (
	"context"
	"fmt"
	"log"
	"strings"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/events/event"
	notifv2 "github.com/shurcooL/home/internal/exp/service/notification"
	notifv1 "github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

// Service implements notifv1.Service using V2.
type Service struct {
	V2 notifv2.Service

	// NotifyPayloadSource, if non-nil, enables Notify to work.
	NotifyPayloadSource *eventsService
}

// List notifications for authenticated user.
// Returns a permission error if no authenticated user.
func (s Service) List(ctx context.Context, opt notifv1.ListOptions) (notifv1.Notifications, error) {
	optV2 := notifv2.ListOptions{
		All: opt.All,
	}
	if opt.Repo != nil {
		optV2.Namespace = opt.Repo.URI
	}
	notifs, err := s.V2.ListNotifications(ctx, optV2)
	if err != nil {
		return nil, err
	}
	type Thread struct {
		Namespace string
		Type      string
		ID        uint64
	}
	var threads = make(map[Thread]notifv1.Notification)
	for _, n := range notifs {
		var (
			title   string
			icon    notifv1.OcticonID
			color   notifv1.RGB
			htmlURL string
		)
		switch p := n.Payload.(type) {
		case notifv2.Issue:
			title = p.IssueTitle
			icon, color = issueActionIconColor(p.Action)
			htmlURL = p.IssueHTMLURL
		case notifv2.Change:
			title = p.ChangeTitle
			icon, color = changeActionIconColor(p.Action)
			htmlURL = p.ChangeHTMLURL
		case notifv2.IssueComment:
			title = p.IssueTitle
			icon, color = issueStateIconColor(p.IssueState)
			htmlURL = p.CommentHTMLURL
		case notifv2.ChangeComment:
			title = p.ChangeTitle
			icon, color = changeStateIconColor(p.ChangeState)
			htmlURL = p.CommentHTMLURL
		}
		t := Thread{
			Namespace: n.Namespace,
			Type:      n.ThreadType,
			ID:        n.ThreadID,
		}
		if !n.Time.After(threads[t].UpdatedAt) {
			continue
		}
		threads[t] = notifv1.Notification{
			RepoSpec:      notifv1.RepoSpec{URI: n.Namespace},
			ThreadType:    n.ThreadType,
			ThreadID:      n.ThreadID,
			Title:         importPathsToFullPrefix(n.ImportPaths) + title,
			Icon:          icon,
			Color:         color,
			Actor:         n.Actor,
			UpdatedAt:     n.Time,
			Read:          !n.Unread,
			HTMLURL:       htmlURL,
			Participating: n.Participating,
			Mentioned:     n.Mentioned,
		}
	}
	var ns notifv1.Notifications
	for _, n := range threads {
		ns = append(ns, n)
	}
	return ns, nil
}

func importPathsToFullPrefix(paths []string) string {
	var b strings.Builder
	for i, p := range paths {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(p)
	}
	b.WriteString(": ")
	return b.String()
}

func issueActionIconColor(action string) (notifv1.OcticonID, notifv1.RGB) {
	switch action {
	case "opened", "reopened":
		return notifv1.OcticonID("issue-" + action), notifv1.RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
	case "closed":
		return "issue-closed", notifv1.RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
	default:
		panic("unreachable")
	}
}

func issueStateIconColor(s state.Issue) (notifv1.OcticonID, notifv1.RGB) {
	switch s {
	case state.IssueOpen:
		return "issue-opened", notifv1.RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
	case state.IssueClosed:
		return "issue-closed", notifv1.RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
	default:
		panic("unreachable")
	}
}

func changeActionIconColor(action string) (notifv1.OcticonID, notifv1.RGB) {
	switch action {
	case "opened", "reopened":
		return "git-pull-request", notifv1.RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
	case "closed":
		return "git-pull-request", notifv1.RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
	case "merged":
		return "git-pull-request", notifv1.RGB{R: 0x6e, G: 0x54, B: 0x94} // Purple.
	default:
		panic("unreachable")
	}
}

func changeStateIconColor(s state.Change) (notifv1.OcticonID, notifv1.RGB) {
	switch s {
	case state.ChangeOpen:
		return "git-pull-request", notifv1.RGB{R: 0x6c, G: 0xc6, B: 0x44} // Green.
	case state.ChangeClosed:
		return "git-pull-request", notifv1.RGB{R: 0xbd, G: 0x2c, B: 0x00} // Red.
	case state.ChangeMerged:
		return "git-pull-request", notifv1.RGB{R: 0x6e, G: 0x54, B: 0x94} // Purple.
	default:
		panic("unreachable")
	}
}

// Count notifications for authenticated user.
// Returns a permission error if no authenticated user.
func (s Service) Count(ctx context.Context, opt interface{}) (uint64, error) {
	return s.V2.CountNotifications(ctx)
}

// MarkRead marks the specified thread as read.
// Returns a permission error if no authenticated user.
func (s Service) MarkRead(ctx context.Context, repo notifv1.RepoSpec, threadType string, threadID uint64) error {
	return s.V2.MarkThreadRead(ctx, repo.URI, threadType, threadID)
}

// MarkAllRead marks all notifications in the specified repository as read.
// Returns a permission error if no authenticated user.
func (s Service) MarkAllRead(ctx context.Context, repo notifv1.RepoSpec) error {
	return fmt.Errorf("MarkAllRead: not implemented")
}

// Subscribe subscribes subscribers to the specified thread.
// If threadType and threadID are zero, subscribers are subscribed
// to watch the entire repo.
// Returns a permission error if no authenticated user.
func (s Service) Subscribe(ctx context.Context, repo notifv1.RepoSpec, threadType string, threadID uint64, subscribers []users.UserSpec) error {
	return s.V2.SubscribeThread(ctx, repo.URI, threadType, threadID, subscribers)
}

// Notify notifies subscribers of the specified thread of a notification.
// Returns a permission error if no authenticated user.
func (s Service) Notify(ctx context.Context, repo notifv1.RepoSpec, threadType string, threadID uint64, nr notifv1.NotificationRequest) error {
	if s.NotifyPayloadSource == nil {
		return fmt.Errorf("Notify: support for s.NotifyPayloadSource == nil is not implemented")
	}
	// TODO: Add user service and use it to check if current user doesn't match nr.Actor, then return error.
	go func() {
		err := s.V2.NotifyThread(ctx, repo.URI, threadType, threadID, notifv2.NotificationRequest{
			ImportPaths: []string{repo.URI},
			Time:        nr.UpdatedAt,
			Payload:     <-s.NotifyPayloadSource.payload,
		})
		if err != nil {
			log.Println("v2tov1.Service.Notify: V2.NotifyThread:", err)
		}
	}()
	return nil
}

// NewNotifyPayloadSource creates a NotifyPayloadSource for use in Service.
// Its return value implements events.Service and it should be passed to
// issue tracker v1 alongside with the Service. It's used to capture the
// notification payload to implement Service.Notify without losing information.
func NewNotifyPayloadSource() *eventsService {
	return &eventsService{
		payload: make(chan interface{}, 1),
	}
}

type eventsService struct {
	payload chan interface{} // Value is notifv2 payload.
}

// List implements events.Service.
func (eventsService) List(context.Context) ([]event.Event, error) { return nil, nil }

// List implements events.Service.
func (s eventsService) Log(ctx context.Context, e event.Event) error {
	switch p := e.Payload.(type) {
	case event.Issue:
		s.payload <- notifv2.Issue(p)
	case event.Change:
		s.payload <- notifv2.Change(p)
	case event.IssueComment:
		s.payload <- notifv2.IssueComment(p)
	case event.ChangeComment:
		s.payload <- notifv2.ChangeComment(p)
	}
	return nil
}
