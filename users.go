package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/shurcooL/users"
	"golang.org/x/net/context"
)

// TODO: Avoid global.
var usersService users.Service

// Users implementats users.Service.
type Users struct {
	gh *github.Client
}

func (s *Users) Get(ctx context.Context, user users.UserSpec) (users.User, error) {
	const (
		ds = "dmitri.shuralyov.com"
		gh = "github.com"
		tw = "twitter.com"
	)

	switch {
	// TODO: Consider using UserSpec{ID: 1, Domain: ds} as well.
	case user == users.UserSpec{ID: 1924134, Domain: gh}:
		return users.User{
			UserSpec:  user,
			Elsewhere: []users.UserSpec{{ID: 21361484, Domain: tw}},
			Login:     "shurcooL",
			Name:      "Dmitri Shuralyov",
			AvatarURL: "https://dmitri.shuralyov.com/avatar.jpg",
			HTMLURL:   "https://dmitri.shuralyov.com",
			SiteAdmin: true,
		}, nil

	case user.Domain == "github.com":
		ghUser, _, err := gitHubUsersGetByID(s.gh, int(user.ID))
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

func (s *Users) GetAuthenticated(ctx context.Context) (*users.UserSpec, error) {
	req, ok := ctx.Value(requestKey).(*http.Request)
	if !ok {
		return nil, nil
	}
	u, err := getUser(req)
	if err != nil {
		// Cannot return an error here. If getUser failed, that means no authenticated user.
		log.Println("ds.com: (*Users) GetAuthenticated:", err)
		return nil, nil
	}
	if u == nil {
		return nil, nil
	}
	return &users.UserSpec{
		ID:     u.ID,
		Domain: "github.com",
	}, nil
}

func (*Users) Edit(ctx context.Context, er users.EditRequest) (users.User, error) {
	return users.User{}, errors.New("Edit is not implemented")
}

// gitHubUsersGetByID fetches a GitHub user based on their userID.
func gitHubUsersGetByID(gh *github.Client, userID int) (*github.User, *github.Response, error) {
	req, err := gh.NewRequest("GET", fmt.Sprintf("/users?since=%v&per_page=1", userID-1), nil)
	if err != nil {
		return nil, nil, err
	}
	var users []github.User
	resp, err := gh.Do(req, &users)
	if err != nil {
		return nil, resp, err
	}
	if len(users) != 1 {
		return nil, resp, fmt.Errorf("expected 1 user, got %v users", len(users))
	}
	if users[0].ID == nil {
		return nil, resp, fmt.Errorf("got user with nil user ID: %#v", users[0])
	}
	if *users[0].ID != userID {
		return nil, resp, fmt.Errorf("expected user ID %v, got user ID %v", userID, *users[0].ID)
	}
	return &users[0], resp, nil
}
