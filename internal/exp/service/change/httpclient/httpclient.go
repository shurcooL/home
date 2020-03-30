// Package httpclient contains change.Service implementation over HTTP.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/home/internal/exp/service/change/httproute"
	"golang.org/x/net/context/ctxhttp"
)

// NewChange creates a client that implements change.Service remotely over HTTP.
// If a nil httpClient is provided, http.DefaultClient will be used.
// scheme and host can be empty strings to target local service.
func NewChange(httpClient *http.Client, scheme, host string) change.Service {
	return &Change{
		client: httpClient,
		baseURL: &url.URL{
			Scheme: scheme,
			Host:   host,
		},
	}
}

// Change implements change.Service remotely over HTTP.
// Use NewChange for creation; zero value of Change is unfit for use.
type Change struct {
	client  *http.Client // HTTP client for API requests. If nil, http.DefaultClient should be used.
	baseURL *url.URL     // Base URL for API requests.

	change.Service // For the rest of the methods that are not implemented.
}

func (c *Change) EditComment(ctx context.Context, repo string, id uint64, cr change.CommentRequest) (change.Comment, error) {
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
	resp, err := ctxhttp.PostForm(ctx, c.client, c.baseURL.ResolveReference(&u).String(), data)
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
