// Package httphandler contains an API handler for issues.Service.
package httphandler

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"strconv"

	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/reactions"
)

func init() {
	// For Issues.ListTimeline.
	gob.Register(issues.Comment{})
	gob.Register(issues.Event{})
}

// Issues is an API handler for issues.Service.
// It returns errors compatible with httperror package.
type Issues struct {
	Issues issues.Service
}

func (h Issues) List(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := issues.RepoSpec{URI: q.Get("RepoURI")}
	opt := issues.IssueListOptions{State: issues.StateFilter(q.Get("OptState"))}
	is, err := h.Issues.List(req.Context(), repo, opt)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: is}
}

func (h Issues) Count(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := issues.RepoSpec{URI: q.Get("RepoURI")}
	opt := issues.IssueListOptions{State: issues.StateFilter(q.Get("OptState"))}
	count, err := h.Issues.Count(req.Context(), repo, opt)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: count}
}

func (h Issues) ListTimeline(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := issues.RepoSpec{URI: q.Get("RepoURI")}
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ID query parameter: %v", err)}
	}
	var opt *issues.ListOptions
	if s, err := strconv.Atoi(q.Get("Opt.Start")); err == nil {
		if opt == nil {
			opt = new(issues.ListOptions)
		}
		opt.Start = s
	}
	if l, err := strconv.Atoi(q.Get("Opt.Length")); err == nil {
		if opt == nil {
			opt = new(issues.ListOptions)
		}
		opt.Length = l
	}
	tis, err := h.Issues.ListTimeline(req.Context(), repo, id, opt)
	if err != nil {
		return err
	}
	return gob.NewEncoder(w).Encode(tis)
}

func (h Issues) Create(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodPost {
		return httperror.Method{Allowed: []string{http.MethodPost}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := issues.RepoSpec{URI: q.Get("RepoURI")}
	issue, err := h.Issues.Create(req.Context(), repo, issues.Issue{
		Title: q.Get("Title"),
		Comment: issues.Comment{
			Body: q.Get("Body"),
		},
	})
	if err != nil {
		// TODO: Return error via JSON.
		return err
	}
	return httperror.JSONResponse{V: issue}
}

func (h Issues) EditComment(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return httperror.Method{Allowed: []string{"POST"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := issues.RepoSpec{URI: q.Get("RepoURI")}
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ID query parameter: %v", err)}
	}
	if err := req.ParseForm(); err != nil {
		return httperror.BadRequest{Err: err}
	}
	var cr issues.CommentRequest
	cr.ID, err = strconv.ParseUint(req.PostForm.Get("ID"), 10, 64) // TODO: Automate this conversion process.
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ID form parameter: %v", err)}
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
	return httperror.JSONResponse{V: is}
}
