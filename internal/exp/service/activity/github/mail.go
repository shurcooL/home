package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"path"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"dmitri.shuralyov.com/go/prefixtitle"
	"dmitri.shuralyov.com/route/github"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/httpfs/vfsutil"
	"github.com/shurcooL/users"
	"golang.org/x/build/maintner/reclog"
	"golang.org/x/mod/modfile"
)

func (s *Service) loadAndPoll() (err error) {
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
	st.HandledTime = time.Now().Add(2 * -72 * time.Hour) // TODO: Be able to process more from past.

	for {
		segs, err := diskSegments(s.notifMail)
		if err != nil {
			return err
		}

		var ghEvents []githubEvent
		err = walkMail(context.Background(), s.notifMail, segs, st.HandledSegs, func(m *mail.Message) error {
			reason := m.Header.Get("X-GitHub-Reason")
			if reason == "" {
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
			id, err := parseMessage(m)
			if err != nil {
				return err
			}
			if id == (githubEventID{}) {
				// Not a recognized event type, skip.
				fmt.Printf("skipping unsupported GitHub message: %q (orig=%q)\n",
					trimAngle(m.Header.Get("Message-ID")),
					trimAngle(m.Header.Get("X-Google-Original-Message-ID")))
				return nil
			}
			ghEvents = append(ghEvents, githubEvent{
				githubEventID: id,
				Reason:        reason,
			})
			return nil
		})
		if err != nil {
			return err
		}

		// TODO: if too many len(ghEvents), skip older?
		//       currently 329
		//       or better do it in fetchAndConvert when len(notifs)/len(events) reaches 100 individually,
		//       but then need to provide ghEvent in order sorted from latest to oldest
		//       or fetchAndConvert should iterate in reverse order
		//       another win could be to convert fetchAndConvert to batch all queries in one GraphQL query
		fmt.Printf("populating more detail for %d github mail events\n", len(ghEvents))

		if len(ghEvents) > 0 {
			notifs, events, err := fetchAndConvert(context.Background(), s.clV4, ghEvents, s.user, s.rtr)
			if err != nil {
				log.Println("fetchAndConvert:", err)
				s.errorMu.Lock()
				s.error = fmt.Errorf("fetchAndConvert: %v", err)
				s.errorMu.Unlock()
			} else {
				s.mail.mu.Lock()
				// TODO: clean out too-old events more efficiently
				{
					s.mail.notifs = append(s.mail.notifs, notifs...)
					sort.SliceStable(s.mail.notifs, func(i, j int) bool { return s.mail.notifs[i].Time.After(s.mail.notifs[j].Time) })
					if len(s.mail.notifs) > 100 {
						s.mail.notifs = s.mail.notifs[:100]
					}
				}
				{
					s.mail.events = append(s.mail.events, events...)
					sort.SliceStable(s.mail.events, func(i, j int) bool { return s.mail.events[i].Time.After(s.mail.events[j].Time) })
					if len(s.mail.events) > 100 {
						s.mail.events = s.mail.events[:100]
					}
				}
				s.mail.mu.Unlock()

				// Add notifications to lastReadAt map, so we can
				// notify observers if said notification becomes read.
				s.notifs.mu.Lock()
				for _, n := range notifs {
					th := thread{n.Namespace, n.ThreadType, n.ThreadID}
					if t, ok := s.notifs.lastReadAt[th]; !ok || n.Time.After(t) {
						s.notifs.lastReadAt[th] = n.Time
					}
				}
				s.notifs.mu.Unlock()

				// Notify streaming observers.
				s.mail.chsMu.Lock()
				for ctx, ch := range s.mail.chs {
					if ctx.Err() != nil {
						delete(s.mail.chs, ctx)
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
				s.mail.chsMu.Unlock()

				st.HandledSegs = segs
				log.Println("set github HandledSegs")
				//err := jsonEncodeFile(context.Background(), s.fs, "pollnotifs.json", st)
				//if err != nil {
				//	return err
				//}

				s.errorMu.Lock()
				s.error = nil
				s.errorMu.Unlock()
			}
		}

		<-s.notifEvents
	}
}

// TODO: rename to something that fits notifications and events
type githubEvent struct {
	githubEventID
	Reason string // "your_activity", etc.
}

// OwnActivity reports whether this is an event rather a notification.
func (ge *githubEvent) OwnActivity() bool {
	return ge.Reason == "your_activity"
}

type githubEventID struct {
	// TODO: support other event types

	// Set only for issue opened and PR opened events.
	Owner, Repo string

	// Issue opened.
	IssueID uint64

	// Issue comment.
	IssueCommentID uint64

	// Issue events.
	ClosedEventID   uint64
	ReopenedEventID uint64

	// Pull request opened.
	PullRequestID uint64

	// Pull request comment.
	PRCommentID uint64

	// TODO: figure out what to do here, are there different types? review vs review comment vs inline, etc.
	// Pull request review.
	PRReviewID uint64

	// Pull request events.
	PRClosedEventID   uint64
	PRMergedEventID   uint64
	PRReopenedEventID uint64
}

// parseMessage parses a mail message from GitHub. It supports these types:
//
// 	Message-ID: <golang/go/issues/33986@github.com> - issue opened
// 	Message-ID: <golang/go/issue/30612/issue_event/2594639223@github.com> - issue closed/reopened
// 	Message-ID: <russross/blackfriday/issues/491/526805638@github.com> - issue comment
//
// 	Message-ID: <sourcegraph/go-vcs/pull/114@github.com> - PR opened
// 	Message-ID: <sourcegraph/go-vcs/pull/114/issue_event/2590234930@github.com> - PR closed/merged/reopened, review requested
// 	Message-ID: <sourcegraph/go-vcs/pull/114/c525709670@github.com> - PR comment
// 	Message-ID: <sourcegraph/go-vcs/pull/114/review/280337816@github.com> - PR review
//
func parseMessage(m *mail.Message) (githubEventID, error) {
	messageID := trimAngle(m.Header.Get("Message-ID"))
	if orig := m.Header.Get("X-Google-Original-Message-ID"); orig != "" {
		messageID = trimAngle(orig)
	}
	if !strings.HasSuffix(messageID, "@github.com") {
		return githubEventID{}, fmt.Errorf("no @github.com suffix in messageID %q", messageID)
	}
	parts := strings.Split(messageID[:len(messageID)-len("@github.com")], "/") // TODO: no allocs
	switch {
	case len(parts) == 4 && parts[2] == "issues":
		// An issue.
		issueID, err := strconv.ParseUint(parts[3], 10, 64)
		if err != nil {
			return githubEventID{}, err
		}
		return githubEventID{Owner: parts[0], Repo: parts[1], IssueID: issueID}, nil
	case len(parts) == 6 && parts[2] == "issue" && parts[4] == "issue_event":
		// An issue event.
		eventID, err := strconv.ParseUint(parts[5], 10, 64)
		if err != nil {
			return githubEventID{}, err
		}
		plain, err := parseTextPlain(m)
		if err != nil {
			return githubEventID{}, err
		}
		switch {
		case strings.HasPrefix(plain, "Closed "):
			return githubEventID{ClosedEventID: eventID}, nil
		case strings.HasPrefix(plain, "Reopened "):
			return githubEventID{ReopenedEventID: eventID}, nil
		case strings.HasPrefix(plain, "Assigned "):
			// TODO: consider handling it, etc.
			return githubEventID{}, nil
		default:
			return githubEventID{}, fmt.Errorf("unknown event type in %q", plain)
		}
	case len(parts) == 5 && parts[2] == "issues":
		// An issue comment.
		issueCommentID, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			return githubEventID{}, err
		}
		return githubEventID{IssueCommentID: issueCommentID}, nil

	case len(parts) == 4 && parts[2] == "pull":
		// A pull request.
		prID, err := strconv.ParseUint(parts[3], 10, 64)
		if err != nil {
			return githubEventID{}, err
		}
		return githubEventID{Owner: parts[0], Repo: parts[1], PullRequestID: prID}, nil
	case len(parts) == 5 && parts[2] == "pull" && strings.HasPrefix(parts[4], "c"):
		// A pull request comment.
		prCommentID, err := strconv.ParseUint(parts[4][len("c"):], 10, 64)
		if err != nil {
			return githubEventID{}, err
		}
		return githubEventID{PRCommentID: prCommentID}, nil
	case len(parts) == 6 && parts[2] == "pull" && parts[4] == "review":
		// A pull request review.
		prReviewID, err := strconv.ParseUint(parts[5], 10, 64)
		if err != nil {
			return githubEventID{}, err
		}
		// TODO: parse text to determine whatever needs to be determined?
		return githubEventID{PRReviewID: prReviewID}, nil
	case len(parts) == 6 && parts[2] == "pull" && parts[4] == "issue_event":
		// A pull request event.
		eventID, err := strconv.ParseUint(parts[5], 10, 64)
		if err != nil {
			return githubEventID{}, err
		}
		plain, err := parseTextPlain(m)
		if err != nil {
			return githubEventID{}, err
		}
		switch {
		case strings.HasPrefix(plain, "Closed "):
			return githubEventID{PRClosedEventID: eventID}, nil
		case strings.HasPrefix(plain, "Merged "):
			return githubEventID{PRMergedEventID: eventID}, nil
		case strings.HasPrefix(plain, "Reopened "):
			return githubEventID{PRReopenedEventID: eventID}, nil
		case strings.HasPrefix(plain, "Assigned "):
			// TODO: consider handling it, etc.
			return githubEventID{}, nil
		case strings.Contains(plain, " requested your review "):
			// TODO: consider handling it, etc.
			return githubEventID{}, nil
		default:
			return githubEventID{}, fmt.Errorf("unknown event type in %q", plain)
		}
	case len(parts) == 4 && parts[2] == "comments": // A comment on a gist. E.g., "dmitshur/gist:6927554/comments/3076334".
		// TODO: consider supporting comments on gists
		return githubEventID{}, nil

	default:
		return githubEventID{}, nil
	}
}

type notifAndURL struct {
	notification.Notification
	url            func(context.Context) string
	GitHubThreadID string // TODO: this worked for GitHub notifs API; maybe switch to beacon URL for mail notifs?
}

func (n notifAndURL) WithURL(ctx context.Context) notification.Notification {
	switch p := n.Payload.(type) {
	case notification.Issue:
		p.IssueHTMLURL = n.url(ctx)
		n.Payload = p
		return n.Notification
	case notification.IssueComment:
		p.CommentHTMLURL = n.url(ctx)
		n.Payload = p
		return n.Notification
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

type eventAndURL struct {
	ID githubEventID
	event.Event
	url func(context.Context) string
}

func (e eventAndURL) WithURL(ctx context.Context) event.Event {
	switch p := e.Payload.(type) {
	case event.Issue:
		p.IssueHTMLURL = e.url(ctx)
		e.Payload = p
		return e.Event
	case event.IssueComment:
		p.CommentHTMLURL = e.url(ctx)
		e.Payload = p
		return e.Event
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

// fetchAndConvert fetches additional information from GitHub API
// and converts GitHub notifications and events to own format.
func fetchAndConvert(
	ctx context.Context,
	clV4 *githubv4.Client,
	ghEvents []githubEvent,
	user users.User,
	rtr github.Router,
) ([]notifAndURL, []eventAndURL, error) {
	// TODO: see if it's worth arranging each ghEvents entry to be batched into a single GraphQL call

	var (
		notifs []notifAndURL
		events []eventAndURL
	)
	for _, e := range ghEvents {
		e := e

		// TODO: try to reduce verbosity further, factor out similar looking code, etc.
		participating := e.Reason != "subscribed" // According to https://developer.github.com/v3/activity/notifications/#notification-reasons, "subscribed" reason means "you're watching the repository", and all other reasons imply participation.

		switch {
		case e.IssueID != 0:
			var q struct {
				Repository struct {
					Name  string
					Owner struct{ Login string }
					GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
					Issue struct {
						CreatedAt time.Time
						Author    *githubV4Actor
						Title     string
						Body      string
					} `graphql:"issue(number:$issueNumber)"`
				} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
			}
			variables := map[string]interface{}{
				"repositoryOwner": githubv4.String(e.Owner),
				"repositoryName":  githubv4.String(e.Repo),
				"issueNumber":     githubv4.Int(e.IssueID),
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve ") { // E.g., because the repo or issue was deleted.
				log.Printf("fetchAndConvert: issue %s/%s/%d was not found: %v\n", e.Owner, e.Repo, e.IssueID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}

			modulePath := modulePath(q.Repository.GoMod, q.Repository.Owner.Login, q.Repository.Name)
			importPaths, issueTitle := prefixtitle.ParseIssue(modulePath, q.Repository.Issue.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + q.Repository.Owner.Login + "/" + q.Repository.Name,
						ThreadType: "Issue", // TODO: const?
						ThreadID:   e.IssueID,

						ImportPaths: importPaths,
						Time:        q.Repository.Issue.CreatedAt,
						Actor:       ghActor(q.Repository.Issue.Author),

						Payload: notification.Issue{
							Action:     "opened",
							IssueTitle: issueTitle,
							IssueBody:  q.Repository.Issue.Body,
						},

						Participating: participating,
						Mentioned:     e.Reason == "mention" && strings.Contains(q.Repository.Issue.Body, "@"+user.Login),
					},
					url: func(ctx context.Context) string {
						return rtr.IssueURL(ctx, q.Repository.Owner.Login, q.Repository.Name, e.IssueID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      q.Repository.Issue.CreatedAt,
						Actor:     ghActor(q.Repository.Issue.Author),
						Container: importPaths[0],
						Payload: event.Issue{
							Action:     "opened",
							IssueTitle: issueTitle,
							IssueBody:  q.Repository.Issue.Body,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.IssueURL(ctx, q.Repository.Owner.Login, q.Repository.Name, e.IssueID)
					},
				})
			}

		case e.IssueCommentID != 0:
			var q struct {
				Node struct {
					IssueComment struct {
						CreatedAt time.Time
						Author    *githubV4Actor
						Issue     struct {
							Number uint64
							Title  string
							State  githubv4.IssueState
						}
						Body       string
						Repository struct {
							Name  string
							Owner struct{ Login string }
							GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
						}
					} `graphql:"...on IssueComment"`
				} `graphql:"node(id:$issueCommentID)"`
			}
			variables := map[string]interface{}{
				"issueCommentID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("012:IssueComment%d", e.IssueCommentID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: issue comment %s/%s/%d/%d was not found: %v\n", e.Owner, e.Repo, e.IssueID, e.IssueCommentID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}
			ic := q.Node.IssueComment

			modulePath := modulePath(ic.Repository.GoMod, ic.Repository.Owner.Login, ic.Repository.Name)
			importPaths, issueTitle := prefixtitle.ParseIssue(modulePath, ic.Issue.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + ic.Repository.Owner.Login + "/" + ic.Repository.Name,
						ThreadType: "Issue", // TODO: const?
						ThreadID:   ic.Issue.Number,

						ImportPaths: importPaths,
						Time:        ic.CreatedAt,
						Actor:       ghActor(ic.Author),

						Payload: notification.IssueComment{
							IssueTitle:  issueTitle,
							IssueState:  ghIssueState(ic.Issue.State),
							CommentBody: ic.Body,
						},

						Participating: participating,
						Mentioned:     e.Reason == "mention" && strings.Contains(ic.Body, "@"+user.Login),
					},
					url: func(ctx context.Context) string {
						return rtr.IssueCommentURL(ctx, ic.Repository.Owner.Login, ic.Repository.Name, ic.Issue.Number, e.IssueCommentID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      ic.CreatedAt,
						Actor:     ghActor(ic.Author),
						Container: importPaths[0],
						Payload: event.IssueComment{
							IssueTitle:  issueTitle,
							IssueState:  ghIssueState(ic.Issue.State),
							CommentBody: ic.Body,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.IssueCommentURL(ctx, ic.Repository.Owner.Login, ic.Repository.Name, ic.Issue.Number, e.IssueCommentID)
					},
				})
			}

		case e.ClosedEventID != 0:
			var q struct {
				Node struct {
					ClosedEvent struct {
						CreatedAt time.Time
						Actor     *githubV4Actor
						Closable  struct {
							Issue struct {
								Number     uint64
								Title      string
								State      githubv4.IssueState
								Repository struct {
									Name  string
									Owner struct{ Login string }
									GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
								}
							} `graphql:"...on Issue"`
						}
					} `graphql:"...on ClosedEvent"`
				} `graphql:"node(id:$closedEventID)"`
			}
			variables := map[string]interface{}{
				"closedEventID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("011:ClosedEvent%d", e.ClosedEventID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: closed event %d was not found: %v\n", e.ClosedEventID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}
			ce := q.Node.ClosedEvent

			modulePath := modulePath(ce.Closable.Issue.Repository.GoMod, ce.Closable.Issue.Repository.Owner.Login, ce.Closable.Issue.Repository.Name)
			importPaths, issueTitle := prefixtitle.ParseIssue(modulePath, ce.Closable.Issue.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + ce.Closable.Issue.Repository.Owner.Login + "/" + ce.Closable.Issue.Repository.Name,
						ThreadType: "Issue", // TODO: const?
						ThreadID:   ce.Closable.Issue.Number,

						ImportPaths: importPaths,
						Time:        ce.CreatedAt,
						Actor:       ghActor(ce.Actor),

						Payload: notification.Issue{
							Action:     "closed",
							IssueTitle: issueTitle,
						},

						Participating: participating,
					},
					url: func(ctx context.Context) string {
						return rtr.IssueEventURL(ctx,
							ce.Closable.Issue.Repository.Owner.Login,
							ce.Closable.Issue.Repository.Name,
							ce.Closable.Issue.Number,
							e.ClosedEventID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      ce.CreatedAt,
						Actor:     ghActor(ce.Actor),
						Container: importPaths[0],
						Payload: event.Issue{
							Action:     "closed",
							IssueTitle: issueTitle,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.IssueEventURL(ctx,
							ce.Closable.Issue.Repository.Owner.Login,
							ce.Closable.Issue.Repository.Name,
							ce.Closable.Issue.Number,
							e.ClosedEventID)
					},
				})
			}
		case e.ReopenedEventID != 0:
			var q struct {
				Node struct {
					ReopenedEvent struct {
						CreatedAt time.Time
						Actor     *githubV4Actor
						Closable  struct {
							Issue struct {
								Number     uint64
								Title      string
								State      githubv4.IssueState
								Repository struct {
									Name  string
									Owner struct{ Login string }
									GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
								}
							} `graphql:"...on Issue"`
						}
					} `graphql:"...on ReopenedEvent"`
				} `graphql:"node(id:$reopenedEventID)"`
			}
			variables := map[string]interface{}{
				"reopenedEventID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("013:ReopenedEvent%d", e.ReopenedEventID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: reopened event %d was not found: %v\n", e.ReopenedEventID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}
			re := q.Node.ReopenedEvent

			modulePath := modulePath(re.Closable.Issue.Repository.GoMod, re.Closable.Issue.Repository.Owner.Login, re.Closable.Issue.Repository.Name)
			importPaths, issueTitle := prefixtitle.ParseIssue(modulePath, re.Closable.Issue.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + re.Closable.Issue.Repository.Owner.Login + "/" + re.Closable.Issue.Repository.Name,
						ThreadType: "Issue", // TODO: const?
						ThreadID:   re.Closable.Issue.Number,

						ImportPaths: importPaths,
						Time:        re.CreatedAt,
						Actor:       ghActor(re.Actor),

						Payload: notification.Issue{
							Action:     "reopened",
							IssueTitle: issueTitle,
						},

						Participating: participating,
					},
					url: func(ctx context.Context) string {
						return rtr.IssueEventURL(ctx,
							re.Closable.Issue.Repository.Owner.Login,
							re.Closable.Issue.Repository.Name,
							re.Closable.Issue.Number,
							e.ReopenedEventID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      re.CreatedAt,
						Actor:     ghActor(re.Actor),
						Container: importPaths[0],
						Payload: event.Issue{
							Action:     "reopened",
							IssueTitle: issueTitle,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.IssueEventURL(ctx,
							re.Closable.Issue.Repository.Owner.Login,
							re.Closable.Issue.Repository.Name,
							re.Closable.Issue.Number,
							e.ReopenedEventID)
					},
				})
			}

		case e.PullRequestID != 0:
			var q struct {
				Repository struct {
					Name        string
					Owner       struct{ Login string }
					GoMod       *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
					PullRequest struct {
						CreatedAt time.Time
						Author    *githubV4Actor
						Title     string
						Body      string
					} `graphql:"pullRequest(number:$prNumber)"`
				} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
			}
			variables := map[string]interface{}{
				"repositoryOwner": githubv4.String(e.Owner),
				"repositoryName":  githubv4.String(e.Repo),
				"prNumber":        githubv4.Int(e.PullRequestID),
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: PR %s/%s/%d was not found: %v\n", e.Owner, e.Repo, e.PullRequestID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}

			modulePath := modulePath(q.Repository.GoMod, q.Repository.Owner.Login, q.Repository.Name)
			importPaths, changeTitle := prefixtitle.ParseChange(modulePath, q.Repository.PullRequest.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + q.Repository.Owner.Login + "/" + q.Repository.Name,
						ThreadType: "PullRequest", // TODO: const?
						ThreadID:   e.PullRequestID,

						ImportPaths: importPaths,
						Time:        q.Repository.PullRequest.CreatedAt,
						Actor:       ghActor(q.Repository.PullRequest.Author),

						Payload: notification.Change{
							Action:      "opened",
							ChangeTitle: changeTitle,
							ChangeBody:  q.Repository.PullRequest.Body,
						},

						Participating: participating,
						Mentioned: e.Reason == "mention" && strings.Contains(q.Repository.PullRequest.Body, "@"+user.Login) ||
							e.Reason == "review_requested",
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestURL(ctx, q.Repository.Owner.Login, q.Repository.Name, e.PullRequestID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      q.Repository.PullRequest.CreatedAt,
						Actor:     ghActor(q.Repository.PullRequest.Author),
						Container: importPaths[0],
						Payload: event.Change{
							Action:      "opened",
							ChangeTitle: changeTitle,
							ChangeBody:  q.Repository.PullRequest.Body,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestURL(ctx, q.Repository.Owner.Login, q.Repository.Name, e.PullRequestID)
					},
				})
			}

		case e.PRCommentID != 0:
			var q struct {
				Node struct {
					IssueComment struct {
						CreatedAt   time.Time
						Author      *githubV4Actor
						PullRequest struct {
							Number uint64
							Title  string
							State  githubv4.PullRequestState
						}
						Body       string
						Repository struct {
							Name  string
							Owner struct{ Login string }
							GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
						}
					} `graphql:"...on IssueComment"`
				} `graphql:"node(id:$prCommentID)"`
			}
			variables := map[string]interface{}{
				"prCommentID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("012:IssueComment%d", e.PRCommentID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: PR comment %s/%s/%d/%d was not found: %v\n", e.Owner, e.Repo, e.IssueID, e.PRCommentID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}
			ic := q.Node.IssueComment

			modulePath := modulePath(ic.Repository.GoMod, ic.Repository.Owner.Login, ic.Repository.Name)
			importPaths, changeTitle := prefixtitle.ParseChange(modulePath, ic.PullRequest.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + ic.Repository.Owner.Login + "/" + ic.Repository.Name,
						ThreadType: "PullRequest", // TODO: const?
						ThreadID:   ic.PullRequest.Number,

						ImportPaths: importPaths,
						Time:        ic.CreatedAt,
						Actor:       ghActor(ic.Author),

						Payload: notification.ChangeComment{
							ChangeTitle: changeTitle,
							ChangeState: ghChangeState(ic.PullRequest.State),
							CommentBody: ic.Body,
						},

						Participating: participating,
						Mentioned:     e.Reason == "mention" && strings.Contains(ic.Body, "@"+user.Login),
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestCommentURL(ctx, ic.Repository.Owner.Login, ic.Repository.Name, ic.PullRequest.Number, e.PRCommentID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      ic.CreatedAt,
						Actor:     ghActor(ic.Author),
						Container: importPaths[0],
						Payload: event.ChangeComment{
							ChangeTitle: changeTitle,
							ChangeState: ghChangeState(ic.PullRequest.State),
							CommentBody: ic.Body,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestCommentURL(ctx, ic.Repository.Owner.Login, ic.Repository.Name, ic.PullRequest.Number, e.PRCommentID)
					},
				})
			}

		case e.PRReviewID != 0:
			var q struct {
				Node struct {
					PullRequestReview struct {
						CreatedAt   time.Time
						Author      *githubV4Actor
						PullRequest struct {
							Number uint64
							Title  string
							State  githubv4.PullRequestState
						}
						Body              string
						State             githubv4.PullRequestReviewState
						AuthorAssociation githubv4.CommentAuthorAssociation
						Comments          struct {
							Nodes []struct {
								Body string
							}
						} `graphql:"comments(first:10)"`
						Repository struct {
							Name  string
							Owner struct{ Login string }
							GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
						}
					} `graphql:"...on PullRequestReview"`
				} `graphql:"node(id:$prReviewID)"`
			}
			variables := map[string]interface{}{
				"prReviewID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("017:PullRequestReview%d", e.PRReviewID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: PR review %s/%s/%d/%d was not found: %v\n", e.Owner, e.Repo, e.IssueID, e.PRReviewID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}
			prr := q.Node.PullRequestReview

			modulePath := modulePath(prr.Repository.GoMod, prr.Repository.Owner.Login, prr.Repository.Name)
			importPaths, changeTitle := prefixtitle.ParseChange(modulePath, prr.PullRequest.Title)

			reviewState, ok := ghReviewState(prr.State, prr.AuthorAssociation)
			if !ok {
				log.Printf("fetchAndConvert: PR review %s/%s/%d/%d had not ok ReviewState\n", e.Owner, e.Repo, e.IssueID, e.PRReviewID)
				continue
			}
			body := prr.Body
			for _, c := range prr.Comments.Nodes {
				body += "\n\n" + c.Body
			}

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + prr.Repository.Owner.Login + "/" + prr.Repository.Name,
						ThreadType: "PullRequest", // TODO: const?
						ThreadID:   prr.PullRequest.Number,

						ImportPaths: importPaths,
						Time:        prr.CreatedAt,
						Actor:       ghActor(prr.Author),

						Payload: notification.ChangeComment{
							ChangeTitle:   changeTitle,
							ChangeState:   ghChangeState(prr.PullRequest.State),
							CommentBody:   body,
							CommentReview: reviewState,
						},

						Participating: participating,
						Mentioned:     e.Reason == "mention" && strings.Contains(prr.Body, "@"+user.Login),
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestReviewURL(ctx, prr.Repository.Owner.Login, prr.Repository.Name, prr.PullRequest.Number, e.PRReviewID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      prr.CreatedAt,
						Actor:     ghActor(prr.Author),
						Container: importPaths[0],
						Payload: event.ChangeComment{
							ChangeTitle:   changeTitle,
							ChangeState:   ghChangeState(prr.PullRequest.State),
							CommentBody:   body,
							CommentReview: reviewState,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestReviewURL(ctx, prr.Repository.Owner.Login, prr.Repository.Name, prr.PullRequest.Number, e.PRReviewID)
					},
				})
			}

		case e.PRClosedEventID != 0:
			var q struct {
				Node struct {
					ClosedEvent struct {
						CreatedAt time.Time
						Actor     *githubV4Actor
						Closable  struct {
							PullRequest struct {
								Number     uint64
								Title      string
								State      githubv4.PullRequestState
								Repository struct {
									Name  string
									Owner struct{ Login string }
									GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
								}
							} `graphql:"...on PullRequest"`
						}
					} `graphql:"...on ClosedEvent"`
				} `graphql:"node(id:$closedEventID)"`
			}
			variables := map[string]interface{}{
				"closedEventID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("011:ClosedEvent%d", e.PRClosedEventID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: closed event %d was not found: %v\n", e.PRClosedEventID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}
			ce := q.Node.ClosedEvent

			modulePath := modulePath(ce.Closable.PullRequest.Repository.GoMod, ce.Closable.PullRequest.Repository.Owner.Login, ce.Closable.PullRequest.Repository.Name)
			importPaths, changeTitle := prefixtitle.ParseChange(modulePath, ce.Closable.PullRequest.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + ce.Closable.PullRequest.Repository.Owner.Login + "/" + ce.Closable.PullRequest.Repository.Name,
						ThreadType: "PullRequest", // TODO: const?
						ThreadID:   ce.Closable.PullRequest.Number,

						ImportPaths: importPaths,
						Time:        ce.CreatedAt,
						Actor:       ghActor(ce.Actor),

						Payload: notification.Change{
							Action:      "closed",
							ChangeTitle: changeTitle,
						},

						Participating: participating,
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestEventURL(ctx,
							ce.Closable.PullRequest.Repository.Owner.Login,
							ce.Closable.PullRequest.Repository.Name,
							ce.Closable.PullRequest.Number,
							e.PRClosedEventID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      ce.CreatedAt,
						Actor:     ghActor(ce.Actor),
						Container: importPaths[0],
						Payload: event.Change{
							Action:      "closed",
							ChangeTitle: changeTitle,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestEventURL(ctx,
							ce.Closable.PullRequest.Repository.Owner.Login,
							ce.Closable.PullRequest.Repository.Name,
							ce.Closable.PullRequest.Number,
							e.PRClosedEventID)
					},
				})
			}
		case e.PRMergedEventID != 0:
			var q struct {
				Node struct {
					MergedEvent struct {
						CreatedAt   time.Time
						Actor       *githubV4Actor
						PullRequest struct {
							Number     uint64
							Title      string
							State      githubv4.PullRequestState
							Repository struct {
								Name  string
								Owner struct{ Login string }
								GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
							}
						}
					} `graphql:"...on MergedEvent"`
				} `graphql:"node(id:$mergedEventID)"`
			}
			variables := map[string]interface{}{
				"mergedEventID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("011:MergedEvent%d", e.PRMergedEventID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: merged event %d was not found: %v\n", e.PRMergedEventID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}
			me := q.Node.MergedEvent

			modulePath := modulePath(me.PullRequest.Repository.GoMod, me.PullRequest.Repository.Owner.Login, me.PullRequest.Repository.Name)
			importPaths, changeTitle := prefixtitle.ParseChange(modulePath, me.PullRequest.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + me.PullRequest.Repository.Owner.Login + "/" + me.PullRequest.Repository.Name,
						ThreadType: "PullRequest", // TODO: const?
						ThreadID:   me.PullRequest.Number,

						ImportPaths: importPaths,
						Time:        me.CreatedAt,
						Actor:       ghActor(me.Actor),

						Payload: notification.Change{
							Action:      "merged",
							ChangeTitle: changeTitle,
						},

						Participating: participating,
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestEventURL(ctx,
							me.PullRequest.Repository.Owner.Login,
							me.PullRequest.Repository.Name,
							me.PullRequest.Number,
							e.PRMergedEventID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      me.CreatedAt,
						Actor:     ghActor(me.Actor),
						Container: importPaths[0],
						Payload: event.Change{
							Action:      "merged",
							ChangeTitle: changeTitle,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestEventURL(ctx,
							me.PullRequest.Repository.Owner.Login,
							me.PullRequest.Repository.Name,
							me.PullRequest.Number,
							e.PRMergedEventID)
					},
				})
			}
		case e.PRReopenedEventID != 0:
			var q struct {
				Node struct {
					ReopenedEvent struct {
						CreatedAt time.Time
						Actor     *githubV4Actor
						Closable  struct {
							PullRequest struct {
								Number     uint64
								Title      string
								State      githubv4.PullRequestState
								Repository struct {
									Name  string
									Owner struct{ Login string }
									GoMod *goModFragment `graphql:"object(expression:\"HEAD:go.mod\")"`
								}
							} `graphql:"...on PullRequest"`
						}
					} `graphql:"...on ReopenedEvent"`
				} `graphql:"node(id:$reopenedEventID)"`
			}
			variables := map[string]interface{}{
				"reopenedEventID": githubv4.ID(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("013:ReopenedEvent%d", e.PRReopenedEventID)))), // HACK, TODO: Confirm StdEncoding vs URLEncoding.
			}
			err := clV4.Query(ctx, &q, variables)
			if err != nil && strings.HasPrefix(err.Error(), "Could not resolve to a node ") { // E.g., because the repo was deleted.
				log.Printf("fetchAndConvert: reopened event %d was not found: %v\n", e.PRReopenedEventID, err)
				continue
			} else if err != nil {
				return nil, nil, err
			}
			re := q.Node.ReopenedEvent

			modulePath := modulePath(re.Closable.PullRequest.Repository.GoMod, re.Closable.PullRequest.Repository.Owner.Login, re.Closable.PullRequest.Repository.Name)
			importPaths, changeTitle := prefixtitle.ParseChange(modulePath, re.Closable.PullRequest.Title)

			if !e.OwnActivity() {
				notifs = append(notifs, notifAndURL{
					Notification: notification.Notification{
						Namespace:  "github.com/" + re.Closable.PullRequest.Repository.Owner.Login + "/" + re.Closable.PullRequest.Repository.Name,
						ThreadType: "PullRequest", // TODO: const?
						ThreadID:   re.Closable.PullRequest.Number,

						ImportPaths: importPaths,
						Time:        re.CreatedAt,
						Actor:       ghActor(re.Actor),

						Payload: notification.Change{
							Action:      "reopened",
							ChangeTitle: changeTitle,
						},

						Participating: participating,
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestEventURL(ctx,
							re.Closable.PullRequest.Repository.Owner.Login,
							re.Closable.PullRequest.Repository.Name,
							re.Closable.PullRequest.Number,
							e.PRReopenedEventID)
					},
					GitHubThreadID: "", // TODO
				})
			} else {
				events = append(events, eventAndURL{
					ID: e.githubEventID,
					Event: event.Event{
						Time:      re.CreatedAt,
						Actor:     ghActor(re.Actor),
						Container: importPaths[0],
						Payload: event.Change{
							Action:      "reopened",
							ChangeTitle: changeTitle,
						},
					},
					url: func(ctx context.Context) string {
						return rtr.PullRequestEventURL(ctx,
							re.Closable.PullRequest.Repository.Owner.Login,
							re.Closable.PullRequest.Repository.Name,
							re.Closable.PullRequest.Number,
							e.PRReopenedEventID)
					},
				})
			}
		}
	}
	return notifs, events, nil
}

// modulePath returns the module path for the specified repository.
// If the repository has no go.mod file, or if the go.mod file fails to parse,
// then "github.com/"+owner+"/"+repo is returned as the module path.
//
// For the main Go repository (i.e., https://github.com/golang/go),
// the empty string is returned.
func modulePath(goMod *goModFragment, owner, repo string) (modulePath string) {
	if owner == "golang" && repo == "go" {
		// Use empty string as the module path for the main Go repository.
		return ""
	}
	if goMod == nil {
		// No go.mod file, so the module path must be equal to the repo path.
		return "github.com/" + owner + "/" + repo
	}
	modulePath = modfile.ModulePath([]byte(goMod.Blob.Text))
	if modulePath == "" {
		// No module path found in go.mod file, so fall back to using the repo path.
		return "github.com/" + owner + "/" + repo
	}
	return modulePath
}

type goModFragment struct {
	Blob struct {
		Text string
	} `graphql:"...on Blob"`
}

type githubV4Actor struct {
	User struct {
		DatabaseID uint64
	} `graphql:"...on User"`
	Bot struct {
		DatabaseID uint64
	} `graphql:"...on Bot"`
	Login     string
	AvatarURL string `graphql:"avatarUrl(size:96)"`
	URL       string
}

func ghActor(actor *githubV4Actor) users.User {
	if actor == nil {
		return ghost // Deleted user, replace with https://github.com/ghost.
	}
	return users.User{
		UserSpec: users.UserSpec{
			ID:     actor.User.DatabaseID | actor.Bot.DatabaseID,
			Domain: "github.com",
		},
		Login:     actor.Login,
		AvatarURL: actor.AvatarURL,
		HTMLURL:   actor.URL,
	}
}

// ghost is https://github.com/ghost, a replacement for deleted users.
var ghost = users.User{
	UserSpec: users.UserSpec{
		ID:     10137,
		Domain: "github.com",
	},
	Login:     "ghost",
	AvatarURL: "https://avatars3.githubusercontent.com/u/10137?v=4",
	HTMLURL:   "https://github.com/ghost",
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

func parseTextPlain(m *mail.Message) (string, error) {
	var boundary string
	if mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type")); err != nil {
		return "", err
	} else if !strings.HasPrefix(mediaType, "multipart/") {
		return "", fmt.Errorf("Content-Type not multipart/*: %q", mediaType)
	} else {
		boundary = params["boundary"]
	}
	mr := multipart.NewReader(m.Body, boundary)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return "", fmt.Errorf("no text/plain part in message")
		} else if err != nil {
			return "", err
		}
		if mediaType, _, err := mime.ParseMediaType(p.Header.Get("Content-Type")); err != nil {
			return "", err
		} else if mediaType != "text/plain" {
			continue
		}
		switch p.Header.Get("Content-Transfer-Encoding") {
		case "7bit", "":
			var buf bytes.Buffer
			_, err := io.Copy(&buf, p)
			return buf.String(), err
		case "base64":
			var buf bytes.Buffer
			_, err := io.Copy(&buf, base64.NewDecoder(base64.StdEncoding, p))
			return buf.String(), err
		default:
			return "", fmt.Errorf("unsupported Content-Transfer-Encoding value %q", p.Header.Get("Content-Transfer-Encoding"))
		}
	}
}

func trimAngle(s string) string {
	return strings.TrimSuffix(strings.TrimPrefix(s, "<"), ">")
}
