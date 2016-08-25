package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/shurcooL/reactions"
	"github.com/shurcooL/reactions/fs"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func newReactionsService(root webdav.FileSystem, users users.Service) (reactions.Service, error) {
	return fs.NewService(root, users)
}

type reactHandler struct {
	rs reactions.Service
}

func (h reactHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" && req.Method != "POST" {
		return MethodError{Allowed: []string{"GET", "POST"}}
	}

	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		return HTTPError{Code: http.StatusBadRequest, err: err}
	}

	reactableURL := req.Form.Get("reactableURL")
	reactableID := req.Form.Get("reactableID")

	switch req.Method {
	case "GET":
		reactions, err := h.rs.Get(req.Context(), reactableURL, reactableID)
		if err != nil {
			return err
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(reactions)
		return err
	case "POST":
		tr := reactions.ToggleRequest{
			Reaction: reactions.EmojiID(req.PostForm.Get("reaction")),
		}
		reactions, err := h.rs.Toggle(req.Context(), reactableURL, reactableID, tr)
		if err != nil {
			return err
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(reactions)
		return err
	default:
		panic("unreachable")
	}
}
