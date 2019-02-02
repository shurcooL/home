package main

import (
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
}
