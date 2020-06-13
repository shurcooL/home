package fs

import (
	"context"
	"fmt"
	"strings"
	"time"

	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/users"
)

// threadType is the notifications thread type for this service.
const threadType = "issues"

func (*service) ThreadType(context.Context, issues.RepoSpec) (string, error) { return threadType, nil }

// subscribe subscribes user and anyone mentioned in body to the issue.
func (s *service) subscribe(ctx context.Context, repo issues.RepoSpec, issueID uint64, user users.UserSpec, body string) error {
	if s.notification == nil {
		return nil
	}

	subscribers := []users.UserSpec{user}

	// TODO: Find mentioned users in body.
	/*mentions, err := mentions(ctx, body)
	if err != nil {
		return err
	}
	subscribers = append(subscribers, mentions...)*/
	_ = body

	return s.notification.SubscribeThread(ctx, repo.URI, threadType, issueID, subscribers)
}

// markRead marks the specified issue as read for current user.
func (s *service) markRead(ctx context.Context, repo issues.RepoSpec, issueID uint64) error {
	if s.notification == nil {
		return nil
	}

	return s.notification.MarkThreadRead(ctx, repo.URI, threadType, issueID)
}

// notifyIssue notifies all subscribed users about an issue.
func (s *service) notifyIssue(ctx context.Context, repo issues.RepoSpec, issueID uint64, fragment string, issue issue, action string, time time.Time) error {
	if s.notification == nil {
		return nil
	}

	nr := notification.NotificationRequest{
		ImportPaths: []string{repo.URI},
		Time:        time,
		Payload: notification.Issue{
			Action:       action,
			IssueTitle:   issue.Title,
			IssueBody:    issue.Body,
			IssueHTMLURL: htmlURL(repo.URI, issueID, fragment),
		},
	}
	return s.notification.NotifyThread(ctx, repo.URI, threadType, issueID, nr)
}

// notifyIssueComment notifies all subscribed users about an issue comment.
func (s *service) notifyIssueComment(ctx context.Context, repo issues.RepoSpec, issueID uint64, fragment string, time time.Time, body string) error {
	if s.notification == nil {
		return nil
	}

	// TODO, THINK: Is this the best place/time? It's also being done in s.notify...
	// Get issue from storage for to populate event fields.
	var issue issue
	err := jsonDecodeFile(ctx, s.fs, issueCommentPath(repo, issueID, 0), &issue)
	if err != nil {
		return err
	}

	nr := notification.NotificationRequest{
		ImportPaths: []string{repo.URI},
		Time:        time,
		Payload: notification.IssueComment{
			IssueTitle:     issue.Title,
			IssueState:     issue.State,
			CommentBody:    body,
			CommentHTMLURL: htmlURL(repo.URI, issueID, fragment),
		},
	}
	return s.notification.NotifyThread(ctx, repo.URI, threadType, issueID, nr)
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
