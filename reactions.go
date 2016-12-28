package main

import (
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/reactions/fs"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func newReactionsService(root webdav.FileSystem, users users.Service) (reactions.Service, error) {
	return fs.NewService(root, users)
}
