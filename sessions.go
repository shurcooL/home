package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/go-github/github"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/oauth2"
	githuboauth2 "golang.org/x/oauth2/github"
)

var githubConfig = oauth2.Config{
	ClientID:     os.Getenv("HOME_GH_CLIENT_ID"),
	ClientSecret: os.Getenv("HOME_GH_CLIENT_SECRET"),
	Scopes:       nil,
	Endpoint:     githuboauth2.Endpoint,
}

// TODO: Persist? In a secure way?
var sessions = struct {
	mu       sync.Mutex
	sessions map[string]user // Access Token -> User.
}{sessions: make(map[string]user)}

func cryptoRandBytes() []byte {
	b := make([]byte, 256)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

const (
	accessTokenCookieName = "accessToken"
	stateCookieName       = "state"
	returnCookieName      = "return" // TODO, THINK.

	returnQueryName = "return"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "github.com/shurcooL/home context value " + k.name }

// userContextKey is a context key. It can be used to access the user
// that the context is tied to. The associated value will be of type *user.
var userContextKey = &contextKey{"user"}

// user is a GitHub user (i.e., domain is "github.com").
type user struct {
	ID uint64

	expiry      time.Time
	accessToken string // Internal access token. Needed to be able to clear session when this user signs out.
}

var errBadAccessToken = errors.New("bad access token")

// getUser either returns a valid user (possibly nil) and nil error,
// or nil user and errBadAccessToken.
func getUser(req *http.Request) (*user, error) {
	cookie, err := req.Cookie(accessTokenCookieName)
	if err == http.ErrNoCookie {
		return nil, nil // No user.
	} else if err != nil {
		return nil, errBadAccessToken
	}
	accessTokenBytes, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, errBadAccessToken
	}
	accessToken := string(accessTokenBytes)
	var u *user
	sessions.mu.Lock()
	if user, ok := sessions.sessions[accessToken]; ok {
		if time.Now().Before(user.expiry) {
			u = &user
		} else {
			delete(sessions.sessions, accessToken) // This is unlikely to happen because cookie expires by then.
		}
	}
	sessions.mu.Unlock()
	if u == nil {
		return nil, errBadAccessToken
	}
	return u, nil // Existing user.
}

// Redirect is an error type used for representing a simple HTTP redirection.
type Redirect struct {
	URL string
}

func (r Redirect) Error() string { return fmt.Sprintf("redirecting to %s", r.URL) }

func IsRedirect(err error) bool {
	_, ok := err.(Redirect)
	return ok
}

// HTTPError is an error type used for representing a non-nil error with a status code.
type HTTPError struct {
	Code int
	err  error // Not nil.
}

// Error returns HTTPError.err.Error().
func (h HTTPError) Error() string { return h.err.Error() }

func IsHTTPError(err error) bool {
	_, ok := err.(HTTPError)
	return ok
}

// JSONResponse is an error type used for representing a JSON response.
type JSONResponse struct {
	V interface{}
}

func (JSONResponse) Error() string { return "JSONResponse" }

func IsJSONResponse(err error) bool {
	_, ok := err.(JSONResponse)
	return ok
}

// HeaderWriter interface is used to construct an HTTP response header and trailer.
type HeaderWriter interface {
	// Header returns the header map that will be sent by
	// WriteHeader. Changing the header after a call to
	// WriteHeader (or Write) has no effect unless the modified
	// headers were declared as trailers by setting the
	// "Trailer" header before the call to WriteHeader (see example).
	// To suppress implicit response headers, set their value to nil.
	Header() http.Header
}

// SetCookie adds a Set-Cookie header to the provided HeaderWriter's headers.
// The provided cookie must have a valid Name. Invalid cookies may be
// silently dropped.
func SetCookie(w HeaderWriter, cookie *http.Cookie) {
	if v := cookie.String(); v != "" {
		w.Header().Add("Set-Cookie", v)
	}
}

type handler struct {
	handler func(w HeaderWriter, req *http.Request, user *user) ([]*html.Node, error)
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// HACK: Manually check that method is allowed for the given path.
	switch req.URL.Path {
	default:
		if req.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method should be GET", http.StatusMethodNotAllowed)
			return
		}
	case "/login/github":
		if req.Method != "POST" {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method should be POST", http.StatusMethodNotAllowed)
			return
		}
	case "/logout":
		if req.Method != "POST" {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method should be POST", http.StatusMethodNotAllowed)
			return
		}
	}

	// TODO: Factor this out into user middleware?
	u, err := getUser(req)
	if err == errBadAccessToken {
		// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
		//       E.g., that will happen when you're logging in. First, errBadAccessToken happens, then a successful login results in setting accessTokenCookieName to a new value.
		http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
	}
	req = req.WithContext(context.WithValue(req.Context(), userContextKey, u))

	nodes, err := h.handler(w, req, u)
	switch {
	case IsRedirect(err):
		http.Redirect(w, req, string(err.(Redirect).URL), http.StatusSeeOther)
	case IsHTTPError(err):
		http.Error(w, err.Error(), err.(HTTPError).Code)
	case os.IsNotExist(err):
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
	case os.IsPermission(err):
		if u == nil {
			loginURL := (&url.URL{
				Path:     "/login",
				RawQuery: url.Values{returnQueryName: {req.URL.String()}}.Encode(),
			}).String()
			http.Redirect(w, req, loginURL, http.StatusSeeOther)
			return
		}
		log.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
	default:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, string(htmlg.Render(nodes...)))
	case IsJSONResponse(err):
		w.Header().Set("Content-Type", "application/json")
		jw := json.NewEncoder(w)
		jw.SetIndent("", "\t")
		err := jw.Encode(err.(JSONResponse).V)
		if err != nil {
			log.Println("error encoding JSONResponse:", err)
		}
	case err != nil:
		log.Println(err)
		// TODO: Only display error details to SiteAdmin users?
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// TODO, THINK: Clean this up.
func sanitizeReturn(returnURL string) string {
	u, err := url.Parse(returnURL)
	if err != nil {
		return "/"
	}
	if u.Scheme != "" || u.Opaque != "" || u.User != nil || u.Host != "" {
		return "/"
	}
	if u.Path == "" {
		return "/"
	}
	return (&url.URL{Path: u.Path, RawQuery: u.RawQuery}).String()
}

type SessionsHandler struct {
	users users.Service
}

func (h SessionsHandler) Serve(w HeaderWriter, req *http.Request, u *user) ([]*html.Node, error) {
	// Simple switch-based router for now. For a larger project, a more sophisticated router should be used.
	switch {
	case req.Method == "POST" && req.URL.Path == "/login/github":
		returnURL := sanitizeReturn(req.PostFormValue("return"))

		if u != nil {
			return nil, Redirect{URL: returnURL}
		}

		state := base64.RawURLEncoding.EncodeToString(cryptoRandBytes()) // GitHub doesn't handle all non-ascii bytes in state, so use base64.
		SetCookie(w, &http.Cookie{Path: "/callback/github", Name: stateCookieName, Value: state, HttpOnly: true, Secure: *productionFlag})

		// TODO, THINK.
		SetCookie(w, &http.Cookie{Path: "/callback/github", Name: returnCookieName, Value: returnURL, HttpOnly: true, Secure: *productionFlag})

		url := githubConfig.AuthCodeURL(state)
		return nil, Redirect{URL: url}

	case req.Method == "GET" && req.URL.Path == "/callback/github":
		if u != nil {
			return nil, Redirect{URL: "/"}
		}

		ghUser, err := func() (*github.User, error) {
			// Validate state (to prevent CSRF).
			cookie, err := req.Cookie(stateCookieName)
			if err != nil {
				return nil, err
			}
			SetCookie(w, &http.Cookie{Path: "/callback/github", Name: stateCookieName, MaxAge: -1})
			state := req.FormValue("state")
			if cookie.Value != state {
				return nil, errors.New("state doesn't match")
			}

			token, err := githubConfig.Exchange(oauth2.NoContext, req.FormValue("code"))
			if err != nil {
				return nil, err
			}
			tc := githubConfig.Client(oauth2.NoContext, token)
			gh := github.NewClient(tc)

			user, _, err := gh.Users.Get("")
			if err != nil {
				return nil, err
			}
			if user.ID == nil || *user.ID == 0 {
				return nil, errors.New("user id is nil/0")
			}
			if user.Login == nil || *user.Login == "" {
				return nil, errors.New("user login is unset/empty")
			}
			return user, nil
		}()
		if err != nil {
			log.Println(err)
			return nil, HTTPError{Code: http.StatusUnauthorized, err: err}
		}

		accessToken := string(cryptoRandBytes())
		expiry := time.Now().Add(7 * 24 * time.Hour)
		sessions.mu.Lock()
		// Clean up expired sesions.
		for token, user := range sessions.sessions {
			if time.Now().Before(user.expiry) {
				continue
			}
			delete(sessions.sessions, token)
		}
		// Add new session.
		sessions.sessions[accessToken] = user{
			ID:          uint64(*ghUser.ID),
			expiry:      expiry,
			accessToken: accessToken,
		}
		sessions.mu.Unlock()

		// TODO, THINK.
		returnURL, err := func() (string, error) {
			cookie, err := req.Cookie(returnCookieName)
			if err != nil {
				return "", err
			}
			SetCookie(w, &http.Cookie{Path: "/callback/github", Name: returnCookieName, MaxAge: -1})
			return sanitizeReturn(cookie.Value), nil
		}()
		if err != nil {
			log.Println("/callback/github: problem with returnCookieName:", err)
			returnURL = "/"
		}

		// TODO: Is base64 the best encoding for cookie values? Factor it out maybe?
		encodedAccessToken := base64.RawURLEncoding.EncodeToString([]byte(accessToken))
		SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, Value: encodedAccessToken, Expires: expiry, HttpOnly: true, Secure: *productionFlag})
		return nil, Redirect{URL: returnURL}

	case req.Method == "POST" && req.URL.Path == "/logout":
		if u != nil {
			sessions.mu.Lock()
			delete(sessions.sessions, u.accessToken)
			sessions.mu.Unlock()
		}

		SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
		return nil, Redirect{URL: sanitizeReturn(req.PostFormValue("return"))}

	case req.Method == "GET" && req.URL.Path == "/api/userspec":
		// Authorization check.
		if u == nil {
			return nil, JSONResponse{users.UserSpec{}}
		}
		return nil, JSONResponse{users.UserSpec{ID: u.ID, Domain: "github.com"}}

	case req.Method == "GET" && req.URL.Path == "/api/user":
		// Authorization check.
		if u == nil {
			return nil, JSONResponse{users.UserSpec{}}
		}
		user, err := h.users.Get(context.TODO(), users.UserSpec{ID: u.ID, Domain: "github.com"})
		if err != nil {
			log.Println("/sessions: h.users.Get:", err)
			return nil, err
		}
		return nil, JSONResponse{user}

	case req.Method == "GET" && req.URL.Path == "/login":
		returnURL := sanitizeReturn(req.URL.Query().Get(returnQueryName))

		if u != nil {
			return nil, Redirect{URL: returnURL}
		}

		centered := &html.Node{
			Type: html.ElementNode, Data: atom.Div.String(),
			Attr: []html.Attribute{{Key: atom.Style.String(), Val: `margin-top: 100px; text-align: center;`}},
		}
		signInViaGitHub := PostButton{
			Action:    "/login/github",
			Text:      "Sign in via GitHub",
			ReturnURL: returnURL,
		}
		for _, n := range signInViaGitHub.Render() {
			centered.AppendChild(n)
		}
		return []*html.Node{centered}, nil

	case req.Method == "GET" && req.URL.Path == "/sessions":
		// Authorization check.
		if u == nil {
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}
		user, err := h.users.Get(context.TODO(), users.UserSpec{ID: u.ID, Domain: "github.com"})
		if err != nil {
			log.Println("/sessions: h.users.Get:", err)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}
		if !user.SiteAdmin {
			log.Printf("/sessions: non-SiteAdmin %q tried to access\n", user.Login)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}

		var nodes []*html.Node
		sessions.mu.Lock()
		for _, u := range sessions.sessions {
			user, err := h.users.Get(context.TODO(), users.UserSpec{ID: u.ID, Domain: "github.com"})
			if err != nil {
				return nil, err
			}
			nodes = append(nodes,
				htmlg.Div(htmlg.Text(fmt.Sprintf("Login: %q Domain: %q expiry: %v accessToken: %q...", user.Login, user.Domain, humanize.Time(u.expiry), base64.RawURLEncoding.EncodeToString([]byte(u.accessToken))[:20]))),
			)
		}
		if len(sessions.sessions) == 0 {
			nodes = append(nodes,
				htmlg.Div(htmlg.Text("-")),
			)
		}
		sessions.mu.Unlock()
		return nodes, nil

	default:
		return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrNotExist}
	}
}
