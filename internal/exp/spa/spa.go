// +build go1.14

// Package spa implements a single-page application
// used on the dmitri.shuralyov.com website.
//
// It is capable of
// serving page HTML on the frontend and backend, and
// setting page state on the frontend.
package spa

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	statepkg "dmitri.shuralyov.com/state"
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/home/internal/exp/app/changesapp"
	"github.com/shurcooL/home/internal/exp/app/issuesapp"
	"github.com/shurcooL/home/internal/exp/app/notifsapp"
	"github.com/shurcooL/home/internal/exp/service/change"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// An App is a single-page application.
type App interface {
	// ServePage renders the page HTML for reqURL to w,
	// and returns the state it computed.
	//
	// It returns an error of OutOfScopeError type
	// if reqURL is out of scope for the app.
	//
	// The returned state must implement the PageState interface.
	ServePage(ctx context.Context, w io.Writer, reqURL *url.URL) (interface{}, error)

	// SetupPage sets up the frontend page state,
	// using state returned from ServePage.
	SetupPage(ctx context.Context, state interface{})
}

type PageState interface {
	// RequestURL returns the request URL for the page.
	RequestURL() *url.URL
}

func NewApp(
	codeService interface {
		// ListDirectories lists directories in sorted order.
		ListDirectories(ctx context.Context) ([]*code.Directory, error)

		// GetDirectory looks up a directory by specified import path.
		// If the directory doesn't exist, os.ErrNotExist is returned.
		GetDirectory(ctx context.Context, importPath string) (*code.Directory, error)
	},
	issueService issues.Service,
	changeService change.Service,
	notifService notification.Service,
	userService users.Service,
	redirect func(*url.URL), // Only needed on frontend.
) *app {
	issuesApp := issuesapp.New(
		issueService,
		userService,
		redirect,
		issuesapp.Options{
			Notification: notifService,
			BodyTop: func(ctx context.Context, st issuesapp.State) ([]htmlg.Component, error) {
				var nc uint64
				if st.CurrentUser.ID != 0 {
					var err error
					nc, err = notifService.CountNotifications(ctx)
					if err != nil {
						log.Println("notifService.CountNotifications:", err)
					}
				}

				header := homecomponent.Header{
					CurrentUser:       st.CurrentUser,
					NotificationCount: nc,
					ReturnURL:         st.ReqURL.String(),
				}

				switch {
				default:
					return []htmlg.Component{header}, nil

				case strings.HasPrefix(st.RepoSpec.URI, "dmitri.shuralyov.com/"):
					// TODO: Maybe try to avoid fetching openIssues twice...
					t0 := time.Now()
					d, err := codeService.GetDirectory(ctx, st.RepoSpec.URI)
					if err != nil {
						return nil, err
					}
					openIssues, err := issueService.Count(ctx, st.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueOpen)})
					if err != nil {
						return nil, err
					}
					openChanges, err := changeService.Count(ctx, st.RepoSpec.URI, change.ListOptions{Filter: change.FilterOpen})
					if err != nil {
						return nil, err
					}
					fmt.Println("counting packages & open issues & changes took:", time.Since(t0).Milliseconds(), "ms", "for:", st.RepoSpec.URI)

					heading := htmlg.NodeComponent{
						Type: html.ElementNode, Data: atom.H2.String(),
						FirstChild: htmlg.Text(st.RepoSpec.URI + "/..."),
					}
					repoPath := strings.TrimPrefix(st.RepoSpec.URI, "dmitri.shuralyov.com")
					tabnav := homecomponent.RepositoryTabNav(homecomponent.IssuesTab, repoPath, d.RepoPackages, openIssues, openChanges)
					return []htmlg.Component{header, heading, tabnav}, nil

				// TODO: Dedup with changes (maybe; mind the githubURL difference).
				case strings.HasPrefix(st.RepoSpec.URI, "github.com/"):
					// TODO: Maybe try to avoid fetching openIssues twice...
					t0 := time.Now()
					openIssues, err := issueService.Count(ctx, st.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueOpen)})
					if err != nil {
						return nil, err
					}
					openChanges, err := changeService.Count(ctx, st.RepoSpec.URI, change.ListOptions{Filter: change.FilterOpen})
					if err != nil {
						return nil, err
					}
					fmt.Println("counting open issues & changes took:", time.Since(t0).Milliseconds(), "ms", "for:", st.RepoSpec.URI)

					heading := &html.Node{
						Type: html.ElementNode, Data: atom.H2.String(),
					}
					heading.AppendChild(htmlg.Text(st.RepoSpec.URI + "/..."))
					var githubURL string
					switch {
					case st.RepoSpec.URI != "github.com/shurcooL/issuesapp" && st.RepoSpec.URI != "github.com/shurcooL/notificationsapp" &&
						st.IssueID == 0:
						githubURL = fmt.Sprintf("https://%s/issues", st.RepoSpec.URI)
					case st.RepoSpec.URI != "github.com/shurcooL/issuesapp" && st.RepoSpec.URI != "github.com/shurcooL/notificationsapp" &&
						st.IssueID != 0:
						githubURL = fmt.Sprintf("https://%s/issues/%d", st.RepoSpec.URI, st.IssueID)
					default:
						githubURL = "https://" + st.RepoSpec.URI
					}
					heading.AppendChild(&html.Node{
						Type: html.ElementNode, Data: atom.A.String(),
						Attr: []html.Attribute{
							{Key: atom.Href.String(), Val: githubURL},
							{Key: atom.Class.String(), Val: "gray"},
							{Key: atom.Style.String(), Val: "margin-left: 10px;"},
						},
						FirstChild: octicon.SetSize(octicon.MarkGitHub(), 24),
					})
					tabnav := homecomponent.TabNav{
						Tabs: []homecomponent.Tab{
							{
								Content: contentCounter{
									Content: iconText{Icon: octicon.IssueOpened, Text: "Issues"},
									Count:   int(openIssues),
								},
								URL: "/issues/" + st.RepoSpec.URI, OnClick: "Open(event, this)",
								Selected: true,
							},
							{
								Content: contentCounter{
									Content: iconText{Icon: octicon.GitPullRequest, Text: "Changes"},
									Count:   int(openChanges),
								},
								URL: "/changes/" + st.RepoSpec.URI, OnClick: "Open(event, this)",
							},
						},
					}
					return []htmlg.Component{header, htmlg.NodeComponent(*heading), tabnav}, nil
				}
			},
		},
	)
	changesApp := changesapp.New(
		changeService,
		userService,
		redirect,
		changesapp.Options{
			Notification: notifService,
			BodyTop: func(ctx context.Context, st changesapp.State) ([]htmlg.Component, error) {
				var nc uint64
				if st.CurrentUser.ID != 0 {
					var err error
					nc, err = notifService.CountNotifications(ctx)
					if err != nil {
						log.Println("notifService.CountNotifications:", err)
					}
				}

				header := homecomponent.Header{
					CurrentUser:       st.CurrentUser,
					NotificationCount: nc,
					ReturnURL:         st.ReqURL.String(),
				}

				switch {
				default:
					return []htmlg.Component{header}, nil

				case strings.HasPrefix(st.RepoSpec, "dmitri.shuralyov.com/"):
					// TODO: Maybe try to avoid fetching openChanges twice...
					t0 := time.Now()
					d, err := codeService.GetDirectory(ctx, st.RepoSpec)
					if err != nil {
						return nil, err
					}
					openIssues, err := issueService.Count(ctx, issues.RepoSpec{URI: st.RepoSpec}, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueOpen)})
					if err != nil {
						return nil, err
					}
					openChanges, err := changeService.Count(ctx, st.RepoSpec, change.ListOptions{Filter: change.FilterOpen})
					if err != nil {
						return nil, err
					}
					fmt.Println("counting packages & open issues & changes took:", time.Since(t0).Milliseconds(), "ms", "for:", st.RepoSpec)

					heading := htmlg.NodeComponent{
						Type: html.ElementNode, Data: atom.H2.String(),
						FirstChild: htmlg.Text(st.RepoSpec + "/..."),
					}
					repoPath := strings.TrimPrefix(st.RepoSpec, "dmitri.shuralyov.com")
					tabnav := homecomponent.RepositoryTabNav(homecomponent.ChangesTab, repoPath, d.RepoPackages, openIssues, openChanges)
					return []htmlg.Component{header, heading, tabnav}, nil

				// TODO: Dedup with issues (maybe; mind the githubURL difference).
				case strings.HasPrefix(st.RepoSpec, "github.com/"):
					// TODO: Maybe try to avoid fetching openChanges twice...
					t0 := time.Now()
					openIssues, err := issueService.Count(ctx, issues.RepoSpec{URI: st.RepoSpec}, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueOpen)})
					if err != nil {
						return nil, err
					}
					openChanges, err := changeService.Count(ctx, st.RepoSpec, change.ListOptions{Filter: change.FilterOpen})
					if err != nil {
						return nil, err
					}
					fmt.Println("counting open issues & changes took:", time.Since(t0).Milliseconds(), "ms", "for:", st.RepoSpec)

					heading := &html.Node{
						Type: html.ElementNode, Data: atom.H2.String(),
					}
					heading.AppendChild(htmlg.Text(st.RepoSpec + "/..."))
					var githubURL string
					switch st.ChangeID {
					case 0:
						githubURL = fmt.Sprintf("https://%s/pulls", st.RepoSpec)
					default:
						githubURL = fmt.Sprintf("https://%s/pull/%d", st.RepoSpec, st.ChangeID)
					}
					heading.AppendChild(&html.Node{
						Type: html.ElementNode, Data: atom.A.String(),
						Attr: []html.Attribute{
							{Key: atom.Href.String(), Val: githubURL},
							{Key: atom.Class.String(), Val: "gray"},
							{Key: atom.Style.String(), Val: "margin-left: 10px;"},
						},
						FirstChild: octicon.SetSize(octicon.MarkGitHub(), 24),
					})
					tabnav := homecomponent.TabNav{
						Tabs: []homecomponent.Tab{
							{
								Content: contentCounter{
									Content: iconText{Icon: octicon.IssueOpened, Text: "Issues"},
									Count:   int(openIssues),
								},
								URL: "/issues/" + st.RepoSpec, OnClick: "Open(event, this)",
							},
							{
								Content: contentCounter{
									Content: iconText{Icon: octicon.GitPullRequest, Text: "Changes"},
									Count:   int(openChanges),
								},
								URL: "/changes/" + st.RepoSpec, OnClick: "Open(event, this)",
								Selected: true,
							},
						},
					}
					return []htmlg.Component{header, htmlg.NodeComponent(*heading), tabnav}, nil

				case strings.HasPrefix(st.RepoSpec, "go.googlesource.com/"):
					project := st.RepoSpec[len("go.googlesource.com/"):]

					// TODO: Maybe try to avoid fetching openChanges twice...
					t0 := time.Now()
					openChanges, err := changeService.Count(ctx, st.RepoSpec, change.ListOptions{Filter: change.FilterOpen})
					if err != nil {
						return nil, err
					}
					fmt.Println("counting open changes took:", time.Since(t0).Milliseconds(), "ms", "for:", st.RepoSpec)

					heading := &html.Node{
						Type: html.ElementNode, Data: atom.H2.String(),
					}
					heading.AppendChild(htmlg.Text(st.RepoSpec + "/..."))
					var gerritURL string
					switch st.ChangeID {
					case 0:
						gerritURL = fmt.Sprintf("https://go-review.googlesource.com/q/project:%s+status:open", project)
					default:
						gerritURL = fmt.Sprintf("https://go-review.googlesource.com/c/%s/+/%d", project, st.ChangeID)
					}
					heading.AppendChild(&html.Node{
						Type: html.ElementNode, Data: atom.A.String(),
						Attr: []html.Attribute{
							{Key: atom.Href.String(), Val: gerritURL},
							{Key: atom.Class.String(), Val: "gray"},
							{Key: atom.Style.String(), Val: "margin-left: 10px;"},
						},
						FirstChild: octicon.SetSize(octicon.Squirrel(), 24),
					})
					tabnav := homecomponent.TabNav{
						Tabs: []homecomponent.Tab{
							{
								Content: contentCounter{
									Content: iconText{Icon: octicon.GitPullRequest, Text: "Changes"},
									Count:   int(openChanges),
								},
								URL: "/changes/" + st.RepoSpec, OnClick: "Open(event, this)",
								Selected: true,
							},
						},
					}
					return []htmlg.Component{header, htmlg.NodeComponent(*heading), tabnav}, nil
				}
			},
		},
	)
	notifsApp := notifsapp.New(
		notifService,
		userService,
		notifsapp.Options{
			BodyTop: func(ctx context.Context, st notifsapp.State) ([]htmlg.Component, error) {
				var nc uint64
				if st.CurrentUser.UserSpec != (users.UserSpec{}) {
					var err error
					nc, err = notifService.CountNotifications(ctx)
					if err != nil {
						log.Println("notifService.CountNotifications:", err)
					}
				}
				header := homecomponent.Header{
					CurrentUser:       st.CurrentUser,
					NotificationCount: nc,
					ReturnURL:         st.ReqURL.String(),
				}
				return []htmlg.Component{header}, nil
			},
		},
	)
	return &app{
		IssuesApp:  issuesApp,
		ChangesApp: changesApp,
		NotifsApp:  notifsApp,
	}
}

type app struct {
	IssuesApp  App
	ChangesApp App
	NotifsApp  App
}

func (a *app) ServePage(ctx context.Context, w io.Writer, reqURL *url.URL) (interface{}, error) {
	switch {
	case strings.HasPrefix(reqURL.Path, "/issues/"),
		strings.HasSuffix(reqURL.Path, "$issues"), strings.Contains(reqURL.Path, "$issues/"):
		return a.IssuesApp.ServePage(ctx, w, reqURL)
	case strings.HasPrefix(reqURL.Path, "/changes/"),
		strings.HasSuffix(reqURL.Path, "$changes"), strings.Contains(reqURL.Path, "$changes/"):
		return a.ChangesApp.ServePage(ctx, w, reqURL)
	case reqURL.Path == "/notifications",
		strings.HasPrefix(reqURL.Path, "/notifications/") && reqURL.Path != "/notifications/status":
		return a.NotifsApp.ServePage(ctx, w, reqURL)
	default:
		return nil, OutOfScopeError{URL: reqURL}
	}
}

func (a *app) SetupPage(ctx context.Context, state interface{}) {
	// TODO: Make this safer and better.
	switch reqURL := state.(PageState).RequestURL(); {
	case strings.HasPrefix(reqURL.Path, "/issues/"),
		strings.HasSuffix(reqURL.Path, "$issues"), strings.Contains(reqURL.Path, "$issues/"):
		a.IssuesApp.SetupPage(ctx, state)
	case strings.HasPrefix(reqURL.Path, "/changes/"),
		strings.HasSuffix(reqURL.Path, "$changes"), strings.Contains(reqURL.Path, "$changes/"):
		a.ChangesApp.SetupPage(ctx, state)
	case reqURL.Path == "/notifications",
		strings.HasPrefix(reqURL.Path, "/notifications/") && reqURL.Path != "/notifications/status":
		a.NotifsApp.SetupPage(ctx, state)
	}
}

// OutOfScopeError is an error returned when the requested page
// is out of scope for the single-page application and therefore
// cannot be served by it directly.
type OutOfScopeError struct {
	// URL is the URL of the requested page.
	URL *url.URL
}

func (o OutOfScopeError) Error() string { return fmt.Sprintf("%s is out of scope", o.URL) }

// IsOutOfScope reports whether err is an OutOfScopeError error.
func IsOutOfScope(err error) (OutOfScopeError, bool) {
	e, ok := err.(OutOfScopeError)
	return e, ok
}

// TODO: Dedup.

type contentCounter struct {
	Content htmlg.Component
	Count   int
}

func (cc contentCounter) Render() []*html.Node {
	var ns []*html.Node
	ns = append(ns, cc.Content.Render()...)
	ns = append(ns, htmlg.SpanClass("counter", htmlg.Text(fmt.Sprint(cc.Count))))
	return ns
}

// iconText is an icon with text on the right.
// Icon must be not nil.
type iconText struct {
	Icon func() *html.Node // Must be not nil.
	Text string
}

func (it iconText) Render() []*html.Node {
	icon := htmlg.Span(it.Icon())
	icon.Attr = append(icon.Attr, html.Attribute{
		Key: atom.Style.String(), Val: "margin-right: 4px;",
	})
	text := htmlg.Text(it.Text)
	return []*html.Node{icon, text}
}
