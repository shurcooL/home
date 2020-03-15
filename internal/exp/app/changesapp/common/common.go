// Package common contains common code for backend and frontend.
package common

import (
	"github.com/shurcooL/users"
)

type State struct {
	BaseURI      string
	ReqPath      string
	RepoSpec     string
	CurrentUser  users.User
	DisableUsers bool

	ChangeID uint64 `json:",omitempty"` // ChangeID is the current change ID, or 0 if not applicable (e.g., current page is /changes).
	PrevSHA  string `json:",omitempty"` // PrevSHA is the previous commit SHA, or empty if not applicable (e.g., current page is not /{changeID}/files/{commitID}).
	NextSHA  string `json:",omitempty"` // NextSHA is the next commit SHA, or empty if not applicable (e.g., current page is not /{changeID}/files/{commitID}).
}
