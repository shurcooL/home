package githubapi

import (
	"context"

	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/notifications"
)

// threadType is the notifications thread type for this service.
const threadType = "Issue"

func (service) ThreadType(context.Context, issues.RepoSpec) (string, error) { return threadType, nil }

// markRead marks the specified issue as read for current user.
func (s service) markRead(ctx context.Context, repo issues.RepoSpec, id uint64) error {
	if s.notifications == nil {
		return nil
	}

	return s.notifications.MarkRead(ctx, notifications.RepoSpec(repo), threadType, id)
}
