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
	"syscall"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

func initRepositories(root string, notifications notifications.Service, users users.Service) error {
	gitUploadPack, err := exec.LookPath("git-upload-pack")
	if err != nil {
		return err
	}
	gitReceivePack, err := exec.LookPath("git-receive-pack")
	if err != nil {
		return err
	}

	repo := "dmitri.shuralyov.com/kebabcase"
	packageHandler := cookieAuth{httputil.ErrorHandler(users, (&packageHandler{
		Repo:          repo,
		notifications: notifications,
		users:         users,
	}).ServeHTTP)}
	h := &gitHandler{
		GitUploadPack:  gitUploadPack,
		GitReceivePack: gitReceivePack,
		users:          users,

		Path:    repo[len("dmitri.shuralyov.com"):],
		RepoDir: filepath.Join(root, filepath.FromSlash(repo)),
		NonGit:  packageHandler,
	}
	http.Handle("/kebabcase", h)
	http.Handle("/kebabcase/", h)
	return nil
}

type gitHandler struct {
	GitUploadPack  string // Path to git-upload-pack binary.
	GitReceivePack string // Path to git-receive-pack binary.
	users          users.Service

	// Repo-specific fields.
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

	case h.Path + "/info/refs?service=git-receive-pack":
		if req.Method != http.MethodGet {
			httperror.HandleMethod(w, httperror.Method{Allowed: []string{http.MethodGet}})
			return
		}

		// Authorization check.
		user, _ := lookUpUserViaBasicAuth(req, h.users)
		if user == nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		} else if !user.SiteAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}

		cmd := exec.CommandContext(req.Context(), h.GitReceivePack, "--advertise-refs", ".")
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
			log.Printf("git-receive-pack command failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-git-receive-pack-advertisement")
		_, err = io.WriteString(w, "001f# service=git-receive-pack\n0000")
		if err != nil {
			log.Println(err)
			return
		}
		_, err = io.Copy(w, &buf)
		if err != nil {
			log.Println(err)
		}
	case h.Path + "/git-receive-pack":
		if req.Method != http.MethodPost {
			httperror.HandleMethod(w, httperror.Method{Allowed: []string{http.MethodPost}})
			return
		}
		if req.Header.Get("Content-Type") != "application/x-git-receive-pack-request" {
			err := fmt.Errorf("unexpected Content-Type: %v", req.Header.Get("Content-Type"))
			httperror.HandleBadRequest(w, httperror.BadRequest{Err: err})
			return
		}

		// Authorization check.
		user, _ := lookUpUserViaBasicAuth(req, h.users)
		if user == nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		} else if !user.SiteAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}

		cmd := exec.CommandContext(req.Context(), h.GitReceivePack, "--stateless-rpc", ".")
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
		if err != nil {
			log.Printf("git-receive-pack command failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
		_, err = io.Copy(w, &buf)
		if err != nil {
			log.Println(err)
		}

	default:
		if req.URL.Path != h.Path {
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		}
		h.NonGit.ServeHTTP(w, req)
	}
}
