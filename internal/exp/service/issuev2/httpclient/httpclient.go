// Package httpclient contains issuev2.Service implementation over HTTP.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/shurcooL/home/internal/exp/service/issuev2"
	"github.com/shurcooL/home/internal/exp/service/issuev2/httproute"
	"golang.org/x/net/context/ctxhttp"
)

// NewIssueV2 creates a client that implements issuev2.Service remotely over HTTP.
// If a nil httpClient is provided, http.DefaultClient will be used.
// scheme and host can be empty strings to target local service.
// A trailing "/" is added to path if there isn't one.
func NewIssueV2(httpClient *http.Client, scheme, host, path string) issuev2.Service {
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

// issueClient implements issuev2.Service remotely over HTTP.
type issueClient struct {
	client  *http.Client // HTTP client for API requests. If nil, http.DefaultClient should be used.
	baseURL *url.URL     // Base URL for API requests. Path must have a trailing "/".

	issuev2.Service // TODO: remove
}

func (n *issueClient) CreateIssue(ctx context.Context, r issuev2.CreateIssueRequest) (issuev2.Issue, error) {
	u := url.URL{
		Path: httproute.CreateIssue,
		RawQuery: url.Values{ // TODO: Automate this conversion process.
			"ImportPath": {r.ImportPath},
			"Title":      {r.Title},
			"Body":       {r.Body},
		}.Encode(),
	}
	resp, err := ctxhttp.Post(ctx, n.client, n.baseURL.ResolveReference(&u).String(), "", nil)
	if err != nil {
		return issuev2.Issue{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return issuev2.Issue{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var i issuev2.Issue
	err = json.NewDecoder(resp.Body).Decode(&i)
	return i, err
}
