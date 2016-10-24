// Package httphandler contains API handlers used by home.
package httphandler

import (
	"net/http"
	"strconv"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/reactions"
)

// Issues is an API handler for issues.Service.
type Issues struct {
	Issues issues.Service
}

func (h Issues) List(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := issues.RepoSpec{URI: q.Get("RepoURI")}
	opt := issues.IssueListOptions{State: issues.StateFilter(q.Get("OptState"))}
	is, err := h.Issues.List(req.Context(), repo, opt)
	if err != nil {
		return err
	}
	return httputil.JSONResponse{is}
}

func (h Issues) ListComments(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := issues.RepoSpec{URI: q.Get("RepoURI")}
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	is, err := h.Issues.ListComments(req.Context(), repo, id, nil)
	if err != nil {
		return err
	}
	return httputil.JSONResponse{is}
}

func (h Issues) EditComment(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return httputil.MethodError{Allowed: []string{"POST"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := issues.RepoSpec{URI: q.Get("RepoURI")}
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	if err := req.ParseForm(); err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	var cr issues.CommentRequest
	cr.ID, err = strconv.ParseUint(req.PostForm.Get("ID"), 10, 64) // TODO: Automate this conversion process.
	if err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	if body := req.PostForm["Body"]; len(body) != 0 {
		cr.Body = &body[0]
	}
	if reaction := req.PostForm["Reaction"]; len(reaction) != 0 {
		r := reactions.EmojiID(reaction[0])
		cr.Reaction = &r
	}
	is, err := h.Issues.EditComment(req.Context(), repo, id, cr)
	if err != nil {
		return err
	}
	return httputil.JSONResponse{is}
}
