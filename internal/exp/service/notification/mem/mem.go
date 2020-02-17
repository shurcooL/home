// Package mem implements notification.Service in memory.
package mem

import (
	"context"
	"os"
	"sync"

	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/users"
)

func NewService(dmitshur users.User, users users.Service) *Service {
	return &Service{
		dmitshur: dmitshur,
		users:    users,

		chs: make(map[context.Context]chan<- []notification.Notification),
	}
}

type Service struct {
	dmitshur users.User
	users    users.Service

	notifsMu sync.Mutex
	notifs   []notification.Notification

	chsMu sync.Mutex
	chs   map[context.Context]chan<- []notification.Notification
}

func (s *Service) ListNotifications(ctx context.Context, opt notification.ListOptions) ([]notification.Notification, error) {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return nil, err
	} else if u != s.dmitshur.UserSpec {
		return nil, os.ErrPermission
	}

	var notifs []notification.Notification
	s.notifsMu.Lock()
	notifs = append(notifs, s.notifs...)
	s.notifsMu.Unlock()
	return notifs, nil
}

func (s *Service) StreamNotifications(ctx context.Context, ch chan<- []notification.Notification) error {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return err
	} else if u != s.dmitshur.UserSpec {
		return os.ErrPermission
	}

	s.chsMu.Lock()
	s.chs[ctx] = ch
	s.chsMu.Unlock()
	return nil
}

func (*Service) CountNotifications(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (*Service) MarkNotificationRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	return nil
}

func (s *Service) Notify(ctx context.Context, n notification.Notification) error {
	if n.Actor.UserSpec == s.dmitshur.UserSpec {
		return nil
	}

	s.notifsMu.Lock()
	s.notifs = append(s.notifs, n)
	s.notifsMu.Unlock()

	// Notify streaming observers.
	s.chsMu.Lock()
	for ctx, ch := range s.chs {
		if ctx.Err() != nil {
			delete(s.chs, ctx)
			continue
		}
		select {
		case ch <- []notification.Notification{n}:
		default:
		}
	}
	s.chsMu.Unlock()
	return nil
}
