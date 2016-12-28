package idiomaticgo_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/shurcooL/home/idiomaticgo"
	"github.com/shurcooL/issues/fs"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func BenchmarkRenderBodyInnerHTML(b *testing.B) {
	users := mockUsers{}
	notifications := mockNotifications{}
	issues, err := fs.NewService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "issues")),
		notifications, users)
	if err != nil {
		b.Fatal(err)
	}
	authenticatedUser, err := users.GetAuthenticated(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	returnURL := "http://localhost:8080/idiomatic-go"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := idiomaticgo.RenderBodyInnerHTML(context.Background(), ioutil.Discard, issues, notifications, authenticatedUser, returnURL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

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

type mockNotifications struct{ notifications.Service }

func (mockNotifications) Count(_ context.Context, opt interface{}) (uint64, error) { return 0, nil }
