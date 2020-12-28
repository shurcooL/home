package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"dmitri.shuralyov.com/go/prefixtitle"
	"dmitri.shuralyov.com/route/github"
	"dmitri.shuralyov.com/state"
	githubv3 "github.com/google/go-github/github"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/users"
	"golang.org/x/mod/modfile"
)

func (s *Service) pollList() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("internal panic: %v\n\n%s", e, debug.Stack())
		}
	}()

	for {
		s.list.mu.Lock()
		repos := make(map[int64]repository, len(s.list.repos))
		for id, r := range s.list.repos {
			repos[id] = r
		}
		commits := make(map[string]event.Commit, len(s.list.commits))
		for sha, c := range s.list.commits {
			commits[sha] = c
		}
		s.list.mu.Unlock()
		events, repos, commits, prs, eventIDs, pollInterval, fetchError := s.fetchEvents(context.Background(), repos, commits)
		if fetchError != nil {
			log.Println("fetchEvents:", fetchError)
		}
		s.list.mu.Lock()
		if fetchError == nil {
			s.list.events, s.list.repos, s.list.commits, s.list.prs, s.list.eventIDs = events, repos, commits, prs, eventIDs
		}
		s.list.fetchError = fetchError
		s.list.mu.Unlock()

		if pollInterval < time.Minute {
			pollInterval = time.Minute
		}
		time.Sleep(pollInterval)
	}
}

// fetchEvents fetches events, repository module paths, mentioned commits and PRs from GitHub.
// Provided repos and commits must be non-nil, and they're used as a starting point.
// Only missing repos and commits are fetched, and unused ones are removed at the end.
func (s *Service) fetchEvents(
	ctx context.Context,
	repos map[int64]repository, // Repo ID -> Module Path.
	commits map[string]event.Commit, // SHA -> Commit.
) (
	events []*githubv3.Event,
	_ map[int64]repository, // repos.
	_ map[string]event.Commit, // commits.
	prs map[string]bool, // PR API URL -> Pull Request merged.
	eventIDs map[*githubv3.Event]uint64, // Event -> event ID.
	pollInterval time.Duration,
	err error,
) {
	// TODO: Investigate this:
	//       Events support pagination, however the per_page option is unsupported. The fixed page size is 30 items. Fetching up to ten pages is supported, for a total of 300 events.
	events, resp, err := s.clV3.Activity.ListEventsPerformedByUser(ctx, s.user.Login, true, &githubv3.ListOptions{PerPage: 100})
	if err != nil {
		return nil, nil, nil, nil, nil, 0, err
	}
	if pi, err := strconv.Atoi(resp.Header.Get("X-Poll-Interval")); err == nil {
		pollInterval = time.Duration(pi) * time.Second
	}

	// Iterate over all events and fetch additional information
	// needed based on their contents.
	prs = make(map[string]bool)
	eventIDs = make(map[*githubv3.Event]uint64)
	usedRepos := make(map[int64]bool)    // A set of used repo IDs.
	usedCommits := make(map[string]bool) // A set of used commit SHAs.
	for _, e := range events {
		payload, err := e.ParsePayload()
		if err != nil {
			return nil, nil, nil, nil, nil, 0, fmt.Errorf("fetchEvents: ParsePayload failed: %v", err)
		}

		// Fetch the module path for this repository if not already known.
		usedRepos[*e.Repo.ID] = true
		if _, ok := repos[*e.Repo.ID]; !ok {
			modulePath, err := s.fetchModulePath(ctx, *e.Repo.ID, "github.com/"+*e.Repo.Name)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchModulePath: repository id=%d name=%q was not found: %v\n", *e.Repo.ID, *e.Repo.Name, err)
				modulePath = "github.com/" + *e.Repo.Name
			} else if err != nil {
				return nil, nil, nil, nil, nil, 0, fmt.Errorf("fetchModulePath: %v", err)
			}
			repos[*e.Repo.ID] = repository{ModulePath: modulePath}
		}

		// Fetch the mentioned commits and PRs that aren't already known.
		switch p := payload.(type) {
		case *githubv3.PushEvent:
			for _, c := range p.Commits {
				usedCommits[*c.SHA] = true
				if _, ok := commits[*c.SHA]; ok {
					continue
				}
				commit, err := s.fetchCommit(ctx, *e.Repo.ID, *c.SHA)
				if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
					log.Printf("fetchEvents: commit %s@%s was not found: %v\n", *e.Repo.Name, *c.SHA, err)

					avatarURL := "https://secure.gravatar.com/avatar?d=mm&f=y&s=96"
					if *c.Author.Email == s.user.Email {
						avatarURL = s.user.AvatarURL
					}
					commit = event.Commit{
						SHA:             *c.SHA,
						Message:         *c.Message,
						AuthorAvatarURL: avatarURL,
					}
				} else if err != nil {
					return nil, nil, nil, nil, nil, 0, fmt.Errorf("fetchCommit: %v", err)
				}
				commits[*c.SHA] = commit
			}
		case *githubv3.CommitCommentEvent:
			usedCommits[*p.Comment.CommitID] = true
			if _, ok := commits[*p.Comment.CommitID]; ok {
				continue
			}
			commit, err := s.fetchCommit(ctx, *e.Repo.ID, *p.Comment.CommitID)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchEvents: commit %s@%s was not found: %v\n", *e.Repo.Name, *p.Comment.CommitID, err)

				commit = event.Commit{
					SHA:             *p.Comment.CommitID,
					AuthorAvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96",
				}
			} else if err != nil {
				return nil, nil, nil, nil, nil, 0, fmt.Errorf("fetchCommit: %v", err)
			}
			commits[*p.Comment.CommitID] = commit

		case *githubv3.IssueCommentEvent:
			if p.Issue.PullRequestLinks == nil {
				continue
			}
			if _, ok := prs[*p.Issue.PullRequestLinks.URL]; ok {
				continue
			}
			merged, err := s.fetchPullRequestMerged(ctx, *p.Issue.PullRequestLinks.URL)
			if err != nil {
				return nil, nil, nil, nil, nil, 0, fmt.Errorf("fetchPullRequestMerged: %v", err)
			}
			prs[*p.Issue.PullRequestLinks.URL] = merged

		case *githubv3.IssuesEvent:
			if *p.Action == "closed" || *p.Action == "reopened" {
				id, err := s.fetchEventID(ctx, e)
				if err != nil {
					log.Printf("fetchEvents: fetchEventID: %v\n", err)
				} else {
					eventIDs[e] = id
				}
			}
		case *githubv3.PullRequestEvent:
			if *p.Action == "closed" || *p.Action == "reopened" {
				id, err := s.fetchEventID(ctx, e)
				if err != nil {
					log.Printf("fetchEvents: fetchEventID: %v\n", err)
				} else {
					eventIDs[e] = id
				}
			}
		}
	}

	// Remove unused repos and commits.
	for id := range repos {
		if !usedRepos[id] {
			delete(repos, id)
		}
	}
	for sha := range commits {
		if !usedCommits[sha] {
			delete(commits, sha)
		}
	}

	return events, repos, commits, prs, eventIDs, pollInterval, nil
}

// goRepoID is the repository ID of the github.com/golang/go repository.
const goRepoID = 23096959

// fetchModulePath fetches the module path for the specified repository.
// repoPath is returned as the module path if the repository has no go.mod file,
// or if the go.mod file fails to parse.
//
// For the main Go repository (i.e., https://github.com/golang/go),
// the empty string is returned as the module path without using network.
func (s *Service) fetchModulePath(ctx context.Context, repoID int64, repoPath string) (modulePath string, _ error) {
	if repoID == goRepoID {
		// Use empty string as the module path for the main Go repository.
		return "", nil
	}

	// TODO: It'd be better to batch and fetch all module paths at once (in fetchEvents loop),
	//       rather than making an individual query for each.
	//       See https://github.com/shurcooL/githubv4/issues/17.

	var q struct {
		Node struct {
			Repository struct {
				Object *struct {
					Blob struct {
						Text string
					} `graphql:"...on Blob"`
				} `graphql:"object(expression:\"HEAD:go.mod\")"`
			} `graphql:"...on Repository"`
		} `graphql:"node(id:$repoID)"`
	}
	variables := map[string]interface{}{
		"repoID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("010:Repository%d", repoID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
	}
	err := s.clV4.Query(ctx, &q, variables)
	if err != nil {
		return "", err
	}
	if q.Node.Repository.Object == nil {
		// No go.mod file, so the module path must be equal to the repo path.
		return repoPath, nil
	}
	modulePath = modfile.ModulePath([]byte(q.Node.Repository.Object.Blob.Text))
	if modulePath == "" {
		// No module path found in go.mod file, so fall back to using the repo path.
		return repoPath, nil
	}
	return modulePath, nil
}

// fetchCommit fetches the specified commit.
func (s *Service) fetchCommit(ctx context.Context, repoID int64, sha string) (event.Commit, error) {
	// TODO: It'd be better to batch and fetch all commits at once (in fetchEvents loop),
	//       rather than making an individual query for each.
	//       See https://github.com/shurcooL/githubv4/issues/17.

	commitID := fmt.Sprintf("06:Commit%d:%s", repoID, sha)
	var q struct {
		Node struct {
			Commit struct {
				OID     string
				Message string
				Author  struct {
					AvatarURL string `graphql:"avatarUrl(size:96)"`
				}
				URL string
			} `graphql:"...on Commit"`
		} `graphql:"node(id:$commitID)"`
	}
	variables := map[string]interface{}{
		"commitID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(commitID))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
	}
	err := s.clV4.Query(ctx, &q, variables)
	if err != nil {
		return event.Commit{}, err
	}
	return event.Commit{
		SHA:             q.Node.Commit.OID,
		Message:         q.Node.Commit.Message,
		AuthorAvatarURL: q.Node.Commit.Author.AvatarURL,
		HTMLURL:         q.Node.Commit.URL,
	}, nil
}

// fetchPullRequestMerged fetches whether the Pull Request at the API URL is merged
// at current time.
func (s *Service) fetchPullRequestMerged(ctx context.Context, prURL string) (bool, error) {
	// Using https://developer.github.com/v3/pulls/#get-if-a-pull-request-has-been-merged.
	req, err := s.clV3.NewRequest("GET", prURL+"/merge", nil)
	if err != nil {
		return false, err
	}
	resp, err := s.clV3.Do(ctx, req, nil)
	switch e, ok := err.(*githubv3.ErrorResponse); {
	case err == nil && resp.StatusCode == http.StatusNoContent:
		// PR merged.
		return true, nil
	case ok && e.Response.StatusCode == http.StatusNotFound:
		// PR not merged.
		return false, nil
	case err != nil:
		return false, err
	default:
		body, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("unexpected status code: %v body: %q", resp.Status, body)
	}
}

func (s *Service) fetchEventID(ctx context.Context, e *githubv3.Event) (uint64, error) {
	// TODO: It'd be better to batch and fetch all event IDs at once (in fetchEvents loop),
	//       rather than making an individual query for each.
	//       See https://github.com/shurcooL/githubv4/issues/17.

	near := func(a, b time.Time) bool {
		d := a.Sub(b)
		return -3*time.Second <= d && d <= 3*time.Second
	}

	payload, err := e.ParsePayload()
	if err != nil {
		return 0, err
	}
	switch p := payload.(type) {
	case *githubv3.IssuesEvent:
		switch *p.Action {
		case "closed":
			var q struct {
				Node struct {
					Issue struct {
						TimelineItems struct {
							Nodes []struct {
								//Typename string `graphql:"__typename"`
								ClosedEvent struct {
									ID        string
									CreatedAt time.Time
								} `graphql:"...on ClosedEvent"`
							}
						} `graphql:"timelineItems(last:5,itemTypes:[CLOSED_EVENT])"`
					} `graphql:"...on Issue"`
				} `graphql:"node(id:$issueID)"`
			}
			variables := map[string]interface{}{
				"issueID": githubv4.ID(p.Issue.GetNodeID()),
			}
			err := s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return 0, err
			}
			for i := len(q.Node.Issue.TimelineItems.Nodes) - 1; i >= 0; i-- {
				ce := q.Node.Issue.TimelineItems.Nodes[i].ClosedEvent
				if !near(ce.CreatedAt, *p.Issue.UpdatedAt) {
					continue
				}
				return parseClosedEventID(ce.ID)
			}
			return 0, fmt.Errorf("no matching ClosedEvent found for issue %q near %v (out of %d)", p.Issue.GetNodeID(), *p.Issue.UpdatedAt, len(q.Node.Issue.TimelineItems.Nodes))
		case "reopened":
			var q struct {
				Node struct {
					Issue struct {
						TimelineItems struct {
							Nodes []struct {
								//Typename string `graphql:"__typename"`
								ReopenedEvent struct {
									ID        string
									CreatedAt time.Time
								} `graphql:"...on ReopenedEvent"`
							}
						} `graphql:"timelineItems(last:5,itemTypes:[REOPENED_EVENT])"`
					} `graphql:"...on Issue"`
				} `graphql:"node(id:$issueID)"`
			}
			variables := map[string]interface{}{
				"issueID": githubv4.ID(p.Issue.GetNodeID()),
			}
			err := s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return 0, err
			}
			for i := len(q.Node.Issue.TimelineItems.Nodes) - 1; i >= 0; i-- {
				re := q.Node.Issue.TimelineItems.Nodes[i].ReopenedEvent
				if !near(re.CreatedAt, *p.Issue.UpdatedAt) {
					continue
				}
				return parseReopenedEventID(re.ID)
			}
			return 0, fmt.Errorf("no matching ReopenedEvent found for issue %q near %v (out of %d)", p.Issue.GetNodeID(), *p.Issue.UpdatedAt, len(q.Node.Issue.TimelineItems.Nodes))
		}

	case *githubv3.PullRequestEvent:
		switch {
		case *p.Action == "closed" && !*p.PullRequest.Merged:
			var q struct {
				Node struct {
					PullRequest struct {
						TimelineItems struct {
							Nodes []struct {
								//Typename string `graphql:"__typename"`
								ClosedEvent struct {
									ID        string
									CreatedAt time.Time
								} `graphql:"...on ClosedEvent"`
							}
						} `graphql:"timelineItems(last:5,itemTypes:[CLOSED_EVENT])"`
					} `graphql:"...on PullRequest"`
				} `graphql:"node(id:$prID)"`
			}
			variables := map[string]interface{}{
				"prID": githubv4.ID(p.PullRequest.GetNodeID()),
			}
			err := s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return 0, err
			}
			for i := len(q.Node.PullRequest.TimelineItems.Nodes) - 1; i >= 0; i-- {
				ce := q.Node.PullRequest.TimelineItems.Nodes[i].ClosedEvent
				if !near(ce.CreatedAt, *p.PullRequest.UpdatedAt) {
					continue
				}
				return parseClosedEventID(ce.ID)
			}
			return 0, fmt.Errorf("no matching ClosedEvent found for PR %q near %v (out of %d)", p.PullRequest.GetNodeID(), *p.PullRequest.UpdatedAt, len(q.Node.PullRequest.TimelineItems.Nodes))
		case *p.Action == "closed" && *p.PullRequest.Merged:
			var q struct {
				Node struct {
					PullRequest struct {
						TimelineItems struct {
							Nodes []struct {
								//Typename string `graphql:"__typename"`
								MergedEvent struct {
									ID        string
									CreatedAt time.Time
								} `graphql:"...on MergedEvent"`
							}
						} `graphql:"timelineItems(last:5,itemTypes:[MERGED_EVENT])"`
					} `graphql:"...on PullRequest"`
				} `graphql:"node(id:$prID)"`
			}
			variables := map[string]interface{}{
				"prID": githubv4.ID(p.PullRequest.GetNodeID()),
			}
			err := s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return 0, err
			}
			for i := len(q.Node.PullRequest.TimelineItems.Nodes) - 1; i >= 0; i-- {
				me := q.Node.PullRequest.TimelineItems.Nodes[i].MergedEvent
				if !near(me.CreatedAt, *p.PullRequest.UpdatedAt) {
					continue
				}
				return parseMergedEventID(me.ID)
			}
			return 0, fmt.Errorf("no matching MergedEvent found for PR %q near %v (out of %d)", p.PullRequest.GetNodeID(), *p.PullRequest.UpdatedAt, len(q.Node.PullRequest.TimelineItems.Nodes))
		case *p.Action == "reopened":
			var q struct {
				Node struct {
					PullRequest struct {
						TimelineItems struct {
							Nodes []struct {
								//Typename string `graphql:"__typename"`
								ReopenedEvent struct {
									ID        string
									CreatedAt time.Time
								} `graphql:"...on ReopenedEvent"`
							}
						} `graphql:"timelineItems(last:5,itemTypes:[REOPENED_EVENT])"`
					} `graphql:"...on PullRequest"`
				} `graphql:"node(id:$prID)"`
			}
			variables := map[string]interface{}{
				"prID": githubv4.ID(p.PullRequest.GetNodeID()),
			}
			err := s.clV4.Query(ctx, &q, variables)
			if err != nil {
				return 0, err
			}
			for i := len(q.Node.PullRequest.TimelineItems.Nodes) - 1; i >= 0; i-- {
				re := q.Node.PullRequest.TimelineItems.Nodes[i].ReopenedEvent
				if !near(re.CreatedAt, *p.PullRequest.UpdatedAt) {
					continue
				}
				return parseReopenedEventID(re.ID)
			}
			return 0, fmt.Errorf("no matching ReopenedEvent found for PR %q near %v (out of %d)", p.PullRequest.GetNodeID(), *p.PullRequest.UpdatedAt, len(q.Node.PullRequest.TimelineItems.Nodes))
		}
	}
	return 0, fmt.Errorf("unsupported event type")
}

func parseClosedEventID(nodeID string) (uint64, error) {
	b, err := base64.StdEncoding.DecodeString(nodeID)
	if err != nil {
		return 0, err
	}
	if !bytes.HasPrefix(b, []byte("011:ClosedEvent")) {
		return 0, fmt.Errorf("unexpected prefix in ClosedEvent nodeID %q", b)
	}
	return strconv.ParseUint(string(b[len("011:ClosedEvent"):]), 10, 64)
}

func parseMergedEventID(nodeID string) (uint64, error) {
	b, err := base64.StdEncoding.DecodeString(nodeID)
	if err != nil {
		return 0, err
	}
	if !bytes.HasPrefix(b, []byte("011:MergedEvent")) {
		return 0, fmt.Errorf("unexpected prefix in MergedEvent nodeID %q", b)
	}
	return strconv.ParseUint(string(b[len("011:MergedEvent"):]), 10, 64)
}

func parseReopenedEventID(nodeID string) (uint64, error) {
	b, err := base64.StdEncoding.DecodeString(nodeID)
	if err != nil {
		return 0, err
	}
	if !bytes.HasPrefix(b, []byte("013:ReopenedEvent")) {
		return 0, fmt.Errorf("unexpected prefix in ReopenedEvent nodeID %q", b)
	}
	return strconv.ParseUint(string(b[len("013:ReopenedEvent"):]), 10, 64)
}

// convert converts GitHub events. Events must contain valid payloads,
// otherwise convert panics. commits key is SHA.
func convert(
	ctx context.Context,
	events []*githubv3.Event,
	repos map[int64]repository, // Repo ID -> Module Path.
	commits map[string]event.Commit, // SHA -> Commit.
	prs map[string]bool, // PR API URL -> Pull Request merged.
	eventIDs map[*githubv3.Event]uint64, // Event -> event ID.
	router github.Router,
) (
	_ []event.Event,
	// TODO: merge ids into []event.Event slice?
	ids []githubEventID, // Corresponds 1:1 to events.
) {
	var es []event.Event
	for _, e := range events {
		ee := event.Event{
			Time: *e.CreatedAt,
			Actor: users.User{
				UserSpec:  users.UserSpec{ID: uint64(*e.Actor.ID), Domain: "github.com"},
				Login:     *e.Actor.Login,
				AvatarURL: *e.Actor.AvatarURL,
			},
		}
		var eventID githubEventID

		modulePath := repos[*e.Repo.ID].ModulePath
		owner, repo := splitOwnerRepo(*e.Repo.Name)
		payload, err := e.ParsePayload()
		if err != nil {
			panic(fmt.Errorf("internal error: convert given a githubv3.Event with an invalid payload: %v", err))
		}
		switch p := payload.(type) {
		case *githubv3.IssuesEvent:
			var body, htmlURL string
			switch *p.Action {
			case "opened":
				body = *p.Issue.Body
				htmlURL = router.IssueURL(ctx, owner, repo, uint64(*p.Issue.Number))
				eventID.Owner, eventID.Repo, eventID.IssueID = owner, repo, uint64(*p.Issue.Number)
			case "closed":
				htmlURL = router.IssueEventURL(ctx, owner, repo, uint64(*p.Issue.Number), eventIDs[e])
				eventID.ClosedEventID = eventIDs[e]
			case "reopened":
				htmlURL = router.IssueEventURL(ctx, owner, repo, uint64(*p.Issue.Number), eventIDs[e])
				eventID.ReopenedEventID = eventIDs[e]

				//default:
				//log.Println("convert: unsupported *githubv3.IssuesEvent action:", *p.Action)
			}
			paths, title := prefixtitle.ParseIssue(modulePath, *p.Issue.Title)
			ee.Container = paths[0]
			ee.Payload = event.Issue{
				Action:       *p.Action,
				IssueTitle:   title,
				IssueBody:    body,
				IssueHTMLURL: htmlURL,
			}
		case *githubv3.PullRequestEvent:
			var action, body string
			switch {
			case *p.Action == "opened":
				action = "opened"
				body = *p.PullRequest.Body
				eventID.Owner, eventID.Repo, eventID.PullRequestID = owner, repo, uint64(*p.PullRequest.Number)
			case *p.Action == "closed" && !*p.PullRequest.Merged:
				action = "closed"
				eventID.PRClosedEventID = eventIDs[e]
			case *p.Action == "closed" && *p.PullRequest.Merged:
				action = "merged"
				eventID.PRMergedEventID = eventIDs[e]
			case *p.Action == "reopened":
				action = "reopened"
				eventID.PRReopenedEventID = eventIDs[e]

				//default:
				//log.Println("convert: unsupported *githubv3.PullRequestEvent PullRequest.State:", *p.PullRequest.State, "PullRequest.Merged:", *p.PullRequest.Merged)
			}
			paths, title := prefixtitle.ParseChange(modulePath, *p.PullRequest.Title)
			ee.Container = paths[0]
			ee.Payload = event.Change{
				Action:        action,
				ChangeTitle:   title,
				ChangeBody:    body,
				ChangeHTMLURL: router.PullRequestURL(ctx, owner, repo, uint64(*p.PullRequest.Number)),
			}

		case *githubv3.IssueCommentEvent:
			switch p.Issue.PullRequestLinks {
			case nil: // Issue.
				switch *p.Action {
				case "created":
					var issueState state.Issue
					switch *p.Issue.State {
					case "open":
						issueState = state.IssueOpen
					case "closed":
						issueState = state.IssueClosed
					default:
						log.Printf("convert: unsupported *githubv3.IssueCommentEvent (issue): Issue.State=%v\n", *p.Issue.State)
						continue
					}
					paths, title := prefixtitle.ParseIssue(modulePath, *p.Issue.Title)
					ee.Container = paths[0]
					ee.Payload = event.IssueComment{
						IssueTitle:     title,
						IssueState:     issueState,
						CommentBody:    *p.Comment.Body,
						CommentHTMLURL: router.IssueCommentURL(ctx, owner, repo, uint64(*p.Issue.Number), uint64(*p.Comment.ID)),
					}
					eventID.IssueCommentID = uint64(*p.Comment.ID)

					//default:
					//e.WIP = true
					//e.Action = component.Text(fmt.Sprintf("%v on an issue in", *p.Action))
				}
			default: // Pull Request.
				switch *p.Action {
				case "created":
					var changeState state.Change
					// Note, State is PR state at the time of event, but merged is PR merged at current time.
					// So, only check merged when State is closed. It's an approximation, but good enough in majority of cases.
					switch merged := prs[*p.Issue.PullRequestLinks.URL]; {
					case *p.Issue.State == "open":
						changeState = state.ChangeOpen
					case *p.Issue.State == "closed" && !merged:
						changeState = state.ChangeClosed
					case *p.Issue.State == "closed" && merged:
						changeState = state.ChangeMerged
					default:
						log.Printf("convert: unsupported *githubv3.IssueCommentEvent (pr): merged=%v Issue.State=%v\n", prs[*p.Issue.PullRequestLinks.URL], *p.Issue.State)
						continue
					}
					paths, title := prefixtitle.ParseChange(modulePath, *p.Issue.Title)
					ee.Container = paths[0]
					ee.Payload = event.ChangeComment{
						ChangeTitle:    title,
						ChangeState:    changeState,
						CommentBody:    *p.Comment.Body,
						CommentHTMLURL: router.PullRequestCommentURL(ctx, owner, repo, uint64(*p.Issue.Number), uint64(*p.Comment.ID)),
					}
					eventID.PRCommentID = uint64(*p.Comment.ID)

					//default:
					//e.WIP = true
					//e.Action = component.Text(fmt.Sprintf("%v on a pull request in", *p.Action))
				}
			}
		case *githubv3.PullRequestReviewEvent:
			switch *p.Action {
			case "created":
				var changeState state.Change
				switch {
				case p.PullRequest.MergedAt == nil && *p.PullRequest.State == "open":
					changeState = state.ChangeOpen
				case p.PullRequest.MergedAt == nil && *p.PullRequest.State == "closed":
					changeState = state.ChangeClosed
				case p.PullRequest.MergedAt != nil:
					changeState = state.ChangeMerged
				default:
					log.Printf("convert: unsupported *githubv3.PullRequestReviewEvent: PullRequest.MergedAt=%v PullRequest.State=%v\n", p.PullRequest.MergedAt, *p.PullRequest.State)
					continue
				}
				var reviewState state.Review
				switch *p.Review.State {
				case "approved":
					reviewState = state.ReviewPlus2
				case "commented":
					reviewState = state.ReviewNoScore
				default:
					log.Printf("convert: PR review %s/%s/%d/%d had not ok ReviewState\n", owner, repo, *p.PullRequest.Number, *p.Review.ID)
					continue
				}
				if reviewState == state.ReviewNoScore && p.Review.Body == nil {
					// No score, no body. Skip this empty review.
					// (The content is likely in review comments.)
					continue
				}
				paths, title := prefixtitle.ParseChange(modulePath, *p.PullRequest.Title)
				ee.Container = paths[0]
				ee.Payload = event.ChangeComment{
					ChangeTitle:    title,
					ChangeState:    changeState,
					CommentBody:    p.Review.GetBody(),
					CommentReview:  reviewState,
					CommentHTMLURL: router.PullRequestReviewURL(ctx, owner, repo, uint64(*p.PullRequest.Number), uint64(*p.Review.ID)),
				}
				eventID.PRReviewID = uint64(*p.Review.ID)

				//default:
				//basicEvent.WIP = true
				//e.Action = component.Text(fmt.Sprintf("%v on a pull request in", *p.Action))
			}
		case *githubv3.PullRequestReviewCommentEvent:
			switch *p.Action {
			case "created":
				var changeState state.Change
				switch {
				case p.PullRequest.MergedAt == nil && *p.PullRequest.State == "open":
					changeState = state.ChangeOpen
				case p.PullRequest.MergedAt == nil && *p.PullRequest.State == "closed":
					changeState = state.ChangeClosed
				case p.PullRequest.MergedAt != nil:
					changeState = state.ChangeMerged
				default:
					log.Printf("convert: unsupported *githubv3.PullRequestReviewCommentEvent: PullRequest.MergedAt=%v PullRequest.State=%v\n", p.PullRequest.MergedAt, *p.PullRequest.State)
					continue
				}
				paths, title := prefixtitle.ParseChange(modulePath, *p.PullRequest.Title)
				ee.Container = paths[0]
				ee.Payload = event.ChangeComment{
					ChangeTitle:    title,
					ChangeState:    changeState,
					CommentBody:    *p.Comment.Body,
					CommentHTMLURL: router.PullRequestReviewCommentURL(ctx, owner, repo, uint64(*p.PullRequest.Number), uint64(*p.Comment.ID)),
				}
				eventID.PRReviewID = uint64(*p.Comment.PullRequestReviewID)

				//default:
				//basicEvent.WIP = true
				//e.Action = component.Text(fmt.Sprintf("%v on a pull request in", *p.Action))
			}

		case *githubv3.CommitCommentEvent:
			c := commits[*p.Comment.CommitID]
			subject, body := splitCommitMessage(c.Message)
			paths, title := prefixtitle.ParseChange(modulePath, subject)
			ee.Container = paths[0]
			c.Message = joinCommitMessage(title, body)
			ee.Payload = event.CommitComment{
				Commit:      c,
				CommentBody: *p.Comment.Body,
			}

		case *githubv3.PushEvent:
			var cs []event.Commit
			for _, c := range p.Commits {
				cs = append(cs, commits[*c.SHA])
			}
			ee.Container = modulePath
			ee.Payload = event.Push{
				Branch:        strings.TrimPrefix(*p.Ref, "refs/heads/"),
				Head:          *p.Head,
				Before:        *p.Before,
				Commits:       cs,
				HeadHTMLURL:   "https://github.com/" + *e.Repo.Name + "/commit/" + *p.Head,
				BeforeHTMLURL: "https://github.com/" + *e.Repo.Name + "/commit/" + *p.Before,
			}

		case *githubv3.WatchEvent:
			ee.Container = modulePath
			ee.Payload = event.Star{}

		case *githubv3.CreateEvent:
			switch *p.RefType {
			case "repository":
				ee.Container = modulePath
				ee.Payload = event.Create{
					Type:        "repository",
					Description: *p.Description,
				}
			case "branch", "tag":
				ee.Container = modulePath
				ee.Payload = event.Create{
					Type: *p.RefType,
					Name: *p.Ref,
				}

				//default:
				//basicEvent.WIP = true
				//e.Action = component.Text(fmt.Sprintf("created %v in", *p.RefType))
				//e.Details = code{
				//	Text: *p.Ref,
				//}
			}
		case *githubv3.ForkEvent:
			ee.Container = modulePath
			ee.Payload = event.Fork{
				Container: "github.com/" + *p.Forkee.FullName,
			}
		case *githubv3.DeleteEvent:
			ee.Container = modulePath
			ee.Payload = event.Delete{
				Type: *p.RefType, // TODO: Verify *p.RefType?
				Name: *p.Ref,
			}

		case *githubv3.GollumEvent:
			var pages []event.Page
			for _, p := range p.Pages {
				pages = append(pages, event.Page{
					Action:         *p.Action,
					SHA:            *p.SHA,
					Title:          *p.Title,
					HTMLURL:        *p.HTMLURL + "/" + *p.SHA,
					CompareHTMLURL: *p.HTMLURL + "/_compare/" + *p.SHA + "^..." + *p.SHA,
				})
			}
			ee.Container = modulePath
			ee.Payload = event.Wiki{
				Pages: pages,
			}

		case *githubv3.MemberEvent:
			// Unsupported event type, skip it.
			continue

		default:
			log.Printf("convert: unexpected event type: %T\n", p)
			continue
		}

		es = append(es, ee)
		ids = append(ids, eventID)
	}
	return es, ids
}

// splitOwnerRepo splits "owner/repo" into "owner" and "repo".
func splitOwnerRepo(ownerRepo string) (owner, repo string) {
	i := strings.IndexByte(ownerRepo, '/')
	return ownerRepo[:i], ownerRepo[i+1:]
}

// repository represents a GitHub repository.
type repository struct {
	// ModulePath is the module path of the module at the root of the repository.
	ModulePath string
}

// splitCommitMessage splits commit message s into subject and body, if any.
func splitCommitMessage(s string) (subject, body string) {
	i := strings.Index(s, "\n\n")
	if i == -1 {
		return strings.ReplaceAll(s, "\n", " "), ""
	}
	return strings.ReplaceAll(s[:i], "\n", " "), s[i+2:]
}

// joinCommitMessage joins commit subject and body into a commit message.
// The empty string value for body represents no body.
func joinCommitMessage(subject, body string) string {
	if body == "" {
		return subject
	}
	return subject + "\n\n" + body
}
