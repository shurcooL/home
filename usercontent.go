package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	pathpkg "path"

	"github.com/satori/go.uuid"
	"github.com/shurcooL/home/httputil"
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
		return httputil.MethodError{Allowed: []string{"POST"}}
	}

	type uploadResponse struct {
		URL   string `json:",omitempty"`
		Error string `json:",omitempty"`
	}

	user, err := uc.users.GetAuthenticated(req.Context())
	if err != nil {
		return httputil.JSONResponse{V: uploadResponse{Error: err.Error()}}
	}
	if user.ID == 0 {
		return httputil.JSONResponse{V: uploadResponse{Error: os.ErrPermission.Error()}}
	}

	if contentType := req.Header.Get("Content-Type"); contentType != "image/png" {
		return httputil.JSONResponse{V: uploadResponse{Error: fmt.Sprintf("Content-Type %q is not supported", contentType)}}
	}

	dir := fmt.Sprintf("/%d@%s", user.ID, user.Domain)
	err = vfsutil.MkdirAll(uc.store, dir, 0755)
	if err != nil {
		return httputil.JSONResponse{V: uploadResponse{Error: err.Error()}}
	}

	name := uuid.NewV4().String() + ".png"
	path := pathpkg.Join(dir, name)
	f, err := uc.store.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return httputil.JSONResponse{V: uploadResponse{Error: err.Error()}}
	}

	const maxSizeBytes = 10 * 1024 * 1024
	body := http.MaxBytesReader(w, req.Body, maxSizeBytes) // The http.Server will close the request body, the handler does not need to.
	_, err = io.Copy(f, body)
	if err != nil {
		f.Close()
		uc.store.RemoveAll(path)
		return httputil.JSONResponse{V: uploadResponse{Error: err.Error()}}
	}
	err = f.Close()
	if err != nil {
		uc.store.RemoveAll(path)
		return httputil.JSONResponse{V: uploadResponse{Error: err.Error()}}
	}

	return httputil.JSONResponse{V: uploadResponse{URL: pathpkg.Join("/usercontent", path)}}
}

func (uc userContentHandler) Serve(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
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
