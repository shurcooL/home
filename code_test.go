package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/shurcooL/events"
	codepkg "github.com/shurcooL/home/internal/code"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/notification"
)

func TestCodeHandler(t *testing.T) {
	mux := http.NewServeMux()

	reposDir := filepath.Join("internal", "code", "testdata", "repositories")
	notification := struct{ notification.Service }{} // Mock.
	events := struct{ events.Service }{}             // Mock.
	users := mockUsers{}
	code, err := codepkg.NewService(reposDir, notification, events, users)
	if err != nil {
		t.Fatal("code.NewService:", err)
	}
	codeHandler := codeHandler{code, reposDir, nil, nil, zeroIssueCounter{}, zeroChangeCounter{}, notification, users, nil}
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			t.Fatal("root path not supported")
		}
		// Serve code pages for existing repos/packages, if the request matches.
		if ok := codeHandler.ServeCodeMaybe(w, req); ok {
			return
		}
		t.Fatal("non-codeHandler paths not supported")
	})

	for _, tt := range [...]struct {
		url      string
		method   string
		wantType string
		wantBody string
	}{
		{
			url:      "/kebabcase",
			method:   http.MethodGet,
			wantType: "text/html; charset=utf-8",
			wantBody: "<html>\n\t<head>\n\t\t<title>Package kebabcase</title>\n\t\t<link href=\"/icon.png\" rel=\"icon\" type=\"image/png\">\n\t\t<meta name=\"viewport\" content=\"width=device-width\">\n\t\t<link href=\"/assets/fonts/fonts.css\" rel=\"stylesheet\" type=\"text/css\">\n\t\t<link href=\"/assets/package/style.css\" rel=\"stylesheet\" type=\"text/css\">\n\t</head>\n\t<body><div style=\"max-width: 800px; margin: 0 auto 100px auto;\"><style type=\"text/css\">\nheader.header {\n\tfont-family: inherit;\n\tfont-size: 14px;\n\tmargin-top: 30px;\n\tmargin-bottom: 30px;\n}\n\nheader.header a {\n\tcolor: rgb(35, 35, 35);\n\ttext-decoration: none;\n}\nheader.header a:hover {\n\tcolor: #4183c4;\n}\nheader.header a.Login {\n\tcolor: #4183c4;\n\ttext-decoration: none;\n}\nheader.header a.Login:hover {\n\ttext-decoration: underline;\n}\n\nheader.header ul.nav {\n\tdisplay: inline-block;\n\tmargin-top: 0;\n\tmargin-bottom: 0;\n\tpadding-left: 0;\n}\nheader.header li.nav {\n\tdisplay: inline-block;\n\tmargin-left: 20px;\n\tfont-weight: bold;\n}\nheader.header .smaller {\n\tfont-size: 12px;\n}\n\nheader.header .user {\n\tfloat: right;\n\tpadding-top: 8px;\n}</style><header class=\"header\"><a href=\"/\" style=\"display: inline-block;\" class=\"Logo\"><svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 200 200\" width=\"32\" height=\"32\" style=\"fill: currentColor;\nstroke: currentColor;\nvertical-align: middle;\"><circle cx=\"100\" cy=\"100\" r=\"90\" stroke-width=\"20\" fill=\"none\"></circle><circle cx=\"100\" cy=\"100\" r=\"60\"></circle></svg></a><ul class=\"nav\"><li class=\"nav\"><a href=\"/packages\">Packages</a></li><li class=\"nav\"><a href=\"/blog\">Blog</a></li><li class=\"nav smaller\"><a href=\"/idiomatic-go\">Idiomatic Go</a></li><li class=\"nav\"><a href=\"/talks\">Talks</a></li><li class=\"nav\"><a href=\"/projects\">Projects</a></li><li class=\"nav\"><a href=\"/resume\">Resume</a></li><li class=\"nav\"><a href=\"/about\">About</a></li></ul><span class=\"user\"><a class=\"Login\" href=\"/login?return=%2Fkebabcase\">Sign in via URL</a></span></header><h2>dmitri.shuralyov.com/kebabcase/...</h2><div class=\"tabnav\"><nav class=\"tabnav-tabs\"><a href=\"/kebabcase/...\" class=\"tabnav-tab\"><span style=\"margin-right: 4px;\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"16\" height=\"16\" viewBox=\"0 0 16 16\" style=\"fill: currentColor; vertical-align: top;\"><path d=\"M1 4.27v7.47c0 .45.3.84.75.97l6.5 1.73c.16.05.34.05.5 0l6.5-1.73c.45-.13.75-.52.75-.97V4.27c0-.45-.3-.84-.75-.97l-6.5-1.74a1.4 1.4 0 00-.5 0L1.75 3.3c-.45.13-.75.52-.75.97zm7 9.09l-6-1.59V5l6 1.61v6.75zM2 4l2.5-.67L11 5.06l-2.5.67L2 4zm13 7.77l-6 1.59V6.61l2-.55V8.5l2-.53V5.53L15 5v6.77zm-2-7.24L6.5 2.8l2-.53L15 4l-2 .53z\"></path></svg></span>Packages<span class=\"counter\">1</span></a><a href=\"/kebabcase/...$history\" class=\"tabnav-tab\"><span style=\"margin-right: 4px;\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"16\" height=\"16\" viewBox=\"0 0 14 16\" style=\"fill: currentColor; vertical-align: top;\"><path d=\"M8 13H6V6h5v2H8v5zM7 1C4.81 1 2.87 2.02 1.59 3.59L0 2v4h4L2.5 4.5C3.55 3.17 5.17 2.3 7 2.3c3.14 0 5.7 2.56 5.7 5.7s-2.56 5.7-5.7 5.7A5.71 5.71 0 011.3 8c0-.34.03-.67.09-1H.08C.03 7.33 0 7.66 0 8c0 3.86 3.14 7 7 7s7-3.14 7-7-3.14-7-7-7z\"></path></svg></span>History</a><a href=\"/kebabcase/...$issues\" class=\"tabnav-tab\" onclick=\"Open(event, this)\"><span style=\"margin-right: 4px;\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"16\" height=\"16\" viewBox=\"0 0 14 16\" style=\"fill: currentColor; vertical-align: top;\"><path d=\"M7 2.3c3.14 0 5.7 2.56 5.7 5.7s-2.56 5.7-5.7 5.7A5.71 5.71 0 011.3 8c0-3.14 2.56-5.7 5.7-5.7zM7 1C3.14 1 0 4.14 0 8s3.14 7 7 7 7-3.14 7-7-3.14-7-7-7zm1 3H6v5h2V4zm0 6H6v2h2v-2z\"></path></svg></span>Issues<span class=\"counter\">0</span></a><a href=\"/kebabcase/...$changes\" class=\"tabnav-tab\" onclick=\"Open(event, this)\"><span style=\"margin-right: 4px;\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"16\" height=\"16\" viewBox=\"0 0 12 16\" style=\"fill: currentColor; vertical-align: top;\"><path d=\"M11 11.28V5c-.03-.78-.34-1.47-.94-2.06C9.46 2.35 8.78 2.03 8 2H7V0L4 3l3 3V4h1c.27.02.48.11.69.31.21.2.3.42.31.69v6.28A1.993 1.993 0 0010 15a1.993 1.993 0 001-3.72zm-1 2.92c-.66 0-1.2-.55-1.2-1.2 0-.65.55-1.2 1.2-1.2.65 0 1.2.55 1.2 1.2 0 .65-.55 1.2-1.2 1.2zM4 3c0-1.11-.89-2-2-2a1.993 1.993 0 00-1 3.72v6.56A1.993 1.993 0 002 15a1.993 1.993 0 001-3.72V4.72c.59-.34 1-.98 1-1.72zm-.8 10c0 .66-.55 1.2-1.2 1.2-.65 0-1.2-.55-1.2-1.2 0-.65.55-1.2 1.2-1.2.65 0 1.2.55 1.2 1.2zM2 4.2C1.34 4.2.8 3.65.8 3c0-.65.55-1.2 1.2-1.2.65 0 1.2.55 1.2 1.2 0 .65-.55 1.2-1.2 1.2z\"></path></svg></span>Changes<span class=\"counter\">0</span></a></nav></div><h1>Package kebabcase</h1><p><code>import &#34;dmitri.shuralyov.com/kebabcase&#34;</code></p><h3>Overview</h3><p>\nPackage kebabcase provides a parser for identifier names\nusing kebab-case naming convention.\n</p>\n<p>\nReference: <a href=\"https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers\">https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers</a>.\n</p>\n<h3>Installation</h3><p><pre>go get -u dmitri.shuralyov.com/kebabcase</pre></p><h3><a href=\"https://pkg.go.dev/dmitri.shuralyov.com/kebabcase\">Documentation</a></h3><h3><a href=\"https://gotools.org/dmitri.shuralyov.com/kebabcase\">Code</a></h3><h3><a href=\"/LICENSE\">License</a></h3></div></body></html>",
		},
		{
			url:      "/kebabcase",
			method:   http.MethodHead,
			wantType: "text/html; charset=utf-8",
			wantBody: "",
		},
		{
			url:      "/kebabcase?go-get=1",
			method:   http.MethodGet,
			wantType: "text/plain; charset=utf-8",
			wantBody: `<meta name="go-import" content="dmitri.shuralyov.com/kebabcase git https://dmitri.shuralyov.com/kebabcase">
<meta name="go-import" content="dmitri.shuralyov.com/kebabcase mod https://dmitri.shuralyov.com/api/module">
<meta name="go-source" content="dmitri.shuralyov.com/kebabcase https://dmitri.shuralyov.com/kebabcase https://gotools.org/dmitri.shuralyov.com/kebabcase https://gotools.org/dmitri.shuralyov.com/kebabcase#{file}-L{line}">`,
		},
		{
			url:      "/kebabcase?go-get=1",
			method:   http.MethodHead,
			wantType: "text/plain; charset=utf-8",
			wantBody: "",
		},
	} {
		req := httptest.NewRequest(tt.method, tt.url, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		resp := rr.Result()
		if got, want := resp.StatusCode, http.StatusOK; got != want {
			t.Errorf("%s %s: got status code %d %s, want %d %s", tt.method, tt.url, got, http.StatusText(got), want, http.StatusText(want))
		}
		if got, want := resp.Header.Get("Content-Type"), tt.wantType; got != want {
			t.Errorf("%s %s: got Content-Type header %q, want %q", tt.method, tt.url, got, want)
		}
		if got, want := rr.Body.String(), tt.wantBody; got != want {
			t.Errorf("%s %s: body not equal:\n got = %q\nwant = %q", tt.method, tt.url, got, want)
		}
	}
}

// zeroIssueCounter implements issues.Service that always returns 0 issue count.
type zeroIssueCounter struct{ issues.Service }

func (zeroIssueCounter) Count(context.Context, issues.RepoSpec, issues.IssueListOptions) (uint64, error) {
	return 0, nil
}
