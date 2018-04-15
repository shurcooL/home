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

	"github.com/shurcooL/home/resume"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/reactions/fs"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

var updateFlag = flag.Bool("update", false, "Update golden files.")

// TestBodyInnerHTML validates that resume.RenderBodyInnerHTML renders the body inner HTML as expected.
func TestBodyInnerHTML(t *testing.T) {
	var buf bytes.Buffer
	err := resume.RenderBodyInnerHTML(context.TODO(), &buf, shurcool, mockReactions{}, mockNotifications{}, alice, "/")
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
	notifications := mockNotifications{}
	authenticatedUser, err := users.GetAuthenticated(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	returnURL := "http://localhost:8080/resume"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := resume.RenderBodyInnerHTML(context.Background(), ioutil.Discard, shurcool, reactions, notifications, authenticatedUser, returnURL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

var (
	shurcool = users.User{
		UserSpec: users.UserSpec{ID: 1924134, Domain: "github.com"},
		Name:     "Dmitri Shuralyov",
		Email:    "dmitri@shuralyov.com",
	}

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

type mockNotifications struct{ notifications.Service }

func (mockNotifications) Count(_ context.Context, opt interface{}) (uint64, error) { return 0, nil }

type mockUsers struct{ users.Service }

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	if user.ID == 0 {
		return users.User{}, fmt.Errorf("user %v not found", user)
	}
	return users.User{
		UserSpec:  user,
		Login:     fmt.Sprintf("%d@%s", user.ID, user.Domain),
		AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96",
		HTMLURL:   "",
	}, nil
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
