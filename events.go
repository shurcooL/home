package main

import (
	"context"
	"sort"

	"github.com/shurcooL/events"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/events/fs"
	"github.com/shurcooL/events/githubapi"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func newEventsService(root webdav.FileSystem, users users.Service) (events.Service, error) {
	shurcool, err := users.Get(context.Background(), shurcool)
	if err != nil {
		return nil, err
	}
	local, err := fs.NewService(root, shurcool)
	if err != nil {
		return nil, err
	}
	github, err := githubapi.NewService(unauthenticatedGitHubClient, shurcool)
	if err != nil {
		return nil, err
	}
	return multiEvents{
		github, // Events from GitHub API.
		local,  // Events from local store.
	}, nil
}

// multiEvents is a union of multiple events.Services.
type multiEvents []events.Service

// List lists newest 100 events from all services.
//
// It keeps going even if there are errors encountered, but reports them at the end.
func (m multiEvents) List(ctx context.Context) ([]event.Event, error) {
	var events []event.Event
	var errors []error
	for _, s := range m {
		e, err := s.List(ctx)
		if err != nil {
			errors = append(errors, err)
		}
		events = append(events, e...)
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Time.After(events[j].Time)
	})
	if len(events) > 100 {
		events = events[:100]
	}
	if len(errors) > 0 {
		return events, errors[0]
	}
	return events, nil
}

// Log logs the event to all services.
//
// It keeps going even if there are errors encountered, but reports them at the end.
func (m multiEvents) Log(ctx context.Context, event event.Event) error {
	var errors []error
	for _, s := range m {
		err := s.Log(ctx, event)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}
