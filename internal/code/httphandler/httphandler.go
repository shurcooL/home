// Package httphandler contains an API handler for issues.Service.
package httphandler

import (
	"net/http"

	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/httperror"
)

// Code is an API handler for code.Service.
// It returns errors compatible with httperror package.
type Code struct {
	Code *code.Service
}

func (h Code) ListDirectories(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	dirs, err := h.Code.ListDirectories(req.Context())
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: dirs}
}

func (h Code) GetDirectory(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	importPath := q.Get("ImportPath")
	dir, err := h.Code.GetDirectory(req.Context(), importPath)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: dir}
}
