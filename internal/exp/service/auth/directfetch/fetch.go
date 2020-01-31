// Package directfetch provides a direct implementation of auth.FetchService.
package directfetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	githubv3 "github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/home/indieauth"
	"github.com/shurcooL/home/internal/exp/service/auth"
	"github.com/shurcooL/users"
	"willnorris.com/go/microformats"
)

type service struct {
	cl *http.Client     // Unauthenticated HTTP client for use by unauthenticated requests.
	gh *githubv3.Client // Unauthenticated GitHub API client for use by unauthenticated requests.
}

// NewService creates an auth.FetchService that fetches directly
// by using unauthenticated HTTP and GitHub API clients.
func NewService() auth.FetchService {
	return service{
		cl: &http.Client{Transport: &httpcache.Transport{Cache: httpcache.NewMemoryCache()}},
		gh: githubv3.NewClient(&http.Client{Transport: &httpcache.Transport{Cache: httpcache.NewMemoryCache()}}),
	}
}

// FetchUserProfile implements auth.FetchService.
func (s service) FetchUserProfile(ctx context.Context, me *url.URL) (auth.UserProfile, error) {
	// Do a direct fetch using the HTTP client s.cl.
	// Set the timeout and detect when it happens.
	fetchCtx, cancel := context.WithTimeout(ctx, auth.FetchTimeout)
	ia, doc, fetchError := indieauth.FetchUserProfile(fetchCtx, s.cl, me)
	cancel()
	if errors.Is(fetchError, context.DeadlineExceeded) {
		return auth.UserProfile{}, fmt.Errorf("user profile not found at %q because it took more than %v to respond", me, auth.FetchTimeout)
	} else if fetchError != nil {
		return auth.UserProfile{}, fetchError
	}

	u := auth.UserProfile{UserProfile: ia}

	data := microformats.ParseNode(doc, ia.CanonicalMe)
	if photo, err := photoURL(data); err != nil {
		return u, err
	} else {
		// TODO, THINK: Consider making a HEAD request to photo URL to ensure it's not 404, maybe? At what level should this be done, if at all?
		u.AvatarURL = photo
	}
	if login, ok := githubLogin(data); ok {
		u.GitHubLogin = login
	}

	return u, nil
}

// FetchGitHubUser implements auth.FetchService.
func (s service) FetchGitHubUser(ctx context.Context, login string) (_ users.User, websiteURL string, _ error) {
	// Do a direct fetch using the GitHub client s.gh.
	// Set the timeout and detect when it happens.
	fetchCtx, cancel := context.WithTimeout(ctx, auth.FetchTimeout)
	ghUser, resp, fetchError := s.gh.Users.Get(fetchCtx, login)
	cancel()
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return users.User{}, "", fmt.Errorf("GitHub user %q doesn't exist", login)
	} else if errors.Is(fetchError, context.DeadlineExceeded) {
		return users.User{}, "", fmt.Errorf("GitHub user %q not found because GitHub took more than %v to respond", login, auth.FetchTimeout)
	} else if fetchError != nil {
		return users.User{}, "", fetchError
	} else if ghUser.GetType() != "User" {
		return users.User{}, "", fmt.Errorf("%q is a GitHub %v; need a GitHub User", login, ghUser.GetType())
	}

	if ghUser.ID == nil || *ghUser.ID <= 0 {
		return users.User{}, "", errors.New("GitHub user ID is nil or nonpositive")
	}
	if ghUser.Login == nil || *ghUser.Login == "" {
		return users.User{}, "", errors.New("GitHub user Login is nil or empty")
	}
	if ghUser.AvatarURL == nil {
		return users.User{}, "", errors.New("GitHub user AvatarURL is nil")
	}
	if ghUser.HTMLURL == nil {
		return users.User{}, "", errors.New("GitHub user HTMLURL is nil")
	}
	return users.User{
		UserSpec:  users.UserSpec{ID: uint64(*ghUser.ID), Domain: "github.com"},
		Login:     *ghUser.Login,
		AvatarURL: *ghUser.AvatarURL,
		HTMLURL:   *ghUser.HTMLURL,
	}, ghUser.GetBlog(), nil
}
