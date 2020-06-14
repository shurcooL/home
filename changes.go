package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"dmitri.shuralyov.com/route/github"
	"github.com/andygrunwald/go-gerrit"
	"github.com/gregjones/httpcache"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/home/internal/exp/service/change/fs"
	"github.com/shurcooL/home/internal/exp/service/change/gerritapi"
	"github.com/shurcooL/home/internal/exp/service/change/githubapi"
	"github.com/shurcooL/home/internal/exp/service/change/httphandler"
	"github.com/shurcooL/home/internal/exp/service/change/httproute"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

func newChangeService(reactions reactions.Service, users users.Service, router github.Router) change.Service {
	local := &fs.Service{Reactions: reactions}
	dmitshurGitHubChange := githubapi.NewService(
		dmitshurPublicRepoGHV3,
		dmitshurPublicRepoGHV4,
		router,
	)
	gerritClient, err := gerrit.NewClient( // TODO: Auth.
		"https://go-review.googlesource.com/",
		&http.Client{Transport: httpcache.NewMemoryCacheTransport()},
	)
	if err != nil {
		panic(fmt.Errorf("internal error: gerrit.NewClient returned non-nil error: %v", err))
	}
	gerritChange := gerritapi.NewService(gerritClient)
	return dmitshurSeesExternalChanges{
		local:                local,
		dmitshurGitHubChange: dmitshurGitHubChange,
		dmitshurGerritChange: gerritChange,
		users:                users,
	}
}

type changeCounter interface {
	// Count changes.
	Count(ctx context.Context, repo string, opt change.ListOptions) (uint64, error)
}

// initChanges registers handlers for the change service HTTP API,
// and handlers for the changes app.
func initChanges(mux *http.ServeMux, changeService change.Service, changesApp httperror.Handler, users users.Service) {
	// Register change service HTTP API endpoints.
	apiHandler := httphandler.Change{Change: changeService}
	mux.Handle(path.Join("/api/change", httproute.List), headerAuth{httputil.ErrorHandler(users, apiHandler.List)})
	mux.Handle(path.Join("/api/change", httproute.Count), headerAuth{httputil.ErrorHandler(users, apiHandler.Count)})
	mux.Handle(path.Join("/api/change", httproute.Get), headerAuth{httputil.ErrorHandler(users, apiHandler.Get)})
	mux.Handle(path.Join("/api/change", httproute.ListTimeline), headerAuth{httputil.ErrorHandler(users, apiHandler.ListTimeline)})
	mux.Handle(path.Join("/api/change", httproute.ListCommits), headerAuth{httputil.ErrorHandler(users, apiHandler.ListCommits)})
	mux.Handle(path.Join("/api/change", httproute.GetDiff), headerAuth{httputil.ErrorHandler(users, apiHandler.GetDiff)})
	mux.Handle(path.Join("/api/change", httproute.EditComment), headerAuth{httputil.ErrorHandler(users, apiHandler.EditComment)})
	mux.Handle(path.Join("/api/change", httproute.ThreadType), headerAuth{httputil.ErrorHandler(users, apiHandler.ThreadType)})

	changesHandler := cookieAuth{httputil.ErrorHandler(users, changesApp.ServeHTTP)}
	mux.Handle("/changes/", changesHandler)
}

// dmitshurSeesExternalChanges gives dmitshur access to changes on GitHub and Gerrit,
// in addition to local ones.
type dmitshurSeesExternalChanges struct {
	local                change.Service
	dmitshurGitHubChange change.Service
	dmitshurGerritChange change.Service
	users                users.Service
}

func (s dmitshurSeesExternalChanges) List(ctx context.Context, repo string, opt change.ListOptions) ([]change.Change, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return nil, err
	}
	return service.List(ctx, repo, opt)
}

func (s dmitshurSeesExternalChanges) Count(ctx context.Context, repo string, opt change.ListOptions) (uint64, error) {
	if repo == "github.com/shurcooL/issuesapp" || repo == "github.com/shurcooL/notificationsapp" {
		// For the gh+ds hybrid packages specifically,
		// don't do an authorization check and allow unauthenticated users
		// to use the dmitshur-authenticated GitHub change service.
		//
		// It's safe to do so because counting changes is a read-only operation.
		//
		// This is needed to display the number of open changes in the tabnav
		// when viewing the gh+ds hybrid packages.
		return s.dmitshurGitHubChange.Count(ctx, repo, opt)
	}

	service, err := s.service(ctx, repo)
	if err != nil {
		return 0, err
	}
	return service.Count(ctx, repo, opt)
}

func (s dmitshurSeesExternalChanges) Get(ctx context.Context, repo string, id uint64) (change.Change, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return change.Change{}, err
	}
	return service.Get(ctx, repo, id)
}

func (s dmitshurSeesExternalChanges) ListTimeline(ctx context.Context, repo string, id uint64, opt *change.ListTimelineOptions) ([]interface{}, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return nil, err
	}
	return service.ListTimeline(ctx, repo, id, opt)
}

func (s dmitshurSeesExternalChanges) ListCommits(ctx context.Context, repo string, id uint64) ([]change.Commit, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return nil, err
	}
	return service.ListCommits(ctx, repo, id)
}

func (s dmitshurSeesExternalChanges) GetDiff(ctx context.Context, repo string, id uint64, opt *change.GetDiffOptions) ([]byte, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return nil, err
	}
	return service.GetDiff(ctx, repo, id, opt)
}

func (s dmitshurSeesExternalChanges) EditComment(ctx context.Context, repo string, id uint64, cr change.CommentRequest) (change.Comment, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return change.Comment{}, err
	}
	return service.EditComment(ctx, repo, id, cr)
}

func (s dmitshurSeesExternalChanges) ThreadType(ctx context.Context, repo string) (string, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return "", err
	}
	return service.ThreadType(ctx, repo)
}

func (s dmitshurSeesExternalChanges) service(ctx context.Context, repo string) (change.Service, error) {
	switch {
	default:
		return s.local, nil
	case strings.HasPrefix(repo, "github.com/"):
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser != dmitshur {
			return nil, os.ErrPermission
		}
		return s.dmitshurGitHubChange, nil
	case strings.HasPrefix(repo, "go.googlesource.com/"):
		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser != dmitshur {
			return nil, os.ErrPermission
		}
		return s.dmitshurGerritChange, nil
	}
}
