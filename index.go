package main

import (
	"net/http"

	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

func initIndex(assets http.Handler, notifications notifications.Service, users users.Service) http.Handler {
	// TODO: Implement.
	return nil
}
