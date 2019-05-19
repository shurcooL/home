package code_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/dirhash"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/mod"
)

func TestModuleHandler(t *testing.T) {
	notifications := mockNotifications{}
	events := &mockEvents{}
	users := mockUsers{}
	service, err := code.NewService(filepath.Join("testdata", "repositories"), notifications, events, users)
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

	for _, tt := range []struct {
		url        string
		wantType   string
		wantBody   string
		wantSum    string // Expected checksum for module .zip (as in go.sum).
		wantModSum string // Expected checksum for go.mod file (as in go.sum).
	}{
		// Module emptyrepo tests.
		{
			url:      "/api/module/dmitri.shuralyov.com/emptyrepo/@v/list",
			wantType: "text/plain; charset=utf-8",
			wantBody: "",
		},

		// Module kebabcase tests.
		{
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/list",
			wantType: "text/plain; charset=utf-8",
			wantBody: `v0.0.0-20170912031248-a1d95f8919b5
v0.0.0-20170914162131-bf160e40a791
`,
		},
		{
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-a1d95f8919b5.info",
			wantType: "application/json",
			wantBody: `{
	"Version": "v0.0.0-20170912031248-a1d95f8919b5",
	"Time": "2017-09-12T03:12:48Z"
}
`,
		},
		{
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170914162131-bf160e40a791.info",
			wantType: "application/json",
			wantBody: `{
	"Version": "v0.0.0-20170914162131-bf160e40a791",
	"Time": "2017-09-14T16:21:31Z"
}
`,
		},
		{
			url:        "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-a1d95f8919b5.mod",
			wantType:   "text/plain; charset=utf-8",
			wantBody:   "module dmitri.shuralyov.com/kebabcase\n",
			wantModSum: "h1:zlZLgG71KSMQ+9XWuKJgSRws1h0iMspYv2y69MUzNFo=",
		},
		{
			url:        "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170914162131-bf160e40a791.mod",
			wantType:   "text/plain; charset=utf-8",
			wantBody:   "module dmitri.shuralyov.com/kebabcase\n",
			wantModSum: "h1:zlZLgG71KSMQ+9XWuKJgSRws1h0iMspYv2y69MUzNFo=",
		},
		{
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170912031248-a1d95f8919b5.zip",
			wantType: "application/zip",
			wantSum:  "h1:xUU8cZj0tfJxDjfyJ6xLLh6G615T10e16A1mxCoygiI=",
		},
		{
			url:      "/api/module/dmitri.shuralyov.com/kebabcase/@v/v0.0.0-20170914162131-bf160e40a791.zip",
			wantType: "application/zip",
			wantSum:  "h1:Lz+BA1qBebmQ4Ev2oGecFqNFK4jq5orgAPanU0rsL98=",
		},

		// Module scratch tests.
		{
			url:      "/api/module/dmitri.shuralyov.com/scratch/@v/list",
			wantType: "text/plain; charset=utf-8",
			wantBody: `v0.0.0-20171129001319-b205cb69d5d7
v0.0.0-20180121202958-53695465092b
v0.0.0-20180125023930-cdbe493822d6
v0.0.0-20180326031431-f628922a6885
`,
		},
	} {
		t.Run(tt.url[1:], func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			resp := rr.Result()
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
				gotSum, err := mod.HashZip(bytes.NewReader(rr.Body.Bytes()), dirhash.DefaultHash)
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
