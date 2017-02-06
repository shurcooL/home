package httputil

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
)

// ErrorHandler factors error handling out of the HTTP handler.
func ErrorHandler(users users.Service, handler func(w http.ResponseWriter, req *http.Request) error) http.Handler {
	return &errorHandler{handler: handler, users: users}
}

type errorHandler struct {
	handler func(w http.ResponseWriter, req *http.Request) error
	users   users.Service
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
		if code == http.StatusBadRequest {
			error += "\n\n" + err.Error()
		} else if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
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
		http.Error(w, error, http.StatusUnauthorized)
		return
	}

	log.Println(err)
	error := "500 Internal Server Error"
	if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
		error += "\n\n" + err.Error()
	}
	http.Error(w, error, http.StatusInternalServerError)
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
