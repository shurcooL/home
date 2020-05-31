package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shurcooL/issues"
	"golang.org/x/net/webdav"
)

// Test that visiting /blog/1822 gives a 404 Not Found error (rather than 500 or something else).
func TestBlogNotFound(t *testing.T) {
	mux := http.NewServeMux()

	users, _, err := newUsersService(webdav.NewMemFS())
	if err != nil {
		t.Fatal(err)
	}
	issuesService, err := newIssuesService(webdav.NewMemFS(), nil, nil, users, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = initBlog(mux, issuesService, issues.RepoSpec{URI: "dmitri.shuralyov.com/blog"}, nil, users)
	if err != nil {
		t.Fatal(err)
	}

	for _, url := range [...]string{
		"/blog/1822",
		"/blog/1822?issuesapp=1",
	} {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		resp := rr.Result()
		if got, want := resp.StatusCode, http.StatusNotFound; got != want {
			t.Errorf("GET %s: got status code %d %s, want %d %s", url, got, http.StatusText(got), want, http.StatusText(want))
		}
		if got, want := resp.Header.Get("Content-Type"), "text/plain; charset=utf-8"; got != want {
			t.Errorf("GET %s: got Content-Type header %q, want %q", url, got, want)
		}
		if got, want := rr.Body.String(), "404 Not Found\n"; got != want {
			t.Errorf("GET %s: got body %q, want %q", url, got, want)
		}
	}
}
