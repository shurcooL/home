// Package fs implements notification.Service using a virtual filesystem.
package fs

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/users"
	"github.com/shurcooL/webdavfs/vfsutil"
	"golang.org/x/net/webdav"
)

// NewService creates a virtual filesystem-backed notification.Service,
// using root for storage.
func NewService(root webdav.FileSystem, us users.Service) notification.Service {
	return &service{
		fs:    root,
		users: us,
		chs: make(map[struct {
			Ctx  context.Context
			User users.UserSpec
		}]chan<- []notification.Notification),
	}
}

type service struct {
	fsMu sync.RWMutex
	fs   webdav.FileSystem

	users users.Service

	chsMu sync.Mutex
	chs   map[struct {
		Ctx  context.Context
		User users.UserSpec
	}]chan<- []notification.Notification
}

func (s *service) ListNotifications(ctx context.Context, opt notification.ListOptions) ([]notification.Notification, error) {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return nil, err
	}
	if currentUser.ID == 0 {
		return nil, os.ErrPermission
	}

	s.fsMu.RLock()
	defer s.fsMu.RUnlock()

	var ns []notification.Notification

	fis, err := vfsutil.ReadDir(ctx, s.fs, notificationsDir(currentUser))
	if os.IsNotExist(err) {
		fis = nil
	} else if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		var nds []notificationDisk
		err := jsonDecodeAllFile(ctx, s.fs, notificationPath(currentUser, fi.Name()), &nds)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %v", notificationPath(currentUser, fi.Name()), err)
		}

		for _, n := range nds {
			if opt.Namespace != "" && n.Namespace != opt.Namespace {
				// All notifications have the same namespace,
				// so if the first one doesn't match, break out.
				// TODO: Can this check be factored out of the loop?
				break
			}

			// TODO: Maybe deduce threadType and threadID from fi.Name() rather than adding that to encoded JSON...
			ns = append(ns, notification.Notification{
				Namespace:   n.Namespace,
				ThreadType:  n.ThreadType,
				ThreadID:    n.ThreadID,
				ImportPaths: n.ImportPaths,
				Time:        n.Time,
				Actor:       s.user(ctx, n.Actor.UserSpec()),
				Payload:     n.Payload,
				Unread:      true,
				// TODO: Participating?
				// TODO: Mentioned?
			})
		}
	}

	if opt.All {
		fis, err := vfsutil.ReadDir(ctx, s.fs, readDir(currentUser))
		if os.IsNotExist(err) {
			fis = nil
		} else if err != nil {
			return nil, err
		}
		for _, fi := range fis {
			var nds []notificationDisk
			err := jsonDecodeAllFile(ctx, s.fs, readPath(currentUser, fi.Name()), &nds)
			if err != nil {
				return nil, fmt.Errorf("error reading %s: %v", readPath(currentUser, fi.Name()), err)
			}

			for _, n := range nds {
				// Delete and skip old read notifications.
				if time.Since(n.Time) > 90*24*time.Hour {
					err := s.fs.RemoveAll(ctx, readPath(currentUser, fi.Name()))
					if err != nil {
						return nil, err
					}
					// TODO: Can this check be factored out of the loop?
					break
				}

				if opt.Namespace != "" && n.Namespace != opt.Namespace {
					// All notifications have the same namespace,
					// so if the first one doesn't match, break out.
					// TODO: Can this check be factored out of the loop?
					break
				}

				// TODO: Maybe deduce threadType and threadID from fi.Name() rather than adding that to encoded JSON...
				ns = append(ns, notification.Notification{
					Namespace:   n.Namespace,
					ThreadType:  n.ThreadType,
					ThreadID:    n.ThreadID,
					ImportPaths: n.ImportPaths,
					Time:        n.Time,
					Actor:       s.user(ctx, n.Actor.UserSpec()),
					Payload:     n.Payload,
					Unread:      false,
					// TODO: Participating?
					// TODO: Mentioned?
				})
			}
		}

		// THINK: Consider using the dir-less vfs abstraction for doing this implicitly? Less code here.
		// If the user has no more read notifications left, remove the empty directory.
		switch notifications, err := vfsutil.ReadDir(ctx, s.fs, readDir(currentUser)); {
		case err != nil && !os.IsNotExist(err):
			return nil, err
		case err == nil && len(notifications) == 0:
			err := s.fs.RemoveAll(ctx, readDir(currentUser))
			if err != nil {
				return nil, err
			}
		}
	}

	return ns, nil
}

func (s *service) StreamNotifications(ctx context.Context, ch chan<- []notification.Notification) error {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return err
	}
	if currentUser.ID == 0 {
		return os.ErrPermission
	}

	s.chsMu.Lock()
	s.chs[struct {
		Ctx  context.Context
		User users.UserSpec
	}{ctx, currentUser}] = ch
	s.chsMu.Unlock()

	return nil
}

func (s *service) CountNotifications(ctx context.Context) (uint64, error) {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return 0, err
	}
	if currentUser.ID == 0 {
		return 0, os.ErrPermission
	}

	s.fsMu.RLock()
	defer s.fsMu.RUnlock()

	// TODO: Consider reading/parsing entries, in case there's .DS_Store, etc., that should be skipped?
	notifications, err := vfsutil.ReadDir(ctx, s.fs, notificationsDir(currentUser))
	if os.IsNotExist(err) {
		notifications = nil
	} else if err != nil {
		return 0, err
	}
	return uint64(len(notifications)), nil
}

func (s *service) NotifyThread(ctx context.Context, namespace, threadType string, threadID uint64, nr notification.NotificationRequest) error {
	//currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	currentUser, err := s.users.GetAuthenticated(ctx)
	if err != nil {
		return err
	}
	if currentUser.ID == 0 {
		return os.ErrPermission
	}

	s.fsMu.Lock()
	defer s.fsMu.Unlock()

	type subscription struct {
		Participating bool
	}
	var subscribers = make(map[users.UserSpec]subscription)

	// Repo watchers.
	fis, err := vfsutil.ReadDir(ctx, s.fs, subscribersDir(namespace, "", 0))
	if os.IsNotExist(err) {
		fis = nil
	} else if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}
		subscriber, err := unmarshalUserSpec(fi.Name())
		if err != nil {
			continue
		}
		subscribers[subscriber] = subscription{Participating: false}
	}

	// Thread subscribers. Iterate over them after repo watchers,
	// so that their participating status takes higher precedence.
	fis, err = vfsutil.ReadDir(ctx, s.fs, subscribersDir(namespace, threadType, threadID))
	if os.IsNotExist(err) {
		fis = nil
	} else if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}
		subscriber, err := unmarshalUserSpec(fi.Name())
		if err != nil {
			continue
		}
		subscribers[subscriber] = subscription{Participating: true}
	}

	// Notify streaming observers, if they're subscribed to the thread.
	s.chsMu.Lock()
	for cu, ch := range s.chs {
		if cu.Ctx.Err() != nil {
			delete(s.chs, cu)
			continue
		}
		if _, ok := subscribers[cu.User]; !ok {
			continue
		}
		if cu.User == currentUser.UserSpec {
			// Don't notify user of their own actions.
			continue
		}
		select {
		case ch <- []notification.Notification{{
			Namespace:   namespace,
			ThreadType:  threadType,
			ThreadID:    threadID,
			ImportPaths: nr.ImportPaths,
			Time:        nr.Time,
			Actor:       currentUser,
			Payload:     nr.Payload,
			Unread:      true,
			// TODO: Participating?
			// TODO: Mentioned?
		}}:
		default:
		}
	}
	s.chsMu.Unlock()

	for subscriber /*, subscription*/ := range subscribers {
		if currentUser.ID != 0 && subscriber == currentUser.UserSpec {
			// Don't notify user of their own actions.
			continue
		}

		// Delete read notification with same key, if any.
		/*err = s.fs.RemoveAll(ctx, readPath(subscriber, notificationKey(namespace, threadType, threadID)))
		if err != nil && !os.IsNotExist(err) {
			return err
		}*/

		// Create notificationsDir for subscriber in case it doesn't already exist.
		err = s.fs.Mkdir(ctx, notificationsDir(subscriber), 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}

		// TODO: Maybe deduce threadType and threadID from fi.Name() rather than adding that to encoded JSON...
		n := notificationDisk{
			Namespace:   namespace,
			ThreadType:  threadType,
			ThreadID:    threadID,
			ImportPaths: nr.ImportPaths,
			Time:        nr.Time,
			Actor:       fromUserSpec(currentUser.UserSpec), //fromUserSpec(nr.Actor), // TODO: Why not use current user?
			Payload:     nr.Payload,

			// TODO.
			//Participating: subscription.Participating,
		}
		err = jsonAppendFile(ctx, s.fs, notificationPath(subscriber, notificationKey(namespace, threadType, threadID)), n)
		// TODO: Maybe in future read previous value, and use it to preserve some fields, like earliest HTML URL.
		//       Maybe that shouldn't happen here though.
		if err != nil {
			return fmt.Errorf("error writing %s: %v", notificationPath(subscriber, notificationKey(namespace, threadType, threadID)), err)
		}
	}

	return nil
}

func (s *service) SubscribeThread(ctx context.Context, namespace, threadType string, threadID uint64, subscribers []users.UserSpec) error {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return err
	}
	if currentUser.ID == 0 {
		return os.ErrPermission
	}

	s.fsMu.Lock()
	defer s.fsMu.Unlock()

	for _, subscriber := range subscribers {
		err := createEmptyFile(ctx, s.fs, subscriberPath(namespace, threadType, threadID, subscriber))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) MarkThreadRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	currentUser, err := s.users.GetAuthenticatedSpec(ctx)
	if err != nil {
		return err
	}
	if currentUser.ID == 0 {
		return os.ErrPermission
	}

	s.fsMu.Lock()
	defer s.fsMu.Unlock()

	// Return early if the notification doesn't exist, before creating readDir for currentUser.
	key := notificationKey(namespace, threadType, threadID)
	_, err = vfsutil.Stat(ctx, s.fs, notificationPath(currentUser, key))
	if os.IsNotExist(err) {
		return nil
	}

	// Notify streaming observers.
	s.chsMu.Lock()
	for cu, ch := range s.chs {
		if cu.Ctx.Err() != nil {
			delete(s.chs, cu)
			continue
		}
		select {
		case ch <- []notification.Notification{{
			Namespace:  namespace,
			ThreadType: threadType,
			ThreadID:   threadID,
			Unread:     false,
		}}:
		default:
		}
	}
	s.chsMu.Unlock()

	// Create readDir for currentUser in case it doesn't already exist.
	err = s.fs.Mkdir(ctx, readDir(currentUser), 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	// Move notification thread content to file in read directory.
	err = appendFile(ctx, s.fs, readPath(currentUser, key), notificationPath(currentUser, key))
	if err != nil {
		return err
	}
	err = s.fs.RemoveAll(ctx, notificationPath(currentUser, key))
	if err != nil {
		return err
	}

	// THINK: Consider using the dir-less vfs abstraction for doing this implicitly? Less code here.
	// If the user has no more unread notifications left, remove the empty directory.
	switch notifications, err := vfsutil.ReadDir(ctx, s.fs, notificationsDir(currentUser)); {
	case err != nil && !os.IsNotExist(err):
		return err
	case err == nil && len(notifications) == 0:
		err := s.fs.RemoveAll(ctx, notificationsDir(currentUser))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) user(ctx context.Context, user users.UserSpec) users.User {
	u, err := s.users.Get(ctx, user)
	if err != nil {
		return users.User{
			UserSpec:  user,
			Login:     fmt.Sprintf("%d@%s", user.ID, user.Domain),
			AvatarURL: "",
			HTMLURL:   "",
		}
	}
	return u
}
