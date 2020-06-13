// Package httpclient contains issues.Service implementation over HTTP.
package httpclient

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/issue/httproute"
	"golang.org/x/net/context/ctxhttp"
)

func init() {
	// For Issues.ListTimeline.
	gob.Register(issues.Comment{})
	gob.Register(issues.Event{})

	// For issues.Close.Closer.
	gob.Register(issues.Change{})
	gob.Register(issues.Commit{})
}

// NewIssues creates a client that implements issues.Service remotely over HTTP.
// If a nil httpClient is provided, http.DefaultClient will be used.
// scheme and host can be empty strings to target local service.
// A trailing "/" is added to path if there isn't one.
func NewIssues(httpClient *http.Client, scheme, host, path string) issues.Service {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return &issueClient{
		client: httpClient,
		baseURL: &url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   path,
		},
	}
}

// issueClient implements issues.Service remotely over HTTP.
type issueClient struct {
	client  *http.Client // HTTP client for API requests. If nil, http.DefaultClient should be used.
	baseURL *url.URL     // Base URL for API requests. Path must have a trailing "/".
}

func (ic *issueClient) List(ctx context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) ([]issues.Issue, error) {
	u := url.URL{
		Path: httproute.List,
		RawQuery: url.Values{
			"RepoURI":  {repo.URI},
			"OptState": {string(opt.State)},
		}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, ic.client, ic.baseURL.ResolveReference(&u).String())
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

func (ic *issueClient) Count(ctx context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) (uint64, error) {
	u := url.URL{
		Path: httproute.Count,
		RawQuery: url.Values{
			"RepoURI":  {repo.URI},
			"OptState": {string(opt.State)},
		}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, ic.client, ic.baseURL.ResolveReference(&u).String())
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

func (ic *issueClient) Get(ctx context.Context, repo issues.RepoSpec, id uint64) (issues.Issue, error) {
	q := url.Values{
		"RepoURI": {repo.URI},
		"ID":      {fmt.Sprint(id)},
	}
	u := url.URL{
		Path:     httproute.Get,
		RawQuery: q.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, ic.client, ic.baseURL.ResolveReference(&u).String())
	if err != nil {
		return issues.Issue{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return issues.Issue{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var i issues.Issue
	err = json.NewDecoder(resp.Body).Decode(&i)
	return i, err
}

func (ic *issueClient) ListTimeline(ctx context.Context, repo issues.RepoSpec, id uint64, opt *issues.ListOptions) ([]interface{}, error) {
	q := url.Values{
		"RepoURI": {repo.URI},
		"ID":      {fmt.Sprint(id)},
	}
	if opt != nil {
		q.Set("Opt.Start", fmt.Sprint(opt.Start))
		q.Set("Opt.Length", fmt.Sprint(opt.Length))
	}
	u := url.URL{
		Path:     httproute.ListTimeline,
		RawQuery: q.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, ic.client, ic.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var tis []interface{}
	err = gob.NewDecoder(resp.Body).Decode(&tis)
	return tis, err
}

func (ic *issueClient) Create(ctx context.Context, repo issues.RepoSpec, issue issues.Issue) (issues.Issue, error) {
	u := url.URL{
		Path: httproute.Create,
		RawQuery: url.Values{ // TODO: Automate this conversion process.
			"RepoURI": {repo.URI},
			"Title":   {issue.Title},
			"Body":    {issue.Body},
		}.Encode(),
	}
	resp, err := ctxhttp.Post(ctx, ic.client, ic.baseURL.ResolveReference(&u).String(), "", nil)
	if err != nil {
		return issues.Issue{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return issues.Issue{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var i issues.Issue
	err = json.NewDecoder(resp.Body).Decode(&i)
	return i, err
}

func (ic *issueClient) CreateComment(ctx context.Context, repo issues.RepoSpec, id uint64, comment issues.Comment) (issues.Comment, error) {
	u := url.URL{
		Path: httproute.CreateComment,
		RawQuery: url.Values{
			"RepoURI": {repo.URI},
			"ID":      {fmt.Sprint(id)},
		}.Encode(),
	}
	data := url.Values{ // TODO: Automate this conversion process.
		"Body": {comment.Body},
	}
	resp, err := ctxhttp.PostForm(ctx, ic.client, ic.baseURL.ResolveReference(&u).String(), data)
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

func (ic *issueClient) Edit(ctx context.Context, repo issues.RepoSpec, id uint64, ir issues.IssueRequest) (issues.Issue, []issues.Event, error) {
	u := url.URL{
		Path: httproute.Edit,
		RawQuery: url.Values{
			"RepoURI": {repo.URI},
			"ID":      {fmt.Sprint(id)},
		}.Encode(),
	}
	data := url.Values{} // TODO: Automate this conversion process.
	if ir.State != nil {
		data.Set("State", string(*ir.State))
	}
	if ir.Title != nil {
		data.Set("Title", *ir.Title)
	}
	resp, err := ctxhttp.PostForm(ctx, ic.client, ic.baseURL.ResolveReference(&u).String(), data)
	if err != nil {
		return issues.Issue{}, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return issues.Issue{}, nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	dec := json.NewDecoder(resp.Body)
	var i issues.Issue
	err = dec.Decode(&i)
	if err != nil {
		return issues.Issue{}, nil, err
	}
	var es []issues.Event
	err = dec.Decode(&es)
	if err != nil {
		return issues.Issue{}, nil, err
	}
	return i, es, nil
}

func (ic *issueClient) EditComment(ctx context.Context, repo issues.RepoSpec, id uint64, cr issues.CommentRequest) (issues.Comment, error) {
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
	resp, err := ctxhttp.PostForm(ctx, ic.client, ic.baseURL.ResolveReference(&u).String(), data)
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

func (cc *issueClient) ThreadType(ctx context.Context, repo issues.RepoSpec) (string, error) {
	u := url.URL{
		Path:     httproute.ThreadType,
		RawQuery: url.Values{"Repo": {repo.URI}}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, cc.client, cc.baseURL.ResolveReference(&u).String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var tt string
	err = json.NewDecoder(resp.Body).Decode(&tt)
	return tt, err
}
