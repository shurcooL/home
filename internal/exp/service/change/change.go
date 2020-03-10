// Package change provides a change service definition.
package change

import (
	"context"
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

// Service defines methods of a change tracking service.
type Service interface {
	// List changes.
	List(ctx context.Context, repo string, opt ListOptions) ([]Change, error)
	// Count changes.
	Count(ctx context.Context, repo string, opt ListOptions) (uint64, error)

	// Get a change.
	Get(ctx context.Context, repo string, id uint64) (Change, error)

	// ListTimeline lists timeline items (change.Comment, change.Review, change.TimelineItem) for specified change id.
	ListTimeline(ctx context.Context, repo string, id uint64, opt *ListTimelineOptions) ([]interface{}, error)
	// ListCommits lists change commits, from first to last.
	ListCommits(ctx context.Context, repo string, id uint64) ([]Commit, error)
	// Get a change diff.
	GetDiff(ctx context.Context, repo string, id uint64, opt *GetDiffOptions) ([]byte, error)

	// EditComment edits a comment.
	EditComment(ctx context.Context, repo string, id uint64, cr CommentRequest) (Comment, error)
}

// Change represents a change in a repository.
type Change struct {
	ID        uint64
	State     state.Change
	Title     string
	Labels    []issues.Label
	Author    users.User
	CreatedAt time.Time
	Replies   int // Number of replies to this change (not counting the mandatory change description comment).

	Commits      int // Number of commits (not populated during list operation).
	ChangedFiles int // Number of changed files (not populated during list operation).
}

type Commit struct {
	SHA        string
	Message    string // TODO: Consider splitting into Subject, Body.
	Author     users.User
	AuthorTime time.Time
}

// ListOptions are options for list and count operations.
type ListOptions struct {
	Filter StateFilter
}

// StateFilter is a filter by state.
type StateFilter string

const (
	// FilterOpen is a state filter that includes open changes.
	FilterOpen StateFilter = "open"
	// FilterClosedMerged is a state filter that includes closed and merged changes.
	FilterClosedMerged StateFilter = "closed|merged"
	// FilterAll is a state filter that includes all changes.
	FilterAll StateFilter = "all"
)

// ListTimelineOptions controls pagination.
type ListTimelineOptions struct {
	// Start is the index of first result to retrieve, zero-indexed.
	Start int

	// Length is the number of results to include.
	Length int
}

type GetDiffOptions struct {
	// Commit is the commit ID of the commit to fetch.
	Commit string
}

// CommentRequest is a request to edit a comment.
type CommentRequest struct {
	ID       string
	Reaction *reactions.EmojiID // If not nil, toggle this reaction.
}
