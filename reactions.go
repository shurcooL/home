package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/reactions"
	"github.com/shurcooL/reactions/fs"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func newReactionsService(root webdav.FileSystem, users users.Service) (reactions.Service, error) {
	rs, err := fs.NewService(root, users)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

type reactHandler struct {
	rs reactions.Service
}

func (h reactHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method should be GET or POST", http.StatusMethodNotAllowed)
		return
	}

	if err := req.ParseForm(); err != nil {
		log.Println("req.ParseForm:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.WithValue(context.Background(), requestContextKey, req) // TODO, THINK: Is this the best place? Can it be generalized? Isn't it error prone otherwise?
	reactableURL := req.Form.Get("reactableURL")
	reactableID := req.Form.Get("reactableID")

	switch req.Method {
	case "GET":
		reactions, err := h.rs.Get(ctx, reactableURL, reactableID)
		if os.IsPermission(err) { // TODO: Move this to a higher level (and upate all other similar code too).
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("h.rs.Get:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(reactions)
		if err != nil {
			log.Println(err)
		}
	case "POST":
		tr := reactions.ToggleRequest{
			Reaction: reactions.EmojiID(req.PostForm.Get("reaction")),
		}
		reactions, err := h.rs.Toggle(ctx, reactableURL, reactableID, tr)
		if os.IsPermission(err) { // TODO: Move this to a higher level (and upate all other similar code too).
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Println("h.rs.Toggle:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(reactions)
		if err != nil {
			log.Println(err)
		}
	}
}
