// Package httpclient contains issues.Service implementation over HTTP.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/code/httproute"
	"golang.org/x/net/context/ctxhttp"
)

// NewCode creates a client that implements code.Service remotely over HTTP.
// If a nil httpClient is provided, http.DefaultClient will be used.
// scheme and host can be empty strings to target local service.
// A trailing "/" is added to path if there isn't one.
func NewCode(httpClient *http.Client, scheme, host, path string) *codeClient {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return &codeClient{
		client: httpClient,
		baseURL: &url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   path,
		},
	}
}

// codeClient implements code.Service remotely over HTTP.
type codeClient struct {
	client  *http.Client // HTTP client for API requests. If nil, http.DefaultClient should be used.
	baseURL *url.URL     // Base URL for API requests. Path must have a trailing "/".
}

func (cc *codeClient) ListDirectories(ctx context.Context) ([]*code.Directory, error) {
	u := url.URL{Path: httproute.ListDirectories}
	resp, err := ctxhttp.Get(ctx, cc.client, cc.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var dirs []*code.Directory
	err = json.NewDecoder(resp.Body).Decode(&dirs)
	return dirs, err
}

func (cc *codeClient) GetDirectory(ctx context.Context, importPath string) (*code.Directory, error) {
	u := url.URL{
		Path:     httproute.GetDirectory,
		RawQuery: url.Values{"ImportPath": {importPath}}.Encode(),
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
	var dir *code.Directory
	err = json.NewDecoder(resp.Body).Decode(&dir)
	return dir, err
}
