// Package githubapi implements a change.Service using GitHub API clients.
// Aside from ability to leave reactions, it is read-only.
package githubapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"dmitri.shuralyov.com/route/github"
	"dmitri.shuralyov.com/state"
	githubv3 "github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

// NewService creates a GitHub-backed change.Service using given GitHub clients.
// At this time it infers the current user from GitHub clients (their authentication info),
// and cannot be used to serve multiple users. Both GitHub clients must use same authentication info.
//
// If router is nil, github.DotCom router is used, which links to subjects on github.com.
func NewService(clientV3 *githubv3.Client, clientV4 *githubv4.Client, router github.Router) change.Service {
	if router == nil {
		router = github.DotCom{}
	}
	return service{
		clV3: clientV3,
		clV4: clientV4,
		rtr:  router,
	}
}

type service struct {
	clV3 *githubv3.Client // GitHub REST API v3 client.
	clV4 *githubv4.Client // GitHub GraphQL API v4 client.
	rtr  github.Router
}

// We use 0 as a special ID for the comment that is the PR description. This comment is edited differently.
const prDescriptionCommentID string = "0"

func (s service) List(ctx context.Context, rs string, opt change.ListOptions) ([]change.Change, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return nil, err
	}
	var states *[]githubv4.PullRequestState
	switch opt.Filter {
	case change.FilterOpen:
		states = &[]githubv4.PullRequestState{githubv4.PullRequestStateOpen}
	case change.FilterClosedMerged:
		states = &[]githubv4.PullRequestState{githubv4.PullRequestStateClosed, githubv4.PullRequestStateMerged}
	case change.FilterAll:
		states = nil // No states to filter the PRs by.
	default:
		// TODO: Map to 400 Bad Request HTTP error.
		return nil, fmt.Errorf("invalid change.ListOptions.Filter value: %q", opt.Filter)
	}
	var q struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					Number uint64
					State  githubv4.PullRequestState
					Title  string
					Labels struct {
						Nodes []struct {
							Name  string
							Color string
						}
					} `graphql:"labels(first:100)"`
					Author    *githubV4Actor
					CreatedAt githubv4.DateTime
					Comments  struct {
						TotalCount int
					}
				}
			} `graphql:"pullRequests(first:30,orderBy:{field:CREATED_AT,direction:DESC},states:$prStates)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"prStates":        states,
	}
	err = s.clV4.Query(ctx, &q, variables)
	if err != nil {
		return nil, err
	}
	var is []change.Change
	for _, pr := range q.Repository.PullRequests.Nodes {
		var labels []issues.Label
		for _, l := range pr.Labels.Nodes {
			labels = append(labels, issues.Label{
				Name:  l.Name,
				Color: ghColor(l.Color),
			})
		}
		is = append(is, change.Change{
			ID:        pr.Number,
			State:     ghPRState(pr.State),
			Title:     pr.Title,
			Labels:    labels,
			Author:    ghActor(pr.Author),
			CreatedAt: pr.CreatedAt.Time,
			Replies:   pr.Comments.TotalCount,
		})
	}
	return is, nil
}

func (s service) Count(ctx context.Context, rs string, opt change.ListOptions) (uint64, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return 0, err
	}
	var states *[]githubv4.PullRequestState
	switch opt.Filter {
	case change.FilterOpen:
		states = &[]githubv4.PullRequestState{githubv4.PullRequestStateOpen}
	case change.FilterClosedMerged:
		states = &[]githubv4.PullRequestState{githubv4.PullRequestStateClosed, githubv4.PullRequestStateMerged}
	case change.FilterAll:
		states = nil // No states to filter the PRs by.
	default:
		// TODO: Map to 400 Bad Request HTTP error.
		return 0, fmt.Errorf("invalid change.ListOptions.Filter value: %q", opt.Filter)
	}
	var q struct {
		Repository struct {
			PullRequests struct {
				TotalCount uint64
			} `graphql:"pullRequests(states:$prStates)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"prStates":        states,
	}
	err = s.clV4.Query(ctx, &q, variables)
	return q.Repository.PullRequests.TotalCount, err
}

func (s service) Get(ctx context.Context, rs string, id uint64) (change.Change, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return change.Change{}, err
	}
	var q struct {
		Repository struct {
			PullRequest struct {
				Number    uint64
				State     githubv4.PullRequestState
				Title     string
				Author    *githubV4Actor
				CreatedAt githubv4.DateTime
				Comments  struct {
					TotalCount int
				}
				Commits struct {
					TotalCount int
				}
				ChangedFiles int
			} `graphql:"pullRequest(number:$prNumber)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"prNumber":        githubv4.Int(id),
	}
	err = s.clV4.Query(ctx, &q, variables)
	if err != nil {
		return change.Change{}, err
	}

	// TODO: Eliminate comment body properties from issues.Issue. It's missing increasingly more fields, like Edited, etc.
	pr := q.Repository.PullRequest
	return change.Change{
		ID:           pr.Number,
		State:        ghPRState(pr.State),
		Title:        pr.Title,
		Author:       ghActor(pr.Author),
		CreatedAt:    pr.CreatedAt.Time,
		Replies:      pr.Comments.TotalCount,
		Commits:      pr.Commits.TotalCount,
		ChangedFiles: pr.ChangedFiles,
	}, nil
}

func (s service) ListCommits(ctx context.Context, rs string, id uint64) ([]change.Commit, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return nil, err
	}
	cs, _, err := s.clV3.PullRequests.ListCommits(ctx, repo.Owner, repo.Repo, int(id), &githubv3.ListOptions{PerPage: 100}) // TODO: Pagination.
	if err != nil {
		return nil, err
	}
	var commits []change.Commit
	for _, c := range cs {
		commits = append(commits, change.Commit{
			SHA:        *c.SHA,
			Message:    *c.Commit.Message,
			Author:     ghV3UserOrGitUser(c.Author, *c.Commit.Author),
			AuthorTime: *c.Commit.Author.Date,
		})
	}
	return commits, nil
}

func (s service) GetDiff(ctx context.Context, rs string, id uint64, opt *change.GetDiffOptions) ([]byte, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return nil, err
	}
	switch opt {
	case nil:
		diff, _, err := s.clV3.PullRequests.GetRaw(ctx, repo.Owner, repo.Repo, int(id), githubv3.RawOptions{Type: githubv3.Diff})
		if err != nil {
			return nil, err
		}
		return []byte(diff), nil
	default:
		diff, _, err := s.clV3.Repositories.GetCommitRaw(ctx, repo.Owner, repo.Repo, opt.Commit, githubv3.RawOptions{Type: githubv3.Diff})
		if err != nil {
			return nil, err
		}
		return []byte(diff), nil
	}
}

func (s service) ListTimeline(ctx context.Context, rs string, id uint64, opt *change.ListTimelineOptions) ([]interface{}, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return nil, err
	}
	type comment struct { // Comment fields.
		Author          *githubV4Actor
		PublishedAt     githubv4.DateTime
		LastEditedAt    *githubv4.DateTime
		Editor          *githubV4Actor
		Body            string
		ReactionGroups  reactionGroups
		ViewerCanUpdate bool
	}
	type event struct { // Common fields for all events.
		Actor     *githubV4Actor
		CreatedAt githubv4.DateTime
	}
	var q struct {
		Repository struct {
			PullRequest struct {
				comment `graphql:"...@include(if:$firstPage)"` // Fetch the PR description only on first page.

				Timeline struct {
					Nodes []struct {
						Typename     string `graphql:"__typename"`
						IssueComment struct {
							DatabaseID uint64
							comment
						} `graphql:"...on IssueComment"`
						ClosedEvent struct {
							event
							Closer struct {
								Typename    string `graphql:"__typename"`
								PullRequest struct {
									State      githubv4.PullRequestState
									Title      string
									Repository struct {
										Owner struct{ Login string }
										Name  string
									}
									Number uint64
								} `graphql:"...on PullRequest"`
								Commit struct {
									OID     string
									Message string
									Author  struct {
										AvatarURL string `graphql:"avatarUrl(size:96)"`
									}
									URL string
								} `graphql:"...on Commit"`
							}
						} `graphql:"...on ClosedEvent"`
						ReopenedEvent struct {
							event
						} `graphql:"...on ReopenedEvent"`
						RenamedTitleEvent struct {
							event
							CurrentTitle  string
							PreviousTitle string
						} `graphql:"...on RenamedTitleEvent"`
						LabeledEvent struct {
							event
							Label struct {
								Name  string
								Color string
							}
						} `graphql:"...on LabeledEvent"`
						UnlabeledEvent struct {
							event
							Label struct {
								Name  string
								Color string
							}
						} `graphql:"...on UnlabeledEvent"`
						ReviewRequestedEvent struct {
							event
							RequestedReviewer struct {
								User *githubV4User `graphql:"...on User"`
							}
						} `graphql:"...on ReviewRequestedEvent"`
						ReviewRequestRemovedEvent struct {
							event
							RequestedReviewer struct {
								User *githubV4User `graphql:"...on User"`
							}
						} `graphql:"...on ReviewRequestRemovedEvent"`
						MergedEvent struct {
							event
							Commit struct {
								OID string
								URL string
							}
							MergeRefName string
						} `graphql:"...on MergedEvent"`
						HeadRefDeletedEvent struct {
							event
							HeadRefName string
						} `graphql:"...on HeadRefDeletedEvent"`
						// TODO: Wait for GitHub to add support.
						//CommentDeletedEvent struct {
						//	event
						//} `graphql:"...on CommentDeletedEvent"`
					}
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage githubv4.Boolean
					}
				} `graphql:"timeline(first:100,after:$timelineCursor)"`

				// Need to use PullRequest.Reviews rather than PullRequest.Timeline.PullRequestReview,
				// because the latter is missing single-inline-reply reviews (as of 2018-02-08).
				Reviews struct {
					Nodes []struct {
						DatabaseID        uint64
						Author            *githubV4Actor
						AuthorAssociation githubv4.CommentAuthorAssociation
						PublishedAt       githubv4.DateTime
						LastEditedAt      *githubv4.DateTime
						Editor            *githubV4Actor
						State             githubv4.PullRequestReviewState
						Body              string
						ReactionGroups    reactionGroups
						ViewerCanUpdate   bool
						Comments          struct {
							Nodes []struct {
								DatabaseID       uint64
								Path             string
								OriginalPosition int // The original line index in the diff to which the comment applies.
								Body             string
								ReactionGroups   reactionGroups
							}
						} `graphql:"comments(first:100)"` // TODO: Pagination... Figure out how to make pagination across 2 resource types work...
					}
				} `graphql:"reviews(first:100)@include(if:$firstPage)"` // TODO: Pagination... Figure out how to make pagination across 2 resource types work...
			} `graphql:"pullRequest(number:$prNumber)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		Viewer githubV4User
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"prNumber":        githubv4.Int(id),
		"firstPage":       githubv4.Boolean(true),
		"timelineCursor":  (*githubv4.String)(nil),
	}
	var timeline []interface{} // Of type change.Comment, change.Review, change.TimelineItem.
	for {
		err := s.clV4.Query(ctx, &q, variables)
		if err != nil {
			return nil, err
		}
		if variables["firstPage"].(githubv4.Boolean) {
			pr := q.Repository.PullRequest.comment // PR description comment.
			var edited *change.Edited
			if pr.LastEditedAt != nil {
				edited = &change.Edited{
					By: ghActor(pr.Editor),
					At: pr.LastEditedAt.Time,
				}
			}
			timeline = append(timeline, change.Comment{
				ID:        prDescriptionCommentID,
				User:      ghActor(pr.Author),
				CreatedAt: pr.PublishedAt.Time,
				Edited:    edited,
				Body:      pr.Body,
				Reactions: ghReactions(pr.ReactionGroups, ghUser(&q.Viewer)),
				Editable:  pr.ViewerCanUpdate,
			})
		}
		for _, node := range q.Repository.PullRequest.Timeline.Nodes {
			if node.Typename != "IssueComment" {
				continue
			}
			comment := node.IssueComment
			var edited *change.Edited
			if comment.LastEditedAt != nil {
				edited = &change.Edited{
					By: ghActor(comment.Editor),
					At: comment.LastEditedAt.Time,
				}
			}
			timeline = append(timeline, change.Comment{
				ID:        fmt.Sprintf("c%d", comment.DatabaseID),
				User:      ghActor(comment.Author),
				CreatedAt: comment.PublishedAt.Time,
				Edited:    edited,
				Body:      comment.Body,
				Reactions: ghReactions(comment.ReactionGroups, ghUser(&q.Viewer)),
				Editable:  comment.ViewerCanUpdate,
			})
		}
		if variables["firstPage"].(githubv4.Boolean) {
			for _, review := range q.Repository.PullRequest.Reviews.Nodes {
				state, ok := ghPRReviewState(review.State, review.AuthorAssociation)
				if !ok {
					continue
				}
				var edited *change.Edited
				if review.LastEditedAt != nil {
					edited = &change.Edited{
						By: ghActor(review.Editor),
						At: review.LastEditedAt.Time,
					}
				}
				var cs []change.InlineComment
				for _, comment := range review.Comments.Nodes {
					cs = append(cs, change.InlineComment{
						ID:        fmt.Sprintf("rc%d", comment.DatabaseID),
						File:      comment.Path,
						Line:      comment.OriginalPosition, // TODO: This isn't line in file, it's line *in the diff*. Take it into account, compute real line, etc.
						Body:      comment.Body,
						Reactions: ghReactions(comment.ReactionGroups, ghUser(&q.Viewer)),
					})
				}
				sort.Slice(cs, func(i, j int) bool {
					if cs[i].File == cs[j].File {
						return cs[i].Line < cs[j].Line
					}
					return cs[i].File < cs[j].File
				})
				timeline = append(timeline, change.Review{
					ID:        fmt.Sprintf("r%d", review.DatabaseID),
					User:      ghActor(review.Author),
					CreatedAt: review.PublishedAt.Time,
					Edited:    edited,
					State:     state,
					Body:      review.Body,
					Reactions: ghReactions(review.ReactionGroups, ghUser(&q.Viewer)),
					Editable:  review.ViewerCanUpdate,
					Comments:  cs,
				})
			}
		}
		for _, event := range q.Repository.PullRequest.Timeline.Nodes {
			e := change.TimelineItem{
				//ID: 0, // TODO.
			}
			switch event.Typename {
			case "ClosedEvent":
				e.Actor = ghActor(event.ClosedEvent.Actor)
				e.CreatedAt = event.ClosedEvent.CreatedAt.Time
				switch event.ClosedEvent.Closer.Typename {
				case "PullRequest":
					pr := event.ClosedEvent.Closer.PullRequest
					e.Payload = change.ClosedEvent{
						Closer: change.Change{
							State: ghPRState(pr.State),
							Title: pr.Title,
						},
						CloserHTMLURL: s.rtr.PullRequestURL(ctx, pr.Repository.Owner.Login, pr.Repository.Name, pr.Number),
					}
				case "Commit":
					c := event.ClosedEvent.Closer.Commit
					e.Payload = change.ClosedEvent{
						Closer: change.Commit{
							SHA:     c.OID,
							Message: c.Message,
							Author:  users.User{AvatarURL: c.Author.AvatarURL},
						},
						CloserHTMLURL: c.URL,
					}
				default:
					e.Payload = change.ClosedEvent{}
				}
			case "ReopenedEvent":
				e.Actor = ghActor(event.ReopenedEvent.Actor)
				e.CreatedAt = event.ReopenedEvent.CreatedAt.Time
				e.Payload = change.ReopenedEvent{}
			case "RenamedTitleEvent":
				e.Actor = ghActor(event.RenamedTitleEvent.Actor)
				e.CreatedAt = event.RenamedTitleEvent.CreatedAt.Time
				e.Payload = change.RenamedEvent{
					From: event.RenamedTitleEvent.PreviousTitle,
					To:   event.RenamedTitleEvent.CurrentTitle,
				}
			case "LabeledEvent":
				e.Actor = ghActor(event.LabeledEvent.Actor)
				e.CreatedAt = event.LabeledEvent.CreatedAt.Time
				e.Payload = change.LabeledEvent{
					Label: issues.Label{
						Name:  event.LabeledEvent.Label.Name,
						Color: ghColor(event.LabeledEvent.Label.Color),
					},
				}
			case "UnlabeledEvent":
				e.Actor = ghActor(event.UnlabeledEvent.Actor)
				e.CreatedAt = event.UnlabeledEvent.CreatedAt.Time
				e.Payload = change.UnlabeledEvent{
					Label: issues.Label{
						Name:  event.UnlabeledEvent.Label.Name,
						Color: ghColor(event.UnlabeledEvent.Label.Color),
					},
				}
			case "ReviewRequestedEvent":
				e.Actor = ghActor(event.ReviewRequestedEvent.Actor)
				e.CreatedAt = event.ReviewRequestedEvent.CreatedAt.Time
				e.Payload = change.ReviewRequestedEvent{
					RequestedReviewer: ghUser(event.ReviewRequestedEvent.RequestedReviewer.User),
				}
			case "ReviewRequestRemovedEvent":
				e.Actor = ghActor(event.ReviewRequestRemovedEvent.Actor)
				e.CreatedAt = event.ReviewRequestRemovedEvent.CreatedAt.Time
				e.Payload = change.ReviewRequestRemovedEvent{
					RequestedReviewer: ghUser(event.ReviewRequestRemovedEvent.RequestedReviewer.User),
				}
			case "MergedEvent":
				e.Actor = ghActor(event.MergedEvent.Actor)
				e.CreatedAt = event.MergedEvent.CreatedAt.Time
				e.Payload = change.MergedEvent{
					CommitID:      event.MergedEvent.Commit.OID,
					CommitHTMLURL: event.MergedEvent.Commit.URL,
					RefName:       event.MergedEvent.MergeRefName,
				}
			case "HeadRefDeletedEvent":
				e.Actor = ghActor(event.HeadRefDeletedEvent.Actor)
				e.CreatedAt = event.HeadRefDeletedEvent.CreatedAt.Time
				e.Payload = change.DeletedEvent{
					Type: "branch",
					Name: event.HeadRefDeletedEvent.HeadRefName,
				}
			// TODO: Wait for GitHub to add support.
			//case "CommentDeletedEvent":
			//	e.Actor = ghActor(event.CommentDeletedEvent.Actor)
			//	e.CreatedAt = event.CommentDeletedEvent.CreatedAt.Time
			default:
				continue
			}
			timeline = append(timeline, e)
		}
		if !q.Repository.PullRequest.Timeline.PageInfo.HasNextPage {
			break
		}
		variables["firstPage"] = githubv4.Boolean(false)
		variables["timelineCursor"] = githubv4.NewString(q.Repository.PullRequest.Timeline.PageInfo.EndCursor)
	}
	// We can't just delegate pagination to GitHub because our timeline items may not match up 1:1,
	// e.g., we want to skip Commit in the timeline, etc. (At least for now; may reconsider later.)
	if opt != nil {
		start := opt.Start
		if start > len(timeline) {
			start = len(timeline)
		}
		end := opt.Start + opt.Length
		if end > len(timeline) {
			end = len(timeline)
		}
		timeline = timeline[start:end]
	}
	return timeline, nil
}

func (s service) EditComment(ctx context.Context, rs string, id uint64, cr change.CommentRequest) (change.Comment, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return change.Comment{}, err
	}

	var comment change.Comment

	if cr.Reaction != nil {
		reactionContent, err := externalizeReaction(*cr.Reaction)
		if err != nil {
			return change.Comment{}, err
		}
		// See if user has already reacted with that reaction.
		// If not, add it. Otherwise, remove it.
		var (
			subjectID        githubv4.ID
			viewerHasReacted bool
			viewer           users.User
		)
		switch {
		case cr.ID == prDescriptionCommentID:
			var q struct {
				Repository struct {
					PullRequest struct {
						ID        githubv4.ID
						Reactions struct {
							ViewerHasReacted bool
						} `graphql:"reactions(content:$reactionContent)"`
					} `graphql:"pullRequest(number:$prNumber)"`
				} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
				Viewer githubV4User
			}
			variables := map[string]interface{}{
				"repositoryOwner": githubv4.String(repo.Owner),
				"repositoryName":  githubv4.String(repo.Repo),
				"prNumber":        githubv4.Int(id),
				"reactionContent": reactionContent,
			}
			err = s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return change.Comment{}, err
			}
			subjectID = q.Repository.PullRequest.ID
			viewerHasReacted = q.Repository.PullRequest.Reactions.ViewerHasReacted
			viewer = ghUser(&q.Viewer)
		case strings.HasPrefix(cr.ID, "c"):
			commentID := "012:IssueComment" + cr.ID[len("c"):]
			var q struct {
				Node struct {
					IssueComment struct {
						ID        githubv4.ID
						Reactions struct {
							ViewerHasReacted bool
						} `graphql:"reactions(content:$reactionContent)"`
					} `graphql:"...on IssueComment"`
				} `graphql:"node(id:$commentID)"`
				Viewer githubV4User
			}
			variables := map[string]interface{}{
				"commentID":       githubv4.ID(base64.StdEncoding.EncodeToString([]byte(commentID))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
				"reactionContent": reactionContent,
			}
			err = s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return change.Comment{}, err
			}
			subjectID = q.Node.IssueComment.ID
			viewerHasReacted = q.Node.IssueComment.Reactions.ViewerHasReacted
			viewer = ghUser(&q.Viewer)
		case strings.HasPrefix(cr.ID, "rc"):
			commentID := "024:PullRequestReviewComment" + cr.ID[len("rc"):]
			var q struct {
				Node struct {
					PullRequestReviewComment struct {
						ID        githubv4.ID
						Reactions struct {
							ViewerHasReacted bool
						} `graphql:"reactions(content:$reactionContent)"`
					} `graphql:"...on PullRequestReviewComment"`
				} `graphql:"node(id:$commentID)"`
				Viewer githubV4User
			}
			variables := map[string]interface{}{
				"commentID":       githubv4.ID(base64.StdEncoding.EncodeToString([]byte(commentID))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
				"reactionContent": reactionContent,
			}
			err = s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return change.Comment{}, err
			}
			subjectID = q.Node.PullRequestReviewComment.ID
			viewerHasReacted = q.Node.PullRequestReviewComment.Reactions.ViewerHasReacted
			viewer = ghUser(&q.Viewer)
		case strings.HasPrefix(cr.ID, "r"):
			reviewID := "017:PullRequestReview" + cr.ID[len("r"):]
			var q struct {
				Node struct {
					PullRequestReview struct {
						ID        githubv4.ID
						Reactions struct {
							ViewerHasReacted bool
						} `graphql:"reactions(content:$reactionContent)"`
					} `graphql:"...on PullRequestReview"`
				} `graphql:"node(id:$reviewID)"`
				Viewer githubV4User
			}
			variables := map[string]interface{}{
				"reviewID":        githubv4.ID(base64.StdEncoding.EncodeToString([]byte(reviewID))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
				"reactionContent": reactionContent,
			}
			err = s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return change.Comment{}, err
			}
			subjectID = q.Node.PullRequestReview.ID
			viewerHasReacted = q.Node.PullRequestReview.Reactions.ViewerHasReacted
			viewer = ghUser(&q.Viewer)
		default:
			return change.Comment{}, fmt.Errorf("EditComment: unrecognized kind of comment ID: %q", cr.ID)
		}

		var rgs reactionGroups
		if !viewerHasReacted {
			// Add reaction.
			var m struct {
				AddReaction struct {
					Subject struct {
						ReactionGroups reactionGroups
					}
				} `graphql:"addReaction(input:$input)"`
			}
			input := githubv4.AddReactionInput{
				SubjectID: subjectID,
				Content:   reactionContent,
			}
			err := s.clV4.Mutate(ctx, &m, input, nil)
			if err != nil {
				return change.Comment{}, err
			}
			rgs = m.AddReaction.Subject.ReactionGroups
		} else {
			// Remove reaction.
			var m struct {
				RemoveReaction struct {
					Subject struct {
						ReactionGroups reactionGroups
					}
				} `graphql:"removeReaction(input:$input)"`
			}
			input := githubv4.RemoveReactionInput{
				SubjectID: subjectID,
				Content:   reactionContent,
			}
			err := s.clV4.Mutate(ctx, &m, input, nil)
			if err != nil {
				return change.Comment{}, err
			}
			rgs = m.RemoveReaction.Subject.ReactionGroups
		}
		// TODO: Consider setting other fields? Now that using GraphQL, not that expensive (same API call).
		//       But not needed for app yet...
		comment.Reactions = ghReactions(rgs, viewer)
	}

	return comment, nil
}

type repoSpec struct {
	Owner string
	Repo  string
}

func ghRepoSpec(rs string) (repoSpec, error) {
	// The "github.com/" prefix is expected to be included.
	ghOwnerRepo := strings.Split(rs, "/")
	if len(ghOwnerRepo) != 3 || ghOwnerRepo[0] != "github.com" || ghOwnerRepo[1] == "" || ghOwnerRepo[2] == "" {
		return repoSpec{}, fmt.Errorf(`RepoSpec is not of form "github.com/owner/repo": %q`, rs)
	}
	return repoSpec{
		Owner: ghOwnerRepo[1],
		Repo:  ghOwnerRepo[2],
	}, nil
}

type githubV4Actor struct {
	User struct {
		DatabaseID uint64
	} `graphql:"...on User"`
	Bot struct {
		DatabaseID uint64
	} `graphql:"...on Bot"`
	Login     string
	AvatarURL string `graphql:"avatarUrl(size:96)"`
	URL       string
}

func ghActor(actor *githubV4Actor) users.User {
	if actor == nil {
		return ghost // Deleted user, replace with https://github.com/ghost.
	}
	return users.User{
		UserSpec: users.UserSpec{
			ID:     actor.User.DatabaseID | actor.Bot.DatabaseID,
			Domain: "github.com",
		},
		Login:     actor.Login,
		AvatarURL: actor.AvatarURL,
		HTMLURL:   actor.URL,
	}
}

type githubV4User struct {
	DatabaseID uint64
	Login      string
	AvatarURL  string `graphql:"avatarUrl(size:96)"`
	URL        string
}

func ghUser(user *githubV4User) users.User {
	if user == nil {
		return ghost // Deleted user, replace with https://github.com/ghost.
	}
	return users.User{
		UserSpec: users.UserSpec{
			ID:     user.DatabaseID,
			Domain: "github.com",
		},
		Login:     user.Login,
		AvatarURL: user.AvatarURL,
		HTMLURL:   user.URL,
	}
}

func ghV3UserOrGitUser(user *githubv3.User, gitUser githubv3.CommitAuthor) users.User {
	if user == nil {
		return users.User{
			Name:      *gitUser.Name,
			Email:     *gitUser.Email,
			AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96", // TODO: Use email.
		}
	}
	if *user.ID == 0 {
		return ghost // Deleted user, replace with https://github.com/ghost.
	}
	return users.User{
		UserSpec: users.UserSpec{
			ID:     uint64(*user.ID),
			Domain: "github.com",
		},
		Login:     *user.Login,
		AvatarURL: *user.AvatarURL,
		HTMLURL:   *user.HTMLURL,
	}
}

// ghost is https://github.com/ghost, a replacement for deleted users.
var ghost = users.User{
	UserSpec: users.UserSpec{
		ID:     10137,
		Domain: "github.com",
	},
	Login:     "ghost",
	AvatarURL: "https://avatars3.githubusercontent.com/u/10137?v=4",
	HTMLURL:   "https://github.com/ghost",
}

// ghPRState converts a GitHub PullRequestState to state.Change.
func ghPRState(st githubv4.PullRequestState) state.Change {
	switch st {
	case githubv4.PullRequestStateOpen:
		return state.ChangeOpen
	case githubv4.PullRequestStateClosed:
		return state.ChangeClosed
	case githubv4.PullRequestStateMerged:
		return state.ChangeMerged
	default:
		panic("unreachable")
	}
}

// ghPRReviewState converts a GitHub PullRequestReviewState to state.Review, if it's supported.
func ghPRReviewState(st githubv4.PullRequestReviewState, aa githubv4.CommentAuthorAssociation) (_ state.Review, ok bool) {
	// TODO: This is a heuristic. Author can be a member of the organization that
	// owns the repository, but it's not known whether they have push access or not.
	// TODO: Use https://developer.github.com/v3/repos/collaborators/#review-a-users-permission-level perhaps?
	// Or wait for equivalent to be available via API v4?
	approver := aa == githubv4.CommentAuthorAssociationOwner ||
		aa == githubv4.CommentAuthorAssociationCollaborator ||
		aa == githubv4.CommentAuthorAssociationMember

	switch {
	case st == githubv4.PullRequestReviewStateApproved && approver:
		return state.ReviewPlus2, true
	case st == githubv4.PullRequestReviewStateApproved && !approver:
		return state.ReviewPlus1, true
	case st == githubv4.PullRequestReviewStateCommented:
		return state.ReviewNoScore, true
	case st == githubv4.PullRequestReviewStateChangesRequested && !approver:
		return state.ReviewMinus1, true
	case st == githubv4.PullRequestReviewStateChangesRequested && approver:
		return state.ReviewMinus2, true
	case st == githubv4.PullRequestReviewStateDismissed:
		// PullRequestReviewStateDismissed are reviews that have been retroactively dismissed.
		// Display them as a regular comment review for now (we can't know the original state).
		// THINK: Consider displaying these more distinctly.
		return state.ReviewNoScore, true
	case st == githubv4.PullRequestReviewStatePending:
		// PullRequestReviewStatePending are reviews that are pending (haven't been posted yet).
		// TODO: Consider displaying pending review comments. Figure this out
		//       when adding ability to leave reviews.
		return 0, false
	default:
		panic("unreachable")
	}
}

// ghColor converts a GitHub color hex string like "ff0000"
// into an issues.RGB value.
func ghColor(hex string) issues.RGB {
	var c issues.RGB
	fmt.Sscanf(hex, "%02x%02x%02x", &c.R, &c.G, &c.B)
	return c
}

type reactionGroups []struct {
	Content githubv4.ReactionContent
	Users   struct {
		Nodes      []*githubV4User
		TotalCount int
	} `graphql:"users(first:10)"`
	ViewerHasReacted bool
}

// ghReactions converts []githubv4.ReactionGroup to []reactions.Reaction.
func ghReactions(rgs reactionGroups, viewer users.User) []reactions.Reaction {
	var rs []reactions.Reaction
	for _, rg := range rgs {
		if rg.Users.TotalCount == 0 {
			continue
		}

		// Only return the details of first few users and viewer.
		var us []users.User
		addedViewer := false
		for i := 0; i < rg.Users.TotalCount; i++ {
			if i < len(rg.Users.Nodes) {
				user := ghUser(rg.Users.Nodes[i])
				us = append(us, user)
				if user.UserSpec == viewer.UserSpec {
					addedViewer = true
				}
			} else if i == len(rg.Users.Nodes) {
				// Add viewer last if they've reacted, but haven't been added already.
				if rg.ViewerHasReacted && !addedViewer {
					us = append(us, viewer)
				} else {
					us = append(us, users.User{})
				}
			} else {
				us = append(us, users.User{})
			}
		}

		rs = append(rs, reactions.Reaction{
			Reaction: internalizeReaction(rg.Content),
			Users:    us,
		})
	}
	return rs
}

// internalizeReaction converts githubv4.ReactionContent to reactions.EmojiID.
func internalizeReaction(reaction githubv4.ReactionContent) reactions.EmojiID {
	switch reaction {
	case githubv4.ReactionContentThumbsUp:
		return "+1"
	case githubv4.ReactionContentThumbsDown:
		return "-1"
	case githubv4.ReactionContentLaugh:
		return "smile"
	case githubv4.ReactionContentHooray:
		return "tada"
	case githubv4.ReactionContentConfused:
		return "confused"
	case githubv4.ReactionContentHeart:
		return "heart"
	case githubv4.ReactionContentRocket:
		return "rocket"
	case githubv4.ReactionContentEyes:
		return "eyes"
	default:
		panic("unreachable")
	}
}

// externalizeReaction converts reactions.EmojiID to githubv4.ReactionContent.
func externalizeReaction(reaction reactions.EmojiID) (githubv4.ReactionContent, error) {
	switch reaction {
	case "+1":
		return githubv4.ReactionContentThumbsUp, nil
	case "-1":
		return githubv4.ReactionContentThumbsDown, nil
	case "smile":
		return githubv4.ReactionContentLaugh, nil
	case "tada":
		return githubv4.ReactionContentHooray, nil
	case "confused":
		return githubv4.ReactionContentConfused, nil
	case "heart":
		return githubv4.ReactionContentHeart, nil
	case "rocket":
		return githubv4.ReactionContentRocket, nil
	case "eyes":
		return githubv4.ReactionContentEyes, nil
	default:
		return "", fmt.Errorf("%q is an unsupported reaction", reaction)
	}
}

// githubPRThreadType is the notification thread type for GitHub Pull Requests.
const githubPRThreadType = "PullRequest"

// ThreadType returns the notification thread type for this service.
func (service) ThreadType(repo string) string { return githubPRThreadType }
