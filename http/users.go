package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/shurcooL/users"
	"golang.org/x/net/context/ctxhttp"
)

// Users implements users.Service remotely over HTTP.
type Users struct{}

func (Users) GetAuthenticated(ctx context.Context) (users.User, error) {
	resp, err := ctxhttp.Get(ctx, nil, "/api/user")
	if err != nil {
		return users.User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return users.User{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var u users.User
	err = json.NewDecoder(resp.Body).Decode(&u)
	return u, err
}

func (Users) GetAuthenticatedSpec(ctx context.Context) (users.UserSpec, error) {
	resp, err := ctxhttp.Get(ctx, nil, "/api/userspec")
	if err != nil {
		return users.UserSpec{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return users.UserSpec{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var us users.UserSpec
	err = json.NewDecoder(resp.Body).Decode(&us)
	return us, err
}

func (Users) Get(ctx context.Context, user users.UserSpec) (users.User, error) {
	resp, err := ctxhttp.Get(ctx, nil, fmt.Sprintf("/api/user/%d@%s", user.ID, user.Domain))
	if err != nil {
		return users.User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return users.User{}, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var u users.User
	err = json.NewDecoder(resp.Body).Decode(&u)
	return u, err
}

func (Users) Edit(_ context.Context, er users.EditRequest) (users.User, error) {
	return users.User{}, fmt.Errorf("Edit: not implemented")
}
