package main

import (
	"github.com/shurcooL/events"
	"github.com/shurcooL/events/githubapi"
	"golang.org/x/net/webdav"
)

func newEventsService(root webdav.FileSystem) events.Service {
	return githubapi.NewService(unauthenticatedGitHubClient, "shurcooL")
}
