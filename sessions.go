package main

import (
	cryptorand "crypto/rand"
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
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/go-github/github"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
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

// global server state.
var global = state{sessions: make(map[string]session)}

type state struct {
	mu       sync.Mutex
	sessions map[string]session // Access Token -> User Session.
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
	_, err := cryptorand.Read(b)
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

// session is a user session. Nil session pointer represents no session.
// Non-nil session pointers are expected to have valid users.
type session struct {
	// GitHubUserID is a valid (i.e., non-zero) GitHub user ID. Domain is "github.com".
	GitHubUserID uint64

	Expiry      time.Time
	AccessToken string // Access token. Needed to be able to clear session when the user signs out.
}

func setAccessTokenCookie(w httputil.HeaderWriter, accessToken string, expiry time.Time) {
	// TODO: Is base64 the best encoding for cookie values? Factor it out maybe?
	encodedAccessToken := base64.RawURLEncoding.EncodeToString([]byte(accessToken))
	httputil.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, Value: encodedAccessToken, Expires: expiry, HttpOnly: false, Secure: *productionFlag})
}
func clearAccessTokenCookie(w httputil.HeaderWriter) {
	httputil.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
}

var errBadAccessToken = errors.New("bad access token")

// lookUpSessionViaCookie retrieves the session from req by looking up
// the request's access token (via accessTokenCookieName cookie) in the sessions map.
// It returns a valid session (possibly nil) and nil error,
// or nil session and errBadAccessToken.
func lookUpSessionViaCookie(req *http.Request) (s *session, err error) {
	cookie, err := req.Cookie(accessTokenCookieName)
	if err == http.ErrNoCookie {
		return nil, nil // No session.
	} else if err != nil {
		return nil, errBadAccessToken
	}
	accessTokenBytes, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, errBadAccessToken
	}
	accessToken := string(accessTokenBytes)
	global.mu.Lock()
	if session, ok := global.sessions[accessToken]; ok {
		if time.Now().Before(session.Expiry) {
			s = &session
		} else {
			delete(global.sessions, accessToken) // This is unlikely to happen because cookie expires by then.
		}
	}
	global.mu.Unlock()
	if s == nil {
		return nil, errBadAccessToken
	}
	return s, nil // Existing session.
}

// cookieAuth is a middleware that parses authentication information
// from request cookies, and sets session as a context value.
type cookieAuth struct {
	Handler http.Handler
}

func (mw cookieAuth) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s, err := lookUpSessionViaCookie(req)
	if err == errBadAccessToken {
		// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
		clearAccessTokenCookie(w)
	}
	mw.Handler.ServeHTTP(w, withSession(req, s))
}

// lookUpSessionViaHeader retrieves the session from req by looking up
// the request's access token (via Authorization header) in the sessions map.
// It returns a valid session (possibly nil) and nil error,
// or nil session and errBadAccessToken.
func lookUpSessionViaHeader(req *http.Request) (*session, error) {
	authorization, ok := req.Header["Authorization"]
	if !ok {
		return nil, nil // No session.
	}
	if len(authorization) != 1 {
		return nil, errBadAccessToken
	}
	if !strings.HasPrefix(authorization[0], "Bearer ") {
		return nil, errBadAccessToken
	}
	encodedAccessToken := authorization[0][len("Bearer "):] // THINK: Should access token be base64 encoded?
	accessTokenBytes, err := base64.RawURLEncoding.DecodeString(encodedAccessToken)
	if err != nil {
		return nil, errBadAccessToken
	}
	accessToken := string(accessTokenBytes)
	var s *session
	global.mu.Lock()
	if session, ok := global.sessions[accessToken]; ok {
		if time.Now().Before(session.Expiry) {
			s = &session
		} else {
			delete(global.sessions, accessToken)
		}
	}
	global.mu.Unlock()
	if s == nil {
		return nil, errBadAccessToken
	}
	return s, nil // Existing session.
}

// headerAuth is a middleware that parses authentication information
// from request headers, and sets session as a context value.
type headerAuth struct {
	Handler http.Handler
}

func (mw headerAuth) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s, err := lookUpSessionViaHeader(req)
	if err != nil {
		http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
		return
	}
	mw.Handler.ServeHTTP(w, withSession(req, s))
}

type sessionsHandler struct {
	users     users.Service
	userStore users.Store
}

func (h *sessionsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// HACK: Manually check that method is allowed for the given path.
	switch req.URL.Path {
	default:
		if req.Method != "GET" {
			httperror.HandleMethod(w, httperror.Method{Allowed: []string{"GET"}})
			return
		}
	case "/login/github", "/logout":
		if req.Method != "POST" {
			httperror.HandleMethod(w, httperror.Method{Allowed: []string{"POST"}})
			return
		}
	}

	// TODO: Factor this out into user middleware?
	s, err := lookUpSessionViaCookie(req)
	if err == errBadAccessToken {
		// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
		//       E.g., that will happen when you're logging in. First, errBadAccessToken happens (here), then a successful login results in setting accessTokenCookieName to a new value (in h.serve).
		clearAccessTokenCookie(w)
	}
	req = withSession(req, s)

	nodes, err := h.serve(w, req, s)
	if err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, htmlg.Render(nodes...))
		return
	}
	if err, ok := httperror.IsRedirect(err); ok {
		http.Redirect(w, req, err.URL, http.StatusSeeOther)
		return
	}
	if err, ok := httperror.IsBadRequest(err); ok {
		httperror.HandleBadRequest(w, err)
		return
	}
	if err, ok := httperror.IsHTTP(err); ok {
		code := err.Code
		error := fmt.Sprintf("%d %s", code, http.StatusText(code))
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, code)
		return
	}
	if err, ok := httperror.IsJSONResponse(err); ok {
		w.Header().Set("Content-Type", "application/json")
		jw := json.NewEncoder(w)
		jw.SetIndent("", "\t")
		err := jw.Encode(err.V)
		if err != nil {
			log.Println("error encoding JSONResponse:", err)
		}
		return
	}
	if os.IsNotExist(err) {
		log.Println(err)
		error := "404 Not Found"
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusNotFound)
		return
	}
	if os.IsPermission(err) {
		// TODO: Factor out this os.IsPermission(err) && s == nil check somewhere, if possible. (But this shouldn't apply for APIs.)
		if s == nil {
			loginURL := (&url.URL{
				Path:     "/login",
				RawQuery: url.Values{returnQueryName: {req.RequestURI}}.Encode(),
			}).String()
			http.Redirect(w, req, loginURL, http.StatusSeeOther)
			return
		}
		log.Println(err)
		error := "403 Forbidden"
		if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
			error += "\n\n" + err.Error()
		}
		http.Error(w, error, http.StatusForbidden)
		return
	}

	log.Println(err)
	error := "500 Internal Server Error"
	if user, e := h.users.GetAuthenticated(req.Context()); e == nil && user.SiteAdmin {
		error += "\n\n" + err.Error()
	}
	http.Error(w, error, http.StatusInternalServerError)
}

func (h *sessionsHandler) serve(w httputil.HeaderWriter, req *http.Request, s *session) ([]*html.Node, error) {
	// Simple switch-based router for now. For a larger project, a more sophisticated router should be used.
	switch {
	case req.Method == "POST" && req.URL.Path == "/login/github":
		returnURL := sanitizeReturn(req.PostFormValue("return"))

		if s != nil {
			return nil, httperror.Redirect{URL: returnURL}
		}

		state := base64.RawURLEncoding.EncodeToString(cryptoRandBytes()) // GitHub doesn't handle all non-ASCII bytes in state, so use base64.
		httputil.SetCookie(w, &http.Cookie{Path: "/callback/github", Name: stateCookieName, Value: state, HttpOnly: true, Secure: *productionFlag})

		// TODO, THINK.
		httputil.SetCookie(w, &http.Cookie{Path: "/callback/github", Name: returnCookieName, Value: returnURL, HttpOnly: true, Secure: *productionFlag})

		url := githubConfig.AuthCodeURL(state)
		return nil, httperror.Redirect{URL: url}

	case req.Method == "GET" && req.URL.Path == "/callback/github":
		if s != nil {
			return nil, httperror.Redirect{URL: "/"}
		}

		us, err := func() (users.User, error) {
			// Validate state (to prevent CSRF).
			cookie, err := req.Cookie(stateCookieName)
			if err != nil {
				return users.User{}, err
			}
			httputil.SetCookie(w, &http.Cookie{Path: "/callback/github", Name: stateCookieName, MaxAge: -1})
			state := req.FormValue("state")
			if cookie.Value != state {
				return users.User{}, errors.New("state doesn't match")
			}

			token, err := githubConfig.Exchange(req.Context(), req.FormValue("code"))
			if err != nil {
				return users.User{}, err
			}
			httpClient := githubConfig.Client(req.Context(), token)
			httpClient.Timeout = 5 * time.Second
			gh := github.NewClient(httpClient)

			ghUser, _, err := gh.Users.Get(req.Context(), "")
			if err != nil {
				return users.User{}, err
			}
			if ghUser.ID == nil || *ghUser.ID == 0 {
				return users.User{}, errors.New("GitHub user ID is nil/0")
			}
			if ghUser.Login == nil || *ghUser.Login == "" {
				return users.User{}, errors.New("GitHub user Login is nil/empty")
			}
			if ghUser.AvatarURL == nil {
				return users.User{}, errors.New("GitHub user AvatarURL is nil")
			}
			if ghUser.HTMLURL == nil {
				return users.User{}, errors.New("GitHub user HTMLURL is nil")
			}
			return users.User{
				UserSpec:  users.UserSpec{ID: uint64(*ghUser.ID), Domain: "github.com"},
				Login:     *ghUser.Login,
				AvatarURL: *ghUser.AvatarURL,
				HTMLURL:   *ghUser.HTMLURL,
			}, nil
		}()
		if err != nil {
			log.Println(err)
			// TODO: Redirect to an "problem with logging in" page, if, for example, error came from gh.Users.Get("") due to GitHub being down.
			return nil, httperror.HTTP{Code: http.StatusUnauthorized, Err: err}
		}

		// If the user doesn't already exist, create it.
		err = h.userStore.Create(req.Context(), us)
		switch err {
		case nil, os.ErrExist:
			// Do nothing.
		default:
			log.Println("/callback/github: error creating user:", err)
			return nil, httperror.HTTP{Code: http.StatusInternalServerError, Err: err}
		}

		// Add new session.
		accessToken := string(cryptoRandBytes())
		expiry := time.Now().Add(7 * 24 * time.Hour)
		global.mu.Lock()
		for token, user := range global.sessions { // Clean up expired sesions.
			if time.Now().Before(user.Expiry) {
				continue
			}
			delete(global.sessions, token)
		}
		global.sessions[accessToken] = session{
			GitHubUserID: us.ID,
			Expiry:       expiry,
			AccessToken:  accessToken,
		}
		global.mu.Unlock()

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

		setAccessTokenCookie(w, accessToken, expiry)
		return nil, httperror.Redirect{URL: returnURL}

	case req.Method == "POST" && req.URL.Path == "/logout":
		if s != nil {
			global.mu.Lock()
			delete(global.sessions, s.AccessToken)
			global.mu.Unlock()
		}

		clearAccessTokenCookie(w)
		return nil, httperror.Redirect{URL: sanitizeReturn(req.PostFormValue("return"))}

	case req.Method == "GET" && req.URL.Path == "/login":
		returnURL := sanitizeReturn(req.URL.Query().Get(returnQueryName))

		if s != nil {
			return nil, httperror.Redirect{URL: returnURL}
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
		if s == nil {
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}
		if user, err := h.users.Get(req.Context(), users.UserSpec{ID: s.GitHubUserID, Domain: "github.com"}); err != nil {
			log.Println("/sessions: h.users.Get:", err)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		} else if !user.SiteAdmin {
			log.Printf("/sessions: non-SiteAdmin %q tried to access\n", user.Login)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}

		var ss []session
		global.mu.Lock()
		for _, s := range global.sessions {
			ss = append(ss, s)
		}
		global.mu.Unlock()
		var nodes []*html.Node
		for _, s := range ss {
			user, err := h.users.Get(req.Context(), users.UserSpec{ID: s.GitHubUserID, Domain: "github.com"})
			if err != nil {
				return nil, err
			}
			nodes = append(nodes,
				htmlg.Div(htmlg.Text(fmt.Sprintf("Login: %q Domain: %q expiry: %v accessToken: %q...", user.Login, user.Domain, humanize.Time(s.Expiry), base64.RawURLEncoding.EncodeToString([]byte(s.AccessToken)[:15])))),
			)
		}
		if len(ss) == 0 {
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
