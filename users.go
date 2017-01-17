package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/users"
	"github.com/shurcooL/users/fs"
	"golang.org/x/net/webdav"
)

var shurcool = users.UserSpec{ID: 1924134, Domain: "github.com"}

// unauthenticatedGitHubClient makes unauthenticated calls
// with the OAuth application credentials.
var unauthenticatedGitHubClient = func() *github.Client {
	var transport http.RoundTripper
	if githubConfig.ClientID != "" {
		transport = &github.UnauthenticatedRateLimitedTransport{
			ClientID:     githubConfig.ClientID,
			ClientSecret: githubConfig.ClientSecret,
		}
	}
	transport = &httpcache.Transport{
		Transport:           transport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}
	return github.NewClient(&http.Client{Transport: transport})
}()

func newUsersService(root webdav.FileSystem) (users.Service, users.Store, error) {
	s, err := fs.NewStore(root)
	if err != nil {
		return nil, nil, err
	}
	return Users{store: s}, s, nil
}

// Users implements users.Service.
type Users struct {
	store users.Store
}

func (s Users) Get(ctx context.Context, user users.UserSpec) (users.User, error) {
	return s.store.Get(ctx, user)
}

// userContextKey is a context key. It can be used to access the user
// that the context is tied to. The associated value will be of type *user.
var userContextKey = &contextKey{"user"}

func withUser(req *http.Request, u *user) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), userContextKey, u))
}

func (s Users) GetAuthenticatedSpec(ctx context.Context) (users.UserSpec, error) {
	u, ok := ctx.Value(userContextKey).(*user)
	if !ok {
		return users.UserSpec{}, fmt.Errorf("internal error: userContextKey isn't set on context but Users.GetAuthenticatedSpec is called")
	}
	if u == nil {
		return users.UserSpec{}, nil
	}
	return users.UserSpec{
		ID:     u.ID,
		Domain: "github.com",
	}, nil
}

func (s Users) GetAuthenticated(ctx context.Context) (users.User, error) {
	userSpec, err := s.GetAuthenticatedSpec(ctx)
	if err != nil {
		return users.User{}, err
	}
	if userSpec.ID == 0 {
		return users.User{}, nil
	}
	return s.Get(ctx, userSpec)
}

func (Users) Edit(ctx context.Context, er users.EditRequest) (users.User, error) {
	return users.User{}, errors.New("Edit is not implemented")
}
