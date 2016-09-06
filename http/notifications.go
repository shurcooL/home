package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

// Notifications implements notifications.Service remotely over HTTP.
type Notifications struct{}

func (Notifications) List(_ context.Context, opt notifications.ListOptions) (notifications.Notifications, error) {
	return nil, fmt.Errorf("List: not implemented")
}

func (Notifications) Count(_ context.Context, opt interface{}) (uint64, error) {
	resp, err := http.Get("/api/notifications/count")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return 0, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var u uint64
	err = json.NewDecoder(resp.Body).Decode(&u)
	return u, err
}

func (Notifications) MarkAllRead(_ context.Context, repo notifications.RepoSpec) error {
	return fmt.Errorf("MarkAllRead: not implemented")
}

func (Notifications) Subscribe(_ context.Context, appID string, repo notifications.RepoSpec, threadID uint64, subscribers []users.UserSpec) error {
	return fmt.Errorf("Subscribe: not implemented")
}

func (Notifications) MarkRead(_ context.Context, appID string, repo notifications.RepoSpec, threadID uint64) error {
	return fmt.Errorf("MarkRead: not implemented")
}

func (Notifications) Notify(_ context.Context, appID string, repo notifications.RepoSpec, threadID uint64, nr notifications.NotificationRequest) error {
	return fmt.Errorf("Notify: not implemented")
}
