package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/shurcooL/httperror"
)

func initRepositories(root string) error {
	gitUploadPack, err := exec.LookPath("git-upload-pack")
	if err != nil {
		return err
	}

	repo := "dmitri.shuralyov.com/kebabcase"
	h := &gitHandler{
		GitUploadPack: gitUploadPack,

		Path:    repo[len("dmitri.shuralyov.com"):],
		RepoDir: filepath.Join(root, filepath.FromSlash(repo)),
		NonGit: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.ServeContent(w, req, ".html", time.Time{}, strings.NewReader(`<html>
	<head>
		<meta name="go-import" content="dmitri.shuralyov.com/kebabcase git https://dmitri.shuralyov.com/kebabcase">
		<meta name="go-source" content="dmitri.shuralyov.com/kebabcase https://dmitri.shuralyov.com/kebabcase https://gotools.org/dmitri.shuralyov.com/kebabcase{/dir} https://gotools.org/dmitri.shuralyov.com/kebabcase{/dir}#{file}-L{line}">
	</head>
	<body>
		<div>Install: <code>go get -u dmitri.shuralyov.com/kebabcase</code></div>
		<div><a href="https://godoc.org/dmitri.shuralyov.com/kebabcase">Documentation</a></div>
		<div><a href="https://gotools.org/dmitri.shuralyov.com/kebabcase">Source</a></div>
		<div><a href="/issues/dmitri.shuralyov.com/kebabcase">Issues</a></div>
	</body>
</html>`))
		}),
	}
	http.Handle("/kebabcase", h)
	http.Handle("/kebabcase/", h)
	return nil
}

type gitHandler struct {
	GitUploadPack string // Path to git-upload-pack binary.

	Path    string       // Path corresponding to repo root, without domain. E.g., "/kebabcase".
	RepoDir string       // Path to repository directory on disk.
	NonGit  http.Handler // Handler for non-git requests.
}

func (h *gitHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.String() {
	case h.Path + "/info/refs?service=git-upload-pack":
		if req.Method != http.MethodGet {
			httperror.HandleMethod(w, httperror.Method{Allowed: []string{http.MethodGet}})
			return
		}
		cmd := exec.CommandContext(req.Context(), h.GitUploadPack, "--strict", "--advertise-refs", ".")
		cmd.Dir = h.RepoDir
		var buf bytes.Buffer
		cmd.Stdout = &buf
		err := cmd.Start()
		if os.IsNotExist(err) {
			http.Error(w, "Not found.", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, fmt.Errorf("could not start command: %v", err).Error(), http.StatusInternalServerError)
			return
		}
		err = cmd.Wait()
		if err != nil {
			log.Printf("git-upload-pack command failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		_, err = io.WriteString(w, "001e# service=git-upload-pack\n0000")
		if err != nil {
			log.Println(err)
			return
		}
		_, err = io.Copy(w, &buf)
		if err != nil {
			log.Println(err)
		}
	case h.Path + "/git-upload-pack":
		if req.Method != http.MethodPost {
			httperror.HandleMethod(w, httperror.Method{Allowed: []string{http.MethodPost}})
			return
		}
		if req.Header.Get("Content-Type") != "application/x-git-upload-pack-request" {
			err := fmt.Errorf("unexpected Content-Type: %v", req.Header.Get("Content-Type"))
			httperror.HandleBadRequest(w, httperror.BadRequest{Err: err})
			return
		}
		cmd := exec.CommandContext(req.Context(), h.GitUploadPack, "--strict", "--stateless-rpc", ".")
		cmd.Dir = h.RepoDir
		cmd.Stdin = req.Body
		var buf bytes.Buffer
		cmd.Stdout = &buf
		err := cmd.Start()
		if os.IsNotExist(err) {
			http.Error(w, "Not found.", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, fmt.Errorf("could not start command: %v", err).Error(), http.StatusInternalServerError)
			return
		}
		err = cmd.Wait()
		if ee, _ := err.(*exec.ExitError); ee != nil && ee.Sys().(syscall.WaitStatus).ExitStatus() == 128 {
			// Supposedly this is "fatal: The remote end hung up unexpectedly"
			// due to git clone --depth=1 or so. Ignore this error.
		} else if err != nil {
			log.Printf("git-upload-pack command failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		_, err = io.Copy(w, &buf)
		if err != nil {
			log.Println(err)
		}
	default:
		h.NonGit.ServeHTTP(w, req)
	}
}
