package main

import (
	"context"
	"testing"

	"golang.org/x/net/webdav"
)

// User shurcooL is required for rendering many pages.
// Test that it exists even with an empty user store.
func TestUserShurcool(t *testing.T) {
	users, _, err := newUsersService(webdav.NewMemFS())
	if err != nil {
		t.Fatal(err)
	}
	_, err = users.Get(context.Background(), shurcool)
	if err != nil {
		t.Errorf("users.Get(shurcool): %v", err)
	}
}
