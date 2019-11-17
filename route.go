package main

import (
	"context"
	"fmt"

	"dmitri.shuralyov.com/route/gerrit"
	"dmitri.shuralyov.com/route/github"
	"github.com/shurcooL/users"
)

// dmitshurSeesHomeRouter implements github.Router and gerrit.Router that
// targets GitHub issues, PRs and Gerrit CLs on home apps for dmitshur user, and
// targets GitHub issues, PRs on github.com and Gerrit CLs on googlesource.com for all other users.
type dmitshurSeesHomeRouter struct {
	users users.Service
}

func (r dmitshurSeesHomeRouter) IssueURL(ctx context.Context, owner, repo string, issueID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.IssueURL(ctx, owner, repo, issueID)
	}
	return github.DotCom{}.IssueURL(ctx, owner, repo, issueID)
}

func (r dmitshurSeesHomeRouter) IssueCommentURL(ctx context.Context, owner, repo string, issueID, commentID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.IssueCommentURL(ctx, owner, repo, issueID, commentID)
	}
	return github.DotCom{}.IssueCommentURL(ctx, owner, repo, issueID, commentID)
}

func (r dmitshurSeesHomeRouter) IssueEventURL(ctx context.Context, owner, repo string, issueID, eventID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.IssueEventURL(ctx, owner, repo, issueID, eventID)
	}
	return github.DotCom{}.IssueEventURL(ctx, owner, repo, issueID, eventID)
}

func (r dmitshurSeesHomeRouter) PullRequestURL(ctx context.Context, owner, repo string, prID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.PullRequestURL(ctx, owner, repo, prID)
	}
	return github.DotCom{}.PullRequestURL(ctx, owner, repo, prID)
}

func (r dmitshurSeesHomeRouter) PullRequestCommentURL(ctx context.Context, owner, repo string, prID, commentID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.PullRequestCommentURL(ctx, owner, repo, prID, commentID)
	}
	return github.DotCom{}.PullRequestCommentURL(ctx, owner, repo, prID, commentID)
}

func (r dmitshurSeesHomeRouter) PullRequestReviewURL(ctx context.Context, owner, repo string, prID, reviewID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.PullRequestReviewURL(ctx, owner, repo, prID, reviewID)
	}
	return github.DotCom{}.PullRequestReviewURL(ctx, owner, repo, prID, reviewID)
}

func (r dmitshurSeesHomeRouter) PullRequestReviewCommentURL(ctx context.Context, owner, repo string, prID, reviewCommentID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.PullRequestReviewCommentURL(ctx, owner, repo, prID, reviewCommentID)
	}
	return github.DotCom{}.PullRequestReviewCommentURL(ctx, owner, repo, prID, reviewCommentID)
}

func (r dmitshurSeesHomeRouter) PullRequestEventURL(ctx context.Context, owner, repo string, prID, eventID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.PullRequestEventURL(ctx, owner, repo, prID, eventID)
	}
	return github.DotCom{}.PullRequestEventURL(ctx, owner, repo, prID, eventID)
}

func (r dmitshurSeesHomeRouter) ChangeURL(ctx context.Context, server, project string, changeID uint64) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.ChangeURL(ctx, server, project, changeID)
	}
	return gerrit.GoogleSource{}.ChangeURL(ctx, server, project, changeID)
}

func (r dmitshurSeesHomeRouter) ChangeMessageURL(ctx context.Context, server, project string, changeID uint64, messageID string) string {
	if currentUser, err := r.users.GetAuthenticatedSpec(ctx); err == nil && currentUser == dmitshur {
		return homeRouter{}.ChangeMessageURL(ctx, server, project, changeID, messageID)
	}
	return gerrit.GoogleSource{}.ChangeMessageURL(ctx, server, project, changeID, messageID)
}

// homeRouter implements github.Router and gerrit.Router that
// targets GitHub issues on home's issuesapp, and
// targets GitHub PRs and Gerrit CLs on home's changes app.
//
// THINK: It embeds home, issuesapp, changes app routing details; can it be composed better?
type homeRouter struct{}

func (homeRouter) IssueURL(_ context.Context, owner, repo string, issueID uint64) string {
	return fmt.Sprintf("/issues/github.com/%s/%s/%d", owner, repo, issueID)
}

func (homeRouter) IssueCommentURL(_ context.Context, owner, repo string, issueID, commentID uint64) string {
	return fmt.Sprintf("/issues/github.com/%s/%s/%d#comment-%d", owner, repo, issueID, commentID)
}

func (homeRouter) IssueEventURL(_ context.Context, owner, repo string, issueID, eventID uint64) string {
	return fmt.Sprintf("/issues/github.com/%s/%s/%d#event-%d", owner, repo, issueID, eventID)
}

func (homeRouter) PullRequestURL(_ context.Context, owner, repo string, prID uint64) string {
	return fmt.Sprintf("/changes/github.com/%s/%s/%d", owner, repo, prID)
}

func (homeRouter) PullRequestCommentURL(_ context.Context, owner, repo string, prID, commentID uint64) string {
	return fmt.Sprintf("/changes/github.com/%s/%s/%d#comment-c%d", owner, repo, prID, commentID)
}

func (homeRouter) PullRequestReviewURL(_ context.Context, owner, repo string, prID, reviewID uint64) string {
	return fmt.Sprintf("/changes/github.com/%s/%s/%d#comment-r%d", owner, repo, prID, reviewID)
}

func (homeRouter) PullRequestReviewCommentURL(_ context.Context, owner, repo string, prID, reviewCommentID uint64) string {
	return fmt.Sprintf("/changes/github.com/%s/%s/%d#comment-rc%d", owner, repo, prID, reviewCommentID)
}

func (homeRouter) PullRequestEventURL(_ context.Context, owner, repo string, prID, eventID uint64) string {
	return fmt.Sprintf("/changes/github.com/%s/%s/%d#event-%d", owner, repo, prID, eventID)
}

func (homeRouter) ChangeURL(_ context.Context, server, project string, changeID uint64) string {
	return fmt.Sprintf("/changes/%s/%s/%d", server, project, changeID)
}

func (homeRouter) ChangeMessageURL(_ context.Context, server, project string, changeID uint64, messageID string) string {
	return fmt.Sprintf("/changes/%s/%s/%d#comment-%s", server, project, changeID, messageID)
}
