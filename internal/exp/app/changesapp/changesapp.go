// +build go1.14

// Package changesapp is a change tracking web app.
package changesapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/shurcooL/github_flavored_markdown"
	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/exp/app/changesapp/component"
	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/reactions"
	reactionscomponent "github.com/shurcooL/reactions/component"
	"github.com/shurcooL/users"
	"github.com/sourcegraph/go-diff/diff"
	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
)

func New(cs change.Service, us users.Service, redirect func(*url.URL), opt Options) *app {
	return &app{
		cs:       cs,
		us:       us,
		redirect: redirect,
		opt:      opt,
	}
}

// Options for configuring changes app.
type Options struct {
	// Notification, if not nil, is used to highlight changes containing
	// unread notifications, and to mark changes that are viewed as read.
	Notification notification.Service

	// BodyTop provides components to include at the top of the <body> element. It can be nil.
	BodyTop func(context.Context, State) ([]htmlg.Component, error)
}

type State struct {
	ReqURL      *url.URL
	CurrentUser users.User
	RepoSpec    string
	BaseURL     string // Must have no trailing slash. Can be empty string.

	ChangeID uint64 // ChangeID is the current change ID, or 0 if not applicable (e.g., current page is '/').

	PrevSHA string // PrevSHA is the previous commit SHA, or empty if not applicable (e.g., current page is not /{changeID}/files/{commitID}).
	NextSHA string // NextSHA is the next commit SHA, or empty if not applicable (e.g., current page is not /{changeID}/files/{commitID}).
}

func (s State) RequestURL() *url.URL { return s.ReqURL }

type app struct {
	cs change.Service
	us users.Service

	redirect func(*url.URL) // TODO: Rename to "open" or "navigate" or so?
	opt      Options
}

func (a *app) ServePage(ctx context.Context, w io.Writer, reqURL *url.URL) (interface{}, error) {
	// TODO: Think about best place for this.
	//       On backend, it needs to be done for each request (could be serving different users).
	//       On frontend, it needs to be done only once (given a user can't sign in or out completely on frontend).
	//       Can optimize the frontend query by embedding information in the HTML (like RedLogo).
	authenticatedUser, err := a.us.GetAuthenticated(ctx)
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}

	repoSpec, baseURL, route, err := parseRequest(reqURL, authenticatedUser.UserSpec)
	if err != nil {
		return nil, err
	}

	st := State{
		ReqURL:      reqURL,
		CurrentUser: authenticatedUser,
		RepoSpec:    repoSpec,
		BaseURL:     baseURL,
	}

	// TODO: Don't hardcode the "/assets" or "/assets/changes" prefix here.
	_, err = io.WriteString(w, `<link href="/assets/changes/style.css" rel="stylesheet">`)
	if err != nil {
		return nil, err
	}

	// Handle "/".
	if route == "/" {
		return st, a.serveChanges(ctx, w, st)
	}

	// Handle "/{changeID}" and "/{changeID}/...".
	elems := strings.SplitN(route[1:], "/", 3)
	st.ChangeID, err = strconv.ParseUint(elems[0], 10, 64)
	if err != nil {
		return nil, httperror.HTTP{Code: http.StatusNotFound, Err: fmt.Errorf("invalid change ID %q: %v", elems[0], err)}
	}
	switch {
	// "/{changeID}".
	case len(elems) == 1:
		return st, a.serveChange(ctx, w, st)

	// "/{changeID}/commits".
	case len(elems) == 2 && elems[1] == "commits":
		return st, a.serveChangeCommits(ctx, w, st)

	// "/{changeID}/files".
	case len(elems) == 2 && elems[1] == "files":
		_, _, err := a.serveChangeFiles(ctx, w, st, "")
		return st, err

	// "/{changeID}/files/{commitID}".
	case len(elems) == 3 && elems[1] == "files":
		commitID := elems[2]
		var err error
		st.PrevSHA, st.NextSHA, err = a.serveChangeFiles(ctx, w, st, commitID)
		return st, err

	default:
		return nil, httperror.HTTP{Code: http.StatusNotFound, Err: errors.New("no route")}
	}
}

func parseRequest(reqURL *url.URL, currentUser users.UserSpec) (repoSpec, baseURL, route string, err error) {
	switch i := strings.Index(reqURL.Path, "/...$changes"); {
	case i >= 0:
		repoSpec = "dmitri.shuralyov.com" + reqURL.Path[:i]
		baseURL = reqURL.Path[:i+len("/...$changes")]
		route = reqURL.Path[i+len("/...$changes"):]

	// Parse "/changes/github.com/..." request.
	case strings.HasPrefix(reqURL.Path, "/changes/github.com/"):
		elems := strings.SplitN(reqURL.Path[len("/changes/github.com/"):], "/", 3)
		if len(elems) < 2 || elems[0] == "" || elems[1] == "" {
			return "", "", "", os.ErrNotExist
		}
		if currentUser != dmitshur {
			// Redirect to GitHub.
			switch len(elems) {
			case 2:
				return "", "", "", httperror.Redirect{URL: "https://github.com/" + elems[0] + "/" + elems[1] + "/pulls"}
			default: // 3 or more.
				return "", "", "", httperror.Redirect{URL: "https://github.com/" + elems[0] + "/" + elems[1] + "/pull/" + elems[2]}
			}
		}
		repoSpec = "github.com/" + elems[0] + "/" + elems[1]
		baseURL = "/changes/" + repoSpec
		route = reqURL.Path[len(baseURL):]

	// Parse "/changes/go.googlesource.com/..." request.
	case strings.HasPrefix(reqURL.Path, "/changes/go.googlesource.com/"):
		elems := strings.SplitN(reqURL.Path[len("/changes/go.googlesource.com/"):], "/", 2)
		if len(elems) < 1 || elems[0] == "" {
			return "", "", "", os.ErrNotExist
		}
		if currentUser != dmitshur {
			// Redirect to Gerrit.
			switch len(elems) {
			case 1:
				return "", "", "", httperror.Redirect{URL: fmt.Sprintf("https://go-review.googlesource.com/q/project:%s+status:open", elems[0])}
			default: // 2 or more.
				return "", "", "", httperror.Redirect{URL: fmt.Sprintf("https://go-review.googlesource.com/c/%s/+/%s", elems[0], elems[1])}
			}
		}
		repoSpec = "go.googlesource.com/" + elems[0]
		baseURL = "/changes/" + repoSpec
		route = reqURL.Path[len(baseURL):]

	default:
		return "", "", "", os.ErrNotExist
	}

	switch route {
	case "/":
		// Redirect to base URL without trailing slash.
		if q := reqURL.RawQuery; q != "" {
			baseURL += "?" + q
		}
		return "", "", "", httperror.Redirect{URL: baseURL}
	case "":
		route = "/"
	}
	return repoSpec, baseURL, route, nil
}

var dmitshur = users.UserSpec{ID: 1924134, Domain: "github.com"}

func (a *app) serveChanges(ctx context.Context, w io.Writer, st State) error {
	filter, err := stateFilter(st.ReqURL.Query())
	if err != nil {
		return httperror.BadRequest{Err: err}
	}
	var (
		bodyTop                template.HTML
		cs                     []change.Change
		openCount, closedCount uint64
	)
	g, groupContext := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		bodyTop, err = a.bodyTop(groupContext, st)
		if err != nil {
			return fmt.Errorf("bodyTop: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		cs, err = a.cs.List(groupContext, st.RepoSpec, change.ListOptions{Filter: filter})
		if err != nil {
			return fmt.Errorf("change.List: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		openCount, err = a.cs.Count(groupContext, st.RepoSpec, change.ListOptions{Filter: change.FilterOpen})
		if err != nil {
			return fmt.Errorf("change.Count(open): %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		closedCount, err = a.cs.Count(groupContext, st.RepoSpec, change.ListOptions{Filter: change.FilterClosedMerged})
		if err != nil {
			return fmt.Errorf("change.Count(closed): %w", err)
		}
		return nil
	})
	err = g.Wait()
	if err != nil {
		return err
	}
	var es []component.ChangeEntry
	for _, c := range cs {
		es = append(es, component.ChangeEntry{Change: c, BaseURL: st.BaseURL})
	}
	es = a.augmentUnread(ctx, st, es)
	changes := component.Changes{
		ChangesNav: component.ChangesNav{
			OpenCount:     openCount,
			ClosedCount:   closedCount,
			Path:          st.ReqURL.Path,
			Query:         st.ReqURL.Query(),
			StateQueryKey: stateQueryKey,
		},
		Filter:  filter,
		Entries: es,
	}
	tt, err := t.Clone()
	if err != nil {
		return fmt.Errorf("t.Clone: %v", err)
	}
	err = tt.ExecuteTemplate(w, "changes.html.tmpl", renderState{
		BodyPre: bodyPre, BodyPost: bodyPost,
		BodyTop:     bodyTop,
		BaseURL:     st.BaseURL,
		CurrentUser: st.CurrentUser,
		Changes:     changes,
	})
	return err
}

func (a *app) augmentUnread(ctx context.Context, st State, es []component.ChangeEntry) []component.ChangeEntry {
	if a.opt.Notification == nil {
		return es
	}

	if st.CurrentUser.ID == 0 {
		// Unauthenticated user cannot have any unread issues.
		return es
	}

	threadType, err := a.cs.ThreadType(ctx, st.RepoSpec)
	if err != nil {
		log.Println("augmentUnread: failed to notifications.ThreadType:", err)
		return es
	}

	// TODO: Consider starting to do this in background in parallel with is.List.
	ns, err := a.opt.Notification.ListNotifications(ctx, notification.ListOptions{
		Namespace: st.RepoSpec,
	})
	if err != nil {
		log.Println("augmentUnread: failed to notifications.List:", err)
		return es
	}

	unreadThreads := make(map[uint64]struct{}) // Set of unread thread IDs.
	for _, n := range ns {
		// n.RepoSpec == st.RepoSpec is guaranteed because we filtered in notifications.ListOptions,
		// so we only need to check that n.ThreadType matches.
		if n.ThreadType != threadType {
			continue
		}
		unreadThreads[n.ThreadID] = struct{}{}
	}

	for i, e := range es {
		_, unread := unreadThreads[e.Change.ID]
		es[i].Unread = unread
	}
	return es
}

func (a *app) bodyTop(ctx context.Context, st State) (template.HTML, error) {
	if a.opt.BodyTop == nil {
		return "", nil
	}
	c, err := a.opt.BodyTop(ctx, st)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = htmlg.RenderComponents(&buf, c...)
	if err != nil {
		return "", fmt.Errorf("htmlg.RenderComponents: %v", err)
	}
	return template.HTML(buf.String()), nil
}

func (a *app) serveChange(ctx context.Context, w io.Writer, st State) error {
	bodyTop, err := a.bodyTop(ctx, st)
	if err != nil {
		return fmt.Errorf("bodyTop: %w", err)
	}
	c, err := a.cs.Get(ctx, st.RepoSpec, st.ChangeID)
	if err != nil {
		return err
	}
	ts, err := a.cs.ListTimeline(ctx, st.RepoSpec, st.ChangeID, nil)
	if err != nil {
		return fmt.Errorf("change.ListTimeline: %w", err)
	}
	if a.opt.Notification != nil {
		err := a.markRead(ctx, st)
		if err != nil {
			log.Println("serveChange: failed to markRead:", err)
		}
	}
	var timeline []timelineItem
	for _, item := range ts {
		timeline = append(timeline, timelineItem{item})
	}
	sort.Sort(byCreatedAtID(timeline))
	// Re-parse "comment" template with updated reactionsBar and reactableID template functions.
	tt, err := t.Clone()
	if err != nil {
		return fmt.Errorf("t.Clone: %v", err)
	}
	tt, err = tt.Funcs(template.FuncMap{
		"reactableID": func(commentID string) string {
			return fmt.Sprintf("%d/%s", st.ChangeID, commentID)
		},
		"reactionsBar": func(reactions []reactions.Reaction, reactableID string) htmlg.Component {
			return reactionscomponent.ReactionsBar{
				Reactions:   reactions,
				CurrentUser: st.CurrentUser,
				ID:          reactableID,
			}
		},
		"event": func(e change.TimelineItem) htmlg.Component {
			return component.Event{Event: e, BaseURL: st.BaseURL, ChangeID: st.ChangeID}
		},
	}).Parse(`
{{/* Dot is a change.Comment. */}}
{{define "comment"}}
<div class="list-entry" style="display: flex;">
	<div style="margin-right: 10px;">{{render (avatar .User)}}</div>
	<div style="flex-grow: 1; display: flex; flex-direction: column;">
		<div id="comment-{{.ID}}">
			<div class="list-entry-container list-entry-border">
				<header class="list-entry-header" style="display: flex;">
					<span style="flex-grow: 1;">{{render (user .User)}} commented <a class="black" href="#comment-{{.ID}}" onclick="AnchorScroll(this, event);">{{render (time .CreatedAt)}}</a>
						{{with .Edited}} · <span style="cursor: default;" title="{{.By.Login}} edited this comment {{reltime .At}}.">edited{{if not (equalUsers $.User .By)}} by {{.By.Login}}{{end}}</span>{{end}}
					</span>
					<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
					{{if .Editable}}<span class="right-icon"><a href="javascript:" title="Edit" onclick="EditComment({{` + "`edit`" + ` | json}}, this);">{{octicon "pencil"}}</a></span>{{end}}
				</header>
				<div class="list-entry-body">
					<div class="markdown-body">
						{{with .Body}}
							{{. | gfm}}
						{{else}}
							<i class="gray">No description.</i>
						{{end}}
					</div>
				</div>
			</div>
		</div>
		{{render (reactionsBar .Reactions (reactableID .ID))}}
	</div>
</div>
{{end}}

{{/* Dot is a change.Review. */}}
{{define "review"}}
<div class="list-entry">
	<div style="display: flex;">
		<div style="margin-right: 10px;">{{render (avatar .User)}}</div>
		<div style="flex-grow: 1; display: flex; flex-direction: column;">
			<div id="comment-{{.ID}}">
				<div class="list-entry-container list-entry-border">
					<header class="list-entry-header" style="display: flex;{{if ne .State 0}} padding: 4px;{{end}}{{if not .Body}} border: none;{{end}}">
						{{template "review-icon" .State}}
						<span style="flex-grow: 1;{{if .State}} line-height: 28px;{{end}}">{{render (user .User)}} {{template "review-action" .State}} <a class="black" href="#comment-{{.ID}}" onclick="AnchorScroll(this, event);">{{render (time .CreatedAt)}}</a>
							{{with .Edited}} · <span style="cursor: default;" title="{{.By.Login}} edited this comment {{reltime .At}}.">edited{{if not (equalUsers $.User .By)}} by {{.By.Login}}{{end}}</span>{{end}}
						</span>
						<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
						{{if .Editable}}<span class="right-icon"><a href="javascript:" title="Edit" onclick="EditComment({{` + "`edit`" + ` | json}}, this);">{{octicon "pencil"}}</a></span>{{end}}
					</header>
					{{with .Body}}
					<div class="list-entry-body">
						<div class="markdown-body">
							{{. | gfm}}
						</div>
					</div>
					{{end}}
				</div>
			</div>
			{{render (reactionsBar .Reactions (reactableID .ID))}}
		</div>
	</div>
	{{with .Comments}}
		<div style="margin-left: 80px;">
		{{range .}}
			<div class="list-entry list-entry-container list-entry-border">
				<header style="display: flex;" class="list-entry-header">
					<span style="flex-grow: 1;">{{.File}}:{{.Line}}</span>
					<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
				</header>
				<div class="list-entry-body">
					<div class="markdown-body">{{.Body | gfm}}</div>
				</div>
			</div>
			{{render (reactionsBar .Reactions (reactableID .ID))}}
		{{end}}
		</div>
	{{end}}
</div>
{{end}}
`)
	if err != nil {
		return err
	}
	err = tt.ExecuteTemplate(w, "change.html.tmpl", renderState{
		BodyPre: bodyPre, BodyPost: bodyPost,
		BodyTop:     bodyTop,
		BaseURL:     st.BaseURL,
		CurrentUser: st.CurrentUser,
		ChangeID:    st.ChangeID,
		Change:      c,
		Timeline:    timeline,
	})
	return err
}

func (a *app) markRead(ctx context.Context, st State) error {
	threadType, err := a.cs.ThreadType(ctx, st.RepoSpec)
	if err != nil {
		return err
	}

	if st.CurrentUser.UserSpec == (users.UserSpec{}) {
		// Unauthenticated user cannot mark anything as read.
		return nil
	}

	err = a.opt.Notification.MarkThreadRead(ctx, st.RepoSpec, threadType, st.ChangeID)
	return err
}

func (a *app) serveChangeCommits(ctx context.Context, w io.Writer, st State) error {
	bodyTop, err := a.bodyTop(ctx, st)
	if err != nil {
		return fmt.Errorf("bodyTop: %w", err)
	}
	c, err := a.cs.Get(ctx, st.RepoSpec, st.ChangeID)
	if err != nil {
		return err
	}
	list, err := a.cs.ListCommits(ctx, st.RepoSpec, st.ChangeID)
	if err != nil {
		return err
	}
	tt, err := t.Clone()
	if err != nil {
		return fmt.Errorf("t.Clone: %v", err)
	}
	err = tt.ExecuteTemplate(w, "change-commits.html.tmpl", renderState{
		BodyPre: bodyPre, BodyPost: bodyPost,
		BodyTop:     bodyTop,
		BaseURL:     st.BaseURL,
		CurrentUser: st.CurrentUser,
		ChangeID:    st.ChangeID,
		Change:      c,
	})
	if err != nil {
		return err
	}
	var cs []commit
	for _, c := range list {
		cs = append(cs, commit{Commit: c})
	}
	err = htmlg.RenderComponents(w, commits{Commits: cs})
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, bodyPost)
	return err
}

// serveChangeFiles serves the "/{changeID}/files" and "/{changeID}/files/{commitID}" pages.
// commitID is empty string for all files, or the SHA of a single commit for single-commit view.
func (a *app) serveChangeFiles(ctx context.Context, w io.Writer, st State, commitID string) (prevSHA, nextSHA string, _ error) {
	bodyTop, err := a.bodyTop(ctx, st)
	if err != nil {
		return "", "", fmt.Errorf("bodyTop: %w", err)
	}
	c, err := a.cs.Get(ctx, st.RepoSpec, st.ChangeID)
	if err != nil {
		return "", "", err
	}
	var commit commitMessage
	if commitID != "" {
		// TODO: Avoid calling ListCommits repeatedly when switching between commits via 'p'/'n' shortcuts.
		cs, err := a.cs.ListCommits(ctx, st.RepoSpec, st.ChangeID)
		if err != nil {
			return "", "", err
		}
		i := commitIndex(cs, commitID)
		if i == -1 {
			return "", "", os.ErrNotExist
		}
		subject, body := splitCommitMessage(cs[i].Message)
		commit = commitMessage{
			CommitHash: cs[i].SHA,
			Subject:    subject,
			Body:       body,
			Author:     cs[i].Author,
			AuthorTime: cs[i].AuthorTime,
		}
		if prev := i - 1; prev >= 0 {
			commit.PrevSHA = cs[prev].SHA
		}
		if next := i + 1; next < len(cs) {
			commit.NextSHA = cs[next].SHA
		}
	}
	var opt *change.GetDiffOptions
	if commitID != "" {
		opt = &change.GetDiffOptions{Commit: commitID}
	}
	rawDiff, err := a.cs.GetDiff(ctx, st.RepoSpec, st.ChangeID, opt)
	if err != nil {
		return "", "", err
	}
	fileDiffs, err := diff.ParseMultiFileDiff(rawDiff)
	if err != nil {
		return "", "", err
	}
	tt, err := t.Clone()
	if err != nil {
		return "", "", fmt.Errorf("t.Clone: %v", err)
	}
	err = tt.ExecuteTemplate(w, "change-files.html.tmpl", renderState{
		BodyPre: bodyPre, BodyPost: bodyPost,
		BodyTop:     bodyTop,
		BaseURL:     st.BaseURL,
		CurrentUser: st.CurrentUser,
		ChangeID:    st.ChangeID,
		Change:      c,
	})
	if err != nil {
		return "", "", err
	}
	if commitID != "" {
		err := tt.ExecuteTemplate(w, "CommitMessage", commit)
		if err != nil {
			return "", "", err
		}
	}
	for _, f := range fileDiffs {
		err := tt.ExecuteTemplate(w, "FileDiff", fileDiff{FileDiff: f})
		if err != nil {
			return "", "", err
		}
	}
	_, err = io.WriteString(w, bodyPost)
	if err != nil {
		return "", "", err
	}
	return commit.PrevSHA, commit.NextSHA, nil
}

// commitIndex returns the index of commit with SHA equal to commitID,
// or -1 if not found.
func commitIndex(cs []change.Commit, commitID string) int {
	for i := range cs {
		if cs[i].SHA == commitID {
			return i
		}
	}
	return -1
}

const (
	// stateQueryKey is name of query key for controlling change state filter.
	stateQueryKey = "state"
)

// stateFilter parses the change state filter from query,
// returning an error if the value is unsupported.
func stateFilter(query url.Values) (change.StateFilter, error) {
	selectedTabName := query.Get(stateQueryKey)
	switch selectedTabName {
	case "":
		return change.FilterOpen, nil
	case "closed":
		return change.FilterClosedMerged, nil
	case "all":
		return change.FilterAll, nil
	default:
		return "", fmt.Errorf("unsupported state filter value: %q", selectedTabName)
	}
}

type renderState struct {
	BodyPre, BodyPost template.HTML
	BodyTop           template.HTML
	SignIn            template.HTML

	// TODO: BaseURL is for BodyTop template in "{issues,issue,new-issue}.html.tmpl", maybe can remove?
	BaseURL     string // Must have no trailing slash. Can be empty string.
	CurrentUser users.User

	// TODO: ChangeID is for Tabnav, maybe can remove?
	ChangeID uint64 // ChangeID is the current change ID, or 0 if not applicable (e.g., current page is /changes).

	Changes  component.Changes
	Change   change.Change
	Timeline []timelineItem
}

// TODO: Is there a better place for Tabnav?

// Tabnav renders the tabnav.
func (s renderState) Tabnav(selected string) template.HTML {
	var files htmlg.Component = iconText{Icon: octicon.Diff, Text: "Files"}
	if s.Change.ChangedFiles != 0 {
		files = contentCounter{Content: files, Count: s.Change.ChangedFiles}
	}
	return template.HTML(htmlg.RenderComponentsString(homecomponent.TabNav{
		Tabs: []homecomponent.Tab{
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.CommentDiscussion, Text: "Discussion"},
					Count:   s.Change.Replies,
				},
				URL: fmt.Sprintf("%s/%d", s.BaseURL, s.ChangeID), OnClick: "Open(event, this)",
				Selected: selected == "Discussion",
			},
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.GitCommit, Text: "Commits"},
					Count:   s.Change.Commits,
				},
				URL: fmt.Sprintf("%s/%d/commits", s.BaseURL, s.ChangeID), OnClick: "Open(event, this)",
				Selected: selected == "Commits",
			},
			{
				Content: files,
				URL:     fmt.Sprintf("%s/%d/files", s.BaseURL, s.ChangeID), OnClick: "Open(event, this)",
				Selected: selected == "Files",
			},
		},
	}))
}

var t = template.Must(template.New("").Funcs(template.FuncMap{
	"json": func(v interface{}) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
	"jsonfmt": func(v interface{}) (string, error) {
		b, err := json.MarshalIndent(v, "", "\t")
		return string(b), err
	},
	"reltime":          humanize.Time,
	"gfm":              func(s string) template.HTML { return template.HTML(github_flavored_markdown.Markdown([]byte(s))) },
	"reactionPosition": func(emojiID reactions.EmojiID) string { return reactions.Position(":" + string(emojiID) + ":") },
	"equalUsers": func(a, b users.User) bool {
		return a.UserSpec == b.UserSpec
	},
	"reactableID": func(commentID string) string { panic("reactableID: not implemented") },
	"reactionsBar": func(reactions []reactions.Reaction, reactableID string) htmlg.Component {
		panic("reactionsBar: not implemented")
	},
	"newReaction": func(reactableID string) htmlg.Component {
		return reactionscomponent.NewReaction{
			ReactableID: reactableID,
		}
	},

	"octicon": func(name string) (template.HTML, error) {
		icon := octicon.Icon(name)
		if icon == nil {
			return "", fmt.Errorf("%q is not a valid Octicon symbol name", name)
		}
		var buf bytes.Buffer
		err := html.Render(&buf, icon)
		if err != nil {
			return "", err
		}
		return template.HTML(buf.String()), nil
	},

	"render": func(c htmlg.Component) template.HTML {
		return template.HTML(htmlg.Render(c.Render()...))
	},
	"event":            func(e change.TimelineItem) htmlg.Component { panic("event: not implemented") },
	"changeStateBadge": func(c change.Change) htmlg.Component { return component.ChangeStateBadge{Change: c} },
	"time":             func(t time.Time) htmlg.Component { return component.Time{Time: t} },
	"user":             func(u users.User) htmlg.Component { return component.User{User: u} },
	"avatar":           func(u users.User) htmlg.Component { return component.Avatar{User: u, Size: 48} },
}).Parse(`
{{define "changes.html.tmpl"}}
	{{.BodyPre}}
	{{.BodyTop}}
	{{render .Changes}}
	{{.BodyPost}}
{{end}}

{{define "change.html.tmpl"}}
	{{.BodyPre}}
	{{.BodyTop}}
	{{template "change" .}}
	{{.BodyPost}}
{{end}}

{{define "change-commits.html.tmpl"}}
	{{.BodyPre}}
	{{.BodyTop}}
	<h1>{{.Change.Title}} <span class="gray">#{{.Change.ID}}</span></h1>
	<div id="change-state-badge" style="margin-bottom: 20px;">{{render (changeStateBadge .Change)}}</div>
	{{.Tabnav "Commits"}}
{{end}}

{{define "change-files.html.tmpl"}}
	{{.BodyPre}}
	{{.BodyTop}}
	<h1>{{.Change.Title}} <span class="gray">#{{.Change.ID}}</span></h1>
	<div id="change-state-badge" style="margin-bottom: 20px;">{{render (changeStateBadge .Change)}}</div>
	{{.Tabnav "Files"}}
{{end}}

{{define "change"}}
	<h1>{{.Change.Title}} <span class="gray">#{{.Change.ID}}</span></h1>
	<div id="change-state-badge" style="margin-bottom: 20px;">{{render (changeStateBadge .Change)}}</div>
	{{.Tabnav "Discussion"}}
	{{range .Timeline}}
		{{template "timeline-item" .}}
	{{end}}
{{end}}

{{define "timeline-item"}}
	{{if eq .TemplateName "comment"}}
		{{template "comment" .TimelineItem}}
	{{else if eq .TemplateName "review"}}
		{{template "review" .TimelineItem}}
	{{else if eq .TemplateName "event"}}
		{{render (event .TimelineItem)}}
	{{end}}
{{end}}

{{/* Dot is a change.Comment. */}}
{{define "comment"}}
<div class="list-entry" style="display: flex;">
	<div style="margin-right: 10px;">{{render (avatar .User)}}</div>
	<div style="flex-grow: 1; display: flex; flex-direction: column;">
		<div id="comment-{{.ID}}">
			<div class="list-entry-container list-entry-border">
				<header class="list-entry-header" style="display: flex;">
					<span style="flex-grow: 1;">{{render (user .User)}} commented <a class="black" href="#comment-{{.ID}}" onclick="AnchorScroll(this, event);">{{render (time .CreatedAt)}}</a>
						{{with .Edited}} · <span style="cursor: default;" title="{{.By.Login}} edited this comment {{reltime .At}}.">edited{{if not (equalUsers $.User .By)}} by {{.By.Login}}{{end}}</span>{{end}}
					</span>
					<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
					{{if .Editable}}<span class="right-icon"><a href="javascript:" title="Edit" onclick="EditComment({{` + "`edit`" + ` | json}}, this);">{{octicon "pencil"}}</a></span>{{end}}
				</header>
				<div class="list-entry-body">
					<div class="markdown-body">
						{{with .Body}}
							{{. | gfm}}
						{{else}}
							<i class="gray">No description.</i>
						{{end}}
					</div>
				</div>
			</div>
		</div>
		{{render (reactionsBar .Reactions (reactableID .ID))}}
	</div>
</div>
{{end}}

{{/* Dot is a change.Review. */}}
{{define "review"}}
<div class="list-entry">
	<div style="display: flex;">
		<div style="margin-right: 10px;">{{render (avatar .User)}}</div>
		<div style="flex-grow: 1; display: flex; flex-direction: column;">
			<div id="comment-{{.ID}}">
				<div class="list-entry-container list-entry-border">
					<header class="list-entry-header" style="display: flex;{{if ne .State 0}} padding: 4px;{{end}}{{if not .Body}} border: none;{{end}}">
						{{template "review-icon" .State}}
						<span style="flex-grow: 1;{{if .State}} line-height: 28px;{{end}}">{{render (user .User)}} {{template "review-action" .State}} <a class="black" href="#comment-{{.ID}}" onclick="AnchorScroll(this, event);">{{render (time .CreatedAt)}}</a>
							{{with .Edited}} · <span style="cursor: default;" title="{{.By.Login}} edited this comment {{reltime .At}}.">edited{{if not (equalUsers $.User .By)}} by {{.By.Login}}{{end}}</span>{{end}}
						</span>
						<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
						{{if .Editable}}<span class="right-icon"><a href="javascript:" title="Edit" onclick="EditComment({{` + "`edit`" + ` | json}}, this);">{{octicon "pencil"}}</a></span>{{end}}
					</header>
					{{with .Body}}
					<div class="list-entry-body">
						<div class="markdown-body">
							{{. | gfm}}
						</div>
					</div>
					{{end}}
				</div>
			</div>
			{{render (reactionsBar .Reactions (reactableID .ID))}}
		</div>
	</div>
	{{with .Comments}}
		<div style="margin-left: 80px;">
		{{range .}}
			<div class="list-entry list-entry-container list-entry-border">
				<header style="display: flex;" class="list-entry-header">
					<span style="flex-grow: 1;">{{.File}}:{{.Line}}</span>
					<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
				</header>
				<div class="list-entry-body">
					<div class="markdown-body">{{.Body | gfm}}</div>
				</div>
			</div>
			{{render (reactionsBar .Reactions (reactableID .ID))}}
		{{end}}
		</div>
	{{end}}
</div>
{{end}}

{{/* Dot is state.Review. */}}
{{define "review-icon" }}
	{{if gt . 0}}
		<span class="event-icon" style="color: #fff; background-color: #6cc644;">{{octicon "check"}}</span>
	{{else if lt . 0}}
		<span class="event-icon" style="color: #fff; background-color: #bd2c00;">{{octicon "x"}}</span>
	{{end}}
{{end}}

{{/* Dot is state.Review. */}}
{{define "review-action" }}
	{{if eq . 0}}
		commented
	{{else}}
		reviewed {{printf "%+d" .}}
	{{end}}
{{end}}

{{define "CommitMessage"}}
<div class="list-entry list-entry-border commit-message">
	<header class="list-entry-header">
		<div style="display: flex;">
			<pre style="flex-grow: 1;"><strong>{{.Subject}}</strong>{{with .Body}}

{{.}}{{end}}</pre>
			{{with .PrevSHA}}
				<a href="{{.}}" onclick="Open(event, this)">{{octicon "arrow-left"}}</a>
			{{else}}
				<span style="color: gray;">{{octicon "arrow-left"}}</span>
			{{end}}
			{{with .NextSHA}}
				<a href="{{.}}" onclick="Open(event, this)">{{octicon "arrow-right"}}</a>
			{{else}}
				<span style="color: gray;">{{octicon "arrow-right"}}</span>
			{{end}}
		</div>
	</header>
	<div class="list-entry-body" style="display: flex;">
		<span style="display: inline-block; vertical-align: bottom; margin-right: 5px;">{{.Avatar}}</span>{{/*
		*/}}<span style="flex-grow: 1; display: inline-block;">{{.User}} committed {{.Time}}</span>
		<span>commit <code>{{.CommitHash}}</code></span>
	</div>
</div>
{{end}}

{{define "FileDiff"}}
<div class="list-entry list-entry-border">
	<header class="list-entry-header">{{.Title}}</header>
	<div class="list-entry-body">
		<pre class="highlight-diff">{{.Diff}}</pre>
	</div>
</div>
{{end}}
`))

const (
	bodyPre  = `<div style="max-width: 800px; margin: 0 auto 100px auto;">`
	bodyPost = `</div>`
)
