package http_test

import (
	"github.com/shurcooL/home/http"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

var (
	_ reactions.Service = http.Reactions{}
	_ users.Service     = http.Users{}
)
