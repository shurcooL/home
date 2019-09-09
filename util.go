package main

import (
	"net/http"
	"net/url"
	"os"

	"github.com/shurcooL/httpgzip"
)

// copyRequestAndURL returns a copy of req and its URL field.
func copyRequestAndURL(req *http.Request) *http.Request {
	r := *req
	u := *req.URL
	r.URL = &u
	return &r
}

// stripPrefix returns request r with prefix of length prefixLen stripped from r.URL.Path.
// prefixLen must not be longer than len(r.URL.Path), otherwise stripPrefix panics.
// If r.URL.Path is empty after the prefix is stripped, the path is changed to "/".
func stripPrefix(r *http.Request, prefixLen int) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = r.URL.Path[prefixLen:]
	if r2.URL.Path == "" {
		r2.URL.Path = "/"
	}
	return r2
}

// serveFile opens the file at path and serves it using httpgzip.ServeContent.
func serveFile(w http.ResponseWriter, req *http.Request, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	httpgzip.ServeContent(w, req, fi.Name(), fi.ModTime(), f)
	return nil
}
