package issuesapp

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
	"strconv"
	"strings"
	"time"

	statepkg "dmitri.shuralyov.com/state"
	"github.com/dustin/go-humanize"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/home/internal/exp/app/issuesapp/assets"
	"github.com/shurcooL/home/internal/exp/app/issuesapp/common"
	"github.com/shurcooL/home/internal/exp/app/issuesapp/component"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octicon"
	"github.com/shurcooL/reactions"
	reactionscomponent "github.com/shurcooL/reactions/component"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
)

// TODO: Find a better way for issuesapp to be able to ensure registration of a top-level route:
//
// 	emojisHandler := httpgzip.FileServer(emojis.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
// 	http.Handle("/emojis/", http.StripPrefix("/emojis", emojisHandler))
//
// So that it can depend on it.

// New returns an issues app http.Handler using given services and options.
// If usersService is nil, then there is no way to have an authenticated user.
// Emojis image data is expected to be available at /emojis/emojis.png, unless
// opt.DisableReactions is true.
//
// In order to serve HTTP requests, the returned http.Handler expects each incoming
// request to have 2 parameters provided to it via RepoSpecContextKey and BaseURIContextKey
// context keys. For example:
//
// 	issuesApp := issuesapp.New(...)
//
// 	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
// 		req = req.WithContext(context.WithValue(req.Context(), issuesapp.RepoSpecContextKey, issues.RepoSpec{...}))
// 		req = req.WithContext(context.WithValue(req.Context(), issuesapp.BaseURIContextKey, string(...)))
// 		err := issuesApp.ServeHTTP(w, req)
// 		// Handle error, if any.
// 	})
//
// An HTTP API must be available (currently, only Create and EditComment endpoints are used):
//
// 	// Register HTTP API endpoints.
// 	apiHandler := httphandler.Issues{Issues: service}
// 	http.Handle(path.Join("/api/issue", httproute.List), errorHandler(apiHandler.List))
// 	http.Handle(path.Join("/api/issue", httproute.Count), errorHandler(apiHandler.Count))
// 	http.Handle(path.Join("/api/issue", httproute.ListTimeline), errorHandler(apiHandler.ListTimeline))
// 	http.Handle(path.Join("/api/issue", httproute.Create), errorHandler(apiHandler.Create))
// 	http.Handle(path.Join("/api/issue", httproute.EditComment), errorHandler(apiHandler.EditComment))
func New(service issues.Service, users users.Service, opt Options) interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request) error
} {
	static, err := loadTemplates(common.State{}, opt.BodyPre)
	if err != nil {
		log.Fatalln("loadTemplates failed:", err)
	}
	return &handler{
		is:               service,
		us:               users,
		static:           static,
		assetsFileServer: httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed}),
		gfmFileServer:    httpgzip.FileServer(assets.GFMStyle, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed}),
		Options:          opt,
	}
}

// RepoSpecContextKey is a context key for the request's repository specification.
// That value specifies which repo the issues are to be displayed for.
// The associated value will be of type issues.RepoSpec.
var RepoSpecContextKey = &contextKey{"RepoSpec"}

// BaseURIContextKey is a context key for the request's base URI.
// That value specifies the base URI prefix to use for all absolute URLs.
// The associated value will be of type string.
var BaseURIContextKey = &contextKey{"BaseURI"}

// StateContextKey is a context key for the request's common state.
// That value specifies the common state of the page being rendered.
// The associated value will be of type common.State.
var StateContextKey = &contextKey{"State"}

// Options for configuring issues app.
type Options struct {
	Notifications    notifications.Service // If not nil, issues containing unread notifications are highlighted.
	DisableReactions bool                  // Disable all support for displaying and toggling reactions.

	HeadPre, HeadPost template.HTML
	BodyPre           string // An html/template definition of "body-pre" template.

	// BodyTop provides components to include on top of <body> of page rendered for req. It can be nil.
	// StateContextKey can be used to get the common state value.
	BodyTop func(req *http.Request) ([]htmlg.Component, error)

	// SignIn returns HTML with a link or button to sign in. It can be nil.
	SignIn func(returnURL string) template.HTML
}

// handler handles all requests to issuesapp. It acts like a request multiplexer,
// choosing from various endpoints and parsing the repository ID from URL.
type handler struct {
	is issues.Service
	us users.Service // May be nil if there's no users service.

	assetsFileServer http.Handler
	gfmFileServer    http.Handler

	// static is loaded once in New, and is only for rendering templates that don't use state.
	static *template.Template

	Options
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	if _, ok := req.Context().Value(RepoSpecContextKey).(issues.RepoSpec); !ok {
		return fmt.Errorf("request to %v doesn't have issuesapp.RepoSpecContextKey context key set", req.URL.Path)
	}
	if _, ok := req.Context().Value(BaseURIContextKey).(string); !ok {
		return fmt.Errorf("request to %v doesn't have issuesapp.BaseURIContextKey context key set", req.URL.Path)
	}

	// Handle "/assets/gfm/...".
	if strings.HasPrefix(req.URL.Path, "/assets/gfm/") {
		req = stripPrefix(req, len("/assets/gfm"))
		h.gfmFileServer.ServeHTTP(w, req)
		return nil
	}

	// Handle "/assets/script.js".
	if req.URL.Path == "/assets/script.js" {
		req = stripPrefix(req, len("/assets"))
		h.assetsFileServer.ServeHTTP(w, req)
		return nil
	}

	// Handle (the rest of) "/assets/...".
	if strings.HasPrefix(req.URL.Path, "/assets/") {
		h.assetsFileServer.ServeHTTP(w, req)
		return nil
	}

	// Handle "/".
	if req.URL.Path == "/" {
		return h.IssuesHandler(w, req)
	}

	// Handle "/new".
	if req.URL.Path == "/new" {
		return h.NewIssueHandler(w, req)
	}

	// Handle "/{issueID}" and "/{issueID}/...".
	elems := strings.SplitN(req.URL.Path[1:], "/", 3)
	issueID, err := strconv.ParseUint(elems[0], 10, 64)
	if err != nil {
		return httperror.HTTP{Code: http.StatusNotFound, Err: fmt.Errorf("invalid issue ID %q: %v", elems[0], err)}
	}
	switch {
	// "/{issueID}".
	case len(elems) == 1:
		return h.IssueHandler(w, req, issueID)

	// "/{issueID}/edit".
	case len(elems) == 2 && elems[1] == "edit":
		return h.PostEditIssueHandler(w, req, issueID)

	// "/{issueID}/comment".
	case len(elems) == 2 && elems[1] == "comment":
		return h.PostCommentHandler(w, req, issueID)

	default:
		return httperror.HTTP{Code: http.StatusNotFound, Err: errors.New("no route")}
	}
}

func (h *handler) IssuesHandler(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return httperror.Method{Allowed: []string{http.MethodGet}}
	}
	state, err := h.state(req, 0)
	if err != nil {
		return err
	}
	filter, err := stateFilter(req.URL.Query())
	if err != nil {
		return httperror.BadRequest{Err: err}
	}
	is, err := h.is.List(req.Context(), state.RepoSpec, issues.IssueListOptions{State: filter})
	if err != nil {
		return err
	}
	openCount, err := h.is.Count(req.Context(), state.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueOpen)})
	if err != nil {
		return fmt.Errorf("issues.Count(open): %v", err)
	}
	closedCount, err := h.is.Count(req.Context(), state.RepoSpec, issues.IssueListOptions{State: issues.StateFilter(statepkg.IssueClosed)})
	if err != nil {
		return fmt.Errorf("issues.Count(closed): %v", err)
	}
	var es []component.IssueEntry
	for _, i := range is {
		es = append(es, component.IssueEntry{Issue: i, BaseURI: state.BaseURI})
	}
	es = state.augmentUnread(req.Context(), es, h.is, h.Notifications)
	state.Issues = component.Issues{
		IssuesNav: component.IssuesNav{
			OpenCount:     openCount,
			ClosedCount:   closedCount,
			Path:          state.BaseURI + state.ReqPath,
			Query:         req.URL.Query(),
			StateQueryKey: stateQueryKey,
		},
		Filter:  filter,
		Entries: es,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = h.static.ExecuteTemplate(w, "issues.html.tmpl", &state)
	if err != nil {
		return fmt.Errorf("h.static.ExecuteTemplate: %v", err)
	}
	return nil
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

// TODO: Switch to notification v2 service.
func (s state) augmentUnread(ctx context.Context, es []component.IssueEntry, is issues.Service, notificationsService notifications.Service) []component.IssueEntry {
	if notificationsService == nil {
		return es
	}

	tt, ok := is.(interface {
		ThreadType(issues.RepoSpec) string
	})
	if !ok {
		log.Println("augmentUnread: issues service doesn't implement ThreadType")
		return es
	}

	if s.CurrentUser.ID == 0 {
		// Unauthenticated user cannot have any unread issues.
		return es
	}

	// TODO: Consider starting to do this in background in parallel with is.List.
	ns, err := notificationsService.List(ctx, notifications.ListOptions{
		Repo: &notifications.RepoSpec{URI: s.RepoSpec.URI},
	})
	if err != nil {
		log.Println("augmentUnread: failed to notifications.List:", err)
		return es
	}

	unreadThreads := make(map[uint64]struct{}) // Set of unread thread IDs.
	for _, n := range ns {
		// n.RepoSpec == s.RepoSpec is guaranteed because we filtered in notifications.ListOptions,
		// so we only need to check that n.ThreadType matches.
		if n.ThreadType != tt.ThreadType(s.RepoSpec) {
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

func (h *handler) IssueHandler(w http.ResponseWriter, req *http.Request, issueID uint64) error {
	if req.Method != http.MethodGet {
		return httperror.Method{Allowed: []string{http.MethodGet}}
	}
	state, err := h.state(req, issueID)
	if err != nil {
		return err
	}
	state.Issue, err = h.is.Get(req.Context(), state.RepoSpec, state.IssueID)
	if err != nil {
		return err
	}
	tis, err := h.is.ListTimeline(req.Context(), state.RepoSpec, state.IssueID, nil)
	if err != nil {
		return fmt.Errorf("issues.ListTimeline: %v", err)
	}
	for _, ti := range tis {
		state.Items = append(state.Items, issueItem{ti})
	}
	// Call loadTemplates to set updated reactionsBar, reactableID, etc., template functions.
	t, err := loadTemplates(state.State, h.Options.BodyPre)
	if err != nil {
		return fmt.Errorf("loadTemplates: %v", err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = t.ExecuteTemplate(w, "issue.html.tmpl", &state)
	if err != nil {
		return fmt.Errorf("t.ExecuteTemplate: %v", err)
	}
	return nil
}

func (h *handler) NewIssueHandler(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return httperror.Method{Allowed: []string{http.MethodGet}}
	}
	state, err := h.state(req, 0)
	if err != nil {
		return err
	}
	if state.CurrentUser.ID == 0 {
		return os.ErrPermission
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = h.static.ExecuteTemplate(w, "new-issue.html.tmpl", &state)
	if err != nil {
		return fmt.Errorf("h.static.ExecuteTemplate: %v", err)
	}
	return nil
}

func (h *handler) PostEditIssueHandler(w http.ResponseWriter, req *http.Request, issueID uint64) error {
	if req.Method != http.MethodPost {
		return httperror.Method{Allowed: []string{http.MethodPost}}
	}
	if err := req.ParseForm(); err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("req.ParseForm: %v", err)}
	}

	repoSpec := req.Context().Value(RepoSpecContextKey).(issues.RepoSpec)

	var ir issues.IssueRequest
	err := json.Unmarshal([]byte(req.PostForm.Get("value")), &ir)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("json.Unmarshal 'value': %v", err)}
	}

	issue, events, err := h.is.Edit(req.Context(), repoSpec, issueID, ir)
	if err != nil {
		return err
	}

	resp := make(url.Values)

	// State badge.
	var buf bytes.Buffer
	err = htmlg.RenderComponents(&buf, component.IssueStateBadge{Issue: issue})
	if err != nil {
		return err
	}
	resp.Set("issue-state-badge", buf.String())

	// Toggle button.
	buf.Reset()
	err = h.static.ExecuteTemplate(&buf, "toggle-button", issue.State)
	if err != nil {
		return err
	}
	resp.Set("issue-toggle-button", buf.String())

	// Events.
	for _, event := range events {
		buf.Reset()
		err = htmlg.RenderComponents(&buf, component.Event{Event: event})
		if err != nil {
			return err
		}
		resp.Add("new-event", buf.String())
	}

	_, err = io.WriteString(w, resp.Encode())
	return err
}

func (h *handler) PostCommentHandler(w http.ResponseWriter, req *http.Request, issueID uint64) error {
	if req.Method != http.MethodPost {
		return httperror.Method{Allowed: []string{http.MethodPost}}
	}
	if err := req.ParseForm(); err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("req.ParseForm: %v", err)}
	}
	state, err := h.state(req, issueID)
	if err != nil {
		return err
	}

	comment := issues.Comment{
		Body: req.PostForm.Get("value"),
	}
	comment, err = h.is.CreateComment(req.Context(), state.RepoSpec, issueID, comment)
	if err != nil {
		return err
	}

	// Call loadTemplates to set updated reactionsBar, reactableID, etc., template functions.
	t, err := loadTemplates(state.State, h.Options.BodyPre)
	if err != nil {
		return fmt.Errorf("loadTemplates: %v", err)
	}
	err = t.ExecuteTemplate(w, "comment", comment)
	if err != nil {
		return fmt.Errorf("t.ExecuteTemplate: %v", err)
	}
	return nil
}

func (h *handler) state(req *http.Request, issueID uint64) (state, error) {
	// TODO: Caller still does a lot of work outside to calculate req.URL.Path by
	//       subtracting BaseURI from full original req.URL.Path. We should be able
	//       to compute it here internally by using req.RequestURI and BaseURI.
	reqPath := req.URL.Path
	if reqPath == "/" {
		reqPath = "" // This is needed so that absolute URL for root view, i.e., /issues, is "/issues" and not "/issues/" because of "/issues" + "/".
	}
	b := state{
		State: common.State{
			BaseURI:  req.Context().Value(BaseURIContextKey).(string),
			ReqPath:  reqPath,
			RepoSpec: req.Context().Value(RepoSpecContextKey).(issues.RepoSpec),
			IssueID:  issueID,
		},
	}
	b.HeadPre = h.HeadPre
	b.HeadPost = h.HeadPost
	if h.BodyTop != nil {
		c, err := h.BodyTop(req.WithContext(context.WithValue(req.Context(), StateContextKey, b.State)))
		if err != nil {
			return state{}, err
		}
		var buf bytes.Buffer
		err = htmlg.RenderComponents(&buf, c...)
		if err != nil {
			return state{}, fmt.Errorf("htmlg.RenderComponents: %v", err)
		}
		b.BodyTop = template.HTML(buf.String())
	}

	b.DisableReactions = h.Options.DisableReactions
	b.DisableUsers = h.us == nil
	if h.Options.SignIn != nil {
		returnURL := b.BaseURI + b.ReqPath
		b.SignIn = h.Options.SignIn(returnURL)
	}

	if h.us == nil {
		// No user service provided, so there can never be an authenticated user.
		b.CurrentUser = users.User{}
	} else if user, err := h.us.GetAuthenticated(req.Context()); err == nil {
		b.CurrentUser = user
	} else {
		return state{}, fmt.Errorf("h.us.GetAuthenticated: %v", err)
	}

	b.ForceIssuesApp, _ = strconv.ParseBool(req.URL.Query().Get("issuesapp"))

	return b, nil
}

type state struct {
	HeadPre, HeadPost template.HTML
	BodyTop           template.HTML
	SignIn            template.HTML

	common.State

	Issues component.Issues
	Issue  issues.Issue
	Items  []issueItem

	// ForceIssuesApp reports whether "issuesapp" query is true.
	// This is a temporary solution for external users to use when overriding templates.
	// It's going to go away eventually, so its use is discouraged.
	ForceIssuesApp bool
}

func loadTemplates(state common.State, bodyPre string) (*template.Template, error) {
	t := template.New("").Funcs(template.FuncMap{
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
		"reactableID": func(commentID uint64) string {
			return fmt.Sprintf("%d/%d", state.IssueID, commentID)
		},
		"reactionsBar": func(reactions []reactions.Reaction, reactableID string) htmlg.Component {
			return reactionscomponent.ReactionsBar{
				Reactions:   reactions,
				CurrentUser: state.CurrentUser,
				ID:          reactableID,
			}
		},
		"newReaction": func(reactableID string) htmlg.Component {
			return reactionscomponent.NewReaction{
				ReactableID: reactableID,
			}
		},
		"state": func() common.State { return state },

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
	})
	t, err := vfstemplate.ParseGlob(assets.Assets, t, "/assets/*.tmpl")
	if err != nil {
		return nil, err
	}
	return t.New("body-pre").Parse(bodyPre)
}

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "issuesapp context value " + k.name }

// stripPrefix returns request r with prefix of length prefixLen stripped from r.URL.Path.
// prefixLen must not be longer than len(r.URL.Path), otherwise stripPrefix panics.
// If r.URL.Path is empty after the prefix is stripped, the path is changed to "/".
func stripPrefix(r *http.Request, prefixLen int) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = r.URL.Path[prefixLen:]
	if r2.URL.Path == "" {
		r2.URL.Path = "/"
	}
	return r2
}
