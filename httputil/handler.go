package httputil

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
	"golang.org/x/net/http/httpguts"
)

// ErrorHandler factors error handling out of the HTTP handler.
func ErrorHandler(users users.Service, handler func(w http.ResponseWriter, req *http.Request) error) http.Handler {
	return &errorHandler{handler: handler, users: users}
}

type errorHandler struct {
	handler func(w http.ResponseWriter, req *http.Request) error
	users   interface {
		GetAuthenticated(context.Context) (users.User, error)
	}
}

func (h *errorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rw := &headerResponseWriter{ResponseWriter: w}
	err := h.handler(rw, req)
	if err == nil {
		// Do nothing.
		return
	}
	if err != nil && rw.WroteHeader {
		// The header has already been written, so it's too late to send
		// a different status code. Just log the error and move on.
		log.Println(err)
		return
	}
	if err, ok := httperror.IsMethod(err); ok {
		httperror.HandleMethod(w, err)
		return
	}
	if err, ok := httperror.IsRedirect(err); ok {
		http.Redirect(w, req, err.URL, http.StatusSeeOther)
		return
	}
	if err, ok := httperror.IsBadRequest(err); ok {
		httperror.HandleBadRequest(w, err)
		return
	}
	if err, ok := httperror.IsHTTP(err); ok {
		code := err.Code
		error := fmt.Sprintf("%d %s", code, http.StatusText(code))
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, code)
		return
	}
	if err, ok := httperror.IsJSONResponse(err); ok {
		w.Header().Set("Content-Type", "application/json")
		jw := json.NewEncoder(w)
		jw.SetIndent("", "\t")
		err := jw.Encode(err.V)
		if err != nil {
			log.Println("error encoding JSONResponse:", err)
		}
		return
	}
	if os.IsNotExist(err) {
		log.Println(err)
		error := "404 Not Found"
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusNotFound)
		return
	}
	if os.IsPermission(err) {
		// TODO: Factor in a GetAuthenticatedSpec.ID == 0 check out here. (But this shouldn't apply for APIs.)
		log.Println(err)
		error := "403 Forbidden"
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusForbidden)
		return
	}

	log.Println(err)
	error := "500 Internal Server Error"
	if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
		error += "\n\n" + err.Error()
	}
	http.Error(w, error, http.StatusInternalServerError)
}

// headerResponseWriter wraps a real http.ResponseWriter and captures
// whether or not the header has been written.
type headerResponseWriter struct {
	http.ResponseWriter

	WroteHeader bool // Write or WriteHeader was called.
}

func (rw *headerResponseWriter) Write(p []byte) (n int, err error) {
	rw.WroteHeader = true
	return rw.ResponseWriter.Write(p)
}
func (rw *headerResponseWriter) WriteHeader(code int) {
	rw.WroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

// GzipHandler applies gzip compression on top of handler, unless handler
// has already handled it (i.e., the "Content-Encoding" header is set).
func GzipHandler(handler http.Handler) http.Handler {
	return gzipHandler{handler}
}

type gzipHandler struct {
	handler http.Handler
}

func (h gzipHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// If request doesn't accept gzip encoding, serve without compression.
	if !httpguts.HeaderValuesContainsToken(req.Header["Accept-Encoding"], "gzip") {
		h.handler.ServeHTTP(w, req)
		return
	}

	// Otherwise, use gzipResponseWriter to start gzip compression when WriteHeader
	// is called, but only if the handler didn't already take care of it.
	rw := &gzipResponseWriter{ResponseWriter: w}
	defer rw.Close()
	h.handler.ServeHTTP(rw, req)
}

// gzipResponseWriter starts gzip compression when WriteHeader is called, unless compression
// has already been applied by that time (i.e., the "Content-Encoding" header is set).
type gzipResponseWriter struct {
	http.ResponseWriter

	// These fields are set by setWriterAndCloser
	// during first call to Write or WriteHeader.
	w io.Writer // When set, must be non-nil.
	c io.Closer // May be nil.
}

func (rw *gzipResponseWriter) WriteHeader(code int) {
	if rw.w != nil {
		panic(fmt.Errorf("internal error: gzipResponseWriter: WriteHeader called twice or after Write"))
	}
	rw.setWriterAndCloser()
	rw.ResponseWriter.WriteHeader(code)
}
func (rw *gzipResponseWriter) Write(p []byte) (n int, err error) {
	if rw.w == nil {
		rw.setWriterAndCloser()
	}
	return rw.w.Write(p)
}

func (rw *gzipResponseWriter) setWriterAndCloser() {
	if _, ok := rw.Header()["Content-Encoding"]; ok {
		// Compression already handled by the handler.
		rw.w = rw.ResponseWriter
		return
	}

	// Update headers, start using a gzip writer.
	rw.Header().Set("Content-Encoding", "gzip")
	rw.Header().Del("Content-Length")
	gw := gzip.NewWriter(rw.ResponseWriter)
	rw.w = gw
	rw.c = gw
}

func (rw *gzipResponseWriter) Close() {
	if rw.c == nil {
		return
	}
	err := rw.c.Close()
	if err != nil {
		log.Printf("gzipResponseWriter.Close: error closing *gzip.Writer: %v", err)
	}
}
