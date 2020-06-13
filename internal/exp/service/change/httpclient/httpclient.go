// Package httpclient contains change.Service implementation over HTTP.
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

	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/home/internal/exp/service/change/httproute"
	"golang.org/x/net/context/ctxhttp"
)

func init() {
	// For Change.ListTimeline.
	gob.Register(change.Comment{})
	gob.Register(change.Review{})
	gob.Register(change.TimelineItem{})

	// For change.TimelineItem.Payload.
	gob.Register(change.ClosedEvent{})
	gob.Register(change.ReopenedEvent{})
	gob.Register(change.RenamedEvent{})
	gob.Register(change.CommitEvent{})
	gob.Register(change.LabeledEvent{})
	gob.Register(change.UnlabeledEvent{})
	gob.Register(change.ReviewRequestedEvent{})
	gob.Register(change.ReviewRequestRemovedEvent{})
	gob.Register(change.MergedEvent{})
	gob.Register(change.DeletedEvent{})
}

// NewChange creates a client that implements change.Service remotely over HTTP.
// If a nil httpClient is provided, http.DefaultClient will be used.
// scheme and host can be empty strings to target local service.
// A trailing "/" is added to path if there isn't one.
func NewChange(httpClient *http.Client, scheme, host, path string) change.Service {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return &changeClient{
		client: httpClient,
		baseURL: &url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   path,
		},
	}
}

// changeClient implements change.Service remotely over HTTP.
type changeClient struct {
	client  *http.Client // HTTP client for API requests. If nil, http.DefaultClient should be used.
	baseURL *url.URL     // Base URL for API requests. Path must have a trailing "/".

	change.Service // For the rest of the methods that are not implemented.
}

func (cc *changeClient) List(ctx context.Context, repo string, opt change.ListOptions) ([]change.Change, error) {
	u := url.URL{
		Path: httproute.List,
		RawQuery: url.Values{
			"Repo":      {repo},
			"OptFilter": {string(opt.Filter)},
		}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, cc.client, cc.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var cs []change.Change
	err = json.NewDecoder(resp.Body).Decode(&cs)
	return cs, err
}

func (cc *changeClient) Count(ctx context.Context, repo string, opt change.ListOptions) (uint64, error) {
	u := url.URL{
		Path: httproute.Count,
		RawQuery: url.Values{
			"Repo":      {repo},
			"OptFilter": {string(opt.Filter)},
		}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, cc.client, cc.baseURL.ResolveReference(&u).String())
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

func (cc *changeClient) Get(ctx context.Context, repo string, id uint64) (change.Change, error) {
	u := url.URL{
		Path: httproute.Get,
		RawQuery: url.Values{
			"Repo": {repo},
			"ID":   {fmt.Sprint(id)},
		}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, cc.client, cc.baseURL.ResolveReference(&u).String())
	if err != nil {
		return change.Change{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return change.Change{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var c change.Change
	err = json.NewDecoder(resp.Body).Decode(&c)
	return c, err
}

func (cc *changeClient) ListTimeline(ctx context.Context, repo string, id uint64, opt *change.ListTimelineOptions) ([]interface{}, error) {
	q := url.Values{
		"Repo": {repo},
		"ID":   {fmt.Sprint(id)},
	}
	if opt != nil {
		q.Set("Opt.Start", fmt.Sprint(opt.Start))
		q.Set("Opt.Length", fmt.Sprint(opt.Length))
	}
	u := url.URL{
		Path:     httproute.ListTimeline,
		RawQuery: q.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, cc.client, cc.baseURL.ResolveReference(&u).String())
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

func (cc *changeClient) ListCommits(ctx context.Context, repo string, id uint64) ([]change.Commit, error) {
	u := url.URL{
		Path: httproute.ListCommits,
		RawQuery: url.Values{
			"Repo": {repo},
			"ID":   {fmt.Sprint(id)},
		}.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, cc.client, cc.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var cs []change.Commit
	err = json.NewDecoder(resp.Body).Decode(&cs)
	return cs, err
}

func (cc *changeClient) GetDiff(ctx context.Context, repo string, id uint64, opt *change.GetDiffOptions) ([]byte, error) {
	q := url.Values{
		"Repo": {repo},
		"ID":   {fmt.Sprint(id)},
	}
	if opt != nil {
		q.Set("Opt.Commit", opt.Commit)
	}
	u := url.URL{
		Path:     httproute.GetDiff,
		RawQuery: q.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, cc.client, cc.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	return ioutil.ReadAll(resp.Body)
}

func (cc *changeClient) EditComment(ctx context.Context, repo string, id uint64, cr change.CommentRequest) (change.Comment, error) {
	u := url.URL{
		Path: httproute.EditComment,
		RawQuery: url.Values{
			"Repo": {repo},
			"ID":   {fmt.Sprint(id)},
		}.Encode(),
	}
	data := url.Values{ // TODO: Automate this conversion process.
		"ID": {cr.ID},
	}
	if cr.Reaction != nil {
		data.Set("Reaction", string(*cr.Reaction))
	}
	resp, err := ctxhttp.PostForm(ctx, cc.client, cc.baseURL.ResolveReference(&u).String(), data)
	if err != nil {
		return change.Comment{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return change.Comment{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var comment change.Comment
	err = json.NewDecoder(resp.Body).Decode(&comment)
	return comment, err
}
