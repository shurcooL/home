package main

import (
	"bytes"
	"context"
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

	"github.com/AaronO/go-git-http"
	"github.com/shurcooL/events"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
)

func initRepositories(root string, notifications notifications.Service, events events.Service, usersService users.Service) error {
	gitUploadPack, err := exec.LookPath("git-upload-pack")
	if err != nil {
		return err
	}
	gitReceivePack, err := exec.LookPath("git-receive-pack")
	if err != nil {
		return err
	}

	// TODO: Add support for additional git users.
	gitUsers := make(map[string]users.User)
	shurcool, err := usersService.Get(context.Background(), shurcool)
	if err != nil {
		return err
	}
	gitUsers[strings.ToLower(shurcool.Email)] = shurcool

	repo := "dmitri.shuralyov.com/kebabcase"
	repoDir := filepath.Join(root, filepath.FromSlash(repo))
	packageHandler := cookieAuth{httputil.ErrorHandler(usersService, (&packageHandler{
		Repo:          repo,
		notifications: notifications,
		users:         usersService,
	}).ServeHTTP)}
	commitsHandler := cookieAuth{httputil.ErrorHandler(usersService, (&commitsHandler{
		Repo:          repo,
		RepoDir:       repoDir,
		notifications: notifications,
		users:         usersService,
		gitUsers:      gitUsers,
	}).ServeHTTP)}
	h := &gitHandler{
		GitUploadPack:  gitUploadPack,
		GitReceivePack: gitReceivePack,
		events:         events,
		users:          usersService,
		gitUsers:       gitUsers,

		Repo:    repo,
		Path:    repo[len("dmitri.shuralyov.com"):],
		RepoDir: repoDir,
		Index:   packageHandler,
		Commits: commitsHandler,
	}
	http.Handle("/kebabcase", h)
	http.Handle("/kebabcase/", h)
	return nil
}

type gitHandler struct {
	GitUploadPack  string // Path to git-upload-pack binary.
	GitReceivePack string // Path to git-receive-pack binary.
	events         events.Service
	users          users.Service
	gitUsers       map[string]users.User // Key is lower git author email.

	// Repo-specific fields.
	Repo    string       // Repo root. E.g., "dmitri.shuralyov.com/kebabcase".
	Path    string       // Path corresponding to repo root, without domain. E.g., "/kebabcase".
	RepoDir string       // Path to repository directory on disk.
	Index   http.Handler // Handler for index page.
	Commits http.Handler // Handler for commits page.
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
		rpc := &githttp.RpcReader{
			Reader: req.Body,
			Rpc:    "receive-pack",
		}
		cmd.Stdin = rpc
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

		// Log events.
		now := time.Now()
		for _, e := range rpc.Events {
			evt := event.Event{
				Time:      now,
				Actor:     *user,
				Container: h.Repo,
			}
			const zero = "0000000000000000000000000000000000000000"
			switch {
			case e.Type == githttp.PUSH && e.Last != zero && e.Commit != zero:
				commits, err := listCommits(h.RepoDir, e.Last, e.Commit, h.gitUsers)
				if err != nil {
					log.Println("listCommits:", err)
				}
				evt.Payload = event.Push{
					Branch:  e.Branch,
					Head:    e.Commit,
					Before:  e.Last,
					Commits: commits,
				}
			case e.Type == githttp.PUSH && e.Last == zero && e.Commit != zero:
				evt.Payload = event.Create{
					Type: "branch", Name: e.Branch,
				}
			case e.Type == githttp.PUSH && e.Last != zero && e.Commit == zero:
				evt.Payload = event.Delete{
					Type: "branch", Name: e.Branch,
				}
			case e.Type == githttp.TAG && e.Last == zero && e.Commit != zero:
				evt.Payload = event.Create{
					Type: "tag", Name: e.Tag,
				}
			case e.Type == githttp.TAG && e.Last != zero && e.Commit == zero:
				evt.Payload = event.Delete{
					Type: "tag", Name: e.Tag,
				}
			default:
				log.Printf("unsupported git event: %+v\n", e)
				continue
			}
			err := h.events.Log(req.Context(), evt)
			if err != nil {
				log.Println("h.events.Log:", err)
			}
		}

	default:
		switch req.URL.Path {
		case h.Path:
			h.Index.ServeHTTP(w, req)
		case h.Path + "/commits":
			h.Commits.ServeHTTP(w, req)
		default:
			http.Error(w, "404 Not Found", http.StatusNotFound)
		}
	}
}

// listCommits returns a list of commits in repoDir from base to head.
func listCommits(repoDir, base, head string, gitUsers map[string]users.User) ([]event.Commit, error) {
	r := &gitcmd.Repository{Dir: repoDir}
	cs, _, err := r.Commits(vcs.CommitsOptions{
		Head:    vcs.CommitID(head),
		Base:    vcs.CommitID(base),
		NoTotal: true,
	})
	if err != nil {
		return nil, err
	}
	var commits []event.Commit
	for i := len(cs) - 1; i >= 0; i-- {
		c := cs[i]

		user, ok := gitUsers[strings.ToLower(c.Author.Email)]
		if !ok {
			user = users.User{
				Name:      c.Author.Name,
				Email:     c.Author.Email,
				AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96", // TODO: Use email.
			}
		}

		commits = append(commits, event.Commit{
			SHA:             string(c.ID),
			CommitMessage:   c.Message,
			AuthorAvatarURL: user.AvatarURL,
			// TODO: Set HTMLURL once there's a commit page available.
		})
	}
	return commits, nil
}
