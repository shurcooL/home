// Package common contains common code for backend and frontend.
package common

import (
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/users"
)

type State struct {
	BaseURI          string
	ReqPath          string
	RepoSpec         issues.RepoSpec
	IssueID          uint64 `json:",omitempty"` // IssueID is the current issue ID, or 0 if not applicable (e.g., current page is /new).
	CurrentUser      users.User
	DisableReactions bool
	DisableUsers     bool
}
