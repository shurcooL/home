// Package httphandler contains an API handler for notification.Service.
package httphandler

import (
	"net/http"

	"github.com/shurcooL/home/internal/exp/service/issuev2"
	"github.com/shurcooL/httperror"
)

// IssueV2 is an API handler for issuev2.Service.
// It returns errors compatible with httperror package.
type IssueV2 struct {
	IssueV2 issuev2.Service
}

func (h IssueV2) CreateIssue(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodPost {
		return httperror.Method{Allowed: []string{http.MethodPost}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	issue, err := h.IssueV2.CreateIssue(req.Context(), issuev2.CreateIssueRequest{
		ImportPath: q.Get("ImportPath"),
		Title:      q.Get("Title"),
		Body:       q.Get("Body"),
	})
	if err != nil {
		// TODO: Return error via JSON.
		return err
	}
	return httperror.JSONResponse{V: issue}
}
