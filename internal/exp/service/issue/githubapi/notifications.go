package githubapi

import (
	"context"

	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/notifications"
)

// threadType is the notifications thread type for this service.
const threadType = "Issue"

// ThreadType returns the notifications thread type for this service.
func (service) ThreadType(issues.RepoSpec) string { return threadType }

// markRead marks the specified issue as read for current user.
func (s service) markRead(ctx context.Context, repo issues.RepoSpec, id uint64) error {
	if s.notifications == nil {
		return nil
	}

	return s.notifications.MarkRead(ctx, notifications.RepoSpec(repo), threadType, id)
}
