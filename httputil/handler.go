package httputil

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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
	switch {
	case err != nil && rw.WroteHeader:
		// The header has already been written, so it's too late to send
		// a different status code. Just log the error and move on.
		log.Println(err)
	case IsMethodError(err):
		w.Header().Set("Allow", strings.Join(err.(MethodError).Allowed, ", "))
		error := fmt.Sprintf("405 Method Not Allowed\n\n%v", err)
		http.Error(w, error, http.StatusMethodNotAllowed)
	case IsRedirect(err):
		http.Redirect(w, req, err.(Redirect).URL, http.StatusSeeOther)
	case IsHTTPError(err):
		code := err.(HTTPError).Code
		error := fmt.Sprintf("%d %s", code, http.StatusText(code))
		if code == http.StatusBadRequest {
			error += "\n\n" + err.Error()
		} else if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, code)
	case IsJSONResponse(err):
		w.Header().Set("Content-Type", "application/json")
		jw := json.NewEncoder(w)
		jw.SetIndent("", "\t")
		err := jw.Encode(err.(JSONResponse).V)
		if err != nil {
			log.Println("error encoding JSONResponse:", err)
		}
	case os.IsNotExist(err):
		log.Println(err)
		error := "404 Not Found"
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusNotFound)
	case os.IsPermission(err):
		// TODO: Factor in a GetAuthenticatedSpec.ID == 0 check out here. (But this shouldn't apply for APIs.)
		log.Println(err)
		error := "403 Forbidden"
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusUnauthorized)
	default:
		// Do nothing.
	case err != nil:
		log.Println(err)
		error := "500 Internal Server Error"
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusInternalServerError)
	}
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
