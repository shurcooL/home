package code_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/mod"
	"golang.org/x/mod/sumdb/dirhash"
)

func TestModuleHandler(t *testing.T) {
	notification := mockNotification{}
	events := &mockEvents{}
	users := mockUsers{}
	service, err := code.NewService(filepath.Join("testdata", "repositories"), notification, events, users)
	if err != nil {
		t.Fatal("code.NewService:", err)
	}
	moduleHandler := code.ModuleHandler{Code: service}

	mux := http.NewServeMux()
	mux.Handle("/api/module/", http.StripPrefix("/api/module/", httputil.ErrorHandler(nil, moduleHandler.ServeModule)))
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		t.Error("HTTP server got a non-module proxy request")
		http.NotFound(w, req)
	})

	for _, tt := range [...]struct {
		name         string
		url          string
		wantNotExist bool // If true, expect 404 status code.
		wantType     string
		wantBody     string
		wantSum      string // Expected checksum for module .zip (as in go.sum).
		wantModSum   string // Expected checksum for go.mod file (as in go.sum).
	}{
		// Module emptyrepo tests.
		{
			name:     "emptyrepo version list",
			url:      "/api/module/dmitri.shuralyov.com/emptyrepo/@v/list",
			wantType: "text/plain; charset=utf-8",
			wantBody: "",
		},

		// Module kebabcase tests.
		{
			name:     "kebabcase version list",
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/list",
			wantType: "text/plain; charset=utf-8",
			wantBody: `v0.0.0-20170912031248-a1d95f8919b5
v0.0.0-20170914162131-bf160e40a791
`,
		},
		{
			name:     "kebabcase version 1 info",
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-a1d95f8919b5.info",
			wantType: "application/json",
			wantBody: `{
	"Version": "v0.0.0-20170912031248-a1d95f8919b5",
	"Time": "2017-09-12T03:12:48Z"
}
`,
		},
		{
			name:     "kebabcase version 2 info",
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170914162131-bf160e40a791.info",
			wantType: "application/json",
			wantBody: `{
	"Version": "v0.0.0-20170914162131-bf160e40a791",
	"Time": "2017-09-14T16:21:31Z"
}
`,
		},
		{
			name:       "kebabcase version 1 mod",
			url:        "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-a1d95f8919b5.mod",
			wantType:   "text/plain; charset=utf-8",
			wantBody:   "module dmitri.shuralyov.com/kebabcase\n",
			wantModSum: "h1:zlZLgG71KSMQ+9XWuKJgSRws1h0iMspYv2y69MUzNFo=",
		},
		{
			name:       "kebabcase version 2 mod",
			url:        "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170914162131-bf160e40a791.mod",
			wantType:   "text/plain; charset=utf-8",
			wantBody:   "module dmitri.shuralyov.com/kebabcase\n",
			wantModSum: "h1:zlZLgG71KSMQ+9XWuKJgSRws1h0iMspYv2y69MUzNFo=",
		},
		{
			name:     "kebabcase version 1 zip",
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-a1d95f8919b5.zip",
			wantType: "application/zip",
			wantSum:  "h1:xUU8cZj0tfJxDjfyJ6xLLh6G615T10e16A1mxCoygiI=",
		},
		{
			name:     "kebabcase version 2 zip",
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170914162131-bf160e40a791.zip",
			wantType: "application/zip",
			wantSum:  "h1:Lz+BA1qBebmQ4Ev2oGecFqNFK4jq5orgAPanU0rsL98=",
		},

		// Module scratch tests.
		{
			name:     "scratch version list",
			url:      "/api/module/dmitri.shuralyov.com/scratch/@v/list",
			wantType: "text/plain; charset=utf-8",
			wantBody: `v0.0.0-20171129001319-b205cb69d5d7
v0.0.0-20180121202958-53695465092b
v0.0.0-20180125023930-cdbe493822d6
v0.0.0-20180326031431-f628922a6885
`,
		},

		// Versions that do not exist must serve 404.
		{
			name:         "wrong timestamp",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-11111111111111-a1d95f8919b5.info",
			wantNotExist: true,
		},
		{
			name:         "wrong revision",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-aaaaaaaaaaaa.info",
			wantNotExist: true,
		},
		{
			name:         "revision length is not 12",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-a1d95f8919b.info",
			wantNotExist: true,
		},
		{
			name:         "wrong separator",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248.a1d95f8919b5.info",
			wantNotExist: true,
		},
		{
			name:         "revision is not all lower-case",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-!a1d95f8919b5.info",
			wantNotExist: true,
		},
		{
			name:         "incompatible version",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-a1d95f8919b5+incompatible.info",
			wantNotExist: true,
		},
		{
			name:         "version v1.0.0",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v1.0.0-20170912031248-a1d95f8919b5.info",
			wantNotExist: true,
		},
		{
			name:         "version v2.0.0",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v2.0.0-20170912031248-a1d95f8919b5.info",
			wantNotExist: true,
		},
		{
			name:         "pseudo-version after v1.2.3-pre",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v1.2.3-pre.0.20170912031248-a1d95f8919b5.info",
			wantNotExist: true,
		},
		{
			name:         "pseudo-version after v1.2.3",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v1.2.4-0.20170912031248-a1d95f8919b5.info",
			wantNotExist: true,
		},
		{
			name:         "commit on non-master branch",
			url:          "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20200225024836-c61324d16db7.info",
			wantNotExist: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			resp := rr.Result()
			if tt.wantNotExist {
				if got, want := resp.StatusCode, http.StatusNotFound; got != want {
					t.Errorf("got status code %d %s, want %d %s", got, http.StatusText(got), want, http.StatusText(want))
				}
				return
			}
			if got, want := resp.StatusCode, http.StatusOK; got != want {
				t.Errorf("got status code %d %s, want %d %s", got, http.StatusText(got), want, http.StatusText(want))
			}
			if got, want := resp.Header.Get("Content-Type"), tt.wantType; got != want {
				t.Errorf("got Content-Type header %q, want %q", got, want)
			}
			if tt.wantType != "application/zip" {
				if got, want := rr.Body.String(), tt.wantBody; got != want {
					t.Errorf("got body:\n%s\nwant:\n%s", got, want)
				}
			}
			if tt.wantSum != "" {
				gotSum, err := mod.HashZip(rr.Body.Bytes(), dirhash.DefaultHash)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := gotSum, tt.wantSum; got != want {
					t.Errorf("got sum %q, want %q", got, want)
				}
			}
			if tt.wantModSum != "" {
				gotModSum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) {
					return ioutil.NopCloser(bytes.NewReader(rr.Body.Bytes())), nil
				})
				if err != nil {
					t.Fatal(err)
				}
				if got, want := gotModSum, tt.wantModSum; got != want {
					t.Errorf("got mod sum %q, want %q", got, want)
				}
			}
		})
	}
}
