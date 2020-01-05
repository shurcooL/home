package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/shurcooL/users"
)

func TestLookUpSessionViaCookie(t *testing.T) {
	defer func() {
		global = state{sessions: make(map[string]session)}
	}()
	var (
		sessionA = session{
			UserSpec:    users.UserSpec{ID: 1, Domain: "example.com"},
			Expiry:      time.Now().Add(6*24*time.Hour + time.Minute),
			AccessToken: "aaa",
		}
		sessionB = session{
			UserSpec:    users.UserSpec{ID: 2, Domain: "example.com"},
			Expiry:      time.Now().Add(6*24*time.Hour - time.Minute),
			AccessToken: "bbb",
		}
	)
	global = state{sessions: map[string]session{
		sessionA.AccessToken: sessionA,
		sessionB.AccessToken: sessionB,
	}}

	tests := []struct {
		in           *http.Request
		wantSession  *session
		wantExtended bool
		wantError    error
	}{
		{
			in:           &http.Request{},
			wantSession:  nil,
			wantExtended: false,
		},
		{
			in: &http.Request{
				Header: http.Header{
					"Cookie": []string{"accessToken=YWFh"}, // Base64-encoded "aaa".
				},
			},
			wantSession:  &sessionA,
			wantExtended: false,
		},
		{
			in: &http.Request{
				Header: http.Header{
					"Cookie": []string{"accessToken=YmJi"}, // Base64-encoded "bbb".
				},
			},
			wantSession: &session{
				UserSpec:    users.UserSpec{ID: 2, Domain: "example.com"},
				Expiry:      time.Now().Add(7 * 24 * time.Hour), // Extended expiry.
				AccessToken: "bbb",
			},
			wantExtended: true,
		},
		{
			in: &http.Request{
				Header: http.Header{
					"Cookie": []string{"accessToken=eA"}, // Base64-encoded "x".
				},
			},
			wantError: errBadAccessToken,
		},
		{
			in: &http.Request{
				Header: http.Header{
					"Cookie": []string{"accessToken=x"}, // Invalid base64 encoding.
				},
			},
			wantError: errBadAccessToken,
		},
	}
	for _, tc := range tests {
		u, extended, err := lookUpSessionViaCookie(tc.in)
		if got, want := err, tc.wantError; !equalError(got, want) {
			t.Fatalf("got error: %v, want: %v", got, want)
		}
		if tc.wantError != nil {
			continue
		}
		if got, want := u, tc.wantSession; !equalSession(got, want) {
			t.Errorf("got session: %v, want: %v", got, want)
		}
		if got, want := extended, tc.wantExtended; got != want {
			t.Errorf("got extended: %v, want: %v", got, want)
		}
	}
}

// equalSession reports whether sessions a and b are considered equal.
// They're equal if both are nil, or both are not nil and have equal fields.
func equalSession(a, b *session) bool {
	return a == nil && b == nil || a != nil && b != nil &&
		a.UserSpec == b.UserSpec &&
		-time.Second < a.Expiry.Sub(b.Expiry) && a.Expiry.Sub(b.Expiry) < time.Second && // Expiry times within a second.
		a.AccessToken == b.AccessToken
}

// equalError reports whether errors a and b are considered equal.
// They're equal if both are nil, or both are not nil and a.Error() == b.Error().
func equalError(a, b error) bool {
	return a == nil && b == nil || a != nil && b != nil && a.Error() == b.Error()
}
