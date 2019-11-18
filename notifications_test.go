package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/webdav"
)

// Test that visiting /notifications without being logged in
// redirects to /login.
func TestNotificationsRedirectsLogin(t *testing.T) {
	mux := http.NewServeMux()

	users, _, err := newUsersService(webdav.NewMemFS())
	if err != nil {
		t.Fatal(err)
	}
	initNotifications(mux, webdav.NewMemFS(), nil, users, nil)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	resp := rr.Result()
	if got, want := resp.StatusCode, http.StatusSeeOther; got != want {
		t.Errorf("got status code %d %s, want %d %s", got, http.StatusText(got), want, http.StatusText(want))
	}
	if got, want := resp.Header.Get("Location"), "/login?return=%2Fnotifications"; got != want {
		t.Errorf("got Location header %q, want %q", got, want)
	}
}
