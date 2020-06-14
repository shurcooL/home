package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"dmitri.shuralyov.com/route/github"
	"github.com/shurcooL/events"
	"github.com/shurcooL/home/httputil"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/issue/fs"
	"github.com/shurcooL/home/internal/exp/service/issue/githubapi"
	"github.com/shurcooL/home/internal/exp/service/issue/httphandler"
	"github.com/shurcooL/home/internal/exp/service/issue/httproute"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

func newIssuesServiceV2(root webdav.FileSystem, notification notification.Service, events events.ExternalService, users users.Service, router github.Router) (issues.Service, error) {
	local, err := fs.NewService(root, notification, events, users)
	if err != nil {
		return nil, err
	}
	dmitshurGitHubIssues := githubapi.NewService(
		dmitshurPublicRepoGHV3,
		dmitshurPublicRepoGHV4,
		notification,
		router,
	)
	return dmitshurSeesExternalIssues{
		local:                local,
		dmitshurGitHubIssues: dmitshurGitHubIssues,
		users:                users,
	}, nil
}

type issueCounter interface {
	// Count issues.
	Count(ctx context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) (uint64, error)
}

// TODO: Remove unused parameters in initIssuesV2.

// initIssuesV2 registers handlers for the issues service HTTP API,
// and handlers for the issues app.
func initIssuesV2(mux *http.ServeMux, issuesService issues.Service, issuesApp httperror.Handler, users users.Service) {
	// Register issue service HTTP API endpoints.
	apiHandler := httphandler.Issues{Issues: issuesService}
	mux.Handle(path.Join("/api/issue", httproute.List), headerAuth{httputil.ErrorHandler(users, apiHandler.List)})
	mux.Handle(path.Join("/api/issue", httproute.Count), headerAuth{httputil.ErrorHandler(users, apiHandler.Count)})
	mux.Handle(path.Join("/api/issue", httproute.Get), headerAuth{httputil.ErrorHandler(users, apiHandler.Get)})
	mux.Handle(path.Join("/api/issue", httproute.ListTimeline), headerAuth{httputil.ErrorHandler(users, apiHandler.ListTimeline)})
	mux.Handle(path.Join("/api/issue", httproute.Create), headerAuth{httputil.ErrorHandler(users, apiHandler.Create)})
	mux.Handle(path.Join("/api/issue", httproute.CreateComment), headerAuth{httputil.ErrorHandler(users, apiHandler.CreateComment)})
	mux.Handle(path.Join("/api/issue", httproute.Edit), headerAuth{httputil.ErrorHandler(users, apiHandler.Edit)})
	mux.Handle(path.Join("/api/issue", httproute.EditComment), headerAuth{httputil.ErrorHandler(users, apiHandler.EditComment)})
	mux.Handle(path.Join("/api/issue", httproute.ThreadType), headerAuth{httputil.ErrorHandler(users, apiHandler.ThreadType)})

	issuesHandler := cookieAuth{httputil.ErrorHandler(users, func(w http.ResponseWriter, req *http.Request) error {
		returnURL := req.URL.Path
		err := issuesApp.ServeHTTP(w, req)
		// TODO: Factor out this os.IsPermission(err) && u == nil check somewhere, if possible. (But this shouldn't apply for APIs.)
		if s := req.Context().Value(sessionContextKey).(*session); os.IsPermission(err) && s == nil {
			loginURL := (&url.URL{
				Path:     "/login",
				RawQuery: url.Values{returnParameterName: {returnURL}}.Encode(),
			}).String()
			return httperror.Redirect{URL: loginURL}
		}
		return err
	})}
	mux.Handle("/issues/", issuesHandler)
}

// dmitshurSeesExternalIssues gives dmitshur access to issues on GitHub,
// in addition to local ones.
type dmitshurSeesExternalIssues struct {
	local                issues.Service
	dmitshurGitHubIssues issues.Service
	users                users.Service
}

func (s dmitshurSeesExternalIssues) List(ctx context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) ([]issues.Issue, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return nil, err
	}
	return service.List(ctx, repo, opt)
}

func (s dmitshurSeesExternalIssues) Count(ctx context.Context, repo issues.RepoSpec, opt issues.IssueListOptions) (uint64, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return 0, err
	}
	return service.Count(ctx, repo, opt)
}

func (s dmitshurSeesExternalIssues) Get(ctx context.Context, repo issues.RepoSpec, id uint64) (issues.Issue, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return issues.Issue{}, err
	}
	return service.Get(ctx, repo, id)
}

func (s dmitshurSeesExternalIssues) ListTimeline(ctx context.Context, repo issues.RepoSpec, id uint64, opt *issues.ListOptions) ([]interface{}, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return nil, err
	}
	return service.ListTimeline(ctx, repo, id, opt)
}

func (s dmitshurSeesExternalIssues) Create(ctx context.Context, repo issues.RepoSpec, issue issues.Issue) (issues.Issue, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return issues.Issue{}, err
	}
	return service.Create(ctx, repo, issue)
}

func (s dmitshurSeesExternalIssues) CreateComment(ctx context.Context, repo issues.RepoSpec, id uint64, comment issues.Comment) (issues.Comment, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return issues.Comment{}, err
	}
	return service.CreateComment(ctx, repo, id, comment)
}

func (s dmitshurSeesExternalIssues) Edit(ctx context.Context, repo issues.RepoSpec, id uint64, ir issues.IssueRequest) (issues.Issue, []issues.Event, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return issues.Issue{}, nil, err
	}
	return service.Edit(ctx, repo, id, ir)
}

func (s dmitshurSeesExternalIssues) EditComment(ctx context.Context, repo issues.RepoSpec, id uint64, cr issues.CommentRequest) (issues.Comment, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return issues.Comment{}, err
	}
	return service.EditComment(ctx, repo, id, cr)
}

func (s dmitshurSeesExternalIssues) ThreadType(ctx context.Context, repo issues.RepoSpec) (string, error) {
	service, err := s.service(ctx, repo)
	if err != nil {
		return "", err
	}
	return service.ThreadType(ctx, repo)
}

func (s dmitshurSeesExternalIssues) service(ctx context.Context, repo issues.RepoSpec) (issues.Service, error) {
	switch {
	default:
		return s.local, nil
	case strings.HasPrefix(repo.URI, "github.com/") &&
		repo.URI != "github.com/shurcooL/issuesapp" &&
		repo.URI != "github.com/shurcooL/notificationsapp":

		currentUser, err := s.users.GetAuthenticatedSpec(ctx)
		if err != nil {
			return nil, err
		}
		if currentUser != dmitshur {
			return nil, os.ErrPermission
		}
		return s.dmitshurGitHubIssues, nil
	}
}
