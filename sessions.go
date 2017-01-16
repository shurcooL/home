package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
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
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
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

var sessions = state{sessions: make(map[string]user)}

type state struct {
	mu       sync.Mutex
	sessions map[string]user // Access Token -> User.
}

// LoadAndRemove first loads state from file at path, then,
// if loading was successful, it removes the file.
func (s *state) LoadAndRemove(path string) error {
	err := s.load(path)
	if err != nil {
		return err
	}
	// Remove only if load was successful.
	return os.Remove(path)
}

func (s *state) load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	s.mu.Lock()
	err = gob.NewDecoder(f).Decode(&s.sessions)
	s.mu.Unlock()
	return err
}

// Save saves state to path with permission 0600.
func (s *state) Save(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	s.mu.Lock()
	err = gob.NewEncoder(f).Encode(s.sessions)
	s.mu.Unlock()
	return err
}

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

// user is a GitHub user (i.e., domain is "github.com").
type user struct {
	ID uint64

	Expiry      time.Time
	AccessToken string // Internal access token. Needed to be able to clear session when this user signs out.
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
		if time.Now().Before(user.Expiry) {
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

// userMiddleware parses authentication information from request headers,
// and sets authenticated user as a context value.
type userMiddleware struct {
	Handler http.Handler
}

func (mw userMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	user, err := getUser(req)
	if err == errBadAccessToken {
		// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
		http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
	}
	mw.Handler.ServeHTTP(w, withUser(req, user))
}

type sessionsHandler struct {
	users users.Service
}

func (h *sessionsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// HACK: Manually check that method is allowed for the given path.
	switch req.URL.Path {
	default:
		if req.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method should be GET", http.StatusMethodNotAllowed)
			return
		}
	case "/login/github", "/logout":
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
	req = withUser(req, u)

	nodes, err := h.serve(w, req, u)
	switch {
	case httputil.IsRedirect(err):
		http.Redirect(w, req, err.(httputil.Redirect).URL, http.StatusSeeOther)
	case httputil.IsHTTPError(err):
		http.Error(w, err.Error(), err.(httputil.HTTPError).Code)
	case os.IsNotExist(err):
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
	case os.IsPermission(err):
		// TODO: Factor this rr.Code == http.StatusUnauthorized && u == nil check out somewhere, if possible.
		if u == nil {
			loginURL := (&url.URL{
				Path:     "/login",
				RawQuery: url.Values{returnQueryName: {req.RequestURI}}.Encode(),
			}).String()
			http.Redirect(w, req, loginURL, http.StatusSeeOther)
			return
		}
		log.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
	default:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, string(htmlg.Render(nodes...)))
	case httputil.IsJSONResponse(err):
		w.Header().Set("Content-Type", "application/json")
		jw := json.NewEncoder(w)
		jw.SetIndent("", "\t")
		err := jw.Encode(err.(httputil.JSONResponse).V)
		if err != nil {
			log.Println("error encoding JSONResponse:", err)
		}
	case err != nil:
		log.Println(err)
		// TODO: Only display error details to SiteAdmin users?
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *sessionsHandler) serve(w httputil.HeaderWriter, req *http.Request, u *user) ([]*html.Node, error) {
	// Simple switch-based router for now. For a larger project, a more sophisticated router should be used.
	switch {
	case req.Method == "POST" && req.URL.Path == "/login/github":
		returnURL := sanitizeReturn(req.PostFormValue("return"))

		if u != nil {
			return nil, httputil.Redirect{URL: returnURL}
		}

		state := base64.RawURLEncoding.EncodeToString(cryptoRandBytes()) // GitHub doesn't handle all non-ascii bytes in state, so use base64.
		httputil.SetCookie(w, &http.Cookie{Path: "/callback/github", Name: stateCookieName, Value: state, HttpOnly: true, Secure: *productionFlag})

		// TODO, THINK.
		httputil.SetCookie(w, &http.Cookie{Path: "/callback/github", Name: returnCookieName, Value: returnURL, HttpOnly: true, Secure: *productionFlag})

		url := githubConfig.AuthCodeURL(state)
		return nil, httputil.Redirect{URL: url}

	case req.Method == "GET" && req.URL.Path == "/callback/github":
		if u != nil {
			return nil, httputil.Redirect{URL: "/"}
		}

		ghUser, err := func() (*github.User, error) {
			// Validate state (to prevent CSRF).
			cookie, err := req.Cookie(stateCookieName)
			if err != nil {
				return nil, err
			}
			httputil.SetCookie(w, &http.Cookie{Path: "/callback/github", Name: stateCookieName, MaxAge: -1})
			state := req.FormValue("state")
			if cookie.Value != state {
				return nil, errors.New("state doesn't match")
			}

			token, err := githubConfig.Exchange(req.Context(), req.FormValue("code"))
			if err != nil {
				return nil, err
			}
			tc := githubConfig.Client(req.Context(), token)
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
			return nil, httputil.HTTPError{Code: http.StatusUnauthorized, Err: err}
		}

		// Add new session.
		accessToken := string(cryptoRandBytes())
		expiry := time.Now().Add(7 * 24 * time.Hour)
		sessions.mu.Lock()
		for token, user := range sessions.sessions { // Clean up expired sesions.
			if time.Now().Before(user.Expiry) {
				continue
			}
			delete(sessions.sessions, token)
		}
		sessions.sessions[accessToken] = user{
			ID:          uint64(*ghUser.ID),
			Expiry:      expiry,
			AccessToken: accessToken,
		}
		sessions.mu.Unlock()

		// TODO, THINK.
		returnURL, err := func() (string, error) {
			cookie, err := req.Cookie(returnCookieName)
			if err != nil {
				return "", err
			}
			httputil.SetCookie(w, &http.Cookie{Path: "/callback/github", Name: returnCookieName, MaxAge: -1})
			return sanitizeReturn(cookie.Value), nil
		}()
		if err != nil {
			log.Println("/callback/github: problem with returnCookieName:", err)
			returnURL = "/"
		}

		// TODO: Is base64 the best encoding for cookie values? Factor it out maybe?
		encodedAccessToken := base64.RawURLEncoding.EncodeToString([]byte(accessToken))
		httputil.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, Value: encodedAccessToken, Expires: expiry, HttpOnly: true, Secure: *productionFlag})
		return nil, httputil.Redirect{URL: returnURL}

	case req.Method == "POST" && req.URL.Path == "/logout":
		if u != nil {
			sessions.mu.Lock()
			delete(sessions.sessions, u.AccessToken)
			sessions.mu.Unlock()
		}

		httputil.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
		return nil, httputil.Redirect{URL: sanitizeReturn(req.PostFormValue("return"))}

	case req.Method == "GET" && req.URL.Path == "/login":
		returnURL := sanitizeReturn(req.URL.Query().Get(returnQueryName))

		if u != nil {
			return nil, httputil.Redirect{URL: returnURL}
		}

		centered := &html.Node{
			Type: html.ElementNode, Data: atom.Div.String(),
			Attr: []html.Attribute{{Key: atom.Style.String(), Val: `margin-top: 100px; text-align: center;`}},
		}
		signInViaGitHub := component.PostButton{
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
		if user, err := h.users.Get(req.Context(), users.UserSpec{ID: u.ID, Domain: "github.com"}); err != nil {
			log.Println("/sessions: h.users.Get:", err)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		} else if !user.SiteAdmin {
			log.Printf("/sessions: non-SiteAdmin %q tried to access\n", user.Login)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}

		var us []user
		sessions.mu.Lock()
		for _, u := range sessions.sessions {
			us = append(us, u)
		}
		sessions.mu.Unlock()
		var nodes []*html.Node
		for _, u := range us {
			user, err := h.users.Get(req.Context(), users.UserSpec{ID: u.ID, Domain: "github.com"})
			if err != nil {
				return nil, err
			}
			nodes = append(nodes,
				htmlg.Div(htmlg.Text(fmt.Sprintf("Login: %q Domain: %q expiry: %v accessToken: %q...", user.Login, user.Domain, humanize.Time(u.Expiry), base64.RawURLEncoding.EncodeToString([]byte(u.AccessToken))[:20]))),
			)
		}
		if len(us) == 0 {
			nodes = append(nodes,
				htmlg.Div(htmlg.Text("-")),
			)
		}
		return nodes, nil

	default:
		return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrNotExist}
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
