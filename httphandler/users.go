package httphandler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
)

// Users is an API handler for users.Service.
type Users struct {
	Users users.Service
}

func (h Users) GetAuthenticatedSpec(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	us, err := h.Users.GetAuthenticatedSpec(req.Context())
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: us}
}

func (h Users) GetAuthenticated(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	u, err := h.Users.GetAuthenticated(req.Context())
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: u}
}

func (h Users) Get(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	user, err := unmarshalUserSpec(strings.TrimPrefix(req.URL.Path, "/api/user/"))
	if err != nil {
		return httperror.BadRequest{Err: err}
	}
	u, err := h.Users.Get(req.Context(), user)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: u}
}

// unmarshalUserSpec parses userSpec, a string like "1@example.com"
// into a users.UserSpec{ID: 1, Domain: "example.com"}.
func unmarshalUserSpec(userSpec string) (users.UserSpec, error) {
	parts := strings.SplitN(userSpec, "@", 2)
	if len(parts) != 2 {
		return users.UserSpec{}, fmt.Errorf("user spec is not 2 parts: %v", len(parts))
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return users.UserSpec{}, err
	}
	return users.UserSpec{ID: id, Domain: parts[1]}, nil
}
