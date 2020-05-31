// Package gerrit implements activity.Service for Gerrit.
package gerrit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"path"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"dmitri.shuralyov.com/go/prefixtitle"
	"dmitri.shuralyov.com/route/gerrit"
	"dmitri.shuralyov.com/service/change"
	"dmitri.shuralyov.com/state"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/httpfs/vfsutil"
	"github.com/shurcooL/users"
	"golang.org/x/build/maintner/reclog"
	"golang.org/x/net/webdav"
)

// NewService creates a Gerrit-backed activity.Service using the given
// Gerrit activity mail filesystem and Gerrit-backed change service.
// It serves the specified user only,
// whose activity mail must be provided,
// and cannot be used to serve multiple users.
//
// newActivityMail delivers a value when there is new mail,
// and must not be closed.
//
// user.Login is used to detect mentions.
func NewService(
	ctx context.Context, wg *sync.WaitGroup,
	fs webdav.FileSystem,
	activityMail http.FileSystem, newActivityMail <-chan struct{},
	cs change.Service,
	user users.User, users users.Service,
	router gerrit.Router,
) (*Service, error) {
	s := &Service{
		fs:          fs,
		notifMail:   activityMail,
		notifEvents: newActivityMail,
		cs:          cs,
		user:        user,
		users:       users,
		rtr:         router,
		lastReadAt:  make(map[thread]time.Time),
		chs:         make(map[context.Context]chan<- []notification.Notification),
	}
	go func() {
		err := s.loadAndPoll(ctx)
		if err != nil {
			log.Println("service/activity/gerrit: loadAndPoll:", err)
			s.errorMu.Lock()
			s.error = err
			s.errorMu.Unlock()
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		s.mu.Lock()
		err := jsonEncodeFile(context.Background(), s.fs, "gerritactivity-lastReadAt.json", s.lastReadAt)
		s.mu.Unlock()
		if err != nil {
			log.Println("service/activity/gerrit: jsonEncodeFile:", err)
		}
	}()
	return s, nil
}

type Service struct {
	fs          webdav.FileSystem // Persistent storage.
	notifMail   http.FileSystem
	notifEvents <-chan struct{} // Never closed.
	cs          change.Service

	user  users.User
	users users.Service
	rtr   gerrit.Router

	mu         sync.Mutex
	events     []eventAndURL
	notifs     []notifAndURL // Most recent notifications are at the front.
	lastReadAt map[thread]time.Time

	chsMu sync.Mutex
	chs   map[context.Context]chan<- []notification.Notification

	errorMu sync.Mutex
	error   error
}

type thread struct {
	Namespace string
	ID        uint64
}

func (t thread) MarshalText() (text []byte, err error) {
	return []byte(fmt.Sprintf("%s#%d", t.Namespace, t.ID)), nil
}

func (t *thread) UnmarshalText(text []byte) error {
	i := bytes.LastIndexByte(text, '#')
	if i == -1 {
		return fmt.Errorf("hash separator ('#') not found")
	}
	ns := string(text[:i])
	id, err := strconv.ParseUint(string(text[i+1:]), 10, 64)
	if err != nil {
		return err
	}
	t.Namespace, t.ID = ns, id
	return nil
}

// List lists events.
func (s *Service) List(ctx context.Context) ([]event.Event, error) {
	s.mu.Lock()
	events := make([]event.Event, len(s.events))
	for i, event := range s.events {
		events[i] = event.WithURL(ctx)
	}
	s.mu.Unlock()
	return events, nil
}

// ListNotifications implements notification.Service.
func (s *Service) ListNotifications(ctx context.Context, opt notification.ListOptions) ([]notification.Notification, error) {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return nil, err
	} else if u != s.user.UserSpec {
		return nil, os.ErrPermission
	}

	// TODO: Filter out notifs from other repos when
	//       opt.Namespace != "".
	var notifs []notification.Notification
	switch opt.All {
	case true:
		s.mu.Lock()
		notifs = make([]notification.Notification, len(s.notifs))
		for i, notif := range s.notifs {
			notifs[i] = notif.WithURL(ctx)
		}
		for i, n := range notifs {
			lastReadAt := s.lastReadAt[thread{n.Namespace, n.ThreadID}]
			notifs[i].Unread = n.Time.After(lastReadAt)
		}
		s.mu.Unlock()
	case false:
		s.mu.Lock()
		for _, n := range s.notifs {
			lastReadAt := s.lastReadAt[thread{n.Namespace, n.ThreadID}]
			unread := n.Time.After(lastReadAt)
			if !unread {
				continue
			}
			n := n.WithURL(ctx)
			n.Unread = true
			notifs = append(notifs, n)
		}
		s.mu.Unlock()
	}
	return notifs, nil
}

// StreamNotifications implements notification.Service.
func (s *Service) StreamNotifications(ctx context.Context, ch chan<- []notification.Notification) error {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return err
	} else if u != s.user.UserSpec {
		return os.ErrPermission
	}

	s.chsMu.Lock()
	s.chs[ctx] = ch
	s.chsMu.Unlock()

	return nil
}

// CountNotifications implements notification.Service.
func (s *Service) CountNotifications(ctx context.Context) (uint64, error) {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return 0, err
	} else if u != s.user.UserSpec {
		return 0, os.ErrPermission
	}

	var count uint64
	s.mu.Lock()
	for _, n := range s.notifs {
		lastReadAt := s.lastReadAt[thread{n.Namespace, n.ThreadID}]
		unread := n.Time.After(lastReadAt)
		if !unread {
			continue
		}
		count++
	}
	s.mu.Unlock()
	return count, nil
}

// Log logs the event.
// event.Time time zone must be UTC.
func (*Service) Log(_ context.Context, event event.Event) error {
	if event.Time.Location() != time.UTC {
		return errors.New("event.Time time zone must be UTC")
	}
	// TODO, THINK: Where should a Log("dmitri.shuralyov.com/foo/bar") event get non-errored? Here, or in home.multiEvents?
	// Nothing to do. Gerrit takes care of this on their end, even when performing actions via API.
	return nil
}

// gerritChangeThreadType is the notification thread type for Gerrit changes.
const gerritChangeThreadType = "Change"

// MarkThreadRead implements notification.Service.
//
// Namespace must be of the form "{server}/{project}".
// E.g., "go.googlesource.com/image".
func (s *Service) MarkThreadRead(ctx context.Context, namespace, threadType string, threadID uint64) error {
	if u, err := s.users.GetAuthenticatedSpec(ctx); err != nil {
		return err
	} else if u != s.user.UserSpec {
		return os.ErrPermission
	}

	if threadType != gerritChangeThreadType {
		return fmt.Errorf("unsupported threadType=%q", threadType)
	}

	th := thread{
		Namespace: namespace,
		ID:        threadID,
	}
	s.mu.Lock()
	s.lastReadAt[th] = time.Now().UTC()
	s.mu.Unlock()

	// Notify streaming observers.
	// TODO: do this only when notification went from unread to read
	s.chsMu.Lock()
	for ctx, ch := range s.chs {
		if ctx.Err() != nil {
			delete(s.chs, ctx)
			continue
		}
		select {
		case ch <- []notification.Notification{{
			Namespace:  namespace,
			ThreadType: gerritChangeThreadType,
			ThreadID:   threadID,
			Unread:     false,
		}}:
		default:
		}
	}
	s.chsMu.Unlock()

	return nil
}

// SubscribeThread implements notification.Service.
func (*Service) SubscribeThread(_ context.Context, namespace, threadType string, threadID uint64, subscribers []users.UserSpec) error {
	// TODO: Do anything? Or not needed?
	return fmt.Errorf("Service.SubscribeThread: not implemented")
}

// NotifyThread implements notification.Service.
func (*Service) NotifyThread(_ context.Context, namespace, threadType string, threadID uint64, nr notification.NotificationRequest) error {
	// TODO: Do anything? Or not needed?
	return fmt.Errorf("Service.NotifyThread: not implemented")
}

type gerritNotif struct {
	Server   string
	Project  string
	ChangeID uint64
}

func (s *Service) loadAndPoll(ctx context.Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("internal panic: %v\n\n%s", e, debug.Stack())
		}
	}()

	// Load initial state.
	var st struct {
		HandledSegs []fileSeg
		HandledTime time.Time
	}
	//err := jsonDecodeFile(context.Background(), s.fs, "pollnotifs.json", &st)
	//if err != nil && !os.IsNotExist(err) {
	//	return err
	//}
	st.HandledTime = time.Now().Add(-72 * time.Hour) // TODO: Be able to process more from past.

	s.mu.Lock()
	err = jsonDecodeFile(ctx, s.fs, "gerritactivity-lastReadAt.json", &s.lastReadAt)
	s.mu.Unlock()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	//var wd = new(mime.WordDecoder)
	//var ap = mail.AddressParser{WordDecoder: wd}

	for {
		segs, err := diskSegments(s.notifMail)
		if err != nil {
			return err
		}

		var grNotifs = make(map[gerritNotif]struct{})
		var latestMail time.Time
		err = walkMail(ctx, s.notifMail, segs, st.HandledSegs, func(m *mail.Message) error {
			if _, ok := m.Header["X-Gerrit-Changeurl"]; !ok {
				// Not a notification from Gerrit. E.g., someone replied
				// to a Gerrit notification from their email client.
				return nil
			}
			date, err := m.Header.Date()
			if err != nil {
				return err
			}
			date = date.UTC()
			if !date.After(st.HandledTime) {
				return nil
			}
			// X-Gerrit-ChangeURL: <https://go-review.googlesource.com/c/go/+/162926>
			//changeURL := trimAngle(m.Header.Get("X-Gerrit-Changeurl"))
			// Extract gerrit project "oauth2" from List-Id header like:
			// List-Id: <gerrit-oauth2.go-review.googlesource.com>
			listID := trimAngle(m.Header.Get("List-Id"))
			i := strings.IndexByte(listID, '.')
			if i == -1 {
				return fmt.Errorf("no dot in listID %q", listID)
			}
			server := strings.Replace(listID[i+1:], "go-review.", "go.", 1)
			project := strings.TrimPrefix(listID[:i], "gerrit-")
			clNumber, err := strconv.ParseUint(m.Header.Get("X-Gerrit-Change-Number"), 10, 64)
			if err != nil {
				return err
			}
			grNotifs[gerritNotif{server, project, clNumber}] = struct{}{}
			if date.After(latestMail) {
				latestMail = date
			}
			return nil
		})
		if err != nil {
			return err
		}

		if len(grNotifs) > 0 {
			events, notifs, err := fetchAndConvert(ctx, s.cs, grNotifs, st.HandledTime, s.user, s.rtr)
			if err != nil {
				log.Println("fetchAndConvert:", err)
			} else {
				s.mu.Lock()
				// TODO: clean out too-old notifs more efficiently
				{
					s.events = append(s.events, events...)
					sort.SliceStable(s.events, func(i, j int) bool { return s.events[i].Time.After(s.events[j].Time) })
					if len(s.events) > 100 {
						s.events = s.events[:100]
					}
				}
				{
					s.notifs = append(s.notifs, notifs...)
					sort.SliceStable(s.notifs, func(i, j int) bool { return s.notifs[i].Time.After(s.notifs[j].Time) })
					if len(s.notifs) > 100 {
						s.notifs = s.notifs[:100]
					}
				}
				for _, e := range events { // Mark threads read based on external self-activity.
					// e.Container no longer matches our namespace, because it's set to import path,
					// but we're still tracking namespaces like "go.googlesource.com/net", etc.
					// So convert from event's import path back to a Gerrit "{server}/{project}" URL.
					namespace := importPathToGerritURL(e.Container)
					th := thread{
						Namespace: namespace,
						ID:        e.changeID,
					}
					if e.Time.After(s.lastReadAt[th]) {
						s.lastReadAt[th] = e.Time
					}
				}
				latestTime := s.notifs[0].Time // THINK: How to be 100% confident this can't panic?
				s.mu.Unlock()

				// Notify streaming observers.
				s.chsMu.Lock()
				for ctx, ch := range s.chs {
					if ctx.Err() != nil {
						delete(s.chs, ctx)
						continue
					}
					ns := make([]notification.Notification, len(notifs))
					for i, n := range notifs {
						ns[i] = n.WithURL(ctx)
						ns[i].Unread = true
					}
					select {
					case ch <- ns:
					default:
					}
				}
				s.chsMu.Unlock()

				if latestMail.Before(st.HandledTime) && latestTime.Before(st.HandledTime) {
					log.Printf("WARNING: latestMail(%v) and latestTime(%v) are before st.HandledTime(%v)!!!\n", latestMail, latestTime, st.HandledTime)
				}
				st.HandledSegs = segs
				st.HandledTime = latestMail
				log.Println("set gerrit HandledTime to:", st.HandledTime)
				if latestTime.After(st.HandledTime) {
					st.HandledTime = latestTime
					log.Println("WARNING: nvm, actually set gerrit HandledTime to:", st.HandledTime)
				}
				//err := jsonEncodeFile(context.Background(), s.fs, "pollnotifs.json", st)
				//if err != nil {
				//	return err
				//}
			}
		}

		select {
		case <-s.notifEvents:
		case <-ctx.Done():
			return nil
		}
	}
}

// fetchAndConvert fetches additional information from
// Gerrit API and converts Gerrit notifications to own format.
func fetchAndConvert(
	ctx context.Context,
	cs change.Service,
	grNotifs map[gerritNotif]struct{},
	handledTime time.Time,
	user users.User,
	rtr gerrit.Router,
) ([]eventAndURL, []notifAndURL, error) {
	// TODO: get this from user.Elsewhere?
	dmitshurOnGerrit := users.UserSpec{
		ID:     6005,
		Domain: "go-review.googlesource.com", // TODO: make UserSpec.Domain not have "-review".
	}

	var (
		events []eventAndURL
		notifs []notifAndURL
	)
	for n := range grNotifs {
		n := n
		notif := notifAndURL{Notification: notification.Notification{
			Namespace:  n.Server + "/" + n.Project,
			ThreadType: gerritChangeThreadType,
			ThreadID:   n.ChangeID,

			//Read:          !*n.Unread, // TODO
			Participating: false, // TODO
		}}

		chg, err := cs.Get(ctx, n.Server+"/"+n.Project, n.ChangeID)
		if err != nil {
			return nil, nil, err
		}
		tis, err := cs.ListTimeline(ctx, n.Server+"/"+n.Project, n.ChangeID, nil)
		if err != nil {
			return nil, nil, err
		}

		// Parse prefixed change title.
		importPaths, changeTitle := prefixtitle.ParseChange(modulePath(n.Server, n.Project), chg.Title)
		notif.ImportPaths = importPaths

		if chg.CreatedAt.After(handledTime) { // Process change description.
			notif := notif
			notif.Time = chg.CreatedAt
			notif.Actor = chg.Author
			notif.Payload = notification.Change{
				Action:      "opened",
				ChangeTitle: changeTitle,
				ChangeBody:  tis[0].(change.Comment).Body,
			}
			notif.url = func(ctx context.Context) string {
				return rtr.ChangeURL(ctx, n.Server, n.Project, n.ChangeID)
			}
			notif.Mentioned = strings.Contains(tis[0].(change.Comment).Body, user.Login)
			if notif.Actor.UserSpec == dmitshurOnGerrit {
				events = append(events, eventAndURL{event.Event{
					Time:      chg.CreatedAt,
					Actor:     user,
					Container: importPaths[0],
					Payload: event.Change{
						Action:      "opened",
						ChangeTitle: changeTitle,
						ChangeBody:  tis[0].(change.Comment).Body,
					},
				}, func(ctx context.Context) string {
					return rtr.ChangeURL(ctx, n.Server, n.Project, n.ChangeID)
				}, n.ChangeID})
			} else {
				notifs = append(notifs, notif)
			}
		}
		for _, ti := range tis[1:] { // Process the remaining timeline items.
			switch t := ti.(type) {
			case change.Comment:
				if !t.CreatedAt.After(handledTime) {
					continue
				}
				notif := notif
				notif.Time = t.CreatedAt
				notif.Actor = t.User
				notif.Payload = notification.ChangeComment{
					ChangeTitle: changeTitle,
					ChangeState: toState(chg.State),
					CommentBody: t.Body,
				}
				notif.url = func(ctx context.Context) string {
					return rtr.ChangeMessageURL(ctx, n.Server, n.Project, n.ChangeID, t.ID)
				}
				notif.Mentioned = strings.Contains(t.Body, user.Login)
				if notif.Actor.UserSpec == dmitshurOnGerrit {
					events = append(events, eventAndURL{event.Event{
						Time:      t.CreatedAt,
						Actor:     user,
						Container: importPaths[0],
						Payload: event.ChangeComment{
							ChangeTitle: changeTitle,
							ChangeState: toState(chg.State),
							CommentBody: t.Body,
						},
					}, func(ctx context.Context) string {
						return rtr.ChangeMessageURL(ctx, n.Server, n.Project, n.ChangeID, t.ID)
					}, n.ChangeID})
				} else {
					notifs = append(notifs, notif)
				}
			case change.Review:
				if !t.CreatedAt.After(handledTime) {
					continue
				}
				body := t.Body
				for _, c := range t.Comments {
					body += "\n\n" + c.Body
				}
				notif := notif
				notif.Time = t.CreatedAt
				notif.Actor = t.User
				notif.Payload = notification.ChangeComment{
					ChangeTitle:   changeTitle,
					ChangeState:   toState(chg.State),
					CommentBody:   body,
					CommentReview: t.State,
				}
				notif.url = func(ctx context.Context) string {
					return rtr.ChangeMessageURL(ctx, n.Server, n.Project, n.ChangeID, t.ID)
				}
				notif.Mentioned = strings.Contains(body, user.Login)
				if notif.Actor.UserSpec == dmitshurOnGerrit {
					events = append(events, eventAndURL{event.Event{
						Time:      t.CreatedAt,
						Actor:     user,
						Container: importPaths[0],
						Payload: event.ChangeComment{
							ChangeTitle:   changeTitle,
							ChangeState:   toState(chg.State),
							CommentBody:   body,
							CommentReview: t.State,
						},
					}, func(ctx context.Context) string {
						return rtr.ChangeMessageURL(ctx, n.Server, n.Project, n.ChangeID, t.ID)
					}, n.ChangeID})
				} else {
					notifs = append(notifs, notif)
				}
			case change.TimelineItem:
				notif := notif
				notif.Time = t.CreatedAt
				notif.Actor = t.Actor
				switch t.Payload.(type) {
				case change.ClosedEvent:
					notif.Payload = notification.Change{
						Action:      "closed",
						ChangeTitle: changeTitle,
					}
					notif.url = func(ctx context.Context) string {
						return rtr.ChangeURL(ctx, n.Server, n.Project, n.ChangeID)
					}
					if notif.Actor.UserSpec == dmitshurOnGerrit {
						events = append(events, eventAndURL{event.Event{
							Time:      t.CreatedAt,
							Actor:     user,
							Container: importPaths[0],
							Payload: event.Change{
								Action:      "closed",
								ChangeTitle: changeTitle,
							},
						}, func(ctx context.Context) string {
							return rtr.ChangeURL(ctx, n.Server, n.Project, n.ChangeID)
						}, n.ChangeID})
					} else {
						notifs = append(notifs, notif)
					}
				case change.MergedEvent:
					notif.Payload = notification.Change{
						Action:      "merged",
						ChangeTitle: changeTitle,
					}
					notif.url = func(ctx context.Context) string {
						return rtr.ChangeURL(ctx, n.Server, n.Project, n.ChangeID)
					}
					if notif.Actor.UserSpec == dmitshurOnGerrit {
						events = append(events, eventAndURL{event.Event{
							Time:      t.CreatedAt,
							Actor:     user,
							Container: importPaths[0],
							Payload: event.Change{
								Action:      "merged",
								ChangeTitle: changeTitle,
							},
						}, func(ctx context.Context) string {
							return rtr.ChangeURL(ctx, n.Server, n.Project, n.ChangeID)
						}, n.ChangeID})
					} else {
						notifs = append(notifs, notif)
					}
				case change.ReopenedEvent:
					// TODO: low priority because Gerrit Change Service doesn't ever emit it, atm
				}
			}
		}
	}
	return events, notifs, nil
}

type eventAndURL struct {
	event.Event
	url      func(context.Context) string
	changeID uint64 // For marking thread as read.
}

func (e eventAndURL) WithURL(ctx context.Context) event.Event {
	switch p := e.Payload.(type) {
	case event.Change:
		p.ChangeHTMLURL = e.url(ctx)
		e.Payload = p
		return e.Event
	case event.ChangeComment:
		p.CommentHTMLURL = e.url(ctx)
		e.Payload = p
		return e.Event
	default:
		return e.Event
	}
}

type notifAndURL struct {
	notification.Notification
	url func(context.Context) string
}

func (n notifAndURL) WithURL(ctx context.Context) notification.Notification {
	switch p := n.Payload.(type) {
	case notification.Change:
		p.ChangeHTMLURL = n.url(ctx)
		n.Payload = p
		return n.Notification
	case notification.ChangeComment:
		p.CommentHTMLURL = n.url(ctx)
		n.Payload = p
		return n.Notification
	default:
		return n.Notification
	}
}

// modulePath returns the module path of the specified Gerrit project.
func modulePath(server, project string) string {
	switch server {
	case "go.googlesource.com":
		switch project {
		case "go":
			// Use empty string as the module path for the main Go repository.
			return ""
		default:
			return "golang.org/x/" + project
		case "dl":
			// dl is a special subrepo, there's no /x/ in its module path.
			return "golang.org/dl"
		case "gddo":
			// There is no golang.org/x/gddo vanity import path,
			// the canonical module path for gddo is on GitHub.
			return "github.com/golang/gddo"
		}
	default:
		// TODO: If need to support arbitrary other Gerrit projects in the future,
		// start fetching the go.mod file and using the module path stated there.
		return server + "/" + project
	}
}

// importPathToGerritURL converts an import path like "net/http" or "golang.org/x/image/font/sfnt"
// to its Gerrit URL like "go.googlesource.com/go" or "go.googlesource.com/image".
func importPathToGerritURL(p string) string {
	switch {
	case strings.HasPrefix(p, "golang.org/x/"):
		proj := p[len("golang.org/x/"):]
		if i := strings.IndexByte(proj, '/'); i != -1 {
			proj = proj[:i]
		}
		return "go.googlesource.com/" + proj
	case p == "golang.org/dl" || strings.HasPrefix(p, "golang.org/dl/"):
		return "go.googlesource.com/dl"
	case p == "github.com/golang/gddo" || strings.HasPrefix(p, "github.com/golang/gddo/"):
		return "go.googlesource.com/gddo"
	case strings.HasPrefix(p, "go.googlesource.com/"):
		// These should not happen anymore, but support them
		// just in case there are legacy events, etc.
		log.Printf("importPathToGerritURL: saw import path %q\n", p)
		return p
	default:
		// All other import paths must be from the standard library.
		return "go.googlesource.com/go"
	}
}

// TODO: make this no longer needed by changing change.Service to use state.Change.
//       Actually, maybe that's not a good idea/won't work out... Need to think more.
func toState(st change.State) state.Change {
	switch st {
	case change.OpenState:
		return state.ChangeOpen
	case change.ClosedState:
		return state.ChangeClosed
	case change.MergedState:
		return state.ChangeMerged
	default:
		panic("unreachable")
	}
}

func diskSegments(fs http.FileSystem) ([]fileSeg, error) {
	fis, err := vfsutil.ReadDir(fs, "/")
	if err != nil {
		return nil, err
	}
	var segs []fileSeg
	for _, fi := range fis {
		name := fi.Name()
		if !strings.HasSuffix(name, ".reclog") {
			continue
		}
		segs = append(segs, fileSeg{
			file: path.Join("/", name),
			size: fi.Size(),
		})
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i].file < segs[j].file })
	return segs, nil
}

type fileSeg struct {
	file string // Absolute path within the http.FileSystem.
	skip int64
	size int64
}

func walkMail(ctx context.Context, fs http.FileSystem, segs, handled []fileSeg, fn func(*mail.Message) error) error {
	for i, seg := range segs {
		if i < len(handled) && seg == handled[i] {
			continue
		} else if i == len(handled)-1 {
			seg.skip = handled[i].size
			log.Printf("processing more of segment i=%v; new bytes = %d\n", i, seg.size-seg.skip)
		}
		err := walkSegMail(ctx, fs, seg, fn)
		if err != nil {
			return fmt.Errorf("walkSegMail(%#v): %v", seg, err)
		}
	}
	return nil
}

func walkSegMail(ctx context.Context, fs http.FileSystem, seg fileSeg, fn func(*mail.Message) error) error {
	f, err := fs.Open(seg.file)
	if err != nil {
		return err
	}
	defer f.Close()
	if seg.skip > 0 {
		_, err := f.Seek(seg.skip, io.SeekStart)
		if err != nil {
			return err
		}
	}
	err = reclog.ForeachRecord(io.LimitReader(f, seg.size-seg.skip), seg.skip, func(off int64, hdr, rec []byte) error {
		m, err := mail.ReadMessage(bytes.NewReader(rec))
		if err != nil {
			return err
		}
		err = fn(m)
		if err != nil {
			return err
		}
		select {
		default:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	return err
}

func trimAngle(s string) string {
	return strings.TrimSuffix(strings.TrimPrefix(s, "<"), ">")
}

// Status reports the status of the service.
// The status is "ok" if everything is okay,
// or an error description otherwise.
func (s *Service) Status() string {
	s.errorMu.Lock()
	err := s.error
	s.errorMu.Unlock()
	if err != nil {
		return err.Error()
	}
	return "ok"
}
