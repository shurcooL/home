// Package issues provides an issues service definition.
package issues

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

// RepoSpec is a specification for a repository.
type RepoSpec struct {
	URI string // URI is clean '/'-separated URI. E.g., "example.com/user/repo".
}

// String implements fmt.Stringer.
func (rs RepoSpec) String() string {
	return rs.URI
}

// Service defines methods of an issue tracking service.
type Service interface {
	// List issues.
	List(ctx context.Context, repo RepoSpec, opt IssueListOptions) ([]Issue, error)
	// Count issues.
	Count(ctx context.Context, repo RepoSpec, opt IssueListOptions) (uint64, error)

	// Get an issue.
	Get(ctx context.Context, repo RepoSpec, id uint64) (Issue, error)

	// ListTimeline lists timeline items (Comment, Event) for specified issue id
	// in chronological order. The issue description comes first in a timeline.
	ListTimeline(ctx context.Context, repo RepoSpec, id uint64, opt *ListOptions) ([]interface{}, error)

	// Create a new issue.
	Create(ctx context.Context, repo RepoSpec, issue Issue) (Issue, error)
	// CreateComment creates a new comment for specified issue id.
	CreateComment(ctx context.Context, repo RepoSpec, id uint64, comment Comment) (Comment, error)

	// Edit the specified issue id.
	Edit(ctx context.Context, repo RepoSpec, id uint64, ir IssueRequest) (Issue, []Event, error)
	// EditComment edits comment of specified issue id.
	EditComment(ctx context.Context, repo RepoSpec, id uint64, cr CommentRequest) (Comment, error)
}

// Issue represents an issue on a repository.
type Issue struct {
	ID     uint64
	State  state.Issue
	Title  string
	Labels []Label
	Comment
	Replies int // Number of replies to this issue (not counting the mandatory issue description comment).
}

// Label represents a label.
type Label struct {
	Name  string
	Color RGB
}

// TODO: Dedup.
//
// RGB represents a 24-bit color without alpha channel.
type RGB struct {
	R, G, B uint8
}

func (c RGB) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	r |= r << 8
	g = uint32(c.G)
	g |= g << 8
	b = uint32(c.B)
	b |= b << 8
	a = uint32(255)
	a |= a << 8
	return
}

// HexString returns a hexadecimal color string. For example, "#ff0000" for red.
func (c RGB) HexString() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// Comment represents a comment left on an issue.
type Comment struct {
	ID        uint64
	User      users.User
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

// IssueRequest is a request to edit an issue.
// To edit the body, use EditComment with comment ID 0.
type IssueRequest struct {
	State *state.Issue
	Title *string
	// TODO: Labels *[]Label
}

// CommentRequest is a request to edit a comment.
type CommentRequest struct {
	ID       uint64
	Body     *string            // If not nil, set the body.
	Reaction *reactions.EmojiID // If not nil, toggle this reaction.
}

// Validate returns non-nil error if the issue is invalid.
func (i Issue) Validate() error {
	if strings.TrimSpace(i.Title) == "" {
		return fmt.Errorf("title can't be blank or all whitespace")
	}
	return nil
}

// Validate returns non-nil error if the issue request is invalid.
func (ir IssueRequest) Validate() error {
	if ir.State != nil {
		switch *ir.State {
		case state.IssueOpen, state.IssueClosed:
		default:
			return fmt.Errorf("bad state")
		}
	}
	if ir.Title != nil {
		if strings.TrimSpace(*ir.Title) == "" {
			return fmt.Errorf("title can't be blank or all whitespace")
		}
	}
	return nil
}

// Validate returns non-nil error if the comment is invalid.
func (c Comment) Validate() error {
	// TODO: Issue descriptions can have blank bodies, support that (primarily for editing comments).
	if strings.TrimSpace(c.Body) == "" {
		return fmt.Errorf("comment body can't be blank or all whitespace")
	}
	return nil
}

// Validate validates the comment edit request, returning an non-nil error if it's invalid.
// requiresEdit reports if the edit request needs edit rights or if it can be done by anyone that can react.
func (cr CommentRequest) Validate() (requiresEdit bool, err error) {
	if cr.Body != nil {
		requiresEdit = true

		// TODO: Issue descriptions can have blank bodies, support that (primarily for editing comments).
		if strings.TrimSpace(*cr.Body) == "" {
			return requiresEdit, fmt.Errorf("comment body can't be blank or all whitespace")
		}
	}
	/*if cr.Reaction != nil {
		// TODO: Maybe validate that the emojiID is one of supported ones.
		//       Or maybe not (unsupported ones can be handled by frontend component).
		//       That way custom emoji can be added/removed, etc. Figure out what the best thing to do is and do it.
	}*/
	return requiresEdit, nil
}
