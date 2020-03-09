// Package httpclient contains change.Service implementation over HTTP.
package httpclient

import (
	"context"
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
