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
// If users is nil, it treats all requests as made by an unauthenticated user.
func ErrorHandler(
	users interface {
		// GetAuthenticated fetches the currently authenticated user,
		// or User{UserSpec: UserSpec{ID: 0}} if there is no authenticated user.
		GetAuthenticated(context.Context) (users.User, error)
	},
	handler func(w http.ResponseWriter, req *http.Request) error,
) http.Handler {
	if users == nil {
		users = noUsers{}
	}
	return &errorHandler{handler: handler, users: users}
}

type errorHandler struct {
	handler func(w http.ResponseWriter, req *http.Request) error
	users   interface {
		GetAuthenticated(context.Context) (users.User, error)
	}
}

func (h *errorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rw := &headerResponseWriter{ResponseWriter: w, Flusher: w.(http.Flusher)}
	err := h.handler(rw, req)
	handleError(w, req, err, h.users, rw.WroteHeader)
}

// ErrorHandleMaybe factors error handling out of the HTTP maybe handler.
// If users is nil, it treats all requests as made by an unauthenticated user.
func ErrorHandleMaybe(
	w http.ResponseWriter, req *http.Request,
	users interface {
		// GetAuthenticated fetches the currently authenticated user,
		// or User{UserSpec: UserSpec{ID: 0}} if there is no authenticated user.
		GetAuthenticated(context.Context) (users.User, error)
	},
	// maybeHandler serves an HTTP request, if it matches.
	// It returns httperror.NotHandle if the HTTP request was explicitly not handled.
	maybeHandler func(w http.ResponseWriter, req *http.Request) error,
) (ok bool) {
	if users == nil {
		users = noUsers{}
	}
	rw := &headerResponseWriter{ResponseWriter: w, Flusher: w.(http.Flusher)}
	err := maybeHandler(rw, req)
	if err == httperror.NotHandle {
		if rw.WroteHeader {
			panic(fmt.Errorf("internal error: maybe handler wrote HTTP header and then returned httperror.NotHandle"))
		}
		// The request was explicitly not handled by the maybe handler.
		// Do nothing, return ok==false.
		return false
	}
	// The request was handled by the maybe handler.
	// Handle error and return ok==true.
	handleError(w, req, err, users, rw.WroteHeader)
	return true
}

// handleError handles error err, which may be nil.
func handleError(
	w http.ResponseWriter, req *http.Request,
	err error,
	users interface {
		// GetAuthenticated fetches the currently authenticated user,
		// or User{UserSpec: UserSpec{ID: 0}} if there is no authenticated user.
		GetAuthenticated(context.Context) (users.User, error)
	},
	wroteHeader bool,
) {
	if err == nil {
		// Do nothing.
		return
	}
	if err != nil && wroteHeader {
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
		if user, e := users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, code)
		return
	}
	if err, ok := httperror.IsJSONResponse(err); ok {
		w.Header().Set("Content-Type", "application/json")
		jw := json.NewEncoder(w)
		jw.SetIndent("", "\t")
		jw.SetEscapeHTML(false)
		err := jw.Encode(err.V)
		if err != nil {
			log.Println("error encoding JSONResponse:", err)
		}
		return
	}
	if os.IsNotExist(err) {
		log.Println(err)
		error := "404 Not Found"
		if user, e := users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusNotFound)
		return
	}
	if os.IsPermission(err) {
		// TODO: Factor in a GetAuthenticatedSpec.ID == 0 check out here. (But this shouldn't apply for APIs.)
		log.Println(err)
		error := "403 Forbidden"
		if user, e := users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusForbidden)
		return
	}

	log.Println(err)
	error := "500 Internal Server Error"
	if user, e := users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
		error += "\n\n" + err.Error()
	}
	http.Error(w, error, http.StatusInternalServerError)
}

// headerResponseWriter wraps a real http.ResponseWriter and captures
// whether or not the header has been written.
type headerResponseWriter struct {
	http.ResponseWriter
	http.Flusher

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
	rw := &gzipResponseWriter{ResponseWriter: w, Flusher: w.(http.Flusher)}
	defer rw.Close()
	h.handler.ServeHTTP(rw, req)
}

// gzipResponseWriter starts gzip compression when WriteHeader is called, unless compression
// has already been applied by that time (i.e., the "Content-Encoding" header is set).
// Close must be called when done with gzipResponseWriter.
type gzipResponseWriter struct {
	http.ResponseWriter
	http.Flusher

	// These fields are set by setWriterAndCloser
	// during first call to Write or WriteHeader.
	w io.Writer   // When set, must be non-nil.
	c flushCloser // May be nil.
}

type flushCloser interface {
	Flush() error
	io.Closer
}

func (rw *gzipResponseWriter) WriteHeader(code int) {
	if rw.w != nil {
		panic(fmt.Errorf("internal error: gzipResponseWriter: WriteHeader called twice or after Write"))
	}
	rw.setWriterAndCloser(code)
	rw.ResponseWriter.WriteHeader(code)
}
func (rw *gzipResponseWriter) Write(p []byte) (n int, err error) {
	if rw.w == nil {
		rw.setWriterAndCloser(http.StatusOK)
	}
	return rw.w.Write(p)
}

func (rw *gzipResponseWriter) setWriterAndCloser(status int) {
	if _, ok := rw.Header()["Content-Encoding"]; ok {
		// Compression already handled by the handler.
		rw.w = rw.ResponseWriter
		return
	}

	if !bodyAllowedForStatus(status) {
		// Body not allowed, don't use gzip.
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

func (rw *gzipResponseWriter) Flush() {
	if rw.c != nil {
		err := rw.c.Flush()
		if err != nil {
			log.Printf("gzipResponseWriter.Flush: error flushing *gzip.Writer: %v", err)
		}
	}
	rw.Flusher.Flush()
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

// bodyAllowedForStatus reports whether a given response status code
// permits a body. See RFC 7230, section 3.3.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status < 200:
		return false
	case status == http.StatusNoContent:
		return false
	case status == http.StatusNotModified:
		return false
	default:
		return true
	}
}

// noUsers implements a subset of the users.Service interface
// relevant to fetching the currently authenticated user.
//
// It does not perform authentication, instead opting to
// always report that there is an unauthenticated user.
type noUsers struct{}

// GetAuthenticated always reports that there is an unauthenticated user.
func (noUsers) GetAuthenticated(context.Context) (users.User, error) {
	return users.User{UserSpec: users.UserSpec{ID: 0, Domain: ""}}, nil
}
