package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	pathpkg "path"
	"strings"

	"github.com/satori/go.uuid"
	"github.com/shurcooL/users"
	"github.com/shurcooL/webdavfs/vfsutil"
	"golang.org/x/net/webdav"
)

type userContentHandler struct {
	store webdav.FileSystem
	users users.Service
}

func (uc userContentHandler) Upload(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return MethodError{Allowed: []string{"POST"}}
	}

	type uploadResponse struct {
		URL   string `json:",omitempty"`
		Error string `json:",omitempty"`
	}

	user, err := uc.users.GetAuthenticated(req.Context())
	if err != nil {
		return JSONResponse{uploadResponse{Error: err.Error()}}
	}
	if user.ID == 0 {
		return JSONResponse{uploadResponse{Error: os.ErrPermission.Error()}}
	}

	if contentType := req.Header.Get("Content-Type"); contentType != "image/png" {
		return JSONResponse{uploadResponse{Error: fmt.Sprintf("Content-Type %q is not supported", contentType)}}
	}

	dir := fmt.Sprintf("/%d@%s", user.ID, user.Domain)
	err = vfsutil.MkdirAll(uc.store, dir, 0755)
	if err != nil {
		return JSONResponse{uploadResponse{Error: err.Error()}}
	}

	name := uuid.NewV4().String() + ".png"
	path := pathpkg.Join(dir, name)
	f, err := uc.store.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return JSONResponse{uploadResponse{Error: err.Error()}}
	}

	const maxSizeBytes = 10 * 1024 * 1024
	body := http.MaxBytesReader(w, req.Body, maxSizeBytes) // The http.Server will close the request body, the handler does not need to.
	_, err = io.Copy(f, body)
	if err != nil {
		f.Close()
		uc.store.RemoveAll(path)
		return JSONResponse{uploadResponse{Error: err.Error()}}
	}
	err = f.Close()
	if err != nil {
		uc.store.RemoveAll(path)
		return JSONResponse{uploadResponse{Error: err.Error()}}
	}

	return JSONResponse{uploadResponse{URL: pathpkg.Join("/usercontent", path)}}
}

func (uc userContentHandler) Serve(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return MethodError{Allowed: []string{"GET"}}
	}

	f, err := vfsutil.Open(uc.store, req.URL.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, req, "", fi.ModTime(), f)
	return nil
}

// errorHandler factors error handling out of the HTTP handler.
type errorHandler struct {
	handler func(w http.ResponseWriter, req *http.Request) error
}

func (h errorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// TODO: Factor this out into user middleware?
	u, err := getUser(req)
	if err == errBadAccessToken {
		// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
		http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
	}
	req = req.WithContext(context.WithValue(req.Context(), userContextKey, u))

	rw := &responseWriter{ResponseWriter: w}
	err = h.handler(rw, req)
	switch {
	case err != nil && rw.WroteHeader:
		// The header has already been written, so it's too late to send
		// a different status code. Just log the error and move on.
		log.Println(err)
	case IsMethodError(err):
		w.Header().Set("Allow", strings.Join(err.(MethodError).Allowed, ", "))
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
	case IsHTTPError(err):
		http.Error(w, err.Error(), err.(HTTPError).Code)
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
		http.Error(w, err.Error(), http.StatusNotFound)
	case os.IsPermission(err):
		log.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
	default:
		// Do nothing.
	case err != nil:
		log.Println(err)
		// TODO: Only display error details to SiteAdmin users?
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

// MethodError is an error type used for methods that aren't allowed.
type MethodError struct {
	Allowed []string // Allowed methods.
}

func (m MethodError) Error() string {
	return fmt.Sprintf("method should be %v", strings.Join(m.Allowed, " or "))
}

func IsMethodError(err error) bool {
	_, ok := err.(MethodError)
	return ok
}
