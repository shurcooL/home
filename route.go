package main

import (
	"context"
	"fmt"

	"dmitri.shuralyov.com/route/github"
	"github.com/shurcooL/users"
)

// shurcoolSeesHomeRouter implements github.Router that
// targets GitHub issues and PRs on home apps for shurcooL user, and
// targets GitHub issues and PRs on github.com for all other users.
type shurcoolSeesHomeRouter struct {
	users users.Service
}

func (r shurcoolSeesHomeRouter) IssueURL(ctx context.Context, owner, repo string, issueID, commentID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == shurcool {
		return homeRouter{}.IssueURL(ctx, owner, repo, issueID, commentID)
	}
	return github.DotCom{}.IssueURL(ctx, owner, repo, issueID, commentID)
}

func (r shurcoolSeesHomeRouter) PullRequestURL(ctx context.Context, owner, repo string, prID, commentID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == shurcool {
		return homeRouter{}.PullRequestURL(ctx, owner, repo, prID, commentID)
	}
	return github.DotCom{}.PullRequestURL(ctx, owner, repo, prID, commentID)
}

// homeRouter implements github.Router that
// targets GitHub issues on home's issuesapp, and
// targets GitHub pull requests on home's changes app.
//
// THINK: It embeds home, issuesapp, changes app routing details; can it be composed better?
type homeRouter struct{}

func (homeRouter) IssueURL(_ context.Context, owner, repo string, issueID, commentID uint64) string {
	var fragment string
	if commentID != 0 {
		fragment = fmt.Sprintf("#comment-%d", commentID)
	}
	return fmt.Sprintf("/issues/github.com/%s/%s/%d%s", owner, repo, issueID, fragment)
}

func (homeRouter) PullRequestURL(_ context.Context, owner, repo string, prID, commentID uint64) string {
	var fragment string
	if commentID != 0 {
		fragment = fmt.Sprintf("#comment-%d", commentID)
	}
	return fmt.Sprintf("/changes/github.com/%s/%s/%d%s", owner, repo, prID, fragment)
}
