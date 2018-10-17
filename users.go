package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	githubv3 "github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/users"
	"github.com/shurcooL/users/fs"
	"golang.org/x/net/webdav"
	"golang.org/x/oauth2"
)

var shurcool = users.UserSpec{ID: 1924134, Domain: "github.com"}

// Authenticated GitHub API clients with public repo scope.
// (Since GraphQL API doesn't support unauthenticated clients at this time.)
var shurcoolPublicRepoGHV3, shurcoolPublicRepoGHV4 = func() (*githubv3.Client, *githubv4.Client) {
	authTransport := &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("HOME_GH_SHURCOOL_PUBLIC_REPO")}),
	}
	cacheTransport := &httpcache.Transport{
		Transport:           authTransport,
		Cache:               httpcache.NewMemoryCache(),
		MarkCachedResponses: true,
	}
	return githubv3.NewClient(&http.Client{Transport: cacheTransport, Timeout: 5 * time.Second}),
		githubv4.NewClient(&http.Client{Transport: authTransport, Timeout: 5 * time.Second})
}()

type userCreator interface {
	// Create creates the specified user.
	// It returns os.ErrExist if the user already exists.
	Create(ctx context.Context, user users.User) error
}

func newUsersService(root webdav.FileSystem) (users.Service, userCreator, error) {
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

func (u Users) Get(ctx context.Context, user users.UserSpec) (users.User, error) {
	return u.store.Get(ctx, user)
}

func (Users) GetAuthenticatedSpec(ctx context.Context) (users.UserSpec, error) {
	s, ok := ctx.Value(sessionContextKey).(*session)
	if !ok {
		return users.UserSpec{}, fmt.Errorf("internal error: sessionContextKey isn't set on context but Users.GetAuthenticatedSpec is called")
	}
	if s == nil {
		return users.UserSpec{}, nil
	}
	return users.UserSpec{
		ID:     s.GitHubUserID,
		Domain: "github.com",
	}, nil
}

func (u Users) GetAuthenticated(ctx context.Context) (users.User, error) {
	userSpec, err := u.GetAuthenticatedSpec(ctx)
	if err != nil {
		return users.User{}, err
	}
	if userSpec.ID == 0 {
		return users.User{}, nil
	}
	return u.Get(ctx, userSpec)
}

func (Users) Edit(ctx context.Context, er users.EditRequest) (users.User, error) {
	return users.User{}, errors.New("Edit is not implemented")
}

// sessionContextKey is a context key. It can be used to access the session
// that the context is tied to. The associated value will be of type *session.
var sessionContextKey = &contextKey{"session"}

func withSession(req *http.Request, s *session) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), sessionContextKey, s))
}

func initGitUsers(usersService users.Service) (gitUsers map[string]users.User, err error) {
	// TODO: Add support for additional git users.
	gitUsers = make(map[string]users.User) // Key is lower git author email.
	shurcool, err := usersService.Get(context.Background(), shurcool)
	if os.IsNotExist(err) {
		log.Printf("initGitUsers: shurcool user does not exist: %v", err)
		return gitUsers, nil
	} else if err != nil {
		return nil, err
	}
	gitUsers[strings.ToLower(shurcool.Email)] = shurcool
	gitUsers[strings.ToLower("shurcooL@gmail.com")] = shurcool // Previous email.
	return gitUsers, nil
}
