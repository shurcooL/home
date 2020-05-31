package fs_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/exp/service/notification/fs"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func Test(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "notificationfs_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatal(err)
		}
	}()

	tempFS := webdav.Dir(tempDir)
	err = tempFS.Mkdir(context.Background(), "notifications", 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = tempFS.Mkdir(context.Background(), "read", 0755)
	if err != nil {
		t.Fatal(err)
	}
	usersService := &mockUsers{Current: users.UserSpec{ID: 1, Domain: "example.org"}}
	s := fs.NewService(tempFS, usersService)

	// List notifications.
	ns, err := s.ListNotifications(context.Background(), notification.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 0 {
		t.Errorf("want no notifications, got: %+v", ns)
	}

	// Subscribe target user to some issues.
	err = s.SubscribeThread(context.Background(), "namespace", "issues", 1,
		[]users.UserSpec{{ID: 1, Domain: "example.org"}})
	if err != nil {
		t.Fatal(err)
	}
	err = s.SubscribeThread(context.Background(), "namespace", "issues", 2,
		[]users.UserSpec{{ID: 1, Domain: "example.org"}})
	if err != nil {
		t.Fatal(err)
	}
	err = s.SubscribeThread(context.Background(), "namespace", "issues", 3,
		[]users.UserSpec{{ID: 1, Domain: "example.org"}})
	if err != nil {
		t.Fatal(err)
	}

	// Make a notification as another user.
	usersService.Current.ID = 2
	err = s.NotifyThread(context.Background(), "namespace", "issues", 1,
		notification.NotificationRequest{
			ImportPaths: []string{"namespace/path"},
			Time:        time.Now(),
			Payload: notification.IssueComment{
				IssueTitle:     "Issue 1",
				IssueState:     state.IssueOpen,
				CommentBody:    "Comment body.",
				CommentHTMLURL: "",
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	usersService.Current.ID = 1

	// List notifications.
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 1 || !ns[0].Unread || ns[0].Payload.(notification.IssueComment).IssueTitle != "Issue 1" {
		t.Errorf(`want 1 unread notification "Issue 1", got: %+v`, ns)
	}

	// Mark it read.
	err = s.MarkThreadRead(context.Background(), "namespace", "issues", 1)
	if err != nil {
		t.Fatal(err)
	}

	// List notifications.
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 0 {
		t.Errorf("want no notifications, got: %+v", ns)
	}
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 1 || ns[0].Unread || ns[0].Payload.(notification.IssueComment).IssueTitle != "Issue 1" {
		t.Errorf(`want 1 read notification "Issue 1", got: %+v`, ns)
	}

	// Make 2 new notifications as another user.
	usersService.Current.ID = 2
	err = s.NotifyThread(context.Background(), "namespace", "issues", 2,
		notification.NotificationRequest{
			ImportPaths: []string{"namespace/path"},
			Time:        time.Now(),
			Payload: notification.IssueComment{
				IssueTitle:     "Issue 2",
				IssueState:     state.IssueOpen,
				CommentBody:    "Comment body.",
				CommentHTMLURL: "",
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	err = s.NotifyThread(context.Background(), "namespace", "issues", 3,
		notification.NotificationRequest{
			ImportPaths: []string{"namespace/path"},
			Time:        time.Now(),
			Payload: notification.IssueComment{
				IssueTitle:     "Issue 3",
				IssueState:     state.IssueOpen,
				CommentBody:    "Comment body.",
				CommentHTMLURL: "",
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	usersService.Current.ID = 1

	// List notifications.
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 2 {
		t.Errorf("want 2 notifications, got: %+v", ns)
	}
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 3 {
		t.Errorf("want 3 notifications, got: %+v", ns)
	}

	// Mark all read.
	for _, n := range ns {
		err = s.MarkThreadRead(context.Background(), n.Namespace, n.ThreadType, n.ThreadID)
		if err != nil {
			t.Fatal(err)
		}
	}

	// List notifications.
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 0 {
		t.Errorf("want no notifications, got: %+v", ns)
	}
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 3 {
		t.Errorf("want 3 notifications, got %d: %+v", len(ns), ns)
	}

	// Repeat a notification as another user.
	usersService.Current.ID = 2
	err = s.NotifyThread(context.Background(), "namespace", "issues", 1,
		notification.NotificationRequest{
			ImportPaths: []string{"namespace/path"},
			Time:        time.Now(),
			Payload: notification.IssueComment{
				IssueTitle:     "Issue 1",
				IssueState:     state.IssueOpen,
				CommentBody:    "Comment body.",
				CommentHTMLURL: "",
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	usersService.Current.ID = 1

	// List notifications.
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 1 || !ns[0].Unread || ns[0].Payload.(notification.IssueComment).IssueTitle != "Issue 1" {
		t.Errorf(`want 1 unread notification "Issue 1", got: %+v`, ns)
	}
	ns, err = s.ListNotifications(context.Background(), notification.ListOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != 4 {
		t.Errorf("want 4 notifications, got %d: %+v", len(ns), ns)
	}
}

type mockUsers struct {
	Current users.UserSpec
	users.Service
}

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	switch {
	case user == users.UserSpec{ID: 1, Domain: "example.org"}:
		return users.User{
			UserSpec: user,
			Login:    "gopher1",
			Name:     "Gopher One",
			Email:    "gopher1@example.org",
		}, nil
	case user == users.UserSpec{ID: 2, Domain: "example.org"}:
		return users.User{
			UserSpec: user,
			Login:    "gopher2",
			Name:     "Gopher Two",
			Email:    "gopher2@example.org",
		}, nil
	default:
		return users.User{}, fmt.Errorf("user %v not found", user)
	}
}

func (m mockUsers) GetAuthenticatedSpec(context.Context) (users.UserSpec, error) {
	return m.Current, nil
}

func (m mockUsers) GetAuthenticated(ctx context.Context) (users.User, error) {
	userSpec, err := m.GetAuthenticatedSpec(ctx)
	if err != nil {
		return users.User{}, err
	}
	if userSpec.ID == 0 {
		return users.User{}, nil
	}
	return m.Get(ctx, userSpec)
}
