// Package httphandler contains an API handler for change.Service.
package httphandler

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"strconv"

	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/reactions"
)

func init() {
	// For Change.ListTimeline.
	gob.Register(change.Comment{})
	gob.Register(change.Review{})
	gob.Register(change.TimelineItem{})

	// For change.TimelineItem.Payload.
	gob.Register(change.ClosedEvent{})
	gob.Register(change.ReopenedEvent{})
	gob.Register(change.RenamedEvent{})
	gob.Register(change.CommitEvent{})
	gob.Register(change.LabeledEvent{})
	gob.Register(change.UnlabeledEvent{})
	gob.Register(change.ReviewRequestedEvent{})
	gob.Register(change.ReviewRequestRemovedEvent{})
	gob.Register(change.MergedEvent{})
	gob.Register(change.DeletedEvent{})
}

// Change is an API handler for change.Service.
// It returns errors compatible with httperror package.
type Change struct {
	Change change.Service
}

func (h Change) List(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := q.Get("Repo")
	opt := change.ListOptions{Filter: change.StateFilter(q.Get("OptFilter"))}
	cs, err := h.Change.List(req.Context(), repo, opt)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: cs}
}

func (h Change) Count(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := q.Get("Repo")
	opt := change.ListOptions{Filter: change.StateFilter(q.Get("OptFilter"))}
	count, err := h.Change.Count(req.Context(), repo, opt)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: count}
}

func (h Change) Get(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := q.Get("Repo")
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ID query parameter: %v", err)}
	}
	i, err := h.Change.Get(req.Context(), repo, id)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: i}
}

func (h Change) ListTimeline(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := q.Get("Repo")
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ID query parameter: %v", err)}
	}
	var opt *change.ListTimelineOptions
	if s, err := strconv.Atoi(q.Get("Opt.Start")); err == nil {
		if opt == nil {
			opt = new(change.ListTimelineOptions)
		}
		opt.Start = s
	}
	if l, err := strconv.Atoi(q.Get("Opt.Length")); err == nil {
		if opt == nil {
			opt = new(change.ListTimelineOptions)
		}
		opt.Length = l
	}
	tis, err := h.Change.ListTimeline(req.Context(), repo, id, opt)
	if err != nil {
		return err
	}
	return gob.NewEncoder(w).Encode(tis)
}

func (h Change) ListCommits(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := q.Get("Repo")
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ID query parameter: %v", err)}
	}
	cs, err := h.Change.ListCommits(req.Context(), repo, id)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: cs}
}

func (h Change) GetDiff(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httperror.Method{Allowed: []string{"GET"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := q.Get("Repo")
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ID query parameter: %v", err)}
	}
	var opt *change.GetDiffOptions
	if c, ok := q["Opt.Commit"]; ok && len(c) == 1 {
		if opt == nil {
			opt = new(change.GetDiffOptions)
		}
		opt.Commit = c[0]
	}
	diff, err := h.Change.GetDiff(req.Context(), repo, id, opt)
	if err != nil {
		return err
	}
	_, err = w.Write(diff)
	return err
}

func (h Change) EditComment(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return httperror.Method{Allowed: []string{"POST"}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	repo := q.Get("Repo")
	id, err := strconv.ParseUint(q.Get("ID"), 10, 64)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ID query parameter: %v", err)}
	}
	if err := req.ParseForm(); err != nil {
		return httperror.BadRequest{Err: err}
	}
	cr := change.CommentRequest{
		ID: req.PostForm.Get("ID"), // TODO: Automate this conversion process.
	}
	if reaction := req.PostForm["Reaction"]; len(reaction) != 0 {
		r := reactions.EmojiID(reaction[0])
		cr.Reaction = &r
	}
	comment, err := h.Change.EditComment(req.Context(), repo, id, cr)
	if err != nil {
		return err
	}
	return httperror.JSONResponse{V: comment}
}
