package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/shurcooL/issues"
)

// Issues implements issues.Service remotely over HTTP.
type Issues struct{}

func (Issues) List(ctx context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) ([]issues.Issue, error) {
	u := url.URL{
		Path: "/api/issues/list",
		RawQuery: url.Values{
			"RepoURI":  {repo.URI},
			"OptState": {string(opt.State)},
		}.Encode(),
	}
	resp, err := http.Get(u.String())
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

func (Issues) Count(_ context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) (uint64, error) {
	return 0, fmt.Errorf("Count: not implemented")
}

func (Issues) Get(_ context.Context, repo issues.RepoSpec, id uint64) (issues.Issue, error) {
	return issues.Issue{}, fmt.Errorf("Get: not implemented")
}

func (Issues) ListComments(_ context.Context, repo issues.RepoSpec, id uint64, opt interface{}) ([]issues.Comment, error) {
	u := url.URL{
		Path: "/api/issues/list-comments",
		RawQuery: url.Values{
			"RepoURI": {repo.URI},
			"ID":      {fmt.Sprint(id)},
		}.Encode(),
	}
	resp, err := http.Get(u.String())
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

func (Issues) ListEvents(_ context.Context, repo issues.RepoSpec, id uint64, opt interface{}) ([]issues.Event, error) {
	return nil, fmt.Errorf("ListEvents: not implemented")
}

func (Issues) Create(_ context.Context, repo issues.RepoSpec, issue issues.Issue) (issues.Issue, error) {
	return issues.Issue{}, fmt.Errorf("Create: not implemented")
}

func (Issues) CreateComment(_ context.Context, repo issues.RepoSpec, id uint64, comment issues.Comment) (issues.Comment, error) {
	return issues.Comment{}, fmt.Errorf("CreateComment: not implemented")
}

func (Issues) Edit(_ context.Context, repo issues.RepoSpec, id uint64, ir issues.IssueRequest) (issues.Issue, []issues.Event, error) {
	return issues.Issue{}, nil, fmt.Errorf("Edit: not implemented")
}

func (Issues) EditComment(_ context.Context, repo issues.RepoSpec, id uint64, cr issues.CommentRequest) (issues.Comment, error) {
	u := url.URL{
		Path: "/api/issues/edit-comment",
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
	resp, err := http.PostForm(u.String(), data)
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
