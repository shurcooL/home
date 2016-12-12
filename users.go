package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/users"
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

func newUsersService() users.Service {
	return Users{gh: unauthenticatedGitHubClient}
}

type usersAPIHandler struct {
	users users.Service
}

func (h usersAPIHandler) GetAuthenticatedSpec(w http.ResponseWriter, req *http.Request) error {
	us, err := h.users.GetAuthenticatedSpec(req.Context())
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: us}
}

func (h usersAPIHandler) GetAuthenticated(w http.ResponseWriter, req *http.Request) error {
	u, err := h.users.GetAuthenticated(req.Context())
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: u}
}

// Users implements users.Service.
type Users struct {
	gh *github.Client
}

func (s Users) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	const (
		ds = "dmitri.shuralyov.com"
		gh = "github.com"
		tw = "twitter.com"
	)

	switch {
	// TODO: Consider using UserSpec{ID: 1, Domain: ds} as well.
	case user == users.UserSpec{ID: 1924134, Domain: gh}: // Dmitri Shuralyov.
		return users.User{
			UserSpec:  user,
			Elsewhere: []users.UserSpec{{ID: 21361484, Domain: tw}},
			Login:     "shurcooL",
			Name:      "Dmitri Shuralyov",
			AvatarURL: "https://dmitri.shuralyov.com/avatar-s.jpg",
			HTMLURL:   "https://dmitri.shuralyov.com",
			SiteAdmin: true,
		}, nil

	case user.Domain == "github.com":
		ghUser, _, err := s.gh.Users.GetByID(int(user.ID))
		if err != nil {
			return users.User{}, err
		}
		if ghUser.Login == nil || ghUser.AvatarURL == nil || ghUser.HTMLURL == nil {
			return users.User{}, fmt.Errorf("github user missing fields: %#v", ghUser)
		}
		return users.User{
			UserSpec:  user,
			Login:     *ghUser.Login,
			AvatarURL: template.URL(*ghUser.AvatarURL),
			HTMLURL:   template.URL(*ghUser.HTMLURL),
		}, nil

	case user == users.UserSpec{ID: 2, Domain: ds}: // Bernardo.
		return users.User{
			UserSpec:  user,
			Login:     "Bernardo",
			Name:      "Bernardo",
			AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96",
		}, nil
	case user == users.UserSpec{ID: 3, Domain: ds}: // Michal Marcinkowski.
		return users.User{
			UserSpec:  user,
			Elsewhere: []users.UserSpec{{ID: 15185890, Domain: tw}},
			Login:     "Michal Marcinkowski",
			Name:      "Michal Marcinkowski",
			AvatarURL: "https://pbs.twimg.com/profile_images/699932252764037123/MZUgYRn5_400x400.jpg", // TODO: Use Twitter API?
		}, nil
	case user == users.UserSpec{ID: 4, Domain: ds}: // Anders Elfgren.
		return users.User{
			UserSpec:  user,
			Login:     "Anders Elfgren",
			Name:      "Anders Elfgren",
			AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96",
		}, nil
	case user == users.UserSpec{ID: 5, Domain: ds}: // benp.
		return users.User{
			UserSpec:  user,
			Login:     "benp",
			AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96",
		}, nil

	default:
		return users.User{}, fmt.Errorf("user %v not found", user)
	}
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
