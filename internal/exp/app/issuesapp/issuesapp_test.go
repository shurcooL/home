package issuesapp_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"testing"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/home/internal/exp/app/issuesapp"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/issue/fs"
	"github.com/shurcooL/home/internal/exp/spa"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
	"github.com/shurcooL/webdavfs/vfsutil"
	"golang.org/x/net/webdav"
)

func TestRoutes(t *testing.T) {
	issuesApp, err := mockIssuesApp()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		url       string
		wantError error
	}{
		{"/test123/...$issues", nil},
		{"/test123/...$issues/", httperror.Redirect{URL: "/test123/...$issues"}},
		{"/test123/...$issues/new", nil},
		{"/test123/...$issues/1", nil},
		{"/test123/...$issues/1/", os.ErrNotExist},
		{"/test123/...$issues/1-foo", os.ErrNotExist},
		{"/test123/...$issues/2", &os.PathError{Op: "open", Path: "dmitri.shuralyov.com/test123/issues/2/0", Err: os.ErrNotExist}},
		{"/test123/...$issues/foo", os.ErrNotExist},
		{"/test123/...$issues/foo/bar", os.ErrNotExist},
		{"/test123/...$issues/1/foo", os.ErrNotExist},
		{"/test123/...$issues/1/foo/bar", os.ErrNotExist},
	}
	for _, tc := range tests {
		reqURL, _ := url.Parse(tc.url)
		_, err := issuesApp.ServePage(context.Background(), ioutil.Discard, reqURL)
		if got, want := err, tc.wantError; !equalError(got, want) {
			t.Errorf("%q: got %v, want %v", tc.url, got, want)
		}
	}
}

func mockIssuesApp() (spa.App, error) {
	repo := issues.RepoSpec{URI: "dmitri.shuralyov.com/test123"}

	mem := webdav.NewMemFS()
	err := vfsutil.MkdirAll(context.Background(), mem, path.Join(repo.URI, "issues"), 0700)
	if err != nil {
		return nil, err
	}

	users := mockUsers{}
	service, err := fs.NewService(mem, nil, nil, users)
	if err != nil {
		return nil, err
	}

	// Create a test issue with some reactions.
	_, err = service.Create(context.Background(), repo, issues.Issue{
		Title: "Some issue about something",
		Comment: issues.Comment{
			Body: "This is a test issue.",
		},
		Labels: []issues.Label{
			{Name: "label", Color: issues.RGB{R: 224, G: 235, B: 245}},
			{Name: "another", Color: issues.RGB{R: 224, G: 235, B: 245}},
		},
	})
	if err != nil {
		return nil, err
	}
	for _, reaction := range [...]reactions.EmojiID{"grinning", "+1", "construction_worker"} {
		_, err = service.EditComment(context.Background(), repo, 1, issues.CommentRequest{
			ID:       0,
			Reaction: &reaction,
		})
		if err != nil {
			return nil, err
		}
	}
	for _, state := range [...]state.Issue{state.IssueClosed, state.IssueOpen} {
		_, _, err = service.Edit(context.Background(), repo, 1, issues.IssueRequest{
			State: &state,
		})
		if err != nil {
			return nil, err
		}
	}
	_, err = service.CreateComment(context.Background(), repo, 1, issues.Comment{
		Body: "This is a test comment.",
	})
	if err != nil {
		return nil, err
	}

	return issuesapp.New(service, users, nil, issuesapp.Options{}), nil
}

type mockUsers struct {
	users.Service
}

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	switch {
	case user == users.UserSpec{ID: 1, Domain: "example.org"}:
		return users.User{
			UserSpec:  user,
			Login:     "gopher",
			Name:      "Sample Gopher",
			Email:     "gopher@example.org",
			AvatarURL: "https://avatars0.githubusercontent.com/u/8566911?v=4&s=32",
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

// equalError reports whether errors a and b are considered equal.
// They are equal if both are nil, or both are not nil and a.Error() == b.Error().
func equalError(a, b error) bool {
	return a == nil && b == nil || a != nil && b != nil && a.Error() == b.Error()
}
