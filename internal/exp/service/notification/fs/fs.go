// Package fs implements notification.Service using a virtual filesystem.
package fs

import (
	"context"

	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/users"
)

// TODO: implement an actual filesystem-based (or otherwise) notification service

// DevNull is an empty placeholder notification.Service implementation.
type DevNull struct{}

func (DevNull) ListNotifications(ctx context.Context, opt notification.ListOptions) ([]notification.Notification, error) {
	return nil, nil
}

func (DevNull) StreamNotifications(ctx context.Context, ch chan<- []notification.Notification) error {
	return nil
}

func (DevNull) CountNotifications(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (DevNull) MarkThreadRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	return nil
}

func (DevNull) SubscribeThread(ctx context.Context, namespace, threadType string, threadID uint64, subscribers []users.UserSpec) error {
	return nil
}

func (DevNull) NotifyThread(ctx context.Context, namespace, threadType string, threadID uint64, nr notification.NotificationRequest) error {
	return nil
}
