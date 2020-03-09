package httpclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/shurcooL/home/httputil"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/issue/fs"
	"github.com/shurcooL/home/internal/exp/service/issue/httpclient"
	"github.com/shurcooL/home/internal/exp/service/issue/httphandler"
	"github.com/shurcooL/home/internal/exp/service/issue/httproute"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
	"golang.org/x/oauth2"
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
	return users.UserSpec{ID: 1, Domain: "example.org"}, nil
}

func (m mockUsers) GetAuthenticated(ctx context.Context) (users.User, error) {
	userSpec, err := m.GetAuthenticatedSpec(ctx)
	if err != nil {
		return users.User{}, err
	}
	if userSpec.ID == 0 {
		return users.User{}, nil
	}
	return m.Get(ctx, userSpec)
}

func init() {
	users := mockUsers{}

	// Create a mock backend service implementation with sample data.
	issuesService, err := fs.NewService(webdav.Dir(filepath.Join("testdata", "issues")), nil, nil, users)
	if err != nil {
		log.Fatalln(err)
	}

	// Register the issues API handler.
	issuesAPIHandler := httphandler.Issues{Issues: issuesService}
	http.Handle(httproute.List, httputil.ErrorHandler(users, issuesAPIHandler.List))
	http.Handle(httproute.Count, httputil.ErrorHandler(users, issuesAPIHandler.Count))
	http.Handle(httproute.ListComments, httputil.ErrorHandler(users, issuesAPIHandler.ListComments))
	http.Handle(httproute.ListEvents, httputil.ErrorHandler(users, issuesAPIHandler.ListEvents))
	http.Handle(httproute.EditComment, httputil.ErrorHandler(users, issuesAPIHandler.EditComment))
}

var issuesClient = httpclient.NewIssues(nil, "", "")

func ExampleNewIssues() {
	issuesClient := httpclient.NewIssues(nil, "http", "localhost:8080")

	// Now you can use any of issuesClient methods.

	// Output:

	_ = issuesClient
}

func ExampleNewIssues_authenticated() {
	// HTTP client with authentication.
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "... your access token ..."},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	issuesClient := httpclient.NewIssues(httpClient, "http", "localhost:8080")

	// Now you can use any of issuesClient methods.

	// Output:

	_ = issuesClient
}

func ExampleIssues_List() {
	is, err := issuesClient.List(context.Background(), issues.RepoSpec{URI: "example.org/repo"}, issues.IssueListOptions{
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
	// 			"CanonicalMe": "",
	// 			"Elsewhere": null,
	// 			"Login": "gopher",
	// 			"Name": "Sample Gopher",
	// 			"Email": "gopher@example.org",
	// 			"AvatarURL": "",
	// 			"HTMLURL": "",
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

func ExampleIssues_Count() {
	count, err := issuesClient.Count(context.Background(), issues.RepoSpec{URI: "example.org/repo"}, issues.IssueListOptions{
		State: issues.AllStates,
	})
	if err != nil {
		log.Fatalln(err)
	}

	printJSON(count)

	// Output:
	// 1
}

func ExampleIssues_ListComments() {
	is, err := issuesClient.ListComments(context.Background(), issues.RepoSpec{URI: "example.org/repo"}, 1, nil)
	if err != nil {
		log.Fatalln(err)
	}

	printJSON(is)

	// Output:
	// [
	// 	{
	// 		"ID": 0,
	// 		"User": {
	// 			"ID": 1,
	// 			"Domain": "example.org",
	// 			"CanonicalMe": "",
	// 			"Elsewhere": null,
	// 			"Login": "gopher",
	// 			"Name": "Sample Gopher",
	// 			"Email": "gopher@example.org",
	// 			"AvatarURL": "",
	// 			"HTMLURL": "",
	// 			"SiteAdmin": false
	// 		},
	// 		"CreatedAt": "2016-09-24T22:00:50.642521756Z",
	// 		"Edited": null,
	// 		"Body": "Sample body.",
	// 		"Reactions": [
	// 			{
	// 				"Reaction": "grinning",
	// 				"Users": [
	// 					{
	// 						"ID": 1,
	// 						"Domain": "example.org",
	// 						"CanonicalMe": "",
	// 						"Elsewhere": null,
	// 						"Login": "gopher",
	// 						"Name": "Sample Gopher",
	// 						"Email": "gopher@example.org",
	// 						"AvatarURL": "",
	// 						"HTMLURL": "",
	// 						"SiteAdmin": false
	// 					}
	// 				]
	// 			},
	// 			{
	// 				"Reaction": "+1",
	// 				"Users": [
	// 					{
	// 						"ID": 2,
	// 						"Domain": "example.org",
	// 						"CanonicalMe": "",
	// 						"Elsewhere": null,
	// 						"Login": "2@example.org",
	// 						"Name": "",
	// 						"Email": "",
	// 						"AvatarURL": "https://secure.gravatar.com/avatar?d=mm\u0026f=y\u0026s=96",
	// 						"HTMLURL": "",
	// 						"SiteAdmin": false
	// 					},
	// 					{
	// 						"ID": 1,
	// 						"Domain": "example.org",
	// 						"CanonicalMe": "",
	// 						"Elsewhere": null,
	// 						"Login": "gopher",
	// 						"Name": "Sample Gopher",
	// 						"Email": "gopher@example.org",
	// 						"AvatarURL": "",
	// 						"HTMLURL": "",
	// 						"SiteAdmin": false
	// 					},
	// 					{
	// 						"ID": 3,
	// 						"Domain": "example.org",
	// 						"CanonicalMe": "",
	// 						"Elsewhere": null,
	// 						"Login": "3@example.org",
	// 						"Name": "",
	// 						"Email": "",
	// 						"AvatarURL": "https://secure.gravatar.com/avatar?d=mm\u0026f=y\u0026s=96",
	// 						"HTMLURL": "",
	// 						"SiteAdmin": false
	// 					}
	// 				]
	// 			},
	// 			{
	// 				"Reaction": "mushroom",
	// 				"Users": [
	// 					{
	// 						"ID": 3,
	// 						"Domain": "example.org",
	// 						"CanonicalMe": "",
	// 						"Elsewhere": null,
	// 						"Login": "3@example.org",
	// 						"Name": "",
	// 						"Email": "",
	// 						"AvatarURL": "https://secure.gravatar.com/avatar?d=mm\u0026f=y\u0026s=96",
	// 						"HTMLURL": "",
	// 						"SiteAdmin": false
	// 					}
	// 				]
	// 			}
	// 		],
	// 		"Editable": true
	// 	},
	// 	{
	// 		"ID": 1,
	// 		"User": {
	// 			"ID": 2,
	// 			"Domain": "example.org",
	// 			"CanonicalMe": "",
	// 			"Elsewhere": null,
	// 			"Login": "2@example.org",
	// 			"Name": "",
	// 			"Email": "",
	// 			"AvatarURL": "https://secure.gravatar.com/avatar?d=mm\u0026f=y\u0026s=96",
	// 			"HTMLURL": "",
	// 			"SiteAdmin": false
	// 		},
	// 		"CreatedAt": "2016-10-02T12:31:50.813167602Z",
	// 		"Edited": null,
	// 		"Body": "Sample reply.",
	// 		"Reactions": null,
	// 		"Editable": false
	// 	},
	// 	{
	// 		"ID": 2,
	// 		"User": {
	// 			"ID": 1,
	// 			"Domain": "example.org",
	// 			"CanonicalMe": "",
	// 			"Elsewhere": null,
	// 			"Login": "gopher",
	// 			"Name": "Sample Gopher",
	// 			"Email": "gopher@example.org",
	// 			"AvatarURL": "",
	// 			"HTMLURL": "",
	// 			"SiteAdmin": false
	// 		},
	// 		"CreatedAt": "2016-10-02T18:51:14.250725508Z",
	// 		"Edited": {
	// 			"By": {
	// 				"ID": 1,
	// 				"Domain": "example.org",
	// 				"CanonicalMe": "",
	// 				"Elsewhere": null,
	// 				"Login": "gopher",
	// 				"Name": "Sample Gopher",
	// 				"Email": "gopher@example.org",
	// 				"AvatarURL": "",
	// 				"HTMLURL": "",
	// 				"SiteAdmin": false
	// 			},
	// 			"At": "2016-10-02T18:57:47.938813179Z"
	// 		},
	// 		"Body": "Sample another reply.",
	// 		"Reactions": null,
	// 		"Editable": true
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

func init() {
	// Allow local HTTP requests without a scheme to hit http.DefaultServeMux directly.
	http.DefaultTransport.(*http.Transport).RegisterProtocol("", localRoundTripper{})
}

// localRoundTripper is an http.RoundTripper that executes HTTP transactions
// by using http.DefaultServeMux directly, instead of going over an HTTP connection.
type localRoundTripper struct{}

func (localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, req)
	return w.Result(), nil
}
