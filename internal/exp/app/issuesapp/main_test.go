package issuesapp_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/exp/app/issuesapp"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/issue/fs"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
	"github.com/shurcooL/webdavfs/vfsutil"
	"golang.org/x/net/webdav"
)

func TestRoutes(t *testing.T) {
	repo := issues.RepoSpec{URI: "example.org"}
	issuesApp, err := mockIssuesApp(repo)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", httputil.ErrorHandler(nil, func(w http.ResponseWriter, req *http.Request) error {
		req = req.WithContext(context.WithValue(req.Context(), issuesapp.RepoSpecContextKey, repo))
		req = req.WithContext(context.WithValue(req.Context(), issuesapp.BaseURIContextKey, "."))
		return issuesApp.ServeHTTP(w, req)
	}))

	tests := []struct {
		method, url string
		wantCode    int
	}{
		{"GET", "/assets/frontend.wasm", http.StatusOK},
		{"POST", "/assets/frontend.wasm", http.StatusMethodNotAllowed},
		{"GET", "/assets/gfm/gfm.css", http.StatusOK},

		{"GET", "/", http.StatusOK},
		{"POST", "/", http.StatusMethodNotAllowed},
		{"GET", "/new", http.StatusOK},
		{"PATCH", "/new", http.StatusMethodNotAllowed},
		{"GET", "/1", http.StatusOK},
		{"POST", "/1", http.StatusMethodNotAllowed},
		{"GET", "/1/", http.StatusNotFound},
		{"GET", "/1-foobar", http.StatusNotFound},
		{"GET", "/2", http.StatusNotFound},
		{"GET", "/foobar", http.StatusNotFound},
		{"GET", "/1/edit", http.StatusMethodNotAllowed},
		{"GET", "/1/comment", http.StatusMethodNotAllowed},
		{"GET", "/1/foobar", http.StatusNotFound},
		{"POST", "/1/comment/0", http.StatusNotFound},
		{"GET", "/1/comment/foobar", http.StatusNotFound},
	}
	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.url, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if got, want := w.Code, tc.wantCode; got != want {
			t.Errorf("%s %q: got %v, want %v", tc.method, tc.url, http.StatusText(got), http.StatusText(want))
		}
	}
}

func mockIssuesApp(repo issues.RepoSpec) (interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request) error
}, error) {
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

	return issuesapp.New(service, users, issuesapp.Options{}), nil
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
