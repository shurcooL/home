package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/shurcooL/home/internal/exp/spa"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

// Test that visiting /notifications without being logged in
// redirects to /login.
func TestNotificationsRedirectsLogin(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "notifications_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatal(err)
		}
	}()

	mux := http.NewServeMux()

	users := mockUsers{}
	ns, _, _, err := newNotificationServiceV2(context.Background(), new(sync.WaitGroup), webdav.NewMemFS(), tempDir, tempDir, users, nil)
	if err != nil {
		t.Fatal(err)
	}
	app := spa.NewApp(ns, users, nil)
	initNotificationsV2(mux, nil, &appHandler{app.NotifsApp}, nil, nil, users)

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

type mockUsers struct{ users.Service }

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	switch {
	case user == dmitshur:
		return users.User{
			UserSpec: users.UserSpec{ID: 1924134, Domain: "github.com"},
			Name:     "Dmitri Shuralyov",
			Email:    "dmitri@shuralyov.com",
		}, nil
	default:
		return users.User{}, fmt.Errorf("user %v not found", user)
	}
}

func (mockUsers) GetAuthenticatedSpec(_ context.Context) (users.UserSpec, error) {
	return users.UserSpec{}, nil
}

func (mockUsers) GetAuthenticated(ctx context.Context) (users.User, error) {
	return users.User{}, nil
}
