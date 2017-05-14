package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test that visiting /projects redirects to /projects/.
func TestProjectsRedirectsTrailingSlash(t *testing.T) {
	mux := http.NewServeMux()

	initProjects(mux, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusMovedPermanently; got != want {
		t.Errorf("got status code %d %s, want %d %s", got, http.StatusText(got), want, http.StatusText(want))
	}
	if got, want := rr.Header().Get("Location"), "/projects/"; got != want {
		t.Errorf("got Location header %q, want %q", got, want)
	}
}
