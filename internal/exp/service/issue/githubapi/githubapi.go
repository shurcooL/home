// Package githubapi implements issues.Service using GitHub API clients.
package githubapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"dmitri.shuralyov.com/route/github"
	"dmitri.shuralyov.com/state"
	githubv3 "github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

// NewService creates a GitHub-backed issues.Service using given GitHub clients.
// It uses notifications service, if not nil. At this time it infers the current user
// from GitHub clients (their authentication info), and cannot be used to serve multiple users.
// Both GitHub clients must use same authentication info.
//
// If router is nil, github.DotCom router is used, which links to subjects on github.com.
func NewService(clientV3 *githubv3.Client, clientV4 *githubv4.Client, notifications notifications.ExternalService, router github.Router) issues.Service {
	if router == nil {
		router = github.DotCom{}
	}
	return service{
		clV3:          clientV3,
		clV4:          clientV4,
		rtr:           router,
		notifications: notifications,
	}
}

type service struct {
	clV3 *githubv3.Client // GitHub REST API v3 client.
	clV4 *githubv4.Client // GitHub GraphQL API v4 client.
	rtr  github.Router

	// notifications may be nil if there's no notifications service.
	notifications notifications.ExternalService
}

// We use 0 as a special ID for the comment that is the issue description. This comment is edited differently.
const issueDescriptionCommentID uint64 = 0

func (s service) List(ctx context.Context, rs issues.RepoSpec, opt issues.IssueListOptions) ([]issues.Issue, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return nil, err
	}
	var states *[]githubv4.IssueState
	switch opt.State {
	case issues.StateFilter(issues.OpenState):
		states = &[]githubv4.IssueState{githubv4.IssueStateOpen}
	case issues.StateFilter(issues.ClosedState):
		states = &[]githubv4.IssueState{githubv4.IssueStateClosed}
	case issues.AllStates:
		states = nil // No states to filter the issues by.
	default:
		// TODO: Map to 400 Bad Request HTTP error.
		return nil, fmt.Errorf("invalid issues.IssueListOptions.State value: %q", opt.State)
	}
	var q struct {
		Repository struct {
			Issues struct {
				Nodes []struct {
					Number uint64
					State  githubv4.IssueState
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
			} `graphql:"issues(first:30,orderBy:{field:CREATED_AT,direction:DESC},states:$issuesStates)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"issuesStates":    states,
	}
	err = s.clV4.Query(ctx, &q, variables)
	if err != nil {
		return nil, err
	}
	var is []issues.Issue
	for _, issue := range q.Repository.Issues.Nodes {
		var labels []issues.Label
		for _, l := range issue.Labels.Nodes {
			labels = append(labels, issues.Label{
				Name:  l.Name,
				Color: ghColor(l.Color),
			})
		}
		is = append(is, issues.Issue{
			ID:     issue.Number,
			State:  ghIssueState(issue.State),
			Title:  issue.Title,
			Labels: labels,
			Comment: issues.Comment{
				User:      ghActor(issue.Author),
				CreatedAt: issue.CreatedAt.Time,
			},
			Replies: issue.Comments.TotalCount,
		})
	}
	return is, nil
}

func (s service) Count(ctx context.Context, rs issues.RepoSpec, opt issues.IssueListOptions) (uint64, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return 0, err
	}
	var states *[]githubv4.IssueState
	switch opt.State {
	case issues.StateFilter(issues.OpenState):
		states = &[]githubv4.IssueState{githubv4.IssueStateOpen}
	case issues.StateFilter(issues.ClosedState):
		states = &[]githubv4.IssueState{githubv4.IssueStateClosed}
	case issues.AllStates:
		states = nil // No states to filter the issues by.
	default:
		// TODO: Map to 400 Bad Request HTTP error.
		return 0, fmt.Errorf("invalid issues.IssueListOptions.State value: %q", opt.State)
	}
	var q struct {
		Repository struct {
			Issues struct {
				TotalCount uint64
			} `graphql:"issues(states:$issuesStates)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"issuesStates":    states,
	}
	err = s.clV4.Query(ctx, &q, variables)
	return q.Repository.Issues.TotalCount, err
}

func (s service) Get(ctx context.Context, rs issues.RepoSpec, id uint64) (issues.Issue, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return issues.Issue{}, err
	}
	var q struct {
		Repository struct {
			Issue struct {
				Number          uint64
				State           githubv4.IssueState
				Title           string
				Author          *githubV4Actor
				CreatedAt       githubv4.DateTime
				ViewerCanUpdate githubv4.Boolean
			} `graphql:"issue(number:$issueNumber)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"issueNumber":     githubv4.Int(id),
	}
	err = s.clV4.Query(ctx, &q, variables)
	if err != nil {
		return issues.Issue{}, err
	}

	// Mark as read. (We know there's an authenticated user since we're using GitHub GraphQL API v4 above.)
	err = s.markRead(ctx, rs, id)
	if err != nil {
		log.Println("service.Get: failed to markRead:", err)
	}

	// TODO: Eliminate comment body properties from issues.Issue. It's missing increasingly more fields, like Edited, etc.
	issue := q.Repository.Issue
	return issues.Issue{
		ID:    issue.Number,
		State: ghIssueState(issue.State),
		Title: issue.Title,
		Comment: issues.Comment{
			User:      ghActor(issue.Author),
			CreatedAt: issue.CreatedAt.Time,
			Editable:  bool(issue.ViewerCanUpdate),
		},
	}, nil
}

func (s service) ListTimeline(ctx context.Context, rs issues.RepoSpec, id uint64, opt *issues.ListOptions) ([]interface{}, error) {
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
			Issue struct {
				comment  `graphql:"...@include(if:$firstPage)"` // Fetch the issue description only on first page.
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
					}
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage githubv4.Boolean
					}
				} `graphql:"timeline(first:100,after:$timelineCursor)"`
			} `graphql:"issue(number:$issueNumber)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		Viewer githubV4User
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"issueNumber":     githubv4.Int(id),
		"firstPage":       githubv4.Boolean(true),
		"timelineCursor":  (*githubv4.String)(nil),
	}
	var timeline []interface{} // Of type issues.Comment and issues.Event.
	for {
		err := s.clV4.Query(ctx, &q, variables)
		if err != nil {
			return timeline, err
		}
		if variables["firstPage"].(githubv4.Boolean) {
			issue := q.Repository.Issue.comment // Issue description comment.
			var edited *issues.Edited
			if issue.LastEditedAt != nil {
				edited = &issues.Edited{
					By: ghActor(issue.Editor),
					At: issue.LastEditedAt.Time,
				}
			}
			timeline = append(timeline, issues.Comment{
				ID:        issueDescriptionCommentID,
				User:      ghActor(issue.Author),
				CreatedAt: issue.PublishedAt.Time,
				Edited:    edited,
				Body:      issue.Body,
				Reactions: ghReactions(issue.ReactionGroups, ghUser(&q.Viewer)),
				Editable:  issue.ViewerCanUpdate,
			})
		}
		for _, n := range q.Repository.Issue.Timeline.Nodes {
			switch n.Typename {
			case "IssueComment":
				comment := n.IssueComment
				var edited *issues.Edited
				if comment.LastEditedAt != nil {
					edited = &issues.Edited{
						By: ghActor(comment.Editor),
						At: comment.LastEditedAt.Time,
					}
				}
				timeline = append(timeline, issues.Comment{
					ID:        comment.DatabaseID,
					User:      ghActor(comment.Author),
					CreatedAt: comment.PublishedAt.Time,
					Edited:    edited,
					Body:      comment.Body,
					Reactions: ghReactions(comment.ReactionGroups, ghUser(&q.Viewer)),
					Editable:  comment.ViewerCanUpdate,
				})
			default:
				et := ghEventType(n.Typename)
				if !et.Valid() {
					continue
				}
				e := issues.Event{
					//ID:   0, // TODO.
					Type: et,
				}
				switch et {
				case issues.Closed:
					e.Actor = ghActor(n.ClosedEvent.Actor)
					e.CreatedAt = n.ClosedEvent.CreatedAt.Time
					switch n.ClosedEvent.Closer.Typename {
					case "PullRequest":
						pr := n.ClosedEvent.Closer.PullRequest
						e.Close = issues.Close{
							Closer: issues.Change{
								State:   ghPRState(pr.State),
								Title:   pr.Title,
								HTMLURL: s.rtr.PullRequestURL(ctx, pr.Repository.Owner.Login, pr.Repository.Name, pr.Number),
							},
						}
					case "Commit":
						c := n.ClosedEvent.Closer.Commit
						e.Close = issues.Close{
							Closer: issues.Commit{
								SHA:             c.OID,
								Message:         c.Message,
								AuthorAvatarURL: c.Author.AvatarURL,
								HTMLURL:         c.URL,
							},
						}
					default:
						e.Close = issues.Close{}
					}
				case issues.Reopened:
					e.Actor = ghActor(n.ReopenedEvent.Actor)
					e.CreatedAt = n.ReopenedEvent.CreatedAt.Time
				case issues.Renamed:
					e.Actor = ghActor(n.RenamedTitleEvent.Actor)
					e.CreatedAt = n.RenamedTitleEvent.CreatedAt.Time
					e.Rename = &issues.Rename{
						From: n.RenamedTitleEvent.PreviousTitle,
						To:   n.RenamedTitleEvent.CurrentTitle,
					}
				case issues.Labeled:
					e.Actor = ghActor(n.LabeledEvent.Actor)
					e.CreatedAt = n.LabeledEvent.CreatedAt.Time
					e.Label = &issues.Label{
						Name:  n.LabeledEvent.Label.Name,
						Color: ghColor(n.LabeledEvent.Label.Color),
					}
				case issues.Unlabeled:
					e.Actor = ghActor(n.UnlabeledEvent.Actor)
					e.CreatedAt = n.UnlabeledEvent.CreatedAt.Time
					e.Label = &issues.Label{
						Name:  n.UnlabeledEvent.Label.Name,
						Color: ghColor(n.UnlabeledEvent.Label.Color),
					}
				default:
					continue
				}
				timeline = append(timeline, e)
			}
		}
		if !q.Repository.Issue.Timeline.PageInfo.HasNextPage {
			break
		}
		variables["firstPage"] = githubv4.Boolean(false)
		variables["timelineCursor"] = githubv4.NewString(q.Repository.Issue.Timeline.PageInfo.EndCursor)
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

func (s service) CreateComment(ctx context.Context, rs issues.RepoSpec, id uint64, c issues.Comment) (issues.Comment, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return issues.Comment{}, err
	}
	var q struct {
		Repository struct {
			Issue struct {
				ID githubv4.ID
			} `graphql:"issue(number:$issueNumber)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"issueNumber":     githubv4.Int(id),
	}
	err = s.clV4.Query(ctx, &q, variables)
	if err != nil {
		return issues.Comment{}, err
	}
	var m struct {
		AddComment struct {
			CommentEdge struct {
				Node struct {
					DatabaseID      githubv4.Int
					Author          *githubV4Actor
					PublishedAt     githubv4.DateTime
					Body            githubv4.String
					ViewerCanUpdate githubv4.Boolean
				}
			}
		} `graphql:"addComment(input:$input)"`
	}
	input := githubv4.AddCommentInput{
		SubjectID: q.Repository.Issue.ID,
		Body:      githubv4.String(c.Body),
	}
	err = s.clV4.Mutate(ctx, &m, input, nil)
	if err != nil {
		return issues.Comment{}, err
	}
	comment := m.AddComment.CommentEdge.Node
	return issues.Comment{
		ID:        uint64(comment.DatabaseID),
		User:      ghActor(comment.Author),
		CreatedAt: comment.PublishedAt.Time,
		Body:      string(comment.Body),
		Editable:  bool(comment.ViewerCanUpdate),
	}, nil
}

func (s service) Create(ctx context.Context, rs issues.RepoSpec, i issues.Issue) (issues.Issue, error) {
	repo, err := ghRepoSpec(rs)
	if err != nil {
		return issues.Issue{}, err
	}
	issue, _, err := s.clV3.Issues.Create(ctx, repo.Owner, repo.Repo, &githubv3.IssueRequest{
		Title: &i.Title,
		Body:  &i.Body,
	})
	if err != nil {
		return issues.Issue{}, err
	}

	return issues.Issue{
		ID:    uint64(*issue.Number),
		State: issues.State(*issue.State),
		Title: *issue.Title,
		Comment: issues.Comment{
			ID:        issueDescriptionCommentID,
			User:      ghV3User(*issue.User),
			CreatedAt: *issue.CreatedAt,
			Editable:  true, // You can always edit issues you've created.
		},
	}, nil
}

func (s service) Edit(ctx context.Context, rs issues.RepoSpec, id uint64, ir issues.IssueRequest) (issues.Issue, []issues.Event, error) {
	// TODO: Why Validate here but not Create, etc.? Figure this out. Might only be needed in fs implementation.
	if err := ir.Validate(); err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return issues.Issue{}, nil, err
	}
	repo, err := ghRepoSpec(rs)
	if err != nil {
		// TODO: Map to 400 Bad Request HTTP error.
		return issues.Issue{}, nil, err
	}

	// Fetch issue state and title before the edit, as well as current user.
	var q struct {
		Repository struct {
			Issue struct {
				State githubv4.IssueState
				Title string
			} `graphql:"issue(number:$issueNumber)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		Viewer githubV4User
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Repo),
		"issueNumber":     githubv4.Int(id),
	}
	err = s.clV4.Query(ctx, &q, variables)
	if err != nil {
		return issues.Issue{}, nil, err
	}
	beforeEdit := q.Repository.Issue

	ghIR := githubv3.IssueRequest{
		Title: ir.Title,
	}
	if ir.State != nil {
		ghIR.State = githubv3.String(string(*ir.State))
	}

	issue, _, err := s.clV3.Issues.Edit(ctx, repo.Owner, repo.Repo, int(id), &ghIR)
	if err != nil {
		return issues.Issue{}, nil, err
	}

	// GitHub API doesn't return the event that will be generated as a result, so we predict what it'll be.
	event := issues.Event{
		// TODO: Figure out if event ID needs to be set, and if so, how to best do that...
		Actor:     ghUser(&q.Viewer),
		CreatedAt: time.Now().UTC(),
	}
	// TODO: A single edit operation can result in multiple events, we should emit multiple events in such cases. We're currently emitting at most one event.
	switch {
	case ir.State != nil && *ir.State != ghIssueState(beforeEdit.State):
		switch *ir.State {
		case issues.OpenState:
			event.Type = issues.Reopened
		case issues.ClosedState:
			event.Type = issues.Closed
		}
	case ir.Title != nil && *ir.Title != beforeEdit.Title:
		event.Type = issues.Renamed
		event.Rename = &issues.Rename{
			From: beforeEdit.Title,
			To:   *ir.Title,
		}
	}
	var events []issues.Event
	if event.Type != "" {
		events = append(events, event)
	}

	return issues.Issue{
		ID:    uint64(*issue.Number),
		State: issues.State(*issue.State),
		Title: *issue.Title,
		Comment: issues.Comment{
			ID:        issueDescriptionCommentID,
			User:      ghV3User(*issue.User),
			CreatedAt: *issue.CreatedAt,
			Editable:  true, // You can always edit issues you've edited.
		},
	}, events, nil
}

func (s service) EditComment(ctx context.Context, rs issues.RepoSpec, id uint64, cr issues.CommentRequest) (issues.Comment, error) {
	// TODO: Why Validate here but not CreateComment, etc.? Figure this out. Might only be needed in fs implementation.
	if _, err := cr.Validate(); err != nil {
		return issues.Comment{}, err
	}
	repo, err := ghRepoSpec(rs)
	if err != nil {
		return issues.Comment{}, err
	}

	if cr.ID == issueDescriptionCommentID {
		var comment issues.Comment

		// Apply edits.
		if cr.Body != nil {
			// Use Issues.Edit() API to edit comment 0 (the issue description).
			issue, _, err := s.clV3.Issues.Edit(ctx, repo.Owner, repo.Repo, int(id), &githubv3.IssueRequest{
				Body: cr.Body,
			})
			if err != nil {
				return issues.Comment{}, err
			}

			var edited *issues.Edited
			/* TODO: Get the actual edited information once GitHub API allows it. Can't use issue.UpdatedAt because of false positives, since it includes the entire issue, not just its comment body.
			if !issue.UpdatedAt.Equal(*issue.CreatedAt) {
				edited = &issues.Edited{
					By: users.User{Login: "Someone"}, //ghV3User(*issue.Actor), // TODO: Get the actual actor once GitHub API allows it.
					At: *issue.UpdatedAt,
				}
			}*/
			// TODO: Consider setting reactions? But it's semi-expensive (to fetch all user details) and not used by app...
			comment.ID = issueDescriptionCommentID
			comment.User = ghV3User(*issue.User)
			comment.CreatedAt = *issue.CreatedAt
			comment.Edited = edited
			comment.Body = *issue.Body
			comment.Editable = true // You can always edit comments you've edited.
		}
		if cr.Reaction != nil {
			reactionContent, err := externalizeReaction(*cr.Reaction)
			if err != nil {
				return issues.Comment{}, err
			}
			// See if user has already reacted with that reaction.
			// If not, add it. Otherwise, remove it.
			var q struct {
				Repository struct {
					Issue struct {
						ID        githubv4.ID
						Reactions struct {
							ViewerHasReacted githubv4.Boolean
						} `graphql:"reactions(content:$reactionContent)"`
					} `graphql:"issue(number:$issueNumber)"`
				} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
				Viewer githubV4User
			}
			variables := map[string]interface{}{
				"repositoryOwner": githubv4.String(repo.Owner),
				"repositoryName":  githubv4.String(repo.Repo),
				"issueNumber":     githubv4.Int(id),
				"reactionContent": reactionContent,
			}
			err = s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return issues.Comment{}, err
			}

			var rgs reactionGroups
			if !q.Repository.Issue.Reactions.ViewerHasReacted {
				// Add reaction.
				var m struct {
					AddReaction struct {
						Subject struct {
							ReactionGroups reactionGroups
						}
					} `graphql:"addReaction(input:$input)"`
				}
				input := githubv4.AddReactionInput{
					SubjectID: q.Repository.Issue.ID,
					Content:   reactionContent,
				}
				err := s.clV4.Mutate(ctx, &m, input, nil)
				if err != nil {
					return issues.Comment{}, err
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
					SubjectID: q.Repository.Issue.ID,
					Content:   reactionContent,
				}
				err := s.clV4.Mutate(ctx, &m, input, nil)
				if err != nil {
					return issues.Comment{}, err
				}
				rgs = m.RemoveReaction.Subject.ReactionGroups
			}
			// TODO: Consider setting other fields? But it's semi-expensive (another API call) and not used by app...
			//       Actually, now that using GraphQL, no longer that expensive (can be same API call).
			comment.Reactions = ghReactions(rgs, ghUser(&q.Viewer))
		}

		return comment, nil
	}

	var comment issues.Comment

	// Apply edits.
	if cr.Body != nil {
		// GitHub API uses comment ID and doesn't need issue ID. Comment IDs are unique per repo (rather than per issue).
		ghComment, _, err := s.clV3.Issues.EditComment(ctx, repo.Owner, repo.Repo, int64(cr.ID), &githubv3.IssueComment{
			Body: cr.Body,
		})
		if err != nil {
			return issues.Comment{}, err
		}

		var edited *issues.Edited
		if !ghComment.UpdatedAt.Equal(*ghComment.CreatedAt) {
			edited = &issues.Edited{
				By: users.User{Login: "Someone"}, //ghV3User(*ghComment.Actor), // TODO: Get the actual actor once GitHub API allows it.
				At: *ghComment.UpdatedAt,
			}
		}
		// TODO: Consider setting reactions? But it's semi-expensive (to fetch all user details) and not used by app...
		comment.ID = uint64(*ghComment.ID)
		comment.User = ghV3User(*ghComment.User)
		comment.CreatedAt = *ghComment.CreatedAt
		comment.Edited = edited
		comment.Body = *ghComment.Body
		comment.Editable = true // You can always edit comments you've edited.
	}
	if cr.Reaction != nil {
		reactionContent, err := externalizeReaction(*cr.Reaction)
		if err != nil {
			return issues.Comment{}, err
		}
		commentID := githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("012:IssueComment%d", cr.ID)))) // HACK, TODO: Confirm StdEncoding vs URLEncoding.
		// See if user has already reacted with that reaction.
		// If not, add it. Otherwise, remove it.
		var q struct {
			Node struct {
				IssueComment struct {
					Reactions struct {
						ViewerHasReacted githubv4.Boolean
					} `graphql:"reactions(content:$reactionContent)"`
				} `graphql:"...on IssueComment"`
			} `graphql:"node(id:$commentID)"`
			Viewer githubV4User
		}
		variables := map[string]interface{}{
			"commentID":       commentID,
			"reactionContent": reactionContent,
		}
		err = s.clV4.Query(ctx, &q, variables)
		if err != nil {
			return issues.Comment{}, err
		}

		var rgs reactionGroups
		if !q.Node.IssueComment.Reactions.ViewerHasReacted {
			// Add reaction.
			var m struct {
				AddReaction struct {
					Subject struct {
						ReactionGroups reactionGroups
					}
				} `graphql:"addReaction(input:$input)"`
			}
			input := githubv4.AddReactionInput{
				SubjectID: commentID,
				Content:   reactionContent,
			}
			err := s.clV4.Mutate(ctx, &m, input, nil)
			if err != nil {
				return issues.Comment{}, err
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
				SubjectID: commentID,
				Content:   reactionContent,
			}
			err := s.clV4.Mutate(ctx, &m, input, nil)
			if err != nil {
				return issues.Comment{}, err
			}
			rgs = m.RemoveReaction.Subject.ReactionGroups
		}
		// TODO: Consider setting other fields? But it's semi-expensive (another API call) and not used by app...
		//       Actually, now that using GraphQL, no longer that expensive (can be same API call).
		comment.Reactions = ghReactions(rgs, ghUser(&q.Viewer))
	}

	return comment, nil
}

type repoSpec struct {
	Owner string
	Repo  string
}

func ghRepoSpec(repo issues.RepoSpec) (repoSpec, error) {
	// The "github.com/" prefix is expected to be included.
	ghOwnerRepo := strings.Split(repo.URI, "/")
	if len(ghOwnerRepo) != 3 || ghOwnerRepo[0] != "github.com" || ghOwnerRepo[1] == "" || ghOwnerRepo[2] == "" {
		return repoSpec{}, fmt.Errorf(`RepoSpec is not of form "github.com/owner/repo": %q`, repo.URI)
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

func ghV3User(user githubv3.User) users.User {
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

// ghIssueState converts a GitHub IssueState to issues.State.
func ghIssueState(state githubv4.IssueState) issues.State {
	switch state {
	case githubv4.IssueStateOpen:
		return issues.OpenState
	case githubv4.IssueStateClosed:
		return issues.ClosedState
	default:
		panic("unreachable")
	}
}

// ghPRState converts a GitHub PullRequestState to state.Change.
func ghPRState(prState githubv4.PullRequestState) state.Change {
	switch prState {
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

func ghEventType(typename string) issues.EventType {
	switch typename {
	case "ReopenedEvent": // TODO: Use githubv4.IssueTimelineItemReopenedEvent or so.
		return issues.Reopened
	case "ClosedEvent": // TODO: Use githubv4.IssueTimelineItemClosedEvent or so.
		return issues.Closed
	case "RenamedTitleEvent":
		return issues.Renamed
	case "LabeledEvent":
		return issues.Labeled
	case "UnlabeledEvent":
		return issues.Unlabeled
	case "CommentDeletedEvent":
		return issues.CommentDeleted
	default:
		return issues.EventType(typename)
	}
}

// ghColor converts a GitHub color hex string like "ff0000"
// into an issues.RGB value.
func ghColor(hex string) issues.RGB {
	var c issues.RGB
	fmt.Sscanf(hex, "%02x%02x%02x", &c.R, &c.G, &c.B)
	return c
}
