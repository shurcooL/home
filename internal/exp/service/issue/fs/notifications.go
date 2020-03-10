package fs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dmitri.shuralyov.com/state"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

// threadType is the notifications thread type for this service.
const threadType = "issues"

// ThreadType returns the notifications thread type for this service.
func (*service) ThreadType(issues.RepoSpec) string { return threadType }

// subscribe subscribes user and anyone mentioned in body to the issue.
func (s *service) subscribe(ctx context.Context, repo issues.RepoSpec, issueID uint64, user users.UserSpec, body string) error {
	if s.notifications == nil {
		return nil
	}

	subscribers := []users.UserSpec{user}

	// TODO: Find mentioned users in body.
	/*mentions, err := mentions(ctx, body)
	if err != nil {
		return err
	}
	subscribers = append(subscribers, mentions...)*/

	return s.notifications.Subscribe(ctx, notifications.RepoSpec(repo), threadType, issueID, subscribers)
}

// markRead marks the specified issue as read for current user.
func (s *service) markRead(ctx context.Context, repo issues.RepoSpec, issueID uint64) error {
	if s.notifications == nil {
		return nil
	}

	return s.notifications.MarkRead(ctx, notifications.RepoSpec(repo), threadType, issueID)
}

// notify notifies all subscribed users of an update that shows up in their Notification Center.
func (s *service) notify(ctx context.Context, repo issues.RepoSpec, issueID uint64, fragment string, actor users.UserSpec, time time.Time) error {
	if s.notifications == nil {
		return nil
	}

	// TODO, THINK: Is this the best place/time?
	// Get issue from storage for to populate notification fields.
	var issue issue
	err := jsonDecodeFile(ctx, s.fs, issueCommentPath(repo, issueID, 0), &issue)
	if err != nil {
		return err
	}

	nr := notifications.NotificationRequest{
		Title:     issue.Title,
		Icon:      notificationIcon(issue.State),
		Color:     notificationColor(issue.State),
		Actor:     actor,
		UpdatedAt: time,
		HTMLURL:   htmlURL(repo.URI, issueID, fragment),
	}

	return s.notifications.Notify(ctx, notifications.RepoSpec(repo), threadType, issueID, nr)
}

// TODO, THINK: Where should the logic to come up with the URL live?
//              It's kinda related to the router/URL scheme of issuesapp...
func htmlURL(repoURI string, issueID uint64, fragment string) string {
	var htmlURL string
	// TODO: Find a good way to factor out this logic and provide it to issues/fs in a reasonable way.
	switch {
	default:
		htmlURL = fmt.Sprintf("https://%s/...$issues/%v", repoURI, issueID)
	case repoURI == "dmitri.shuralyov.com/blog":
		htmlURL = fmt.Sprintf("https://dmitri.shuralyov.com/blog/%v", issueID)
	case repoURI == "dmitri.shuralyov.com/idiomatic-go":
		htmlURL = fmt.Sprintf("https://dmitri.shuralyov.com/idiomatic-go/entries/%v", issueID)
	case strings.HasPrefix(repoURI, "github.com/shurcooL/"):
		htmlURL = fmt.Sprintf("https://dmitri.shuralyov.com/issues/%s/%v", repoURI, issueID)
	}
	if fragment != "" {
		htmlURL += "#" + fragment
	}
	return htmlURL
}

// TODO: This is display/presentation logic; try to factor it out of the backend service implementation.
//       (Have it be provided to the service, maybe? Or another way.)
func notificationIcon(st state.Issue) notifications.OcticonID {
	switch st {
	case state.IssueOpen:
		return "issue-opened"
	case state.IssueClosed:
		return "issue-closed"
	default:
		return ""
	}
}

/* TODO
func (e event) Octicon() string {
	switch e.Event.Type {
	case issues.Reopened:
		return "octicon-primitive-dot"
	case issues.Closed:
		return "octicon-circle-slash"
	default:
		return "octicon-primitive-dot"
	}
}*/

func notificationColor(st state.Issue) notifications.RGB {
	switch st {
	case state.IssueOpen: // Open.
		return notifications.RGB{R: 0x6c, G: 0xc6, B: 0x44}
	case state.IssueClosed: // Closed.
		return notifications.RGB{R: 0xbd, G: 0x2c, B: 0x00}
	default:
		return notifications.RGB{}
	}
}
