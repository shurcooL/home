package indieauth_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/shurcooL/home/indieauth"
)

// Test redirect handling of FetchUserProfile, as specified by IndieAuth:
//
// 	Clients MUST start by making a GET or HEAD request to fetch the user's profile URL
// 	to discover the necessary values. Clients MUST follow HTTP redirects (up to a self-
// 	imposed limit). If an HTTP permament redirect (HTTP 301 or 308) is encountered, the
// 	client MUST use the resulting URL as the canonical profile URL. If an HTTP temporary
// 	redirect (HTTP 302 or 307) is encountered, the client MUST use the previous URL as
// 	the profile URL, but use the redirected-to page for discovery.
//
// See https://indieauth.spec.indieweb.org/#discovery-by-clients
// and https://indieauth.spec.indieweb.org/#redirect-examples.
//
// Also see https://github.com/indieweb/indieauth/issues/36.
func TestFetchUserProfileRedirect(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/1", http.RedirectHandler("/2", http.StatusMovedPermanently))
	mux.Handle("/2", http.RedirectHandler("/3", http.StatusMovedPermanently))
	mux.Handle("/3", http.RedirectHandler("/4", http.StatusFound))
	mux.Handle("/4", http.RedirectHandler("/5", http.StatusFound))
	mux.Handle("/5", http.RedirectHandler("/6", http.StatusMovedPermanently))
	mux.Handle("/6", http.RedirectHandler("/7", http.StatusMovedPermanently))
	mux.Handle("/7", http.RedirectHandler("/8", http.StatusFound))
	mux.Handle("/8", http.RedirectHandler("/9", http.StatusFound))
	mux.Handle("/9", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		mustWrite(w, `<link href="/api/indieauth/authorization" rel="authorization_endpoint">`)
	}))
	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	me, err := url.Parse(ts.URL + "/1")
	if err != nil {
		t.Fatal(err)
	}
	u, _, err := indieauth.FetchUserProfile(context.Background(), ts.Client().Transport, me)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := u.CanonicalMe.Path, "/3"; got != want {
		t.Errorf("got CanonicalMe.Path = %q, want %q", got, want)
	}
	if u.AuthzEndpoint == nil {
		t.Error("got AuthzEndpoint = nil, want non-nil")
	} else if got, want := u.AuthzEndpoint.Path, "/api/indieauth/authorization"; got != want {
		t.Errorf("got AuthzEndpoint.Path = %q, want %q", got, want)
	}
}

// Test that FetchUserProfile returns an error
// on a redirect to the insecure HTTP protocol.
func TestFetchUserProfileRedirectToInsecure(t *testing.T) {
	insecure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		mustWrite(w, `<link href="/api/indieauth/authorization" rel="authorization_endpoint">`)
	}))
	defer insecure.Close()
	secure := httptest.NewTLSServer(http.RedirectHandler(insecure.URL+"/2", http.StatusMovedPermanently))
	defer secure.Close()

	me, err := url.Parse(secure.URL + "/1")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = indieauth.FetchUserProfile(context.Background(), secure.Client().Transport, me)
	if err == nil {
		t.Fatal("got err = nil, want non-nil")
	}
	if got, want := err.Error(), "redirected to insecure URL"; !strings.Contains(got, want) {
		t.Errorf("got %q, doesn't contain %q", got, want)
	}
}

func mustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}
