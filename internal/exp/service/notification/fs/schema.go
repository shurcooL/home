package fs

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/users"
)

func marshalUserSpec(us users.UserSpec) string {
	return fmt.Sprintf("%d@%s", us.ID, us.Domain)
}

// unmarshalUserSpec parses userSpec, a string like "1@example.com"
// into a users.UserSpec{ID: 1, Domain: "example.com"}.
func unmarshalUserSpec(userSpec string) (users.UserSpec, error) {
	parts := strings.SplitN(userSpec, "@", 2)
	if len(parts) != 2 {
		return users.UserSpec{}, fmt.Errorf("user spec is not 2 parts: %v", len(parts))
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return users.UserSpec{}, err
	}
	return users.UserSpec{ID: id, Domain: parts[1]}, nil
}

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

// notificationDisk is an on-disk representation of notification.Notification.
// Unread is omitted from struct because it's encoded in the file path.
type notificationDisk struct {
	Namespace  string
	ThreadType string
	ThreadID   uint64

	ImportPaths []string
	Time        time.Time
	Actor       userSpec

	Payload interface{} // One of notification.{Issue,Change,IssueComment,ChangeComment}.

	// TODO.
	//Participating bool
	//Mentioned     bool
}

// MarshalJSON implements the json.Marshaler interface.
func (n notificationDisk) MarshalJSON() ([]byte, error) {
	v := struct {
		Namespace  string
		ThreadType string
		ThreadID   uint64

		ImportPaths []string
		Time        time.Time
		Actor       userSpec

		Type    string
		Payload interface{}

		//Participating bool
		//Mentioned     bool
	}{
		Namespace:   n.Namespace,
		ThreadType:  n.ThreadType,
		ThreadID:    n.ThreadID,
		ImportPaths: n.ImportPaths,
		Time:        n.Time,
		Actor:       n.Actor,
		//Participating: n.Participating,
		//Mentioned:     n.Mentioned,
	}
	switch p := n.Payload.(type) {
	case notification.Issue:
		v.Type = "issue"
		v.Payload = fromIssue(p)
	case notification.Change:
		v.Type = "change"
		v.Payload = fromChange(p)
	case notification.IssueComment:
		v.Type = "issueComment"
		v.Payload = fromIssueComment(p)
	case notification.ChangeComment:
		v.Type = "changeComment"
		v.Payload = fromChangeComment(p)
	default:
		return nil, fmt.Errorf("notificationDisk.MarshalJSON: invalid payload type %T; notificationDisk was %+v", n.Payload, n)
	}
	return json.Marshal(v)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (n *notificationDisk) UnmarshalJSON(b []byte) error {
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
		Actor       userSpec

		Type    string
		Payload json.RawMessage

		//Participating bool
		//Mentioned     bool
	}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	*n = notificationDisk{
		Namespace:   v.Namespace,
		ThreadType:  v.ThreadType,
		ThreadID:    v.ThreadID,
		ImportPaths: v.ImportPaths,
		Time:        v.Time,
		Actor:       v.Actor,
		//Participating: v.Participating,
		//Mentioned:     v.Mentioned,
	}
	switch v.Type {
	case "issue":
		var p issue
		err := json.Unmarshal(v.Payload, &p)
		if err != nil {
			return err
		}
		n.Payload = p.Issue()
	case "change":
		var p change
		err := json.Unmarshal(v.Payload, &p)
		if err != nil {
			return err
		}
		n.Payload = p.Change()
	case "issueComment":
		var p issueComment
		err := json.Unmarshal(v.Payload, &p)
		if err != nil {
			return err
		}
		n.Payload = p.IssueComment()
	case "changeComment":
		var p changeComment
		err := json.Unmarshal(v.Payload, &p)
		if err != nil {
			return err
		}
		n.Payload = p.ChangeComment()
	default:
		return fmt.Errorf("Notification.UnmarshalJSON: invalid payload type %q", v.Type)
	}
	return nil
}

type issue struct {
	Action       string
	IssueTitle   string
	IssueBody    string
	IssueHTMLURL string
}

func fromIssue(i notification.Issue) issue {
	return issue(i)
}

func (i issue) Issue() notification.Issue {
	return notification.Issue(i)
}

type change struct {
	Action        string
	ChangeTitle   string
	ChangeBody    string
	ChangeHTMLURL string
}

func fromChange(c notification.Change) change {
	return change(c)
}

func (c change) Change() notification.Change {
	return notification.Change(c)
}

type issueComment struct {
	IssueTitle     string
	IssueState     string
	CommentBody    string
	CommentHTMLURL string
}

func fromIssueComment(c notification.IssueComment) issueComment {
	var issueState string
	switch c.IssueState {
	case state.IssueOpen:
		issueState = "open"
	case state.IssueClosed:
		issueState = "closed"
	}
	return issueComment{
		IssueTitle:     c.IssueTitle,
		IssueState:     issueState,
		CommentBody:    c.CommentBody,
		CommentHTMLURL: c.CommentHTMLURL,
	}
}

func (c issueComment) IssueComment() notification.IssueComment {
	var issueState state.Issue
	switch c.IssueState {
	case "open":
		issueState = state.IssueOpen
	case "closed":
		issueState = state.IssueClosed
	}
	return notification.IssueComment{
		IssueTitle:     c.IssueTitle,
		IssueState:     issueState,
		CommentBody:    c.CommentBody,
		CommentHTMLURL: c.CommentHTMLURL,
	}
}

type changeComment struct {
	ChangeTitle    string
	ChangeState    string
	CommentBody    string
	CommentReview  int `json:",omitempty"`
	CommentHTMLURL string
}

func fromChangeComment(c notification.ChangeComment) changeComment {
	var changeState string
	switch c.ChangeState {
	case state.ChangeOpen:
		changeState = "open"
	case state.ChangeClosed:
		changeState = "closed"
	case state.ChangeMerged:
		changeState = "merged"
	}
	var commentReview int
	switch c.CommentReview {
	case state.ReviewPlus2:
		commentReview = +2
	case state.ReviewPlus1:
		commentReview = +1
	case state.ReviewNoScore:
		commentReview = 0
	case state.ReviewMinus1:
		commentReview = -1
	case state.ReviewMinus2:
		commentReview = -2
	}
	return changeComment{
		ChangeTitle:    c.ChangeTitle,
		ChangeState:    changeState,
		CommentBody:    c.CommentBody,
		CommentReview:  commentReview,
		CommentHTMLURL: c.CommentHTMLURL,
	}
}

func (c changeComment) ChangeComment() notification.ChangeComment {
	var changeState state.Change
	switch c.ChangeState {
	case "open":
		changeState = state.ChangeOpen
	case "closed":
		changeState = state.ChangeClosed
	case "merged":
		changeState = state.ChangeMerged
	}
	var commentReview state.Review
	switch c.CommentReview {
	case +2:
		commentReview = state.ReviewPlus2
	case +1:
		commentReview = state.ReviewPlus1
	case 0:
		commentReview = state.ReviewNoScore
	case -1:
		commentReview = state.ReviewMinus1
	case -2:
		commentReview = state.ReviewMinus2
	}
	return notification.ChangeComment{
		ChangeTitle:    c.ChangeTitle,
		ChangeState:    changeState,
		CommentBody:    c.CommentBody,
		CommentReview:  commentReview,
		CommentHTMLURL: c.CommentHTMLURL,
	}
}

// Tree layout:
//
// 	root
// 	├── notifications - unread notifications only
// 	│   └── userSpec
// 	│       └── namespace-threadType-threadID - encoded notification stream
// 	├── read - read notifications only
// 	│   └── userSpec
// 	│       └── namespace-threadType-threadID - encoded notification stream
// 	└── subscribers
// 	    └── namespace
// 	        ├── threadType-threadID
// 	        │   └── userSpec - blank file
// 	        └── userSpec - blank file
//
// ThreadType is primarily needed to separate namespaces of {Namespace, ThreadID}.
// Without ThreadType, a notification about an issue with ThreadID 1 in namespace "a"
// would clash with a notification about a change with ThreadID 1 in namespace "a".

func notificationsDir(user users.UserSpec) string {
	return path.Join("notifications", marshalUserSpec(user))
}

func notificationPath(user users.UserSpec, key string) string {
	return path.Join(notificationsDir(user), key)
}

func readDir(user users.UserSpec) string {
	return path.Join("read", marshalUserSpec(user))
}

func readPath(user users.UserSpec, key string) string {
	return path.Join(readDir(user), key)
}

func notificationKey(namespace, threadType string, threadID uint64) string {
	// TODO: Think about namespace replacement of "/" -> "-", is it optimal?
	return fmt.Sprintf("%s-%s-%d", strings.Replace(namespace, "/", "-", -1), threadType, threadID)
}

func subscribersDir(namespace, threadType string, threadID uint64) string {
	switch {
	default:
		return path.Join("subscribers", namespace, fmt.Sprintf("%s-%d", threadType, threadID))
	case threadType == "" && threadID == 0:
		return path.Join("subscribers", namespace)
	}
}

func subscriberPath(namespace, threadType string, threadID uint64, subscriber users.UserSpec) string {
	return path.Join(subscribersDir(namespace, threadType, threadID), marshalUserSpec(subscriber))
}
