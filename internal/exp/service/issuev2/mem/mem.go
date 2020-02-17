// Package mem implements issuev2.Service in memory.
package mem

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/home/internal/exp/service/issuev2"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/users"
)

func NewService(users users.Service) *Service {
	return &Service{
		users: users,
		is:    make(map[int64]issuev2.Issue),
		ti:    make(map[int64][]interface{}),
	}
}

type Service struct {
	users users.Service

	mu sync.Mutex
	is map[int64]issuev2.Issue // Issue ID -> Issue.
	ti map[int64][]interface{} // Issue ID -> Issue Timeline Items (Comment, Event).

	// TODO: delete
	issues.Service
}

func (s *Service) CreateIssue(ctx context.Context, r issuev2.CreateIssueRequest) (issuev2.Issue, error) {
	u, err := s.users.GetAuthenticated(ctx)
	if err != nil {
		return issuev2.Issue{}, err
	}
	if u.UserSpec == (users.UserSpec{}) {
		return issuev2.Issue{}, os.ErrPermission
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	issue := issuev2.Issue{
		ID:         int64(len(s.is)) + 1,
		Author:     u,
		CreatedAt:  time.Now().UTC(),
		ImportPath: r.ImportPath,
		Title:      r.Title,
		State:      state.IssueOpen,
	}
	s.is[issue.ID] = issue
	s.ti[issue.ID] = append(s.ti[issue.ID], issuev2.Comment{
		ID:        0,
		Author:    issue.Author,
		CreatedAt: issue.CreatedAt,
		Body:      r.Body,
	})
	return issue, nil
}

func (s *Service) CreateIssueComment(ctx context.Context, id int64, r issuev2.CreateIssueCommentRequest) (issuev2.Comment, error) {
	u, err := s.users.GetAuthenticated(ctx)
	if err != nil {
		return issuev2.Comment{}, err
	}
	if u.UserSpec == (users.UserSpec{}) {
		return issuev2.Comment{}, os.ErrPermission
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tis, ok := s.ti[id]
	if !ok {
		return issuev2.Comment{}, os.ErrNotExist
	}
	comment := issuev2.Comment{
		ID:        int64(len(tis)),
		Author:    u,
		CreatedAt: time.Now().UTC(),
		Body:      r.Body,
	}
	s.ti[id] = append(tis, comment)
	return comment, nil
}

func (s *Service) ListIssues(ctx context.Context, pattern string, opt issuev2.ListOptions) ([]issuev2.Issue, error) {
	if opt.State != issues.StateFilter(state.IssueOpen) && opt.State != issues.StateFilter(state.IssueClosed) && opt.State != issues.AllStates {
		return nil, fmt.Errorf("invalid issues.IssueListOptions.State value: %q", opt.State) // TODO: Map to 400 Bad Request HTTP error.
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	match := matchPattern(pattern)
	var is []issuev2.Issue
	for _, i := range s.is {
		if !match(i.ImportPath) {
			continue
		} else if opt.State != issues.AllStates && i.State != state.Issue(opt.State) {
			continue
		}
		is = append(is, i)
	}
	return is, nil
}

func (s *Service) CountIssues(ctx context.Context, pattern string, opt issuev2.CountOptions) (int64, error) {
	if opt.State != issues.StateFilter(state.IssueOpen) && opt.State != issues.StateFilter(state.IssueClosed) && opt.State != issues.AllStates {
		return 0, fmt.Errorf("invalid issues.IssueListOptions.State value: %q", opt.State) // TODO: Map to 400 Bad Request HTTP error.
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	match := matchPattern(pattern)
	var count int64
	for _, i := range s.is {
		if !match(i.ImportPath) {
			continue
		} else if opt.State != issues.AllStates && i.State != state.Issue(opt.State) {
			continue
		}
		count++
	}
	return count, nil
}

func (s *Service) GetIssue(ctx context.Context, id int64) (issuev2.Issue, error) {
	u, err := s.users.GetAuthenticated(ctx)
	if err != nil {
		return issuev2.Issue{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	issue, ok := s.is[id]
	if !ok {
		return issuev2.Issue{}, os.ErrNotExist
	}
	issue.Editable = u.UserSpec == issue.Author.UserSpec || u.SiteAdmin
	return issue, nil
}

func (s *Service) EditIssue(ctx context.Context, id int64, r issuev2.EditIssueRequest) (issuev2.Issue, []issues.Event, error) {
	u, err := s.users.GetAuthenticated(ctx)
	if err != nil {
		return issuev2.Issue{}, nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	i, ok := s.is[id]
	if !ok {
		return issuev2.Issue{}, nil, os.ErrNotExist
	}
	if editable := u.UserSpec == i.Author.UserSpec || u.SiteAdmin; !editable {
		return issuev2.Issue{}, nil, os.ErrPermission
	}
	i.State = r.State
	s.is[id] = i
	return i, nil /* TODO */, nil
}

func (s *Service) ListIssueTimeline(ctx context.Context, id int64, opt *issuev2.ListOptions) ([]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tis, ok := s.ti[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return tis, nil
}

// matchPattern(pattern)(name) reports whether name matches pattern.
// Pattern is a limited glob pattern in which '...' means 'any string',
// foo/... matches foo too, and there is no other special syntax.
// The pattern "all" is a special case and matches all names.
func matchPattern(pattern string) func(name string) bool {
	if pattern == "all" {
		return func(string) bool { return true }
	}
	re := regexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	return regexp.MustCompile(`^` + re + `$`).MatchString
}
