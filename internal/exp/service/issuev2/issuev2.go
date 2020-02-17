// Package issuev2 defines an issue tracking service for Go packages.
// It uses import path patterns as a first-class primitive.
package issuev2

import (
	"context"
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

// Service defines methods of an issue tracking service for Go packages.
type Service interface {
	// TODO: use more secure thingy
	//Create(ctx context.Context, repo RepoSpec, issue Issue) (Issue, error)
	// CreateIssue creates a new issue.
	CreateIssue(ctx context.Context, r CreateIssueRequest) (Issue, error)
	// CreateIssueComment creates a new comment for specified issue id.
	CreateIssueComment(ctx context.Context, id int64, r CreateIssueCommentRequest) (Comment, error)

	// ListIssues lists issues that match the specified pattern.
	ListIssues(ctx context.Context, pattern string, opt ListOptions) ([]Issue, error)
	// CountIssues counts issues that match the specified pattern.
	CountIssues(ctx context.Context, pattern string, opt CountOptions) (int64, error)

	// GetIssue gets an issue with the specified ID.
	GetIssue(ctx context.Context, id int64) (Issue, error)
	// EditIssue edits the specified issue id.
	EditIssue(ctx context.Context, id int64, r EditIssueRequest) (Issue, []issues.Event, error)

	// ListIssueTimeline lists timeline items (Comment, Event) for specified issue id
	// in chronological order. The issue description comes first in a timeline.
	ListIssueTimeline(ctx context.Context, id int64, opt *ListOptions) ([]interface{}, error)

	issues.Service
}

type (
	CreateIssueRequest struct {
		ImportPath string
		Title      string
		Body       string
	}
	CreateIssueCommentRequest struct {
		Body string
	}
	EditIssueRequest struct {
		State state.Issue
	}

	ListOptions struct {
		State issues.StateFilter
	}
	CountOptions struct {
		State issues.StateFilter
	}
)

// Issue represents an issue on a Go package.
type Issue struct {
	ID         int64
	Author     users.User
	CreatedAt  time.Time
	ImportPath string
	Title      string
	State      state.Issue
	Labels     []issues.Label
	Replies    int // Number of replies to this issue (not counting the mandatory issue description comment).
	Editable   bool
}

// Comment represents a comment left on an issue.
type Comment struct {
	ID        int64
	Author    users.User
	CreatedAt time.Time
	Edited    *Edited // Edited is nil if the comment hasn't been edited.
	Body      string
	Reactions []reactions.Reaction
	Editable  bool // Editable represents whether the current user (if any) can perform edit operations on this comment (or the encompassing issue).
}

// Edited provides the actor and timing information for an edited item.
type Edited struct {
	By users.User
	At time.Time
}
