package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"dmitri.shuralyov.com/state"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
	"github.com/shurcooL/webdavfs/vfsutil"
)

// userSpec is an on-disk representation of users.UserSpec.
type userSpec struct {
	ID     uint64
	Domain string `json:",omitempty"`
}

func fromUserSpec(us users.UserSpec) userSpec {
	return userSpec{ID: us.ID, Domain: us.Domain}
}

func (us userSpec) UserSpec() users.UserSpec {
	return users.UserSpec{ID: us.ID, Domain: us.Domain}
}

func (us userSpec) Equal(other users.UserSpec) bool {
	return us.Domain == other.Domain && us.ID == other.ID
}

// rgb is an on-disk representation of issues.RGB.
type rgb struct {
	R, G, B uint8
}

func fromRGB(c issues.RGB) rgb {
	return rgb(c)
}

func (c rgb) RGB() issues.RGB {
	return issues.RGB(c)
}

// issue is an on-disk representation of issues.Issue.
type issue struct {
	State  issues.State
	Title  string
	Labels []label `json:",omitempty"`
	comment
}

// label is an on-disk representation of issues.Label.
type label struct {
	Name  string
	Color rgb
}

// comment is an on-disk representation of issues.Comment.
type comment struct {
	Author    userSpec
	CreatedAt time.Time
	Edited    *edited `json:",omitempty"`
	Body      string
	Reactions []reaction `json:",omitempty"`
}

type edited struct {
	By userSpec
	At time.Time
}

// reaction is an on-disk representation of reactions.Reaction.
type reaction struct {
	EmojiID reactions.EmojiID
	Authors []userSpec // First entry is first person who reacted.
}

// event is an on-disk representation of issues.Event.
type event struct {
	Actor     userSpec
	CreatedAt time.Time
	Type      issues.EventType
	Close     *closeDisk     `json:",omitempty"`
	Rename    *issues.Rename `json:",omitempty"`
	Label     *label         `json:",omitempty"`
}

// closeDisk is an on-disk representation of issues.Close.
// Nil issues.Close.Closer is represented by nil *closeDisk.
type closeDisk struct {
	Closer interface{} // issues.Change, issues.Commit.
}

func (c closeDisk) MarshalJSON() ([]byte, error) {
	var v struct {
		Type   string      // "change", "commit".
		Closer interface{} // change, commit.
	}
	switch p := c.Closer.(type) {
	case issues.Change:
		v.Type = "change"
		v.Closer = fromChange(p)
	case issues.Commit:
		v.Type = "commit"
		v.Closer = fromCommit(p)
	default:
		return nil, fmt.Errorf("closeDisk.MarshalJSON: unsupported Closer type %T", c.Closer)
	}
	return json.Marshal(v)
}

func (c *closeDisk) UnmarshalJSON(b []byte) error {
	// Ignore null, like in the main JSON package.
	if string(b) == "null" {
		return nil
	}
	var v struct {
		Type   string          // "change", "commit".
		Closer json.RawMessage // change, commit.
	}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	*c = closeDisk{}
	switch v.Type {
	case "change":
		var p change
		err := json.Unmarshal(v.Closer, &p)
		if err != nil {
			return err
		}
		c.Closer = p.Change()
	case "commit":
		var p commit
		err := json.Unmarshal(v.Closer, &p)
		if err != nil {
			return err
		}
		c.Closer = p.Commit()
	default:
		return fmt.Errorf("closeDisk.UnmarshalJSON: unsupported Closer type %q", v.Type)
	}
	return nil
}

func fromClose(c issues.Close) *closeDisk {
	if c.Closer == nil {
		return nil
	}
	return (*closeDisk)(&c)
}

func (c *closeDisk) Close() issues.Close {
	if c == nil {
		return issues.Close{Closer: nil}
	}
	return issues.Close(*c)
}

// change is an on-disk representation of issues.Change.
type change struct {
	State   state.Change
	Title   string
	HTMLURL string
}

func fromChange(c issues.Change) change {
	return change(c)
}

func (c change) Change() issues.Change {
	return issues.Change(c)
}

// commit is an on-disk representation of issues.Commit.
type commit struct {
	SHA             string
	Message         string
	AuthorAvatarURL string
	HTMLURL         string
}

func fromCommit(c issues.Commit) commit {
	return commit(c)
}

func (c commit) Commit() issues.Commit {
	return issues.Commit(c)
}

// Tree layout:
//
// 	root
// 	└── domain.com
// 	    └── path
// 	        └── issues
// 	            ├── 1
// 	            │   ├── 0 - encoded issue
// 	            │   ├── 1 - encoded comment
// 	            │   ├── 2
// 	            │   └── events
// 	            │       ├── 1 - encoded event
// 	            │       └── 2
// 	            └── 2
// 	                ├── 0
// 	                └── events

func (s *service) createNamespace(ctx context.Context, repo issues.RepoSpec) error {
	if path.Clean("/"+repo.URI) != "/"+repo.URI {
		return fmt.Errorf("invalid repo.URI (not clean): %q", repo.URI)
	}

	// Only needed for first issue in the repo.
	// THINK: Consider implicit dir adapter?
	return vfsutil.MkdirAll(ctx, s.fs, issuesDir(repo), 0755)
}

// issuesDir is '/'-separated path to issue storage dir.
func issuesDir(repo issues.RepoSpec) string {
	return path.Join(repo.URI, "issues")
}

func issueDir(repo issues.RepoSpec, issueID uint64) string {
	return path.Join(repo.URI, "issues", formatUint64(issueID))
}

func issueCommentPath(repo issues.RepoSpec, issueID, commentID uint64) string {
	return path.Join(repo.URI, "issues", formatUint64(issueID), formatUint64(commentID))
}

// issueEventsDir is '/'-separated path to issue events dir.
func issueEventsDir(repo issues.RepoSpec, issueID uint64) string {
	return path.Join(repo.URI, "issues", formatUint64(issueID), "events")
}

func issueEventPath(repo issues.RepoSpec, issueID, eventID uint64) string {
	return path.Join(repo.URI, "issues", formatUint64(issueID), "events", formatUint64(eventID))
}
