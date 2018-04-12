package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"dmitri.shuralyov.com/service/change"
	"golang.org/x/net/webdav"
)

// Test that visiting /issues/github.com/shurcooL/issuesapp/new without being logged in
// redirects to /login.
func TestNewIssueRedirectsLogin(t *testing.T) {
	mux := http.NewServeMux()

	users, _, err := newUsersService(webdav.NewMemFS())
	if err != nil {
		t.Fatal(err)
	}
	notifications := initNotifications(mux, webdav.NewMemFS(), users, nil)
	issues, err := newIssuesService(webdav.NewMemFS(), notifications, nil, users, nil)
	if err != nil {
		t.Fatal(err)
	}
	initIssues(mux, issues, zeroCounter{}, notifications, users)

	req := httptest.NewRequest(http.MethodGet, "/issues/github.com/shurcooL/issuesapp/new", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusSeeOther; got != want {
		t.Errorf("got status code %d %s, want %d %s", got, http.StatusText(got), want, http.StatusText(want))
	}
	if got, want := rr.Header().Get("Location"), "/login?return=%2Fissues%2Fgithub.com%2FshurcooL%2Fissuesapp%2Fnew"; got != want {
		t.Errorf("got Location header %q, want %q", got, want)
	}
}

// zeroCounter implements changeCounter that always returns 0 change count.
type zeroCounter struct{}

func (zeroCounter) Count(context.Context, string, change.ListOptions) (uint64, error) { return 0, nil }
