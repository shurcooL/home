// Package httpclient contains notification.Service implementation over HTTP.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/exp/service/notification/httproute"
	"golang.org/x/net/context/ctxhttp"
)

// NewNotification creates a client that implements notification.Service remotely over HTTP.
// If a nil httpClient is provided, http.DefaultClient will be used.
// scheme and host can be empty strings to target local service.
// A trailing "/" is added to path if there isn't one.
func NewNotification(httpClient *http.Client, scheme, host, path string) notification.Service {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return &notificationClient{
		client: httpClient,
		baseURL: &url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   path,
		},
	}
}

// notificationClient implements notification.Service remotely over HTTP.
type notificationClient struct {
	client  *http.Client // HTTP client for API requests. If nil, http.DefaultClient should be used.
	baseURL *url.URL     // Base URL for API requests. Path must have a trailing "/".
}

func (n *notificationClient) ListNotifications(ctx context.Context, opt notification.ListOptions) ([]notification.Notification, error) {
	v := url.Values{} // TODO: Automate this conversion process.
	v.Set("Namespace", opt.Namespace)
	if opt.All {
		v.Set("All", "1")
	}
	u := url.URL{
		Path:     httproute.ListNotifications,
		RawQuery: v.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, n.client, n.baseURL.ResolveReference(&u).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var notifs []notification.Notification
	err = json.NewDecoder(resp.Body).Decode(&notifs)
	return notifs, err
}

func (n *notificationClient) StreamNotifications(ctx context.Context, ch chan<- []notification.Notification) error {
	u := url.URL{
		Path: httproute.StreamNotifications,
	}
	resp, err := ctxhttp.Get(ctx, n.client, n.baseURL.ResolveReference(&u).String())
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	dec := json.NewDecoder(resp.Body)
	go func() {
		defer resp.Body.Close()
		for {
			var notifs []notification.Notification
			err := dec.Decode(&notifs)
			if err != nil {
				log.Println("notificationClient.StreamNotifications: dec.Decode:", err)
				return
			}
			select {
			case <-ctx.Done():
				return
			case ch <- notifs:
			}
		}
	}()
	return nil
}

func (n *notificationClient) CountNotifications(ctx context.Context) (uint64, error) {
	u := url.URL{
		Path: httproute.CountNotifications,
	}
	resp, err := ctxhttp.Get(ctx, n.client, n.baseURL.ResolveReference(&u).String())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return 0, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var c uint64
	err = json.NewDecoder(resp.Body).Decode(&c)
	return c, err
}

func (n *notificationClient) MarkNotificationRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	u := url.URL{
		Path: httproute.MarkNotificationRead,
		RawQuery: url.Values{ // TODO: Automate this conversion process.
			"Namespace":  {namespace},
			"ThreadType": {threadType},
			"ThreadID":   {strconv.FormatUint(threadID, 10)},
		}.Encode(),
	}
	resp, err := ctxhttp.Post(ctx, n.client, n.baseURL.ResolveReference(&u).String(), "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	return nil
}
