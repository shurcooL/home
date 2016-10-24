package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	httpapi "github.com/shurcooL/home/http"
	"github.com/shurcooL/home/httphandler"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issues/fs"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

type mockUsers struct {
	users.Service
}

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	switch {
	case user == users.UserSpec{ID: 1, Domain: "example.org"}:
		return users.User{
			UserSpec: user,
			Login:    "gopher",
			Name:     "Sample Gopher",
			Email:    "gopher@example.org",
		}, nil
	default:
		return users.User{}, fmt.Errorf("user %v not found", user)
	}
}

func (mockUsers) GetAuthenticatedSpec(_ context.Context) (users.UserSpec, error) {
	return users.UserSpec{ID: 1}, nil
}

func ExampleIssues_List() {
	issuesService, err := fs.NewService(webdav.Dir(filepath.Join("testdata", "issues")), nil, mockUsers{})
	if err != nil {
		log.Fatalln(err)
	}
	issuesAPIHandler := httphandler.Issues{Issues: issuesService}
	http.Handle("/api/issues/list", httputil.ErrorHandler{issuesAPIHandler.List})
	http.DefaultTransport.(*http.Transport).RegisterProtocol("", localRoundTripper{})

	s := httpapi.Issues{}

	is, err := s.List(context.Background(), issues.RepoSpec{URI: "example.org/repo"}, issues.IssueListOptions{
		State: issues.StateFilter(issues.OpenState),
	})
	if err != nil {
		log.Fatalln(err)
	}

	printJSON(is)

	// Output:
	// [
	// 	{
	// 		"ID": 1,
	// 		"State": "open",
	// 		"Title": "Sample title",
	// 		"Labels": null,
	// 		"User": {
	// 			"ID": 1,
	// 			"Domain": "example.org",
	// 			"Elsewhere": null,
	// 			"Login": "gopher",
	// 			"Name": "Sample Gopher",
	// 			"Email": "gopher@example.org",
	// 			"AvatarURL": "",
	// 			"HTMLURL": "",
	// 			"CreatedAt": "0001-01-01T00:00:00Z",
	// 			"UpdatedAt": "0001-01-01T00:00:00Z",
	// 			"SiteAdmin": false
	// 		},
	// 		"CreatedAt": "2016-09-24T22:00:50.642521756Z",
	// 		"Edited": null,
	// 		"Body": "",
	// 		"Reactions": null,
	// 		"Editable": false,
	// 		"Replies": 2
	// 	}
	// ]
}

// printJSON prints v as JSON encoded with indent to stdout. It panics on any error.
// It's meant to be used by examples to print the output.
func printJSON(v interface{}) {
	w := json.NewEncoder(os.Stdout)
	w.SetIndent("", "\t")
	err := w.Encode(v)
	if err != nil {
		panic(err)
	}
}

type localRoundTripper struct{}

func (localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, req)
	return w.Result(), nil
}
