package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/shurcooL/events"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/go/ctxhttp"
)

// NewEvents creates a client that implements events.Service remotely over HTTP.
// If a nil httpClient is provided, http.DefaultClient will be used.
// scheme and host can be empty strings to target local service.
func NewEvents(httpClient *http.Client, scheme, host string) events.Service {
	return &eventsClient{
		client: httpClient,
		baseURL: &url.URL{
			Scheme: scheme,
			Host:   host,
		},
	}
}

// eventsClient implements events.Service remotely over HTTP.
type eventsClient struct {
	client  *http.Client // HTTP client for API requests. If nil, http.DefaultClient should be used.
	baseURL *url.URL     // Base URL for API requests.
}

func (c *eventsClient) List(ctx context.Context) ([]event.Event, error) {
	u := url.URL{Path: "/api/events/list"}
	resp, err := ctxhttp.Get(ctx, c.client, c.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var v struct {
		Events []event.Event
		Error  *string
	}
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return nil, err
	}
	var eventsError error
	if v.Error != nil {
		eventsError = errors.New(*v.Error)
	}
	return v.Events, eventsError
}

func (c *eventsClient) Log(ctx context.Context, event event.Event) error {
	return fmt.Errorf("eventsClient.Log: not implemented")
}
