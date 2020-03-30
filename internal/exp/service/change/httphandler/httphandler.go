// Package httphandler contains an API handler for change.Service.
package httphandler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/reactions"
)

// Change is an API handler for change.Service.
type Change struct {
	Change change.Service
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
