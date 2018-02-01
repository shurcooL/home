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
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
)

func initGitUsers(usersService users.Service) (gitUsers map[string]users.User, err error) {
	// TODO: Add support for additional git users.
	gitUsers = make(map[string]users.User) // Key is lower git author email.
	shurcool, err := usersService.Get(context.Background(), shurcool)
	if err != nil {
		return nil, err
	}
	gitUsers[strings.ToLower(shurcool.Email)] = shurcool
	gitUsers[strings.ToLower("shurcooL@gmail.com")] = shurcool // Previous email.
	return gitUsers, nil
}

func initGitHandler(code code.Code, reposDir string, events events.Service, users users.Service, gitUsers map[string]users.User) (*gitHandler, error) {
	gitUploadPack, err := exec.LookPath("git-upload-pack")
	if err != nil {
		return nil, err
	}
	gitReceivePack, err := exec.LookPath("git-receive-pack")
	if err != nil {
		return nil, err
	}
	return &gitHandler{
		code:           code,
		reposDir:       reposDir,
		events:         events,
		users:          users,
		gitUsers:       gitUsers,
		gitUploadPack:  gitUploadPack,
		gitReceivePack: gitReceivePack,
	}, nil
}

type gitHandler struct {
	code     code.Code
	reposDir string
	events   events.Service
	users    users.Service
	gitUsers map[string]users.User // Key is lower git author email.

	gitUploadPack  string // Path to git-upload-pack binary.
	gitReceivePack string // Path to git-receive-pack binary.
}

func (h *gitHandler) ServeGitMaybe(w http.ResponseWriter, req *http.Request) (ok bool) {
	switch url := req.URL.String(); {
	case strings.HasSuffix(url, "/info/refs?service=git-upload-pack"):
		repoRoot := "dmitri.shuralyov.com" + url[:len(url)-len("/info/refs?service=git-upload-pack")]
		if dir, ok := h.code.ByImportPath[repoRoot]; !ok || !dir.IsRepository() {
			return false
		}
		h.serveGitInfoRefsUploadPack(w, req, repoInfo{
			Spec: repoRoot,
			Path: repoRoot[len("dmitri.shuralyov.com"):],
			Dir:  filepath.Join(h.reposDir, filepath.FromSlash(repoRoot)),
		})
		return true
	case strings.HasSuffix(url, "/git-upload-pack"):
		repoRoot := "dmitri.shuralyov.com" + url[:len(url)-len("/git-upload-pack")]
		if dir, ok := h.code.ByImportPath[repoRoot]; !ok || !dir.IsRepository() {
			return false
		}
		h.serveGitUploadPack(w, req, repoInfo{
			Spec: repoRoot,
			Path: repoRoot[len("dmitri.shuralyov.com"):],
			Dir:  filepath.Join(h.reposDir, filepath.FromSlash(repoRoot)),
		})
		return true
	case strings.HasSuffix(url, "/info/refs?service=git-receive-pack"):
		repoRoot := "dmitri.shuralyov.com" + url[:len(url)-len("/info/refs?service=git-receive-pack")]
		if dir, ok := h.code.ByImportPath[repoRoot]; !ok || !dir.IsRepository() {
			return false
		}
		h.serveGitInfoRefsReceivePack(w, req, repoInfo{
			Spec: repoRoot,
			Path: repoRoot[len("dmitri.shuralyov.com"):],
			Dir:  filepath.Join(h.reposDir, filepath.FromSlash(repoRoot)),
		})
		return true
	case strings.HasSuffix(url, "/git-receive-pack"):
		repoRoot := "dmitri.shuralyov.com" + url[:len(url)-len("/git-receive-pack")]
		if dir, ok := h.code.ByImportPath[repoRoot]; !ok || !dir.IsRepository() {
			return false
		}
		h.serveGitReceivePack(w, req, repoInfo{
			Spec: repoRoot,
			Path: repoRoot[len("dmitri.shuralyov.com"):],
			Dir:  filepath.Join(h.reposDir, filepath.FromSlash(repoRoot)),
		})
		return true
	default:
		return false
	}
}

func (h *gitHandler) serveGitInfoRefsUploadPack(w http.ResponseWriter, req *http.Request, repo repoInfo) {
	if req.Method != http.MethodGet {
		httperror.HandleMethod(w, httperror.Method{Allowed: []string{http.MethodGet}})
		return
	}
	cmd := exec.CommandContext(req.Context(), h.gitUploadPack, "--strict", "--advertise-refs", ".")
	cmd.Dir = repo.Dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Start()
	if os.IsNotExist(err) {
		http.Error(w, "404 Not Found", http.StatusNotFound)
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
}
func (h *gitHandler) serveGitUploadPack(w http.ResponseWriter, req *http.Request, repo repoInfo) {
	if req.Method != http.MethodPost {
		httperror.HandleMethod(w, httperror.Method{Allowed: []string{http.MethodPost}})
		return
	}
	if req.Header.Get("Content-Type") != "application/x-git-upload-pack-request" {
		err := fmt.Errorf("unexpected Content-Type: %v", req.Header.Get("Content-Type"))
		httperror.HandleBadRequest(w, httperror.BadRequest{Err: err})
		return
	}
	cmd := exec.CommandContext(req.Context(), h.gitUploadPack, "--strict", "--stateless-rpc", ".")
	cmd.Dir = repo.Dir
	cmd.Stdin = req.Body
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Start()
	if os.IsNotExist(err) {
		http.Error(w, "404 Not Found", http.StatusNotFound)
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
}

func (h *gitHandler) serveGitInfoRefsReceivePack(w http.ResponseWriter, req *http.Request, repo repoInfo) {
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

	cmd := exec.CommandContext(req.Context(), h.gitReceivePack, "--advertise-refs", ".")
	cmd.Dir = repo.Dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Start()
	if os.IsNotExist(err) {
		http.Error(w, "404 Not Found", http.StatusNotFound)
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
}
func (h *gitHandler) serveGitReceivePack(w http.ResponseWriter, req *http.Request, repo repoInfo) {
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

	cmd := exec.CommandContext(req.Context(), h.gitReceivePack, "--stateless-rpc", ".")
	cmd.Dir = repo.Dir
	rpc := &githttp.RpcReader{
		Reader: req.Body,
		Rpc:    "receive-pack",
	}
	cmd.Stdin = rpc
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Start()
	if os.IsNotExist(err) {
		http.Error(w, "404 Not Found", http.StatusNotFound)
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
	now := time.Now().UTC()
	for _, e := range rpc.Events {
		evt := event.Event{
			Time:      now,
			Actor:     *user,
			Container: repo.Spec,
		}
		const zero = "0000000000000000000000000000000000000000"
		switch {
		case e.Type == githttp.PUSH && e.Last != zero && e.Commit != zero:
			commits, err := listCommits(repo, e.Last, e.Commit, h.gitUsers)
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
}

// listCommits returns a list of commits in repo from base to head.
func listCommits(repo repoInfo, base, head string, gitUsers map[string]users.User) ([]event.Commit, error) {
	r := &gitcmd.Repository{Dir: repo.Dir}
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
			HTMLURL:         route.RepoCommit(repo.Path) + "/" + string(c.ID),
		})
	}
	return commits, nil
}
