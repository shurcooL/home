package httphandler

import (
	"net/http"

	"github.com/gorilla/schema"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/issues"
)

// Issues is an API handler for issues.Service.
type Issues struct {
	Issues issues.Service
}

func (h Issues) List(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	var q struct {
		RepoURI  string
		OptState issues.StateFilter
	}
	if err := schema.NewDecoder().Decode(&q, req.URL.Query()); err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	repo := issues.RepoSpec{URI: q.RepoURI}
	opt := issues.IssueListOptions{State: q.OptState}
	is, err := h.Issues.List(req.Context(), repo, opt)
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: is}
}

func (h Issues) Count(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	var q struct {
		RepoURI  string
		OptState issues.StateFilter
	}
	if err := schema.NewDecoder().Decode(&q, req.URL.Query()); err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	repo := issues.RepoSpec{URI: q.RepoURI}
	opt := issues.IssueListOptions{State: q.OptState}
	count, err := h.Issues.Count(req.Context(), repo, opt)
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: count}
}

func (h Issues) ListComments(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	var q struct {
		RepoURI string
		ID      uint64
		Opt     *struct {
			Start  int
			Length int
		}
	}
	if err := schema.NewDecoder().Decode(&q, req.URL.Query()); err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	repo := issues.RepoSpec{URI: q.RepoURI}
	var opt *issues.ListOptions
	if q.Opt != nil {
		opt = &issues.ListOptions{
			Start:  q.Opt.Start,
			Length: q.Opt.Length,
		}
	}
	is, err := h.Issues.ListComments(req.Context(), repo, q.ID, opt)
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: is}
}

func (h Issues) EditComment(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return httputil.MethodError{Allowed: []string{"POST"}}
	}
	var q struct {
		RepoURI string
		ID      uint64
	}
	if err := schema.NewDecoder().Decode(&q, req.URL.Query()); err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	repo := issues.RepoSpec{URI: q.RepoURI}
	if err := req.ParseForm(); err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	var cr issues.CommentRequest
	if err := schema.NewDecoder().Decode(&cr, req.PostForm); err != nil {
		return httputil.HTTPError{Code: http.StatusBadRequest, Err: err}
	}
	is, err := h.Issues.EditComment(req.Context(), repo, q.ID, cr)
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: is}
}
