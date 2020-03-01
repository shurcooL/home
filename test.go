package main

import (
	"archive/zip"
	"io"
	"net/http"
)

func init() {
	// For golang.org/issue/18660.
	{
		h := func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, `<meta name="go-import" content="dmitri.shuralyov.com/test/go-get-issue-unicode git https://github.com/dmitshur-test/go-get-issue-unicode">`)
		}
		http.HandleFunc("/test/go-get-issue-unicode", h)
		http.HandleFunc("/test/go-get-issue-unicode/испытание", h)
	}

	// For own module learning.
	{
		h := func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, `<meta name="go-import" content="dmitri.shuralyov.com/test/modtest1 mod https://dmitri.shuralyov.com/test/moduleproxy">`)
		}
		http.HandleFunc("/test/modtest1", h)
		http.HandleFunc("/test/modtest1/inner", h)
		http.HandleFunc("/test/modtest1/inner/p", h)
	}

	// For testing Go module proxy URLs with an empty host. Such module proxy URLs unintentionally
	// worked without error in Go 1.11 and 1.12. Go 1.13 has fixed it, so they no longer do.
	// See https://golang.org/issue/32006#issuecomment-491943083.
	{
		http.HandleFunc("/test/modtest2", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, `<meta name="go-import" content="dmitri.shuralyov.com/test/modtest2 mod https://">`)
		})
		http.HandleFunc("/test/modtest2/@v/list", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, "v0.0.0\n")
		})
		http.HandleFunc("/test/modtest2/@v/v0.0.0.info", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, "{\n\t\"Version\": \"v0.0.0\",\n\t\"Time\": \"2019-05-04T15:44:36Z\"\n}\n")
		})
		http.HandleFunc("/test/modtest2/@v/v0.0.0.mod", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, "module dmitri.shuralyov.com/test/modtest2\n")
		})
		http.HandleFunc("/test/modtest2/@v/v0.0.0.zip", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			z := zip.NewWriter(w)
			for _, file := range [...]struct {
				Name, Body string
			}{
				{"dmitri.shuralyov.com/test/modtest2@v0.0.0/go.mod", "module dmitri.shuralyov.com/test/modtest2\n"},
				{"dmitri.shuralyov.com/test/modtest2@v0.0.0/p.go", "package p\n\n// Life is the answer.\nconst Life = 42\n"},
			} {
				f, err := z.Create(file.Name)
				if err != nil {
					panic(err)
				}
				_, err = f.Write([]byte(file.Body))
				if err != nil {
					panic(err)
				}
			}
			err := z.Close()
			if err != nil {
				panic(err)
			}
		})
	}
}
