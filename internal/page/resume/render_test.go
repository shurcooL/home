package resume_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/home/internal/page/resume"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/reactions/fs"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

var updateFlag = flag.Bool("update", false, "Update golden files.")

// TestBodyInnerHTML validates that resume.RenderBodyInnerHTML renders the body inner HTML as expected.
func TestBodyInnerHTML(t *testing.T) {
	var buf bytes.Buffer
	err := resume.RenderBodyInnerHTML(context.TODO(), &buf, mockReactions{}, mockNotification{}, mockUsers{}, mockTime, alice, "/")
	if err != nil {
		t.Fatal(err)
	}
	got := buf.Bytes()
	if *updateFlag {
		err := ioutil.WriteFile(filepath.Join("testdata", "body-inner.html"), got, 0644)
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := ioutil.ReadFile(filepath.Join("testdata", "body-inner.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("resume.RenderBodyInnerHTML produced output that doesn't match 'testdata/body-inner.html'")
	}
}

func BenchmarkRenderBodyInnerHTML(b *testing.B) {
	users := mockUsers{}
	reactions, err := fs.NewService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "reactions")),
		users)
	if err != nil {
		b.Fatal(err)
	}
	notification := mockNotification{}
	authenticatedUser, err := users.GetAuthenticated(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	returnURL := "http://localhost:8080/resume"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := resume.RenderBodyInnerHTML(context.Background(), ioutil.Discard, reactions, notification, users, mockTime, authenticatedUser, returnURL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var (
	dmitshur = users.User{
		UserSpec: users.UserSpec{ID: 1924134, Domain: "github.com"},
		Name:     "Dmitri Shuralyov",
		Email:    "dmitri@shuralyov.com",
	}

	mockTime = time.Date(2018, time.August, 26, 9, 41, 0, 0, time.UTC)

	alice = users.User{UserSpec: users.UserSpec{ID: 1, Domain: "example.org"}, Login: "Alice"}
	bob   = users.User{UserSpec: users.UserSpec{ID: 2, Domain: "example.org"}, Login: "Bob"}
)

type mockReactions struct{ reactions.Service }

func (mockReactions) List(_ context.Context, uri string) (map[string][]reactions.Reaction, error) {
	if uri != resume.ReactableURL {
		return nil, os.ErrNotExist
	}
	return map[string][]reactions.Reaction{
		"Go": {{
			Reaction: "smile",
			Users:    []users.User{alice, bob},
		}, {
			Reaction: "balloon",
			Users:    []users.User{bob},
		}},
	}, nil
}

type mockNotification struct{ notification.Service }

func (mockNotification) CountNotifications(_ context.Context) (uint64, error) { return 0, nil }

type mockUsers struct{ users.Service }

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	switch {
	case user == dmitshur.UserSpec:
		return dmitshur, nil
	case user == alice.UserSpec:
		return alice, nil
	case user == bob.UserSpec:
		return bob, nil
	case user.ID != 0:
		return users.User{
			UserSpec:  user,
			Login:     fmt.Sprintf("%d@%s", user.ID, user.Domain),
			AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96",
			HTMLURL:   "",
		}, nil
	default:
		return users.User{}, fmt.Errorf("user %v not found", user)
	}
}

func (mockUsers) GetAuthenticatedSpec(_ context.Context) (users.UserSpec, error) {
	return users.UserSpec{ID: 1, Domain: "example.org"}, nil
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
