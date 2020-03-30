package change

import (
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

// Comment represents a comment left on a change.
// TODO: Consider removing in favor of Review with commented state and no inline comments.
type Comment struct {
	ID        string
	User      users.User
	CreatedAt time.Time
	Edited    *Edited // Edited is nil if the comment hasn't been edited.
	Body      string
	Reactions []reactions.Reaction
	Editable  bool // Editable represents whether the current user (if any) can perform edit operations on this comment.
}

// Edited provides the actor and timing information for an edited item.
type Edited struct {
	By users.User
	At time.Time
}

// Review represents a review left on a change.
type Review struct {
	ID        string
	User      users.User
	CreatedAt time.Time
	Edited    *Edited // Edited is nil if the review hasn't been edited.
	State     state.Review
	Body      string // Optional.
	Reactions []reactions.Reaction
	Editable  bool // Editable represents whether the current user (if any) can perform edit operations on this review.
	Comments  []InlineComment
}

// InlineComment represents an inline comment that was left as part of a review.
type InlineComment struct {
	ID        string
	File      string
	Line      int
	Body      string
	Reactions []reactions.Reaction
}

// TimelineItem represents a timeline item.
type TimelineItem struct {
	ID        string // TODO: See if this belongs here.
	Actor     users.User
	CreatedAt time.Time

	// Payload specifies the event type. It's one of:
	// ClosedEvent, ReopenedEvent, ..., MergedEvent, DeletedEvent.
	Payload interface{}
}

type (
	// ClosedEvent is when a change is closed.
	ClosedEvent struct {
		Closer        interface{} // Change (with State, Title), Commit (with SHA, Message, Author.AvatarURL), nil.
		CloserHTMLURL string      // If Closer is not nil.
	}

	// ReopenedEvent is when a change is reopened.
	ReopenedEvent struct{}

	// RenamedEvent is when a change is renamed.
	RenamedEvent struct {
		From string
		To   string
	}

	// CommitEvent is when a change gets a new commit.
	CommitEvent struct {
		SHA     string
		Subject string
	}

	// LabeledEvent is when a change is labeled.
	LabeledEvent struct {
		Label issues.Label
	}
	// UnlabeledEvent is when a change is unlabeled.
	UnlabeledEvent struct {
		Label issues.Label
	}

	ReviewRequestedEvent struct {
		RequestedReviewer users.User
	}
	ReviewRequestRemovedEvent struct {
		RequestedReviewer users.User
	}

	MergedEvent struct {
		CommitID      string
		CommitHTMLURL string // Optional.
		RefName       string
	}

	// DeletedEvent is a delete event.
	// THINK: Merge with "github.com/shurcooL/events/event".Delete?
	DeletedEvent struct {
		Type string // "branch", "comment".
		Name string
	}
)
