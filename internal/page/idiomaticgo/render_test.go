package idiomaticgo_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/shurcooL/home/internal/page/idiomaticgo"
	"github.com/shurcooL/issues/fs"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"github.com/shurcooL/webdavfs/webdavfs"
	"golang.org/x/net/webdav"
)

var updateFlag = flag.Bool("update", false, "Update golden files.")

// issuesFS was generated with:
//
// 	goexec 'vfsgen.Generate(filter.Keep(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "issues")), func(path string, fi os.FileInfo) bool { return (path == "/" || path == "/dmitri.shuralyov.com" || path == "/dmitri.shuralyov.com/idiomatic-go" || strings.HasPrefix(path, "/dmitri.shuralyov.com/idiomatic-go/")) && fi.Name() != ".DS_Store"}), vfsgen.Options{PackageName: "idiomaticgo_test", Filename: "issuesfs_vfsdata_test.go", VariableName: "issuesFS", VariableComment: "issuesFS is issues test data."})'

// TestBodyInnerHTML verifies that idiomaticgo.RenderBodyInnerHTML renders the body inner HTML as expected.
func TestBodyInnerHTML(t *testing.T) {
	users := mockUsers{}
	notifications := mockNotifications{}
	issues, err := fs.NewService(
		webdavfs.New(issuesFS),
		notifications, nil, users)
	if err != nil {
		t.Fatal(err)
	}
	authenticatedUser, err := users.GetAuthenticated(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	returnURL := "http://localhost:8080/idiomatic-go"

	var buf bytes.Buffer
	err = idiomaticgo.RenderBodyInnerHTML(context.Background(), &buf, issues, notifications, authenticatedUser, returnURL)
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
		t.Error("idiomaticgo.RenderBodyInnerHTML produced output that doesn't match 'testdata/body-inner.html'")
	}
}

func BenchmarkRenderBodyInnerHTML(b *testing.B) {
	users := mockUsers{}
	notifications := mockNotifications{}
	issues, err := fs.NewService(
		webdav.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Store", "issues")),
		notifications, nil, users)
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
