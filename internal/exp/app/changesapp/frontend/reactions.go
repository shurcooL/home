// +build js,wasm,go1.14

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/reactions"
)

// ChangeReactions implements reactions.Service on top of change.Service,
// specifically for use by changes app.
//
// The format of ID is "{{.changeID}}/{{.commentID}}".
type ChangeReactions struct {
	Change change.Service
}

// Toggle toggles a reaction.
// id is "{{.changeID}}/{{.commentID}}".
func (ir ChangeReactions) Toggle(ctx context.Context, uri string, id string, tr reactions.ToggleRequest) ([]reactions.Reaction, error) {
	var (
		changeID  uint64
		commentID string
	)
	_, err := fmt.Sscanf(id, "%d/%s", &changeID, &commentID)
	if err != nil {
		return nil, err
	}
	comment, err := ir.Change.EditComment(ctx, uri, changeID, change.CommentRequest{
		ID:       commentID,
		Reaction: &tr.Reaction,
	})
	if err != nil {
		return nil, err
	}
	return comment.Reactions, nil
}

func (ChangeReactions) Get(_ context.Context, uri string, id string) ([]reactions.Reaction, error) {
	return nil, errors.New("ChangeReactions.Get: not implemented")
}

func (ChangeReactions) List(_ context.Context, uri string) (map[string][]reactions.Reaction, error) {
	return nil, errors.New("ChangeReactions.List: not implemented")
}
