// Package httpclient contains issues.Service implementation over HTTP.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/issue/httproute"
	"golang.org/x/net/context/ctxhttp"
)

// NewIssues creates a client that implements issues.Service remotely over HTTP.
// If a nil httpClient is provided, http.DefaultClient will be used.
// scheme and host can be empty strings to target local service.
func NewIssues(httpClient *http.Client, scheme, host string) issues.Service {
	return &Issues{
		client: httpClient,
		baseURL: &url.URL{
			Scheme: scheme,
			Host:   host,
		},
	}
}

// Issues implements issues.Service remotely over HTTP.
// Use NewIssues for creation, zero value of Issues is unfit for use.
type Issues struct {
	client  *http.Client // HTTP client for API requests. If nil, http.DefaultClient should be used.
	baseURL *url.URL     // Base URL for API requests.
}

func (i *Issues) List(ctx context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) ([]issues.Issue, error) {
	u := url.URL{
		Path: httproute.List,
		RawQuery: url.Values{
			"RepoURI":  {repo.URI},
			"OptState": {string(opt.State)},
		}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, i.client, i.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var is []issues.Issue
	err = json.NewDecoder(resp.Body).Decode(&is)
	return is, err
}

func (i *Issues) Count(ctx context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) (uint64, error) {
	u := url.URL{
		Path: httproute.Count,
		RawQuery: url.Values{
			"RepoURI":  {repo.URI},
			"OptState": {string(opt.State)},
		}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, i.client, i.baseURL.ResolveReference(&u).String())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return 0, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var count uint64
	err = json.NewDecoder(resp.Body).Decode(&count)
	return count, err
}

func (*Issues) Get(_ context.Context, repo issues.RepoSpec, id uint64) (issues.Issue, error) {
	return issues.Issue{}, fmt.Errorf("Get: not implemented")
}

func (i *Issues) ListComments(ctx context.Context, repo issues.RepoSpec, id uint64, opt *issues.ListOptions) ([]issues.Comment, error) {
	q := url.Values{
		"RepoURI": {repo.URI},
		"ID":      {fmt.Sprint(id)},
	}
	if opt != nil {
		q.Set("Opt.Start", fmt.Sprint(opt.Start))
		q.Set("Opt.Length", fmt.Sprint(opt.Length))
	}
	u := url.URL{
		Path:     httproute.ListComments,
		RawQuery: q.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, i.client, i.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var cs []issues.Comment
	err = json.NewDecoder(resp.Body).Decode(&cs)
	return cs, err
}

func (i *Issues) ListEvents(ctx context.Context, repo issues.RepoSpec, id uint64, opt *issues.ListOptions) ([]issues.Event, error) {
	q := url.Values{
		"RepoURI": {repo.URI},
		"ID":      {fmt.Sprint(id)},
	}
	if opt != nil {
		q.Set("Opt.Start", fmt.Sprint(opt.Start))
		q.Set("Opt.Length", fmt.Sprint(opt.Length))
	}
	u := url.URL{
		Path:     httproute.ListEvents,
		RawQuery: q.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, i.client, i.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var es []issues.Event
	err = json.NewDecoder(resp.Body).Decode(&es)
	return es, err
}

func (*Issues) Create(_ context.Context, repo issues.RepoSpec, issue issues.Issue) (issues.Issue, error) {
	return issues.Issue{}, fmt.Errorf("Create: not implemented")
}

func (*Issues) CreateComment(_ context.Context, repo issues.RepoSpec, id uint64, comment issues.Comment) (issues.Comment, error) {
	return issues.Comment{}, fmt.Errorf("CreateComment: not implemented")
}

func (*Issues) Edit(_ context.Context, repo issues.RepoSpec, id uint64, ir issues.IssueRequest) (issues.Issue, []issues.Event, error) {
	return issues.Issue{}, nil, fmt.Errorf("Edit: not implemented")
}

func (i *Issues) EditComment(ctx context.Context, repo issues.RepoSpec, id uint64, cr issues.CommentRequest) (issues.Comment, error) {
	u := url.URL{
		Path: httproute.EditComment,
		RawQuery: url.Values{
			"RepoURI": {repo.URI},
			"ID":      {fmt.Sprint(id)},
		}.Encode(),
	}
	data := url.Values{ // TODO: Automate this conversion process.
		"ID": {fmt.Sprint(cr.ID)},
	}
	if cr.Body != nil {
		data.Set("Body", *cr.Body)
	}
	if cr.Reaction != nil {
		data.Set("Reaction", string(*cr.Reaction))
	}
	resp, err := ctxhttp.PostForm(ctx, i.client, i.baseURL.ResolveReference(&u).String(), data)
	if err != nil {
		return issues.Comment{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return issues.Comment{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var c issues.Comment
	err = json.NewDecoder(resp.Body).Decode(&c)
	return c, err
}
