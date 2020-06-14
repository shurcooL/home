// +build go1.14

// Package issuesapp is an issue tracking web app.
package issuesapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	statepkg "dmitri.shuralyov.com/state"
	"github.com/dustin/go-humanize"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/home/internal/exp/app/issuesapp/component"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/reactions"
	reactionscomponent "github.com/shurcooL/reactions/component"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
)

func New(is issues.Service, us users.Service, redirect func(*url.URL), opt Options) *app {
	return &app{
		is:       is,
		us:       us,
		redirect: redirect,
		opt:      opt,
	}
}

// Options for configuring issues app.
type Options struct {
	// Notification, if not nil, is used to highlight issues containing
	// unread notifications, and to mark issues that are viewed as read.
	Notification notification.Service

	// BodyTop provides components to include at the top of the <body> element. It can be nil.
	BodyTop func(context.Context, State) ([]htmlg.Component, error)
}

type State struct {
	ReqURL      *url.URL
	CurrentUser users.User
	RepoSpec    issues.RepoSpec
	BaseURL     string // Must have no trailing slash. Can be empty string.

	IssueID uint64 // IssueID is the current issue ID, or 0 if not applicable (e.g., current page is /new).
}

func (s State) RequestURL() *url.URL { return s.ReqURL }

type app struct {
	is issues.Service
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
		RepoSpec:    issues.RepoSpec{URI: repoSpec},
		BaseURL:     baseURL,
	}

	// TODO: Don't hardcode the "/assets" or "/assets/issues" prefix here.
	_, err = io.WriteString(w, `<link href="/assets/issues/style.css" rel="stylesheet">`)
	if err != nil {
		return nil, err
	}

	// Handle "/".
	if route == "/" {
		return st, a.serveIssues(ctx, w, st)
	}

	// Handle "/new".
	if route == "/new" {
		return st, a.serveNewIssue(ctx, w, st)
	}

	// Handle "/{issueID}".
	st.IssueID, err = strconv.ParseUint(route[1:], 10, 64)
	if err != nil {
		return nil, os.ErrNotExist
	}
	return st, a.serveIssue(ctx, w, st)
}

func parseRequest(reqURL *url.URL, currentUser users.UserSpec) (repoSpec, baseURL, route string, err error) {
	switch i := strings.Index(reqURL.Path, "/...$issues"); {
	case i >= 0:
		repoSpec = "dmitri.shuralyov.com" + reqURL.Path[:i]
		baseURL = reqURL.Path[:i+len("/...$issues")]
		route = reqURL.Path[i+len("/...$issues"):]

	// Parse "/issues/github.com/..." request.
	case strings.HasPrefix(reqURL.Path, "/issues/github.com/"):
		elems := strings.SplitN(reqURL.Path[len("/issues/github.com/"):], "/", 3)
		if len(elems) < 2 || elems[0] == "" || elems[1] == "" {
			return "", "", "", os.ErrNotExist
		}
		repoSpec = "github.com/" + elems[0] + "/" + elems[1]
		onGitHub := repoSpec != "github.com/shurcooL/issuesapp" && repoSpec != "github.com/shurcooL/notificationsapp"
		if onGitHub && currentUser != dmitshur {
			// Redirect to GitHub.
			switch len(elems) {
			case 2:
				return "", "", "", httperror.Redirect{URL: "https://github.com/" + elems[0] + "/" + elems[1] + "/issues"}
			default: // 3 or more.
				return "", "", "", httperror.Redirect{URL: "https://github.com/" + elems[0] + "/" + elems[1] + "/issues/" + elems[2]}
			}
		}
		baseURL = "/issues/" + repoSpec
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

func (a *app) serveIssues(ctx context.Context, w io.Writer, st State) error {
	filter, err := stateFilter(st.ReqURL.Query())
	if err != nil {
		return httperror.BadRequest{Err: err}
	}
	var (
		bodyTop                template.HTML
		is                     []issues.Issue
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
		is, err = a.is.List(groupContext, st.RepoSpec, issues.IssueListOptions{State: filter})
		if err != nil {
			return fmt.Errorf("issues.List: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		openCount, err = a.is.Count(groupContext, st.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueOpen)})
		if err != nil {
			return fmt.Errorf("issues.Count(open): %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		closedCount, err = a.is.Count(groupContext, st.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueClosed)})
		if err != nil {
			return fmt.Errorf("issues.Count(closed): %w", err)
		}
		return nil
	})
	err = g.Wait()
	if err != nil {
		return err
	}
	var es []component.IssueEntry
	for _, i := range is {
		es = append(es, component.IssueEntry{Issue: i, BaseURL: st.BaseURL})
	}
	es = a.augmentUnread(ctx, st, es)
	issues := component.Issues{
		IssuesNav: component.IssuesNav{
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
	err = tt.ExecuteTemplate(w, "issues.html.tmpl", renderState{
		BodyPre: bodyPre, BodyPost: bodyPost,
		BodyTop:     bodyTop,
		BaseURL:     st.BaseURL,
		CurrentUser: st.CurrentUser,
		Issues:      issues,
	})
	return err
}

func (a *app) augmentUnread(ctx context.Context, st State, es []component.IssueEntry) []component.IssueEntry {
	if a.opt.Notification == nil {
		return es
	}

	if st.CurrentUser.ID == 0 {
		// Unauthenticated user cannot have any unread issues.
		return es
	}

	threadType, err := a.is.ThreadType(ctx, st.RepoSpec)
	if err != nil {
		log.Println("augmentUnread: failed to notifications.ThreadType:", err)
		return es
	}

	// TODO: Consider starting to do this in background in parallel with is.List.
	ns, err := a.opt.Notification.ListNotifications(ctx, notification.ListOptions{
		Namespace: st.RepoSpec.URI,
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
		_, unread := unreadThreads[e.Issue.ID]
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

func (a *app) serveNewIssue(ctx context.Context, w io.Writer, st State) error {
	// Check that user is authenticated.
	if st.CurrentUser.UserSpec == (users.UserSpec{}) {
		return os.ErrPermission
	}

	bodyTop, err := a.bodyTop(ctx, st)
	if err != nil {
		return fmt.Errorf("bodyTop: %w", err)
	}
	tt, err := t.Clone()
	if err != nil {
		return fmt.Errorf("t.Clone: %v", err)
	}
	err = tt.ExecuteTemplate(w, "new-issue.html.tmpl", renderState{
		BodyPre: bodyPre, BodyPost: bodyPost,
		BodyTop:     bodyTop,
		BaseURL:     st.BaseURL,
		CurrentUser: st.CurrentUser,
	})
	return err
}

func (a *app) serveIssue(ctx context.Context, w io.Writer, st State) error {
	bodyTop, err := a.bodyTop(ctx, st)
	if err != nil {
		return fmt.Errorf("bodyTop: %w", err)
	}
	issue, err := a.is.Get(ctx, st.RepoSpec, st.IssueID)
	if err != nil {
		return err
	}
	tis, err := a.is.ListTimeline(ctx, st.RepoSpec, st.IssueID, nil)
	if err != nil {
		return fmt.Errorf("issues.ListTimeline: %v", err)
	}
	// TODO: Marking read is currently done in the issue service. Should it be removed there and factored in here?
	/*if a.opt.Notification != nil {
		err := a.markRead(ctx)
		if err != nil {
			log.Println("serveIssue: failed to markRead:", err)
		}
	}*/
	var items []issueItem
	for _, ti := range tis {
		items = append(items, issueItem{ti})
	}
	// Re-parse "comment" template with updated reactionsBar and reactableID template functions.
	tt, err := t.Clone()
	if err != nil {
		return fmt.Errorf("t.Clone: %v", err)
	}
	tt, err = tt.Funcs(template.FuncMap{
		"reactableID": func(commentID uint64) string {
			return fmt.Sprintf("%d/%d", st.IssueID, commentID)
		},
		"reactionsBar": func(reactions []reactions.Reaction, reactableID string) htmlg.Component {
			return reactionscomponent.ReactionsBar{
				Reactions:   reactions,
				CurrentUser: st.CurrentUser,
				ID:          reactableID,
			}
		},
	}).Parse(`
{{/* Dot is an issues.Comment. */}}
{{define "comment"}}
<div class="comment-edit-container">
	<div>
		<div style="float: left; margin-right: 10px;">{{render (avatar .User)}}</div>
		<div id="comment-{{.ID}}" style="display: flex;" class="list-entry">
			<div class="list-entry-container list-entry-border">
				<div class="list-entry-header" style="display: flex;">
					<span class="content">{{render (user .User)}} commented <a class="black" href="#comment-{{.ID}}" onclick="AnchorScroll(this, event);">{{render (time .CreatedAt)}}</a>
						{{with .Edited}} · <span style="cursor: default;" title="{{.By.Login}} edited this comment {{reltime .At}}.">edited{{if not (equalUsers $.User .By)}} by {{.By.Login}}{{end}}</span>{{end}}
					</span>
					<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
					{{if .Editable}}<span class="right-icon"><a href="javascript:" title="Edit" onclick="EditComment({{` + "`edit`" + ` | json}}, this, event);">{{octicon "pencil"}}</a></span>{{end}}
				</div>
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
	</div>
	<div style="display: none;">
		{{template "edit-comment" .}}
	</div>
	{{render (reactionsBar .Reactions (reactableID .ID))}}
</div>
{{end}}
`)
	if err != nil {
		return err
	}
	err = tt.ExecuteTemplate(w, "issue.html.tmpl", renderState{
		BodyPre: bodyPre, BodyPost: bodyPost,
		BodyTop:     bodyTop,
		BaseURL:     st.BaseURL,
		CurrentUser: st.CurrentUser,
		Issue:       issue,
		Items:       items,
	})
	return err
}

const (
	// stateQueryKey is name of query key for controlling issue state filter.
	stateQueryKey = "state"
)

// stateFilter parses the issue state filter from query,
// returning an error if the value is unsupported.
func stateFilter(query url.Values) (issues.StateFilter, error) {
	selectedTabName := query.Get(stateQueryKey)
	switch selectedTabName {
	case "":
		return issues.StateFilter(statepkg.IssueOpen), nil
	case "closed":
		return issues.StateFilter(statepkg.IssueClosed), nil
	case "all":
		return issues.AllStates, nil
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

	Issues component.Issues
	Issue  issues.Issue
	Items  []issueItem
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
	"reactableID": func(commentID uint64) string { panic("reactableID: not implemented") },
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
	"event":           func(e issues.Event) htmlg.Component { return component.Event{Event: e} },
	"issueStateBadge": func(i issues.Issue) htmlg.Component { return component.IssueStateBadge{Issue: i} },
	"time":            func(t time.Time) htmlg.Component { return component.Time{Time: t} },
	"user":            func(u users.User) htmlg.Component { return component.User{User: u} },
	"avatar":          func(u users.User) htmlg.Component { return component.Avatar{User: u, Size: 48} },
}).Parse(`
{{define "issues.html.tmpl"}}
	{{.BodyPre}}
	{{.BodyTop}}
	<div style="text-align: right;"><a href="{{.BaseURL}}/new" onclick="Open(event, this)">New Issue</a></div>
	{{render .Issues}}
	{{.BodyPost}}
{{end}}

{{define "issue.html.tmpl"}}
	{{.BodyPre}}
	{{.BodyTop}}
	{{template "issue" .}}
	{{.BodyPost}}
{{end}}

{{define "new-issue.html.tmpl"}}
	{{.BodyPre}}
	{{.BodyTop}}
	{{template "new-issue" .}}
	{{.BodyPost}}
{{end}}

{{define "issue"}}
	<h1>{{.Issue.Title}} <span class="gray">#{{.Issue.ID}}</span></h1>
	<div id="issue-state-badge" style="margin-bottom: 20px;">{{render (issueStateBadge .Issue)}}</div>
	{{range .Items}}
		{{template "issue-item" .}}
	{{end}}
	<div id="new-item-marker"></div>
	{{template "new-comment" .}}
{{end}}

{{define "issue-item"}}
	{{if eq .TemplateName "comment"}}
		{{template "comment" .IssueItem}}
	{{else if eq .TemplateName "event"}}
		{{render (event .IssueItem)}}
	{{end}}
{{end}}

{{define "new-comment"}}
{{if .CurrentUser.ID}}
	<div id="new-comment-container" class="edit-container list-entry" style="display: flex;">
		<div style="margin-right: 10px;">{{render (avatar .CurrentUser)}}</div>
		<div class="list-entry-border" style="flex-grow: 1;">
			<div class="list-entry-header tabs" style="display: flex;">
				<span style="flex-grow: 1; font-size: 14px;">
					<a class="write-tab-link black tab-link active" tabindex=-1 href="javascript:" onclick="SwitchWriteTab(this);">Write</a>
					<a class="preview-tab-link black tab-link" tabindex=-1 href="javascript:" onclick="MarkdownPreview(this);">Preview</a>
				</span>
				<span class="gray"><span style="margin-right: 6px;">{{octicon "markdown"}}</span>Markdown</span>
			</div>
			<div class="list-entry-body">
				<textarea class="comment-editor" placeholder="Leave a comment." onpaste="PasteHandler(event);" onkeydown="TabSupportKeyDownHandler(this, event);" tabindex=1></textarea>
				<div class="comment-preview markdown-body" style="padding: 11px 11px 10px 11px; min-height: 120px; box-sizing: border-box; border-bottom: 1px solid #eee; display: none;"></div>
				<div style="text-align: right; margin-top: 10px;">
					<button class="btn btn-success btn-small" onclick="PostComment();" tabindex=1>Comment</button>
					{{if .Issue.Editable}}{{template "toggle-button" (print .Issue.State)}}{{end}}
				</div>
			</div>
		</div>
	</div>
{{else if .SignIn}}
	<div class="event" style="margin-top: 20px; margin-bottom: 20px;">
		{{.SignIn}} to comment.
	</div>
{{end}}
{{end}}

{{define "new-issue"}}
<div style="display: flex; margin-top: 20px;" class="edit-container list-entry">
	<div style="margin-right: 10px;">{{render (avatar .CurrentUser)}}</div>
	<div class="list-entry-border" style="flex-grow: 1;">
		<div class="list-entry-header tabs-title">
			<div><input id="title-editor" type="text" placeholder="Title" autofocus></div>
			<div style="display: flex;">
				<span style="flex-grow: 1; font-size: 14px;">
					<a class="write-tab-link black tab-link active" tabindex=-1 href="javascript:" onclick="SwitchWriteTab(this);">Write</a>
					<a class="preview-tab-link black tab-link" tabindex=-1 href="javascript:" onclick="MarkdownPreview(this);">Preview</a>
				</span>
				<span class="gray"><span style="margin-right: 6px;">{{octicon "markdown"}}</span>Markdown</span>
			</div>
		</div>
		<div class="list-entry-body">
			<textarea class="comment-editor" style="min-height: 200px;" placeholder="Leave a comment." onpaste="PasteHandler(event);" onkeydown="TabSupportKeyDownHandler(this, event);"></textarea>
			<div class="comment-preview markdown-body" style="padding: 10px; min-height: 200px; display: none;"></div>
			<div style="text-align: right; margin-top: 10px;">
				<button id="create-issue-button" class="btn btn-success btn-small" disabled="disabled" onclick="CreateNewIssue();">Create Issue</button>
			</div>
		</div>
	</div>
</div>
{{end}}

{{/* Dot is an issues.Comment. */}}
{{define "comment"}}
<div class="comment-edit-container">
	<div>
		<div style="float: left; margin-right: 10px;">{{render (avatar .User)}}</div>
		<div id="comment-{{.ID}}" style="display: flex;" class="list-entry">
			<div class="list-entry-container list-entry-border">
				<div class="list-entry-header" style="display: flex;">
					<span class="content">{{render (user .User)}} commented <a class="black" href="#comment-{{.ID}}" onclick="AnchorScroll(this, event);">{{render (time .CreatedAt)}}</a>
						{{with .Edited}} · <span style="cursor: default;" title="{{.By.Login}} edited this comment {{reltime .At}}.">edited{{if not (equalUsers $.User .By)}} by {{.By.Login}}{{end}}</span>{{end}}
					</span>
					<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
					{{if .Editable}}<span class="right-icon"><a href="javascript:" title="Edit" onclick="EditComment({{` + "`edit`" + ` | json}}, this, event);">{{octicon "pencil"}}</a></span>{{end}}
				</div>
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
	</div>
	<div style="display: none;">
		{{template "edit-comment" .}}
	</div>
	{{render (reactionsBar .Reactions (reactableID .ID))}}
</div>
{{end}}

{{/* TODO: Dedup with new-comment, only buttons differ, so factor them out. */}}
{{define "edit-comment"}}
<div style="display: flex;" class="edit-container list-entry">
	<div style="margin-right: 10px;">{{render (avatar .User)}}</div>
	<div class="list-entry-border" style="flex-grow: 1;">
		<div class="list-entry-header tabs" style="display: flex;">
			<span style="flex-grow: 1; font-size: 14px;">
				<a class="write-tab-link black tab-link active" tabindex=-1 href="javascript:" onclick="SwitchWriteTab(this);">Write</a>
				<a class="preview-tab-link black tab-link" tabindex=-1 href="javascript:" onclick="MarkdownPreview(this);">Preview</a>
			</span>
			<span class="gray"><span style="margin-right: 6px;">{{octicon "markdown"}}</span>Markdown</span>
		</div>
		<div class="list-entry-body">
			<textarea class="comment-editor" placeholder="Leave a comment." onpaste="PasteHandler(event);" onkeydown="TabSupportKeyDownHandler(this, event);" data-id="{{.ID}}" data-raw="{{.Body}}" tabindex=1></textarea>
			<div class="comment-preview markdown-body" style="padding: 11px 11px 10px 11px; min-height: 120px; box-sizing: border-box; border-bottom: 1px solid #eee; display: none;"></div>
			<div style="text-align: right; margin-top: 10px;">
				<button class="btn btn-success btn-small" onclick="EditComment({{` + "`update`" + ` | json}}, this, event);" tabindex=1>Update comment</button>
				<button class="btn btn-danger btn-small" onclick="EditComment({{` + "`cancel`" + ` | json}}, this, event);" tabindex=1>Cancel</button>
			</div>
		</div>
	</div>
</div>
{{end}}

{{/* TODO: Try to use issues.OpenState and issues.ClosedState constants. */}}
{{define "toggle-button"}}
	{{if eq . "open"}}
		{{template "close-button"}}
	{{else if eq . "closed"}}
		{{template "reopen-button"}}
	{{else}}
		{{.}}
	{{end}}
{{end}}

{{define "close-button"}}
<button id="issue-toggle-button" class="btn btn-neutral btn-small" data-1-action="Close Issue" data-2-actions="Comment and close" onclick="ToggleIssueState('closed');" tabindex=1>Close Issue</button>
{{end}}

{{define "reopen-button"}}
<button id="issue-toggle-button" class="btn btn-neutral btn-small" data-1-action="Reopen Issue" data-2-actions="Reopen and comment" onclick="ToggleIssueState('open');" tabindex=1>Reopen Issue</button>
{{end}}
`))

const (
	bodyPre  = `<div style="max-width: 800px; margin: 0 auto 100px auto;">`
	bodyPost = `</div>`
)
