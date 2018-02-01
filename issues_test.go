package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
	notifications, err := initNotifications(mux, webdav.NewMemFS(), users)
	if err != nil {
		t.Fatal(err)
	}
	issues, err := newIssuesService(webdav.NewMemFS(), notifications, nil, users)
	if err != nil {
		t.Fatal(err)
	}
	_, err = initIssues(mux, issues, notifications, users)
	if err != nil {
		t.Fatal(err)
	}

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
