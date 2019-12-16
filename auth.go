package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	githubv3 "github.com/google/go-github/github"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/oauth2"
	githuboauth2 "golang.org/x/oauth2/github"
)

var githubConfig = oauth2.Config{
	ClientID:     os.Getenv("HOME_GH_CLIENT_ID"),
	ClientSecret: os.Getenv("HOME_GH_CLIENT_SECRET"),
	Scopes:       nil,
	Endpoint:     githuboauth2.Endpoint,
}

func initAuth(usersService users.Service, userStore userCreator) {
	logoStyle := "header a.Logo { color: rgb(35, 35, 35); } header a.Logo:hover { color: #4183c4; }"
	if component.RedLogo {
		logoStyle = "header a.Logo { color: red; } header a.Logo:hover { color: darkred; }"
	}
	serveSignInPage := signInPage{
		Logo: template.HTML("<style>" + logoStyle + "</style>" + htmlg.RenderComponentsString(component.Logo{})),
	}.Serve

	type state struct {
		Expiry       time.Time
		EnteredLogin string
		ReturnURL    string
	}
	var statesMu sync.Mutex
	var states = make(map[string]state) // State Key -> State.
	http.Handle("/login", cookieAuth{httputil.ErrorHandler(usersService,
		func(w http.ResponseWriter, req *http.Request) error {
			if err := httputil.AllowMethods(req, http.MethodGet, http.MethodPost); err != nil {
				return err
			}

			returnURL := sanitizeReturn(req.FormValue(returnParameterName))

			if u, err := usersService.GetAuthenticatedSpec(req.Context()); err != nil {
				return err
			} else if u != (users.UserSpec{}) {
				return httperror.Redirect{URL: returnURL}
			}

			switch req.Method {
			case http.MethodGet:
				return serveSignInPage(w, "")
			case http.MethodPost:
				me, err := parseProfileURL(req.PostFormValue("me"))
				log.Printf("parseProfileURL(%q) -> err=%v me=%q\n", req.PostFormValue("me"), err, me)
				if err != nil {
					return serveSignInPage(w, err.Error())
				}
				switch me.Host {
				case "github.com":
					login, ok := parseGitHubLogin(me.Path)
					if !ok {
						return serveSignInPage(w, "GitHub URL must be like https://github.com/example")
					}

					// Do a best-effort preemptive check. Don't use an authenticated client here
					// because unauthenticated requests can force it to exceed GitHub rate limit.
					if u, resp, err := unauthGHV3.Users.Get(req.Context(), login); resp != nil &&
						resp.StatusCode == http.StatusNotFound {
						return serveSignInPage(w, fmt.Sprintf("GitHub user %q doesn't exist", login))
					} else if err == nil && u.GetType() != "User" {
						return serveSignInPage(w, fmt.Sprintf("%q is a GitHub %v; need a GitHub User", login, u.GetType()))
					}

					// Add new state.
					stateKey := base64.RawURLEncoding.EncodeToString(cryptoRandBytes()) // GitHub doesn't handle all non-ASCII bytes in state, so use base64.
					statesMu.Lock()
					for key, s := range states { // Clean up expired states.
						if time.Now().Before(s.Expiry) {
							continue
						}
						delete(states, key)
					}
					states[stateKey] = state{
						Expiry:       time.Now().Add(5 * time.Minute), // Enough time to get password, use 2 factor auth, etc.
						EnteredLogin: login,
						ReturnURL:    returnURL,
					}
					statesMu.Unlock()

					url := githubConfig.AuthCodeURL(stateKey,
						oauth2.SetAuthURLParam("login", login),
						oauth2.SetAuthURLParam("allow_signup", "false"))
					return httperror.Redirect{URL: url}
				default:
					return serveSignInPage(w, "other URL types aren't supported yet, only GitHub URLs like https://github.com/example are supported now")
				}
			default:
				panic("unreachable")
			}
		},
	)})
	http.Handle("/callback/github", cookieAuth{httputil.ErrorHandler(usersService,
		func(w http.ResponseWriter, req *http.Request) error {
			if err := httputil.AllowMethods(req, http.MethodGet); err != nil {
				return err
			}

			if u, err := usersService.GetAuthenticatedSpec(req.Context()); err != nil {
				return err
			} else if u != (users.UserSpec{}) {
				return httperror.Redirect{URL: "/"}
			}

			// Consume state.
			stateKey := req.FormValue("state")
			statesMu.Lock()
			state, ok := states[stateKey]
			delete(states, stateKey)
			statesMu.Unlock()

			// Verify state and expiry.
			if !ok || !time.Now().Before(state.Expiry) {
				return httperror.BadRequest{Err: fmt.Errorf("state not recognized")}
			}

			us, err := func() (users.User, error) {
				token, err := githubConfig.Exchange(req.Context(), req.FormValue("code"))
				if err != nil {
					return users.User{}, err
				}
				httpClient := githubConfig.Client(req.Context(), token)
				httpClient.Timeout = 5 * time.Second
				ghUser, _, err := githubv3.NewClient(httpClient).Users.Get(req.Context(), "")
				if err != nil {
					return users.User{}, err
				}
				if ghUser.ID == nil || *ghUser.ID == 0 {
					return users.User{}, errors.New("GitHub user ID is nil or 0")
				}
				if ghUser.Login == nil || *ghUser.Login == "" {
					return users.User{}, errors.New("GitHub user Login is nil or empty")
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
				log.Println("/callback/github: error getting user from GitHub:", err)
				// Show a problem page, if, for example, error came from gh.Users.Get("") due to GitHub being down.
				return serveSignInPage(w, "there was a problem authenticating via GitHub")
			}

			if state.EnteredLogin != "" && !strings.EqualFold(us.Login, state.EnteredLogin) {
				return serveSignInPage(w, fmt.Sprintf("GitHub authenticated you as %q, doesn't match entered %q", "github.com/"+us.Login, "github.com/"+state.EnteredLogin))
			}

			// If the user doesn't already exist, create it.
			err = userStore.Create(req.Context(), us)
			switch err {
			case nil, os.ErrExist:
				// Do nothing.
			default:
				log.Println("/callback/github: error creating user:", err)
				return httperror.HTTP{Code: http.StatusInternalServerError, Err: err}
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

			setAccessTokenCookie(w, accessToken, expiry)
			return httperror.Redirect{URL: state.ReturnURL}
		},
	)})
	http.Handle("/logout", httputil.ErrorHandler(nil,
		func(w http.ResponseWriter, req *http.Request) error {
			if err := httputil.AllowMethods(req, http.MethodPost); err != nil {
				return err
			}
			if s, _, _ := lookUpSessionViaCookie(req); s != nil {
				global.mu.Lock()
				delete(global.sessions, s.AccessToken)
				global.mu.Unlock()
			}
			clearAccessTokenCookie(w)
			return httperror.Redirect{URL: sanitizeReturn(req.PostFormValue(returnParameterName))}
		},
	))

	http.Handle("/sessions", cookieAuth{httputil.ErrorHandler(usersService,
		func(w http.ResponseWriter, req *http.Request) error {
			if err := httputil.AllowMethods(req, http.MethodGet); err != nil {
				return err
			}

			// Authorization check.
			if u, err := usersService.GetAuthenticated(req.Context()); err != nil {
				return err
			} else if u.UserSpec == (users.UserSpec{}) {
				// TODO: Factor out this os.IsPermission(err) && s == nil check somewhere, if possible. (But this shouldn't apply for APIs.)
				loginURL := (&url.URL{
					Path:     "/login",
					RawQuery: url.Values{returnParameterName: {req.URL.String()}}.Encode(),
				}).String()
				return httperror.Redirect{URL: loginURL}
			} else if !u.SiteAdmin {
				return os.ErrPermission
			}

			var ss []session
			global.mu.Lock()
			for _, s := range global.sessions {
				ss = append(ss, s)
			}
			global.mu.Unlock()
			var nodes []*html.Node
			for _, s := range ss {
				u, err := usersService.Get(req.Context(), users.UserSpec{ID: s.GitHubUserID, Domain: "github.com"})
				if err != nil {
					log.Printf("usersService.Get(%+v): %v\n", users.UserSpec{ID: s.GitHubUserID, Domain: "github.com"}, err)
					u = users.User{
						UserSpec: users.UserSpec{ID: s.GitHubUserID, Domain: "github.com"},
						Login:    fmt.Sprintf("??? (GitHubUserID=%d)", s.GitHubUserID),
					}
				}
				nodes = append(nodes,
					htmlg.Div(htmlg.Text(fmt.Sprintf("Login: %q Domain: %q expiry: %v accessToken: %q...", u.Login, u.Domain, humanize.Time(s.Expiry), base64.RawURLEncoding.EncodeToString([]byte(s.AccessToken)[:15])))),
				)
			}
			if len(ss) == 0 {
				nodes = append(nodes,
					htmlg.Div(htmlg.Text("-")),
				)
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			for _, n := range nodes {
				err := html.Render(w, n)
				if err != nil {
					return err
				}
			}
			return nil
		},
	)})
}

// parseProfileURL parses a user profile URL that a user is identified by.
//
// It verifies the restrictions as described in the IndieAuth specification
// at https://indieauth.spec.indieweb.org/#user-profile-url:
//
// 	Profile URLs
// 		MUST have either an https or http scheme,
// 		MUST contain a path component (/ is a valid path),
// 		MUST NOT contain single-dot or double-dot path segments,
// 		MAY contain a query string component,
// 		MUST NOT contain a fragment component,
// 		MUST NOT contain a username or password component, and
// 		MUST NOT contain a port.
// 	Additionally, hostnames
// 		MUST be domain names and
// 		MUST NOT be ipv4 or ipv6 addresses.
//
// It applies a few additional restrictions for now.
//
func parseProfileURL(me string) (*url.URL, error) {
	if len(me) > 50 {
		return nil, fmt.Errorf("URL should not be longer than 50 bytes (for now)")
	}
	u, err := url.Parse(me)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" {
		// Require "https" scheme. This is stricter than IndieAuth spec requires.
		return nil, fmt.Errorf("URL scheme must be https")
	}
	if u.Path == "" {
		// Canonicalize empty path to "/" to meet the requirement of special URLs.
		//
		// See https://indieauth.spec.indieweb.org/#url-canonicalization
		// and https://url.spec.whatwg.org/#special-scheme.
		u.Path = "/"
	}
	if path.Clean("/"+u.Path) != u.Path || u.RawPath != "" {
		return nil, fmt.Errorf("URL path must be clean")
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("URL must not have a fragment")
	}
	if u.User != nil {
		return nil, fmt.Errorf("URL must not have a username or password")
	}
	if u.Port() != "" {
		return nil, fmt.Errorf("URL must not contain a port")
	}
	if !strings.Contains(u.Host, ".") {
		return nil, fmt.Errorf("URL must be a domain name (contain a dot)")
	}
	if net.ParseIP(u.Host) != nil {
		return nil, fmt.Errorf("URL must not be an IP")
	}
	u.Host = strings.ToLower(u.Host)
	return u, nil
}

func parseGitHubLogin(githubURLPath string) (string, bool) {
	if !strings.HasPrefix(githubURLPath, "/") {
		return "", false
	}
	login := githubURLPath[1:]
	if login == "" {
		return "", false
	}
	for _, b := range []byte(login) {
		ok := ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') || b == '-'
		if !ok {
			return "", false
		}
	}
	if strings.HasPrefix(login, "-") || strings.HasSuffix(login, "-") || strings.Contains(login, "--") {
		return "", false
	}
	return login, true
}

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
	return (&url.URL{Path: u.Path, RawQuery: u.RawQuery, Fragment: u.Fragment}).String()
}

type signInPage struct {
	Logo template.HTML
}

func (p signInPage) Serve(w http.ResponseWriter, errorText string) error {
	// TODO: redirect to /login or some other friendlier URL and show the page there (via query params)?
	// TODO: consider using http.StatusUnauthorized rather than 200 OK status when errorText != ""?
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return signInHTML.Execute(w, struct {
		Logo     template.HTML
		SiteName string
		Error    string
	}{p.Logo, *siteNameFlag, errorText})
}

var signInHTML = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Dmitri Shuralyov - Sign In</title>
		<link href="/icon.png" rel="icon" type="image/png">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<style type="text/css">
body {
	font-family: Go;
	word-break: break-word;
}
input {
	font-family: inherit;
	font-size: 100%;
	box-sizing: border-box;
	width: 100%;
	padding: 5px;
	border: 1px solid lightgray;
	border-radius: 0;
	-webkit-appearance: none;
}
button {
	font-family: inherit;
	font-size: 100%;
	border: 1px solid lightgray;
	border-radius: 4px;
	box-shadow: 0 1px 1px rgba(0, 0, 0, .05);
	width: 100%;
}
header {
	text-align: center;
	margin-top: 50px;
	margin-bottom: 30px;
}
footer {
	text-align: center;
	margin-top: 50px;
	margin-bottom: 50px;
}
header h1 {
	margin-top: 30px;
}
div.error {
	font-size: 87.5%;
	text-align: center;
	background-color: rgb(255, 229, 232);
	border: 1px solid rgb(195, 137, 139);
	border-radius: 5px;
	margin: 20px;
	padding: 15px;
}
form {
	max-width: 355px;
	margin-left: auto;
	margin-right: auto;
	border: 1px solid lightgray;
	border-radius: 5px;
	padding: 15px;
}
form :first-child {
	margin-top: 0;
}
form :last-child {
	margin-bottom: 0;
}
p {
	margin-top: 20px;
	margin-bottom: 20px;
}
ul {
	line-height: 1.4;
}
b {
	font-weight: 500;
}
small {
	font-size: 10px;
}
		</style>
	</head>
	<body>
		<header>
			{{.Logo}}
			<h1>Sign In to {{.SiteName}}</h1>
		</header>
		{{with .Error}}<div class="error">{{.}}</div>{{end}}
		<form method="post" action="/login">
			<p>Enter your URL to sign in.</p>
			<p><input type="url" name="me" value="https://"></p>
			<p style="font-size: 80%; color: gray; margin-bottom: 8px;">Supported authentication methods:</p>
			<ul style="font-size: 80%; color: gray; margin-top: 8px; padding-left: 20px;">
				<li>https://github.com/example<small> — authenticate as <b>example</b> on GitHub</small></li>
			</ul>
			<p><button type="submit">Sign In</button></p>
		</form>
		<footer>
			<p style="font-size: 80%; color: gray;">Problem signing in?
			Please <a href="/about" style="color: gray;">let me know</a> and I'll fix it.</p>
		</footer>
	</body>
</html>
`))
