package httphandler

import (
	"net/http"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/users"
)

// Users is an API handler for users.Service.
type Users struct {
	Users users.Service
}

func (h Users) GetAuthenticatedSpec(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	us, err := h.Users.GetAuthenticatedSpec(req.Context())
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: us}
}

func (h Users) GetAuthenticated(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	u, err := h.Users.GetAuthenticated(req.Context())
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: u}
}
