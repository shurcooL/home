package github

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"dmitri.shuralyov.com/state"
	githubv3 "github.com/google/go-github/github"
	"github.com/google/go-querystring/query"
	"github.com/shurcooL/githubv4"
)

func parseIssueSpec(issueAPIURL string) (_ repoSpec, issueID uint64, _ error) {
	rs, id, err := parseSpec(issueAPIURL, "issues")
	if err != nil {
		return repoSpec{}, 0, err
	}
	issueID, err = strconv.ParseUint(id, 10, 64)
	if err != nil {
		return repoSpec{}, 0, err
	}
	return rs, issueID, nil
}

func parsePullRequestSpec(prAPIURL string) (_ repoSpec, prID uint64, _ error) {
	rs, id, err := parseSpec(prAPIURL, "pulls")
	if err != nil {
		return repoSpec{}, 0, err
	}
	prID, err = strconv.ParseUint(id, 10, 64)
	if err != nil {
		return repoSpec{}, 0, err
	}
	return rs, prID, nil
}

func parseSpec(apiURL, specType string) (_ repoSpec, id string, _ error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return repoSpec{}, "", err
	}
	e := strings.Split(u.Path, "/")
	if len(e) < 5 {
		return repoSpec{}, "", fmt.Errorf("unexpected path (too few elements): %q", u.Path)
	}
	if got, want := e[len(e)-2], specType; got != want {
		return repoSpec{}, "", fmt.Errorf("unexpected path element %q, want %q", got, want)
	}
	return repoSpec{Owner: e[len(e)-4], Repo: e[len(e)-3]}, e[len(e)-1], nil
}

type repoSpec struct {
	Owner string
	Repo  string
}

func ghRepoSpec(namespace string) (repoSpec, error) {
	// The "github.com/" prefix is expected to be included.
	ghOwnerRepo := strings.Split(namespace, "/")
	if len(ghOwnerRepo) != 3 || ghOwnerRepo[0] != "github.com" || ghOwnerRepo[1] == "" || ghOwnerRepo[2] == "" {
		return repoSpec{}, fmt.Errorf(`namespace is not of form "github.com/{owner}/{repo}": %q`, namespace)
	}
	return repoSpec{
		Owner: ghOwnerRepo[1],
		Repo:  ghOwnerRepo[2],
	}, nil
}

// ghIssueState converts a GitHub IssueState to state.Issue.
func ghIssueState(is githubv4.IssueState) state.Issue {
	switch is {
	case githubv4.IssueStateOpen:
		return state.IssueOpen
	case githubv4.IssueStateClosed:
		return state.IssueClosed
	default:
		panic("unreachable")
	}
}

// ghChangeState converts a GitHub PullRequestState to state.Change.
func ghChangeState(cs githubv4.PullRequestState) state.Change {
	switch cs {
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

// ghReviewState converts a GitHub PullRequestReviewState to state.Review, if it's supported.
func ghReviewState(st githubv4.PullRequestReviewState, aa githubv4.CommentAuthorAssociation) (_ state.Review, ok bool) {
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

func ghListNotificationsAllPages(ctx context.Context, cl *githubv3.Client, opt *githubv3.NotificationListOptions, cache bool) ([]*githubv3.Notification, *githubv3.Response, error) {
	var nss []*githubv3.Notification
	var resp *githubv3.Response
	for {
		var ns []*githubv3.Notification
		var err error
		ns, resp, err = ghListNotifications(ctx, cl, opt, cache)
		if err != nil {
			return nil, resp, err
		}
		nss = append(nss, ns...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nss, resp, nil
}

// ghListNotifications is like githubv3.Client.Activity.ListNotifications,
// but gives caller control over whether cache can be used.
func ghListNotifications(ctx context.Context, cl *githubv3.Client, opt *githubv3.NotificationListOptions, cache bool) ([]*githubv3.Notification, *githubv3.Response, error) {
	u := fmt.Sprintf("notifications")
	u, err := ghAddOptions(u, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := cl.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	if !cache {
		req.Header.Set("Cache-Control", "no-cache")
	}

	var notifications []*githubv3.Notification
	resp, err := cl.Do(ctx, req, &notifications)
	return notifications, resp, err
}

// ghListRepositoryNotifications is like githubv3.Client.Activity.ListRepositoryNotifications,
// but gives caller control over whether cache can be used.
func ghListRepositoryNotifications(ctx context.Context, cl *githubv3.Client, owner, repo string, opt *githubv3.NotificationListOptions, cache bool) ([]*githubv3.Notification, *githubv3.Response, error) {
	u := fmt.Sprintf("repos/%v/%v/notifications", owner, repo)
	u, err := ghAddOptions(u, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := cl.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	if !cache {
		req.Header.Set("Cache-Control", "no-cache")
	}

	var notifications []*githubv3.Notification
	resp, err := cl.Do(ctx, req, &notifications)
	return notifications, resp, err
}

// ghAddOptions adds the parameters in opt as URL query parameters to s.
// opt must be a struct (or a pointer to one) whose fields may contain "url" tags.
func ghAddOptions(s string, opt interface{}) (string, error) {
	if v := reflect.ValueOf(opt); v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}
	u.RawQuery = qs.Encode()
	return u.String(), nil
}

func getPollInterval(resp *githubv3.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	pi, err := strconv.Atoi(resp.Header.Get("X-Poll-Interval"))
	return time.Duration(pi) * time.Second, err == nil
}
