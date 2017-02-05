package httphandler

import (
	"log"
	"net/http"

	"github.com/shurcooL/httperror"
	"github.com/shurcooL/reactions"
)

// Reactions is an API handler for reactions.Service.
type Reactions struct {
	Reactions reactions.Service
}

func (h Reactions) GetOrToggle(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" && req.Method != "POST" {
		return httperror.Method{Allowed: []string{"GET", "POST"}}
	}
	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		return httperror.HTTP{Code: http.StatusBadRequest, Err: err}
	}
	reactableURL := req.Form.Get("reactableURL")
	reactableID := req.Form.Get("reactableID")
	switch req.Method {
	case "GET":
		reactions, err := h.Reactions.Get(req.Context(), reactableURL, reactableID)
		if err != nil {
			return err
		}
		return httperror.JSONResponse{V: reactions}
	case "POST":
		tr := reactions.ToggleRequest{
			Reaction: reactions.EmojiID(req.PostForm.Get("reaction")),
		}
		reactions, err := h.Reactions.Toggle(req.Context(), reactableURL, reactableID, tr)
		if err != nil {
			return err
		}
		return httperror.JSONResponse{V: reactions}
	default:
		panic("unreachable")
	}
}
