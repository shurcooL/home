// Package v1tov2 provides an issuev2.Service wrapper
// on top of an issuev1.Service implementation.
package v1tov2

import (
	"context"
	"log"
	"regexp"
	"strings"

	"dmitri.shuralyov.com/go/prefixtitle"
	"dmitri.shuralyov.com/state"
	issuev2 "github.com/shurcooL/home/internal/exp/service/issuev2"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/issues"
	issuev1 "github.com/shurcooL/issues"
)

// Service implements issuev2.Service using issuev1.Service.
type Service struct {
	issuev1.Service
	ListV1Repos func() []issues.RepoSpec
	Notif       notification.FullService
}

func (s Service) CreateIssue(ctx context.Context, issue issuev2.Issue /* TODO: IssueCreate */) (issuev2.Issue, error) {
	i, err := s.Create(ctx, issues.RepoSpec{URI: "dmitri.shuralyov.com/scratch" /* TODO */}, issues.Issue{
		ID:     uint64(issue.ID),
		State:  issues.OpenState, // TODO: issue.State,
		Title:  issue.Title,
		Labels: issue.Labels,
		Comment: issues.Comment{
			User:      issue.Author,
			CreatedAt: issue.CreatedAt,
			Edited:    nil,  // TODO
			Body:      "",   // TODO
			Reactions: nil,  // TODO
			Editable:  true, // TODO
		},
		Replies: issue.Replies,
	})
	if err != nil {
		return issuev2.Issue{}, err
	}
	return issuev2.Issue{
		ID:         int64(i.ID),
		Author:     i.User,
		CreatedAt:  i.CreatedAt,
		ImportPath: "import/path", // TODO
		Title:      i.Title,
		State:      state.IssueOpen, // TODO
		Labels:     i.Labels,
		Replies:    i.Replies,
	}, nil
}

func (s Service) ListIssues(ctx context.Context, pattern string, opt issuev2.ListOptions) ([]issuev2.Issue, error) {
	match := matchPattern(pattern)
	var isV2 []issuev2.Issue
	for _, repo := range s.ListV1Repos() {
		isV1, err := s.List(ctx, repo, issues.IssueListOptions{State: opt.State})
		if err != nil {
			return isV2, err
		}
		for _, i := range isV1 {
			paths, title := prefixtitle.ParseIssue(repo.URI, i.Title)
			if !match(paths[0]) {
				continue
			}
			isV2 = append(isV2, issuev2.Issue{
				ID:         int64(i.ID),
				CreatedAt:  i.CreatedAt,
				Author:     i.User,
				State:      state.Issue(i.State),
				ImportPath: paths[0],
				Title:      title,
				Labels:     i.Labels,
				Replies:    i.Replies,
			})
		}
	}
	return isV2, nil
}

func (s Service) CountIssues(ctx context.Context, pattern string, opt issuev2.CountOptions) (int64, error) {
	match := matchPattern(pattern)
	var countV2 int64
	for _, repo := range s.ListV1Repos() {
		isV1, err := s.List(ctx, repo, issues.IssueListOptions{State: opt.State})
		if err != nil {
			return countV2, err
		}
		for _, i := range isV1 {
			paths, _ := prefixtitle.ParseIssue(repo.URI, i.Title)
			if !match(paths[0]) {
				continue
			}
			countV2++
		}
	}
	return countV2, nil
}

func (s Service) CreateComment(ctx context.Context, repo issues.RepoSpec, id uint64, c issues.Comment) (issues.Comment, error) {
	c, err := s.Service.CreateComment(ctx, repo, id, c)
	if err != nil {
		return issues.Comment{}, err
	}
	// Notify subscribed users.
	// TODO: Come up with a better way to compute fragment; that logic shouldn't be duplicated here from issuesapp router.
	err = s.Notif.Notify(ctx, notification.Notification{
		Namespace:   repo.URI,
		ThreadType:  "issue",
		ThreadID:    id,
		ImportPaths: []string{"dmitri.shuralyov.com/some/import/path"},
		Time:        c.CreatedAt,
		Actor:       c.User,
		Payload: notification.IssueComment{
			IssueTitle:     "issue title",
			IssueState:     state.IssueOpen,
			CommentBody:    c.Body,
			CommentHTMLURL: "",
		},
		Unread:        true,
		Participating: false,
		Mentioned:     false,
	})
	if err != nil {
		log.Println("Service.CreateComment: failed to s.Notif.Notify:", err)
	}
	return c, nil
}

// matchPattern(pattern)(name) reports whether name matches pattern.
// Pattern is a limited glob pattern in which '...' means 'any string',
// foo/... matches foo too, and there is no other special syntax.
// The pattern "all" is a special case and matches all names.
func matchPattern(pattern string) func(name string) bool {
	if pattern == "all" {
		return func(string) bool { return true }
	}
	re := regexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	return regexp.MustCompile(`^` + re + `$`).MatchString
}
