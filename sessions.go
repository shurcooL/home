package main

import (
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

	"github.com/google/go-github/github"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
	"golang.org/x/net/context"
	"golang.org/x/net/html"
	"golang.org/x/oauth2"
	githuboauth2 "golang.org/x/oauth2/github"
)

var gitHubConfig = oauth2.Config{
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
)

// user is a GitHub user (i.e., domain is "github.com").
type user struct {
	ID uint64

	accessToken string // Internal access token. Needed to be able to clear session when this user signs out.
}

var errBadAccessToken = errors.New("bad access token")

// getUser either returns a valid user (possibly nil) and nil error, or
// nil user and errBadAccessToken.
func getUser(req *http.Request) (*user, error) {
	cookie, err := req.Cookie(accessTokenCookieName)
	if err == http.ErrNoCookie {
		return nil, nil // No user.
	} else if err != nil {
		return nil, errBadAccessToken
	}
	decodedAccessToken, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, errBadAccessToken
	}
	accessToken := string(decodedAccessToken)
	var u *user
	sessions.mu.Lock()
	if user, ok := sessions.sessions[accessToken]; ok {
		u = &user
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
	Body []byte
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
	handler func(user *user, w HeaderWriter, req *http.Request) ([]*html.Node, error)
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

	u, err := getUser(req)
	if err == errBadAccessToken {
		// TODO: Is it okay if we later set the same cookie again? Or should we avoid doing this here?
		http.SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
	}

	nodes, err := h.handler(u, w, req)
	switch {
	case IsRedirect(err):
		http.Redirect(w, req, string(err.(Redirect).URL), http.StatusSeeOther)
	case IsHTTPError(err):
		http.Error(w, err.Error(), err.(HTTPError).Code)
	case os.IsNotExist(err):
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
	case os.IsPermission(err):
		log.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
	default:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, string(htmlg.Render(nodes...)))
	case IsJSONResponse(err):
		w.Header().Set("Content-Type", "application/json")
		w.Write(err.(JSONResponse).Body)
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
	return u.Path
}

func SessionsHandler(u *user, w HeaderWriter, req *http.Request) ([]*html.Node, error) {
	// Simple switch-based router for now. For a larger project, a more sophisticated router should be used.
	switch {
	case req.Method == "POST" && req.URL.Path == "/login/github":
		returnURL := sanitizeReturn(req.PostFormValue("return"))

		if u != nil {
			return nil, Redirect{URL: returnURL}
		}

		state := base64.RawURLEncoding.EncodeToString(cryptoRandBytes()) // GitHub doesn't handle all non-ascii bytes in state, so use base64.
		SetCookie(w, &http.Cookie{Path: "/callback/github", Name: stateCookieName, Value: state, HttpOnly: true})

		// TODO, THINK.
		SetCookie(w, &http.Cookie{Path: "/callback/github", Name: returnCookieName, Value: returnURL, HttpOnly: true})

		url := gitHubConfig.AuthCodeURL(state)
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

			token, err := gitHubConfig.Exchange(oauth2.NoContext, req.FormValue("code"))
			if err != nil {
				return nil, err
			}
			tc := gitHubConfig.Client(oauth2.NoContext, token)
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
		sessions.mu.Lock()
		sessions.sessions[accessToken] = user{
			ID:          uint64(*ghUser.ID),
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
		SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, Value: encodedAccessToken, HttpOnly: true})
		return nil, Redirect{URL: returnURL}
	case req.Method == "POST" && req.URL.Path == "/logout":
		if u != nil {
			sessions.mu.Lock()
			delete(sessions.sessions, u.accessToken)
			sessions.mu.Unlock()
		}

		SetCookie(w, &http.Cookie{Path: "/", Name: accessTokenCookieName, MaxAge: -1})
		return nil, Redirect{URL: sanitizeReturn(req.PostFormValue("return"))}
	case req.Method == "GET" && req.URL.Path == "/api/user":
		// Authorization check.
		if u == nil {
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}
		user, err := usersService.Get(context.TODO(), users.UserSpec{ID: u.ID, Domain: "github.com"})
		if err != nil {
			log.Println("/sessions: usersService.Get:", err)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}
		b, err := json.MarshalIndent(user, "", "\t")
		if err != nil {
			return nil, err
		}
		return nil, JSONResponse{Body: b}
	case req.Method == "GET" && req.URL.Path == "/sessions":
		// Authorization check.
		if u == nil {
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}
		user, err := usersService.Get(context.TODO(), users.UserSpec{ID: u.ID, Domain: "github.com"})
		if err != nil {
			log.Println("/sessions: usersService.Get:", err)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}
		if !user.SiteAdmin {
			log.Printf("/sessions: non-SiteAdmin %q tried to access\n", user.Login)
			return nil, &os.PathError{Op: "open", Path: req.URL.String(), Err: os.ErrPermission}
		}

		var nodes []*html.Node
		sessions.mu.Lock()
		for _, u := range sessions.sessions {
			user, err := usersService.Get(context.TODO(), users.UserSpec{ID: u.ID, Domain: "github.com"})
			if err != nil {
				return nil, err
			}
			nodes = append(nodes,
				htmlg.Div(htmlg.Text(fmt.Sprintf("Login: %q Domain: %q accessToken: %q...", user.Login, user.Domain, base64.RawURLEncoding.EncodeToString([]byte(u.accessToken))[:20]))),
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
