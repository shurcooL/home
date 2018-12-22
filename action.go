package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/users"
)

func initAction(code *code.Service, users users.Service) {
	// "Create a New Repo" action.
	http.Handle("/action/new-repo", cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		if err := httputil.AllowMethods(req, http.MethodGet, http.MethodPost); err != nil {
			return err
		}
		switch req.Method {
		case http.MethodGet:
			if user, err := users.GetAuthenticated(req.Context()); err != nil {
				return err
			} else if !user.SiteAdmin {
				return os.ErrPermission
			}
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

			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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

// getSingleValue returns the single value for key in form,
// or an error if there isn't exactly a single value.
func getSingleValue(form url.Values, key string) (string, error) {
	v, ok := form[key]
	if !ok {
		return "", fmt.Errorf("key %q is not set", key)
	}
	if len(v) != 1 {
		return "", fmt.Errorf("key %q has non-single value: %+v", key, v)
	}
	return v[0], nil
}
