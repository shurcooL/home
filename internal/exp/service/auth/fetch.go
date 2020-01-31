// Package auth defines a service for home's user authentication needs.
package auth

import (
	"context"
	"net/url"
	"time"

	"github.com/shurcooL/home/indieauth"
	"github.com/shurcooL/users"
)

// FetchTimeout is the maximum time allotted
// to fetching information about a user.
const FetchTimeout = 10 * time.Second

// FetchService provides a service for fetching information about users
// from various sources, including the independent web and on GitHub.
//
// It operates on a best-effort basis. An authenticated GitHub client
// is not expected to be used because unbounded unauthenticated requests
// can force it to exceed the GitHub rate limit.
type FetchService interface {
	// FetchUserProfile fetches the user profile specified by me,
	// which must be a valid user profile URL. It returns an error
	// if the specified user profile is not available or is malformed,
	// or if communication with the remote server has failed.
	//
	// FetchUserProfile enforces a timeout.
	FetchUserProfile(ctx context.Context, me *url.URL) (UserProfile, error)

	// FetchGitHubUser fetches the GitHub user specified by login,
	// and their website URL. If the user doesn't have
	// a website URL set, the empty string is returned.
	// It returns an error if the specified GitHub user doesn't exist,
	// is malformed, or if communication with GitHub has failed.
	//
	// FetchGitHubUser enforces a timeout.
	FetchGitHubUser(ctx context.Context, login string) (_ users.User, websiteURL string, _ error)
}

// UserProfile is the parsed result of fetching
// a user profile URL with an HTTP GET request.
type UserProfile struct {
	indieauth.UserProfile
	AvatarURL   string // URL of an h-card.photo entry, or empty string if it doesn't exist.
	GitHubLogin string // Stated GitHub profile login, or empty string if it doesn't exist.
}
