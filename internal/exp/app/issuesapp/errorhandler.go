package issuesapp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
)

// errorHandler factors error handling out of the HTTP handler.
type errorHandler struct {
	handler func(w http.ResponseWriter, req *http.Request) error
	users   interface {
		GetAuthenticated(context.Context) (users.User, error)
	} // May be nil if there's no users service.
}

func (h *errorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rw := &responseWriter{ResponseWriter: w}
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
		if user, e := h.getAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, code)
		return
	}
	if os.IsNotExist(err) {
		log.Println(err)
		error := "404 Not Found"
		if user, e := h.getAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusNotFound)
		return
	}
	if os.IsPermission(err) {
		log.Println(err)
		error := "403 Forbidden"
		if user, e := h.getAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusForbidden)
		return
	}

	log.Println(err)
	error := "500 Internal Server Error"
	if user, e := h.getAuthenticated(req.Context()); e == nil && user.SiteAdmin {
		error += "\n\n" + err.Error()
	}
	http.Error(w, error, http.StatusInternalServerError)
}

func (h *errorHandler) getAuthenticated(ctx context.Context) (users.User, error) {
	if h.users == nil {
		return users.User{}, errors.New("no users service")
	}
	return h.users.GetAuthenticated(ctx)
}

// responseWriter wraps a real http.ResponseWriter and captures
// whether or not the header has been written.
type responseWriter struct {
	http.ResponseWriter

	WroteHeader bool // Write or WriteHeader was called.
}

func (rw *responseWriter) Write(p []byte) (n int, err error) {
	rw.WroteHeader = true
	return rw.ResponseWriter.Write(p)
}
func (rw *responseWriter) WriteHeader(code int) {
	rw.WroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}
