// +build ignore

// An app that serves mock issues for development and testing.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/exp/app/issuesv2"
	"github.com/shurcooL/home/internal/exp/service/issuev2/v1tov2"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/issues/fs"
	"github.com/shurcooL/issuesapp/httphandler"
	"github.com/shurcooL/issuesapp/httproute"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/reactions/emojis"
	"github.com/shurcooL/users"
	"github.com/shurcooL/webdavfs/vfsutil"
	"golang.org/x/net/webdav"
)

var (
	httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
)

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	mem := webdav.NewMemFS()
	repo := issues.RepoSpec{URI: "example.org"}
	err := vfsutil.MkdirAll(context.Background(), mem, path.Join(repo.URI, "issues"), 0700)
	if err != nil {
		return err
	}

	users := mockUsers{}
	serviceV1, err := fs.NewService(mem, nil, nil, users)
	if err != nil {
		return err
	}
	service := v1tov2.Service{
		Service:     serviceV1,
		ListV1Repos: func() []issues.RepoSpec { return []issues.RepoSpec{repo} },
	}

	// Create a test issue with some reactions.
	_, err = service.Create(context.Background(), repo, issues.Issue{
		Title: "import/path: this is a test issue",
		Comment: issues.Comment{
			Body: "This is a test issue." + strings.Repeat("\n\n...", 10),
		},
		Labels: []issues.Label{
			{Name: "label", Color: issues.RGB{R: 224, G: 235, B: 245}},
			{Name: "another", Color: issues.RGB{R: 224, G: 235, B: 245}},
		},
	})
	if err != nil {
		return err
	}
	for _, reaction := range []reactions.EmojiID{"grinning", "+1", "construction_worker"} {
		_, err = service.EditComment(context.Background(), repo, 1, issues.CommentRequest{
			ID:       0,
			Reaction: &reaction,
		})
		if err != nil {
			return err
		}
	}
	for _, state := range []issues.State{issues.ClosedState, issues.OpenState} {
		_, _, err = service.Edit(context.Background(), repo, 1, issues.IssueRequest{
			State: &state,
		})
		if err != nil {
			return err
		}
	}
	_, err = service.CreateComment(context.Background(), repo, 1, issues.Comment{
		Body: "This is a test comment.",
	})
	if err != nil {
		return err
	}

	// Create another issue with a different prefix.
	_, err = service.Create(context.Background(), repo, issues.Issue{
		Title:   "somewhere/else: this is an another test issue",
		Comment: issues.Comment{Body: "This is a test issue."},
	})
	if err != nil {
		return err
	}

	// Register HTTP API endpoints.
	apiHandler := httphandler.Issues{Issues: service}
	http.Handle(httproute.List, httputil.ErrorHandler(users, apiHandler.List))
	http.Handle(httproute.Count, httputil.ErrorHandler(users, apiHandler.Count))
	http.Handle(httproute.ListComments, httputil.ErrorHandler(users, apiHandler.ListComments))
	http.Handle(httproute.ListEvents, httputil.ErrorHandler(users, apiHandler.ListEvents))
	http.Handle(httproute.EditComment, httputil.ErrorHandler(users, apiHandler.EditComment))

	opt := issuesv2.Options{
		HeadPre: `<meta name="viewport" content="width=device-width">
<style type="text/css">
	body {
		margin: 20px;
		font-family: sans-serif;
		font-size: 14px;
		line-height: initial;
		color: #373a3c;
	}
	.btn {
		font-size: 11px;
		line-height: 11px;
		border-radius: 4px;
		border: solid #d2d2d2 1px;
		background-color: #fff;
		box-shadow: 0 1px 1px rgba(0, 0, 0, .05);
	}
</style>`,
		BodyPre: `<div style="max-width: 800px; margin: 0 auto 100px auto;">`,
	}
	issuesApp := issuesv2.New(service, users, opt)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		req = req.WithContext(context.WithValue(req.Context(), issuesv2.PatternContextKey, req.URL.Query().Get("pattern")))
		req = req.WithContext(context.WithValue(req.Context(), issuesv2.BaseURIContextKey, "."))
		issuesApp.ServeHTTP(w, req)
	})

	http.HandleFunc("/login/github", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Sorry, this is a read-only instance and it doesn't support signing in.")
	})

	emojisHandler := httpgzip.FileServer(emojis.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	http.Handle("/emojis/", http.StripPrefix("/emojis", emojisHandler))

	log.Println("Started.")

	err = http.ListenAndServe(*httpFlag, nil)
	return err
}

type mockUsers struct {
	users.Service
}

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	switch {
	case user == users.UserSpec{ID: 1, Domain: "example.org"}:
		return users.User{
			UserSpec:  user,
			Login:     "gopher",
			Name:      "Sample Gopher",
			Email:     "gopher@example.org",
			AvatarURL: "https://avatars0.githubusercontent.com/u/8566911?v=4&s=96",
			HTMLURL:   "https://github.com/gopherbot",
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
