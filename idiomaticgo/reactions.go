package idiomaticgo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions"
)

// IssuesReactions implements reactions.Service on top of issues.Service,
// specifically for use by Idiomatic Go page. It hardcodes comment ID value of 0.
type IssuesReactions struct {
	Issues issues.Service
}

// issuesReactionsCommentID is the comment ID that IssuesReactions is hardcoded to use.
const issuesReactionsCommentID = 0

func (ir IssuesReactions) Get(ctx context.Context, uri string, id string) ([]reactions.Reaction, error) {
	// TODO: id is issueID/commentID. Maybe? Not needed for this specific use atm.
	issueID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	cs, err := ir.Issues.ListComments(ctx, issues.RepoSpec{URI: uri}, issueID, &issues.ListOptions{Start: issuesReactionsCommentID, Length: 1})
	if err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return nil, fmt.Errorf("id not found")
	}
	comment := cs[0]
	return comment.Reactions, nil
}

func (ir IssuesReactions) Toggle(ctx context.Context, uri string, id string, tr reactions.ToggleRequest) ([]reactions.Reaction, error) {
	// TODO: id is issueID/commentID. Maybe? Not needed for this specific use atm.
	issueID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	comment, err := ir.Issues.EditComment(ctx, issues.RepoSpec{URI: uri}, issueID, issues.CommentRequest{
		ID:       issuesReactionsCommentID,
		Reaction: &tr.Reaction,
	})
	if err != nil {
		return nil, err
	}
	return comment.Reactions, nil
}
