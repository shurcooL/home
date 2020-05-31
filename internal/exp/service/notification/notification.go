// Package notification provides a notification service definition.
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/users"
)

// TODO: this API is still a work in progress, to be evolved over time

// Service defines methods of a notification service.
type Service interface {
	// ListNotifications lists notifications for authenticated user.
	// A permission error is returned if no authenticated user.
	ListNotifications(ctx context.Context, opt ListOptions) ([]Notification, error)

	// StreamNotifications streams notifications for authenticated user,
	// sending them to ch until context is canceled or an error occurs.
	// A permission error is returned if no authenticated user.
	// TODO: Consider changing signature to single Notification rather than slice, if slice isn't helpful.
	StreamNotifications(ctx context.Context, ch chan<- []Notification) error

	// CountNotifications counts unread notifications for authenticated user.
	// A permission error is returned if no authenticated user.
	CountNotifications(ctx context.Context) (uint64, error)

	// MarkThreadRead marks the specified notification thread as read.
	// A permission error is returned if no authenticated user.
	MarkThreadRead(ctx context.Context, namespace, threadType string, threadID uint64) error

	// SubscribeThread subscribes subscribers to the specified thread.
	// If threadType and threadID are zero, subscribers are subscribed
	// to watch the entire namespace.
	// Returns a permission error if no authenticated user.
	//
	// THINK: Why is MarkRead and MarkAllRead 2 separate methods instead of 1,
	//        but this is combined into one method? Maybe there should be:
	//        SubscribeAll(ctx context.Context, namespace string, subscribers []users.UserSpec) error
	//        Or maybe MarkAllRead should be merged into MarkRead?
	SubscribeThread(ctx context.Context, namespace, threadType string, threadID uint64, subscribers []users.UserSpec) error

	// NotifyThread notifies subscribers of the specified thread of a notification.
	// The authenticated user will be the notification actor.
	// Returns a permission error if no authenticated user.
	NotifyThread(ctx context.Context, namespace, threadType string, threadID uint64, nr NotificationRequest) error
}

// ListOptions are options for ListNotifications.
type ListOptions struct {
	// Namespace is an optional filter. If not empty, only notifications
	// from the specified namespace will be listed.
	Namespace string

	// All specifies whether to include read notifications in addition to
	// unread ones.
	All bool
}

// NotificationRequest represents a request to create a notification.
type NotificationRequest struct {
	ImportPaths []string  // 1 or more.
	Time        time.Time // Canonical initial time of the notification request.

	// Payload specifies the event type. It's one of
	// Issue, Change, IssueComment, or ChangeComment.
	Payload interface{}

	// TODO: Need these?
	//Participating bool // Whether user is participating in the thread, or just watching.
	//Mentioned     bool // Whether user was specifically @mentioned in the content.
}

// Notification represents a notification.
type Notification struct {
	Namespace  string
	ThreadType string
	ThreadID   uint64

	ImportPaths []string // 1 or more.
	Time        time.Time
	Actor       users.User

	// Payload specifies the event type. It's one of
	// Issue, Change, IssueComment, or ChangeComment.
	Payload interface{}

	Unread        bool
	Participating bool // Whether user is participating in the thread, or just watching.
	Mentioned     bool // Whether user was specifically @mentioned in the content.
}

// MarshalJSON implements the json.Marshaler interface.
func (n Notification) MarshalJSON() ([]byte, error) {
	v := struct {
		Namespace  string
		ThreadType string
		ThreadID   uint64

		ImportPaths []string
		Time        time.Time
		Actor       users.User

		Type    string
		Payload interface{}

		Unread        bool
		Participating bool
		Mentioned     bool
	}{
		Namespace:     n.Namespace,
		ThreadType:    n.ThreadType,
		ThreadID:      n.ThreadID,
		ImportPaths:   n.ImportPaths,
		Time:          n.Time,
		Actor:         n.Actor,
		Payload:       n.Payload,
		Unread:        n.Unread,
		Participating: n.Participating,
		Mentioned:     n.Mentioned,
	}
	switch n.Payload.(type) {
	case Issue:
		v.Type = "Issue"
	case Change:
		v.Type = "Change"
	case IssueComment:
		v.Type = "IssueComment"
	case ChangeComment:
		v.Type = "ChangeComment"
	case nil:
		v.Type = "MarkRead" // HACK, THINK, TODO: currently (ab)using nil payload to mean "mark notification read", find a better solution
	default:
		return nil, fmt.Errorf("Notification.MarshalJSON: invalid payload type %T; Notification was %+v", n.Payload, n)
	}
	return json.Marshal(v)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (n *Notification) UnmarshalJSON(b []byte) error {
	// Ignore null, like in the main JSON package.
	if string(b) == "null" {
		return nil
	}
	var v struct {
		Namespace  string
		ThreadType string
		ThreadID   uint64

		ImportPaths []string
		Time        time.Time
		Actor       users.User

		Type    string
		Payload json.RawMessage

		Unread        bool
		Participating bool
		Mentioned     bool
	}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	*n = Notification{
		Namespace:     v.Namespace,
		ThreadType:    v.ThreadType,
		ThreadID:      v.ThreadID,
		ImportPaths:   v.ImportPaths,
		Time:          v.Time,
		Actor:         v.Actor,
		Unread:        v.Unread,
		Participating: v.Participating,
		Mentioned:     v.Mentioned,
	}
	switch v.Type {
	case "Issue":
		var p Issue
		err := json.Unmarshal(v.Payload, &p)
		if err != nil {
			return err
		}
		n.Payload = p
	case "Change":
		var p Change
		err := json.Unmarshal(v.Payload, &p)
		if err != nil {
			return err
		}
		n.Payload = p
	case "IssueComment":
		var p IssueComment
		err := json.Unmarshal(v.Payload, &p)
		if err != nil {
			return err
		}
		n.Payload = p
	case "ChangeComment":
		var p ChangeComment
		err := json.Unmarshal(v.Payload, &p)
		if err != nil {
			return err
		}
		n.Payload = p
	case "MarkRead":
		n.Payload = nil // HACK, THINK, TODO: currently (ab)using nil payload to mean "mark notification read", find a better solution
	default:
		return fmt.Errorf("Notification.UnmarshalJSON: invalid payload type %q", v.Type)
	}
	return nil
}

// Issue is an issue event.
type Issue struct {
	Action       string // "opened", "closed", "reopened".
	IssueTitle   string
	IssueBody    string // Only set when action is "opened".
	IssueHTMLURL string
}

// Change is a change event.
type Change struct {
	Action        string // "opened", "closed", "merged", "reopened".
	ChangeTitle   string
	ChangeBody    string // Only set when action is "opened".
	ChangeHTMLURL string
}

// IssueComment is an issue comment event.
type IssueComment struct {
	IssueTitle     string
	IssueState     state.Issue
	CommentBody    string
	CommentHTMLURL string
}

// ChangeComment is a change comment event.
// A change comment is a review iff CommentReview is non-zero.
type ChangeComment struct {
	ChangeTitle    string
	ChangeState    state.Change
	CommentBody    string
	CommentReview  state.Review
	CommentHTMLURL string
}
