package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shurcooL/events"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

func initAction(code *codeService, users users.Service) {
	// "Create a New Repo" action.
	http.Handle("/action/new-repo", cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		if err := httputil.AllowMethods(req, http.MethodGet, http.MethodPost); err != nil {
			return err
		}
		switch req.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			httpgzip.ServeContent(w, req, "", time.Time{}, strings.NewReader(newRepoHTML))
			return nil
		case http.MethodPost:
			if err := req.ParseForm(); err != nil {
				return httperror.BadRequest{Err: err}
			}
			repoSpec, err := getSingleValue(req.Form, "spec")
			if err != nil {
				return httperror.BadRequest{Err: err}
			}
			repoDescription, err := getSingleValue(req.Form, "description")
			if err != nil {
				return httperror.BadRequest{Err: err}
			}

			createRepoError := code.CreateRepo(req.Context(), repoSpec, repoDescription)

			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(io.MultiWriter(os.Stdout, w), "creating new repo: spec=%q description=%q: err=%v\n", repoSpec, repoDescription, createRepoError)

			return nil
		default:
			panic("unreachable")
		}
	})})
}

const newRepoHTML = `<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Dmitri Shuralyov</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<style type="text/css">
body, input {
	font-family: Go;
}
.wide {
	width: 100%;
	box-sizing: border-box;
}
		</style>
	</head>
	<body>
		<form method="post" action="/action/new-repo">
			<h1>Create a New Repo</h1>
			Repo Spec<br>
			<input class="wide" name="spec" type="text" value="dmitri.shuralyov.com/"><br>
			<br>
			Description<br>
			<input class="wide" name="description" type="text" placeholder="Description goes here." style="width: 100%;"><br>
			<br>
			<input type="submit" value="Create">
		</form>
	</body>
</html>
`

type codeService struct {
	reposDir      string
	notifications notifications.ExternalService
	events        events.ExternalService
	users         users.Service
}

func (s *codeService) CreateRepo(ctx context.Context, repoSpec, repoDescription string) error {
	currentUser, err := s.users.GetAuthenticated(ctx)
	if err != nil {
		return err
	}

	// Authorization check.
	if !currentUser.SiteAdmin {
		return os.ErrPermission
	}

	// Create bare git repo.
	cmd := exec.Command("git", "init", "--bare", filepath.Join(s.reposDir, filepath.FromSlash(repoSpec)))
	err = cmd.Run()
	if err != nil {
		return err
	}

	// Watch the newly created repository.
	err = s.notifications.Subscribe(ctx, notifications.RepoSpec{URI: repoSpec}, "", 0, []users.UserSpec{shurcool})
	if err != nil {
		return err
	}

	// Log a "created repository" event.
	err = s.events.Log(ctx, event.Event{
		Time:      time.Now().UTC(),
		Actor:     currentUser,
		Container: repoSpec,
		Payload: event.Create{
			Type:        "repository",
			Description: repoDescription,
		},
	})
	return err
}

// getSingleValue returns the single value for key in form,
// or an error if there isn't exactly a single value.
func getSingleValue(form url.Values, key string) (string, error) {
	v, ok := form[key]
	if !ok {
		return "", fmt.Errorf("key %q not set", key)
	}
	if len(v) != 1 {
		return "", fmt.Errorf("key %q has non-single value: %+v", key, v)
	}
	return v[0], nil
}
