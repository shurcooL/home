// Package fs implements notification.Service using a virtual filesystem.
package fs

import (
	"context"

	"github.com/shurcooL/home/internal/exp/service/notification"
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

func (DevNull) MarkNotificationRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	return nil
}
