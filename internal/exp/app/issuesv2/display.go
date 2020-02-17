package issuesv2

import (
	"fmt"
	"time"

	"github.com/shurcooL/home/internal/exp/service/issuev2"
	"github.com/shurcooL/issues"
)

// issueItem represents an issue item for display purposes.
type issueItem struct {
	// IssueItem can be one of issuev2.Comment, issues.Event.
	IssueItem interface{}
}

func (i issueItem) TemplateName() string {
	switch i.IssueItem.(type) {
	case issuev2.Comment:
		return "comment"
	case issues.Event:
		return "event"
	default:
		panic(fmt.Errorf("unknown item type %T", i.IssueItem))
	}
}

func (i issueItem) CreatedAt() time.Time {
	switch i := i.IssueItem.(type) {
	case issuev2.Comment:
		return i.CreatedAt
	case issues.Event:
		return i.CreatedAt
	default:
		panic(fmt.Errorf("unknown item type %T", i))
	}
}

func (i issueItem) ID() int64 {
	switch i := i.IssueItem.(type) {
	case issuev2.Comment:
		return i.ID
	case issues.Event:
		return int64(i.ID)
	default:
		panic(fmt.Errorf("unknown item type %T", i))
	}
}

// byCreatedAtID implements sort.Interface.
type byCreatedAtID []issueItem

func (s byCreatedAtID) Len() int { return len(s) }
func (s byCreatedAtID) Less(i, j int) bool {
	if s[i].CreatedAt().Equal(s[j].CreatedAt()) {
		// If CreatedAt time is equal, fall back to ID as a tiebreaker.
		return s[i].ID() < s[j].ID()
	}
	return s[i].CreatedAt().Before(s[j].CreatedAt())
}
func (s byCreatedAtID) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
