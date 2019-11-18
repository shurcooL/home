package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	githubv3 "github.com/google/go-github/github"
	"github.com/shurcooL/home/internal/exp/service/notification"
)

type thread struct {
	Namespace string
	Type      string
	ID        uint64
}

func (s *Service) pollNotifications() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("internal panic: %v\n\n%s", e, debug.Stack())
		}
	}()

	// Poll loop.
	for {
		// List all unread notifications and compare against last time,
		// to find out whether some of them became read externally.
		ghNotifs, resp, err := ghListNotificationsAllPages( // Sorted by most recently updated.
			context.Background(),
			s.clV3,
			&githubv3.NotificationListOptions{ListOptions: githubv3.ListOptions{PerPage: 100}},
			false,
		)
		if ge, ok := err.(*githubv3.ErrorResponse); ok && ge.Response.StatusCode == http.StatusUnauthorized {
			// Permanent error.
			return err
		} else if err != nil {
			log.Println("pollNotifications: sleep because temporary error:", err)
			ghNotifs = nil
		}
		var nextPoll time.Time
		if pi, ok := getPollInterval(resp); ok {
			nextPoll = time.Now().Add(pi)
		} else {
			nextPoll = time.Now().Add(time.Minute)
		}

		if ghNotifs != nil {
			var lastReadAt = make(map[thread]time.Time)
			for _, n := range ghNotifs {
				var id uint64
				switch *n.Subject.Type {
				case "Issue":
					var err error
					_, id, err = parseIssueSpec(*n.Subject.URL)
					if err != nil {
						return fmt.Errorf("failed to parseIssueSpec: %v", err)
					}
				case "PullRequest":
					var err error
					_, id, err = parsePullRequestSpec(*n.Subject.URL)
					if err != nil {
						return fmt.Errorf("failed to parsePullRequestSpec: %v", err)
					}
				default:
					continue
				}
				th := thread{
					Namespace: "github.com/" + *n.Repository.FullName,
					Type:      *n.Subject.Type,
					ID:        id,
				}
				if t, ok := lastReadAt[th]; !ok || n.GetLastReadAt().After(t) {
					lastReadAt[th] = n.GetLastReadAt()
				}
			}

			// Figure out if any notification threads became read.
			var becameRead []notification.Notification
			for th := range s.notifs.lastReadAt { // No need for lock because we're the only writer and just reading here.
				if _, ok := lastReadAt[th]; ok {
					// Still unread. Skip.
					continue
				}
				// The notification became read.
				becameRead = append(becameRead, notification.Notification{
					Namespace:  th.Namespace,
					ThreadType: th.Type,
					ThreadID:   th.ID,
					Unread:     false,
				})
			}

			// Notify streaming observers that some notifications became read.
			s.mail.chsMu.Lock()
			for ctx, ch := range s.mail.chs {
				if ctx.Err() != nil {
					delete(s.mail.chs, ctx)
					continue
				}
				select {
				case ch <- becameRead:
				default:
				}
			}
			s.mail.chsMu.Unlock()

			s.notifs.mu.Lock()
			s.notifs.lastReadAt = lastReadAt
			s.notifs.mu.Unlock()
		}

		time.Sleep(time.Until(nextPoll))
	}
}
