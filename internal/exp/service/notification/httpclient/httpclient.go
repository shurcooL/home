// Package httpclient contains notification.Service implementation over HTTP.
package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/exp/service/notification/httproute"
	"github.com/shurcooL/users"
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
	q := url.Values{} // TODO: Automate this conversion process.
	q.Set("Namespace", opt.Namespace)
	if opt.All {
		q.Set("All", "1")
	}
	u := url.URL{
		Path:     httproute.ListNotifications,
		RawQuery: q.Encode(),
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
	var v struct {
		Notifs []notification.Notification
		Error  *string
	}
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return nil, err
	}
	if e := v.Error; e != nil {
		return v.Notifs, errors.New(*e)
	}
	return v.Notifs, nil
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
		for {
			if dec == nil {
				const backoff = 10 * time.Second
				log.Printf("notificationClient.StreamNotifications: sleeping %v then trying again\n", backoff)
				time.Sleep(backoff)
				resp, err := ctxhttp.Get(ctx, n.client, n.baseURL.ResolveReference(&u).String())
				if err != nil {
					log.Println("notificationClient.StreamNotifications: http.Get:", err)
					continue
				}
				if resp.StatusCode != http.StatusOK {
					body, _ := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					log.Printf("notificationClient.StreamNotifications: did not get acceptable status code: %v body: %q\n", resp.Status, body)
					continue
				}
				dec = json.NewDecoder(resp.Body)
			}

			var notifs []notification.Notification
			err := dec.Decode(&notifs)
			if err != nil {
				log.Println("notificationClient.StreamNotifications: dec.Decode:", err)
				resp.Body.Close()
				dec = nil
				continue
			}
			select {
			case <-ctx.Done():
				resp.Body.Close()
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
	var v struct {
		Count uint64
		Error *string
	}
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return 0, err
	}
	if e := v.Error; e != nil {
		return v.Count, errors.New(*e)
	}
	return v.Count, nil
}

func (n *notificationClient) MarkThreadRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	u := url.URL{
		Path: httproute.MarkThreadRead,
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

func (*notificationClient) SubscribeThread(_ context.Context, namespace, threadType string, threadID uint64, subscribers []users.UserSpec) error {
	return fmt.Errorf("notificationClient.SubscribeThread: not implemented")
}

func (*notificationClient) NotifyThread(_ context.Context, namespace, threadType string, threadID uint64, nr notification.NotificationRequest) error {
	return fmt.Errorf("notificationClient.NotifyThread: not implemented")
}
