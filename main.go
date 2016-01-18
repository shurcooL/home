// home is my personal website.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"time"

	//"github.com/shurcooL/go/raw_file_server"
	"github.com/shurcooL/go-goon"
)

var httpFlag = flag.String("http", ":80", "Listen for HTTP connections on this address.")

func NewRouter() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		if req.URL.Path == "/" {
			req.URL.Path = "/index.html"
		}

		req.URL.Scheme = "https"
		req.URL.Host = "dl.dropboxusercontent.com"
		req.Host = "dl.dropboxusercontent.com"
		req.URL.Path = "/u/8554242/dmitri" + req.URL.Path
	}
	return &httputil.ReverseProxy{Director: director}
}

func main() {
	flag.Parse()

	fmt.Println("Started.")

	mux := http.NewServeMux()
	mux.Handle("/robots.txt", http.NotFoundHandler())
	//mux.Handle("/", http.RedirectHandler("http://goo.gl/bijah", http.StatusTemporaryRedirect))
	//mux.Handle("/", NewRouter())
	mux.Handle("/", http.FileServer(http.Dir(filepath.Join("Dropbox", "Public", "dmitri"))))
	//mux.Handle("/", raw_file_server.NewUsingHttpFs(http.Dir("./Dropbox/Public/dmitri/")))

	//handler := NewCountingHandler(mux)
	handler := mux

	if err := http.ListenAndServe(*httpFlag, handler); err != nil {
		panic(err)
	}
}

// ---

type countingHandler struct {
	count map[string]uint64

	handler http.Handler
}

func NewCountingHandler(handler http.Handler) http.Handler {
	return &countingHandler{
		count:   make(map[string]uint64),
		handler: handler,
	}
}

func (ch *countingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/stats" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, goon.Sdump(ch.count))
		return
	}

	ch.count[time.Now().UTC().Format("2006-01-02 ")+req.Host]++

	// Dump request to stdout.
	if dump, err := httputil.DumpRequest(req, true); err == nil {
		fmt.Println(string(dump))
	}
	goon.DumpExpr(req.URL.Query())
	goon.DumpExpr(req.RemoteAddr)
	fmt.Println("-----")

	ch.handler.ServeHTTP(w, req)
}
