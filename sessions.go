package main

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/users"
)

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

// AddNewSession adds a new session with the specified user.
// userSpec must be a valid existing (i.e., non-zero) user.
func (s *state) AddNewSession(userSpec users.UserSpec) (accessToken string, expiry time.Time) {
	accessToken = string(cryptoRandBytes())
	expiry = time.Now().Add(7 * 24 * time.Hour)

	s.mu.Lock()
	for token, user := range s.sessions { // Clean up expired sessions.
		if time.Now().Before(user.Expiry) {
			continue
		}
		delete(s.sessions, token)
	}
	s.sessions[accessToken] = session{
		UserSpec:    userSpec,
		Expiry:      expiry,
		AccessToken: accessToken,
	}
	s.mu.Unlock()

	return accessToken, expiry
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
	returnParameterName   = "return"
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
	// UserSpec is the spec of a valid existing (i.e., non-zero) user.
	UserSpec users.UserSpec

	Expiry      time.Time
	AccessToken string // Access token. Needed to be able to clear session when the user signs out.
}

func setAccessTokenCookie(w httputil.HeaderWriter, accessToken string, expiry time.Time) {
	// TODO: Is base64 the best encoding for cookie values? Factor it out maybe?
	encodedAccessToken := base64.RawURLEncoding.EncodeToString([]byte(accessToken))
	httputil.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, Value: encodedAccessToken, Expires: expiry, HttpOnly: false, Secure: *secureCookieFlag})
}
func clearAccessTokenCookie(w httputil.HeaderWriter) {
	httputil.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
}

// cookieAuth is a middleware that parses authentication information
// from request cookies, and sets session as a context value.
type cookieAuth struct {
	Handler http.Handler
}

func (mw cookieAuth) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s, extended, err := lookUpSessionViaCookie(req)
	if err == errBadAccessToken {
		// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
		clearAccessTokenCookie(w)
	} else if err == nil && extended {
		// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
		setAccessTokenCookie(w, s.AccessToken, s.Expiry)
	}
	mw.Handler.ServeHTTP(w, withSession(req, s))
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

var errBadAccessToken = errors.New("bad access token")

// lookUpSessionViaCookie retrieves the session from req by looking up
// the request's access token (via accessTokenCookieName cookie) in the sessions map.
// It returns a valid session (possibly nil) and nil error,
// or nil session and errBadAccessToken.
// extended reports whether matched session expiry was extended.
func lookUpSessionViaCookie(req *http.Request) (s *session, extended bool, err error) {
	cookie, err := req.Cookie(accessTokenCookieName)
	if err == http.ErrNoCookie {
		return nil, false, nil // No session.
	} else if err != nil {
		panic(fmt.Errorf("internal error: Request.Cookie is documented to return only nil or ErrNoCookie error, yet it returned %v", err))
	}
	accessTokenBytes, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, false, errBadAccessToken
	}
	accessToken := string(accessTokenBytes)
	global.mu.Lock()
	if session, ok := global.sessions[accessToken]; ok {
		if time.Now().Before(session.Expiry) {
			// Extend expiry if 6 days or less left.
			if time.Until(session.Expiry) <= 6*24*time.Hour {
				session.Expiry = time.Now().Add(7 * 24 * time.Hour)
				global.sessions[accessToken] = session
				extended = true
			}

			s = &session
		} else {
			delete(global.sessions, accessToken) // This is unlikely to happen because cookie expires by then.
		}
	}
	global.mu.Unlock()
	if s == nil {
		return nil, false, errBadAccessToken
	}
	return s, extended, nil // Existing session.
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

// lookUpSessionViaBasicAuth retrieves the session from req by looking up
// the request's access token (via Basic Auth password) in the sessions map,
// getting the associated user via usersService, and verifying that the
// provided Basic Auth username matches the user login.
// It returns a valid session (possibly nil) and nil error,
// or nil session and errBadAccessToken.
func lookUpSessionViaBasicAuth(req *http.Request, usersService users.Service) (*session, error) {
	username, password, ok := req.BasicAuth()
	if !ok {
		return nil, nil // No session.
	}
	encodedAccessToken := password
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
	// Existing session, now get user and verify the username matches.
	user, err := usersService.Get(req.Context(), s.UserSpec)
	if err != nil {
		log.Println("lookUpSessionUserViaBasicAuth: failed to get user:", err)
		return nil, errBadAccessToken
	}
	if username != user.Login {
		return nil, errBadAccessToken
	}
	return s, nil // Existing session.
}
