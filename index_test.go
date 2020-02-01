package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test that the IndieAuth authorization endpoint is
// advertised on index page in an HTTP Link header.
func TestIndexAuthzEndpoint(t *testing.T) {
	for _, tt := range []struct {
		name          string
		authzEndpoint bool
		want          string
	}{
		{name: "off", authzEndpoint: false, want: ""},
		{name: "on", authzEndpoint: true, want: `</api/indieauth/authorization>; rel="authorization_endpoint"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			indexHandler := indexHandler{
				AuthzEndpoint: tt.authzEndpoint,
			}
			req := httptest.NewRequest(http.MethodHead, "/", nil)
			rr := httptest.NewRecorder()
			indexHandler.ServeHTTP(rr, req)
			resp := rr.Result()
			if got, want := resp.StatusCode, http.StatusOK; got != want {
				t.Errorf("HEAD /: got status code %d %s, want %d %s", got, http.StatusText(got), want, http.StatusText(want))
			}
			if got, want := resp.Header.Get("Link"), tt.want; got != want {
				t.Errorf("HEAD /: got Link header %q, want %q", got, want)
			}
		})
	}
}
