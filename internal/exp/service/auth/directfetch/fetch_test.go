package directfetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	urlpkg "net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shurcooL/home/indieauth"
	"github.com/shurcooL/home/internal/exp/service/auth"
)

func TestFetchUserProfile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("local.test/with-photo", func(w http.ResponseWriter, req *http.Request) {
		http.ServeContent(w, req, ".html", time.Time{}, strings.NewReader(`<div class="h-card"><img class="u-photo" src="/photo.jpg"></div>`))
	})
	mux.HandleFunc("local.test/with-photo-and-alt", func(w http.ResponseWriter, req *http.Request) {
		http.ServeContent(w, req, ".html", time.Time{}, strings.NewReader(`<div class="h-card"><img class="u-photo" src="/photo.jpg" alt="Photo"></div>`))
	})
	s := service{http: localRoundTripper{mux}}

	url := func(s string) *urlpkg.URL {
		t.Helper()
		u, err := urlpkg.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	for _, tt := range [...]struct {
		in   *urlpkg.URL
		want auth.UserProfile
	}{
		{
			in: url("https://local.test/with-photo"),
			want: auth.UserProfile{
				UserProfile: indieauth.UserProfile{
					CanonicalMe: url("https://local.test/with-photo"),
				},
				AvatarURL: "https://local.test/photo.jpg",
			},
		},
		{
			in: url("https://local.test/with-photo-and-alt"),
			want: auth.UserProfile{
				UserProfile: indieauth.UserProfile{
					CanonicalMe: url("https://local.test/with-photo-and-alt"),
				},
				AvatarURL: "https://local.test/photo.jpg",
			},
		},
	} {
		t.Run(tt.in.Path[1:], func(t *testing.T) {
			got, err := s.FetchUserProfile(context.Background(), tt.in)
			if err != nil {
				t.Fatal("FetchUserProfile:", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FetchUserProfile: not equal (-want +got):\n%s", diff)
			}
		})
	}
}

// localRoundTripper is an http.RoundTripper that executes HTTP transactions
// by using Handler directly, instead of going over an HTTP connection.
type localRoundTripper struct {
	Handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.Handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Request = req
	return resp, nil
}
