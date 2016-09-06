package http_test

import (
	"github.com/shurcooL/home/http"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

var (
	_ notifications.Service = http.Notifications{}
	_ reactions.Service     = http.Reactions{}
	_ users.Service         = http.Users{}
)
