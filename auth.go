package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	githubv3 "github.com/google/go-github/github"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/home/indieauth"
	"github.com/shurcooL/home/internal/exp/service/auth"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/oauth2"
	githuboauth2 "golang.org/x/oauth2/github"
)

var indieauthClient = indieauth.Client{
	ClientID:    os.Getenv("HOME_IA_CLIENT_ID"),
	RedirectURL: os.Getenv("HOME_IA_REDIRECT_URL"),
}

var githubConfig = oauth2.Config{
	ClientID:     os.Getenv("HOME_GH_CLIENT_ID"),
	ClientSecret: os.Getenv("HOME_GH_CLIENT_SECRET"),
	Scopes:       nil,
	Endpoint:     githuboauth2.Endpoint,
}

func initAuth(fs auth.FetchService, usersService users.Service, userStore userCreator) {
	logoStyle := `header a.Logo { color: rgb(35, 35, 35); } header a.Logo:hover { color: #4183c4; }
@media (prefers-color-scheme: dark) {
	header a.Logo { color: hsl(0, 0%, 90%); } header a.Logo:hover { color: #4183c4; }
}`
	if component.RedLogo {
		logoStyle = `header a.Logo { color: red; } header a.Logo:hover { color: darkred; }
@media (prefers-color-scheme: dark) {
	header a.Logo { color: #a00; } header a.Logo:hover { color: #e00; }
}`
	}
	signInPage := signInPage{
		Logo: template.HTML("<style>" + logoStyle + "</style>" + htmlg.RenderComponentsString(component.Logo{})),
	}
	serveSignInPage := func(w http.ResponseWriter, req *http.Request, errorText string) error {
		return signInPage.Serve(w, req, "", errorText)
	}

	// A semaphore to limit concurrent sign in processes.
	signInSem := make(chan struct{}, 5)

	type iaState struct {
		Expiry        time.Time
		User          users.User // User is the user signing in.
		EnteredHost   string     // Host of entered user profile URL.
		AuthzEndpoint *url.URL   // URL of IndieAuth authorization endpoint.
		ReturnURL     string
		Verifier      string
	}
	var iaStatesMu sync.Mutex
	var iaStates = make(map[string]iaState) // State Key -> IndieAuth State.
	type ghState struct {
		Expiry time.Time
		// User is the user signing in.
		// True GitHub users have Domain set to "github.com".
		// Indie users signing in via RelMeAuth (using GitHub) have Domain set to "dmitri.shuralyov.com",
		// and the CanonicalMe field set to their canonical user profile URL.
		User      users.User
		ReturnURL string
	}
	var ghStatesMu sync.Mutex
	var ghStates = make(map[string]ghState) // State Key -> GitHub State.
	http.Handle("/login", cookieAuth{httputil.ErrorHandler(usersService,
		func(w http.ResponseWriter, req *http.Request) error {
			if err := httputil.AllowMethods(req, http.MethodGet, http.MethodPost); err != nil {
				return err
			}

			returnURL := sanitizeReturn(req.FormValue(returnParameterName))

			if u, err := usersService.GetAuthenticatedSpec(req.Context()); err != nil {
				return err
			} else if u != (users.UserSpec{}) {
				return httperror.Redirect{URL: returnURL.String()}
			}

			serveSignInPage := serveSignInPage
			if returnURL.Path == "/api/indieauth/authorization" {
				// Display "to continue to <target site>" after "Sign in to <site name>".
				clientID, err := indieauth.ParseClientID(returnURL.Query().Get("client_id"))
				if err != nil {
					return httperror.BadRequest{Err: fmt.Errorf("bad client_id value: %v", err)}
				}
				continueTo := displayURL(*clientID)
				serveSignInPage = func(w http.ResponseWriter, req *http.Request, errorText string) error {
					return signInPage.Serve(w, req, continueTo, errorText)
				}
			}

			switch req.Method {
			case http.MethodGet:
				return serveSignInPage(w, req, "")
			case http.MethodPost:
				// Throttle unauthenticated sign in requests.
				select {
				case signInSem <- struct{}{}:
				default:
					return serveSignInPage(w, req, "too many requests to sign in are being made now, please try later")
				}
				defer func() { <-signInSem }()

				// Parse the entered user profile URL.
				me, err := indieauth.ParseUserProfile(req.PostFormValue("me"))
				log.Printf("indieauth.ParseUserProfile(%q) -> err=%v me=%q\n", req.PostFormValue("me"), err, me)
				if err != nil {
					return serveSignInPage(w, req, err.Error())
				}

				var (
					user        users.User // User who wants to sign in.
					enteredHost string     // Host of entered user profile URL.
					authVia     struct {   // Their authentication options.
						AuthzEndpoint *url.URL // URL of IndieAuth authorization endpoint, or nil if not available.
						GitHubLogin   string   // GitHub user login, or empty string if not available.
					}
				)

				// Fetch the entered user (but don't attempt to authenticate them yet).
				// Report an error on failure, or if the user information is malformed.
				switch me.Host {
				// A user on the independent web.
				default:
					indieUser, err := fs.FetchUserProfile(req.Context(), me)
					if err != nil {
						log.Printf("/login: error fetching user profile %q: %v\n", me, err)
						return serveSignInPage(w, req, err.Error())
					}

					// Discover presence on GitHub, if any.
					ghUser, err := func(ctx context.Context, fs auth.FetchService, p auth.UserProfile) (*users.User, error) {
						if p.GitHubLogin == "" {
							// This indie user doesn't have a presence on GitHub.
							return nil, nil
						}
						ghUser, ghUserWebsiteURL, err := fs.FetchGitHubUser(ctx, p.GitHubLogin)
						if err != nil {
							return nil, fmt.Errorf("user profile page at %q has a rel=\"me\" link to https://github.com/%s, but failed to fetch it: %v", p.CanonicalMe, p.GitHubLogin, err)
						}
						// Verify that the GitHub user's Website URL equals the user profile URL.
						if ghUserWebsiteURL == "" {
							return nil, fmt.Errorf("GitHub user %q has no Website URL set, but it needs to match user profile URL %q", ghUser.Login, p.CanonicalMe)
						} else if ghUserWebsite, err := indieauth.ParseUserProfile(ghUserWebsiteURL); err != nil {
							return nil, fmt.Errorf("GitHub user %q Website URL is %q, which is not a valid user profile URL: %v", ghUser.Login, ghUserWebsiteURL, err)
						} else if *ghUserWebsite != *p.CanonicalMe {
							return nil, fmt.Errorf("GitHub user %q Website URL is %q, doesn't match user profile URL %q", ghUser.Login, ghUserWebsite, p.CanonicalMe)
						}
						return &ghUser, nil
					}(req.Context(), fs, indieUser)
					if err != nil {
						return serveSignInPage(w, req, err.Error())
					}

					// Construct the user that is about to sign in.
					var (
						elsewhere []users.UserSpec
						avatarURL string
					)
					if ghUser != nil {
						elsewhere = append(elsewhere, ghUser.UserSpec)
					}
					avatarURL = indieUser.AvatarURL
					if avatarURL == "" && ghUser != nil {
						// Fall back to GitHub avatar.
						avatarURL = ghUser.AvatarURL
					}
					if avatarURL == "" {
						// Fall back to default avatar.
						avatarURL = "https://secure.gravatar.com/avatar?d=mm&f=y&s=96"
					}
					user = users.User{
						UserSpec:    users.UserSpec{Domain: "dmitri.shuralyov.com"},
						CanonicalMe: indieUser.CanonicalMe.String(),

						Elsewhere: elsewhere,

						Login:     displayURL(*indieUser.CanonicalMe),
						AvatarURL: avatarURL,
						HTMLURL:   indieUser.CanonicalMe.String(),
					}
					enteredHost = indieUser.CanonicalMe.Host

					// Populate authentication options.
					authVia.AuthzEndpoint = indieUser.AuthzEndpoint
					if ghUser != nil {
						authVia.GitHubLogin = ghUser.Login
					}

				// GitHub user.
				case "www.github.com":
					return serveSignInPage(w, req, `GitHub URL must omit the "www." subdomain, like https://github.com/example`)
				case "github.com":
					if me.Path == "/" {
						return serveSignInPage(w, req, "GitHub URL must include the user, like https://github.com/example")
					}
					login, ok := parseGitHubLogin(me.Path)
					if !ok {
						return serveSignInPage(w, req, "GitHub URL must be like https://github.com/example")
					}
					ghUser, _, err := fs.FetchGitHubUser(req.Context(), login)
					if err != nil {
						log.Printf("/login: error getting user %q from GitHub: %v\n", login, err)
						return serveSignInPage(w, req, fmt.Sprintf("error getting user %q from GitHub: %v", login, err))
					}

					// TODO: Discover presence on the independent web, if doing so is useful.

					// Construct the user that is about to sign in,
					// and populate authentication options.
					user = ghUser
					authVia.GitHubLogin = ghUser.Login
				}

				// Start authenticating the entered user via one of the authentication
				// options available to them. Prefer IndieAuth first, GitHub second.
				switch {
				// Authenticate via IndieAuth.
				case authVia.AuthzEndpoint != nil:
					// Add new IndieAuth state.
					stateKey := base64.RawURLEncoding.EncodeToString(cryptoRandBytes()) // OAuth 2.0 requires state to be printable ASCII, so use base64. See https://tools.ietf.org/html/rfc6749#appendix-A.5.
					verifier := newVerifier()
					iaStatesMu.Lock()
					for key, s := range iaStates { // Clean up expired IndieAuth states.
						if time.Now().Before(s.Expiry) {
							continue
						}
						delete(iaStates, key)
					}
					iaStates[stateKey] = iaState{
						Expiry:        time.Now().Add(5 * time.Minute), // Enough time to get password, use 2 factor auth, etc.
						User:          user,
						EnteredHost:   enteredHost,
						AuthzEndpoint: authVia.AuthzEndpoint,
						ReturnURL:     returnURL.String(),
						Verifier:      verifier,
					}
					iaStatesMu.Unlock()

					// Build the authentication request URL and redirect to it.
					url := indieauthClient.AuthnReqURL(authVia.AuthzEndpoint, user.CanonicalMe, stateKey, verifier)
					return httperror.Redirect{URL: url}

				// Authenticate via GitHub (either directly, or via RelMeAuth).
				case authVia.AuthzEndpoint == nil && authVia.GitHubLogin != "":
					// Add new GitHub state.
					stateKey := base64.RawURLEncoding.EncodeToString(cryptoRandBytes()) // OAuth 2.0 requires state to be printable ASCII, so use base64. See https://tools.ietf.org/html/rfc6749#appendix-A.5.
					ghStatesMu.Lock()
					for key, s := range ghStates { // Clean up expired GitHub states.
						if time.Now().Before(s.Expiry) {
							continue
						}
						delete(ghStates, key)
					}
					ghStates[stateKey] = ghState{
						Expiry:    time.Now().Add(5 * time.Minute), // Enough time to get password, use 2 factor auth, etc.
						User:      user,
						ReturnURL: returnURL.String(),
					}
					ghStatesMu.Unlock()

					url := githubConfig.AuthCodeURL(stateKey,
						oauth2.SetAuthURLParam("login", authVia.GitHubLogin),
						oauth2.SetAuthURLParam("allow_signup", "false"))
					return httperror.Redirect{URL: url}

				// No supported authentication options found.
				default:
					return serveSignInPage(w, req, fmt.Sprintf("couldn't find any supported way to authenticate you using your website\n"+
						"\n"+
						"to authenticate as %q, you can either:\n"+
						"\n"+
						"• add an IndieAuth authorization endpoint\n"+
						"• add a rel='me' link to your GitHub profile", me))
				}
			default:
				panic("unreachable")
			}
		},
	)})
	http.Handle("/callback/indieauth", cookieAuth{httputil.ErrorHandler(usersService,
		func(w http.ResponseWriter, req *http.Request) error {
			if err := httputil.AllowMethods(req, http.MethodGet); err != nil {
				return err
			}

			if u, err := usersService.GetAuthenticatedSpec(req.Context()); err != nil {
				return err
			} else if u != (users.UserSpec{}) {
				return httperror.Redirect{URL: "/"}
			}

			// Consume IndieAuth state.
			stateKey := req.FormValue("state")
			iaStatesMu.Lock()
			state, ok := iaStates[stateKey]
			delete(iaStates, stateKey)
			iaStatesMu.Unlock()
			user := state.User

			// Verify state and expiry.
			if !ok || !time.Now().Before(state.Expiry) {
				return httperror.BadRequest{Err: fmt.Errorf("state not recognized")}
			}

			// Handle an authentication error, if any.
			if err := req.FormValue("error"); err != "" {
				errorText := "there was a problem authenticating via IndieAuth: " + err
				if desc := req.FormValue("error_description"); desc != "" {
					errorText += "\n\n" + desc
				}
				return serveSignInPage(w, req, errorText)
			}

			// Verify the authorization code by making a POST request to the authorization endpoint.
			me, err := indieauthClient.Verify(req.Context(), state.AuthzEndpoint.String(), state.EnteredHost, req.FormValue("code"), state.Verifier)
			if err != nil {
				return serveSignInPage(w, req, err.Error())
			}
			if me.String() != user.CanonicalMe {
				// TODO, THINK: Disallow any mismatch for now. If allowed, may need to re-fetch all user info? Think more first.
				return serveSignInPage(w, req, fmt.Sprintf("authorization endpoint authenticated you as %q, doesn't match entered %q", me, user.CanonicalMe))
			}

			// Create or update user by their CanonicalMe.
			user, err = userStore.InsertByCanonicalMe(req.Context(), user)
			if err != nil {
				log.Println("/callback/indieauth: error creating or updating user:", err)
				return httperror.HTTP{Code: http.StatusInternalServerError, Err: err}
			}

			// Add new session with user who authenticated via IndieAuth.
			accessToken, expiry := global.AddNewSession(user.UserSpec)
			setAccessTokenCookie(w, accessToken, expiry)

			return httperror.Redirect{URL: state.ReturnURL}
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

			// Consume GitHub state.
			stateKey := req.FormValue("state")
			ghStatesMu.Lock()
			state, ok := ghStates[stateKey]
			delete(ghStates, stateKey)
			ghStatesMu.Unlock()
			user := state.User

			// Verify state and expiry.
			if !ok || !time.Now().Before(state.Expiry) {
				return httperror.BadRequest{Err: fmt.Errorf("state not recognized")}
			}

			// Verify the authenticated GitHub user equals the entered user.
			ghUserSpec, ghUserLogin, err := func() (users.UserSpec, string, error) {
				token, err := githubConfig.Exchange(req.Context(), req.FormValue("code"))
				if err != nil {
					return users.UserSpec{}, "", err
				}
				httpClient := githubConfig.Client(req.Context(), token)
				httpClient.Timeout = 5 * time.Second
				ghUser, _, err := githubv3.NewClient(httpClient).Users.Get(req.Context(), "")
				if err != nil {
					return users.UserSpec{}, "", err
				}
				if ghUser.ID == nil || *ghUser.ID <= 0 {
					return users.UserSpec{}, "", errors.New("GitHub user ID is nil or nonpositive")
				}
				if ghUser.Login == nil || *ghUser.Login == "" {
					return users.UserSpec{}, "", errors.New("GitHub user Login is nil or empty")
				}
				return users.UserSpec{ID: uint64(*ghUser.ID), Domain: "github.com"}, *ghUser.Login, nil
			}()
			if err != nil {
				log.Println("/callback/github: error getting user from GitHub:", err)
				// Show a problem page, if, for example, error came from gh.Users.Get("") due to GitHub being down.
				return serveSignInPage(w, req, "there was a problem authenticating via GitHub")
			}
			switch user.Domain {
			// Indie user signing in via RelMeAuth.
			case "dmitri.shuralyov.com":
				if len(user.Elsewhere) == 0 || user.Elsewhere[0].Domain != "github.com" {
					return fmt.Errorf("internal error: GitHub authenticated you as %q (GitHub ID %d), but can't find your expected GitHub identity", "github.com/"+ghUserLogin, ghUserSpec.ID)
				} else if ghUserSpec != user.Elsewhere[0] {
					return serveSignInPage(w, req, fmt.Sprintf("GitHub authenticated you as %q (GitHub ID %d), doesn't match expected GitHub ID %d", "github.com/"+ghUserLogin, ghUserSpec.ID, user.Elsewhere[0].ID))
				}
			// True GitHub user.
			case "github.com":
				if ghUserSpec != user.UserSpec {
					return serveSignInPage(w, req, fmt.Sprintf("GitHub authenticated you as %q, doesn't match entered %q", "github.com/"+ghUserLogin, "github.com/"+user.Login))
				}
			default:
				panic("unreachable")
			}

			// Create or update user.
			switch user.Domain {
			// Indie user signing in via RelMeAuth.
			case "dmitri.shuralyov.com":
				// Create or update user by their CanonicalMe.
				var err error
				user, err = userStore.InsertByCanonicalMe(req.Context(), user)
				if err != nil {
					log.Println("/callback/github: error creating or updating user:", err)
					return httperror.HTTP{Code: http.StatusInternalServerError, Err: err}
				}
			// True GitHub user.
			case "github.com":
				// TODO: Now is a good time to update user's Elsewhere, Login, AvatarURL, and HTMLURL if needed.
				//       But be mindful of SiteAdmin.

				// If the user doesn't already exist, create it.
				err := userStore.Create(req.Context(), user)
				switch err {
				case nil, os.ErrExist:
					// Do nothing.
				default:
					log.Println("/callback/github: error creating user:", err)
					return httperror.HTTP{Code: http.StatusInternalServerError, Err: err}
				}
			default:
				panic("unreachable")
			}

			// Add new session with user who authenticated via GitHub.
			accessToken, expiry := global.AddNewSession(user.UserSpec)
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
			return httperror.Redirect{URL: sanitizeReturn(req.PostFormValue(returnParameterName)).String()}
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
					RawQuery: url.Values{returnParameterName: {req.RequestURI}}.Encode(),
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
				u, err := usersService.Get(req.Context(), s.UserSpec)
				if err != nil {
					log.Printf("usersService.Get(%+v): %v\n", s.UserSpec, err)
					u = users.User{
						UserSpec: s.UserSpec,
						Login:    fmt.Sprintf("??? (UserSpec=%+v)", s.UserSpec),
					}
				}
				nodes = append(nodes,
					htmlg.Div(htmlg.Text(fmt.Sprintf("Login: %q UserSpec: %d@%q expiry: %v accessToken: %q...", u.Login, u.UserSpec.ID, u.UserSpec.Domain, humanize.Time(s.Expiry), base64.RawURLEncoding.EncodeToString([]byte(s.AccessToken)[:15])))),
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

// newVerifier generates a new code_verifier value.
func newVerifier() string {
	// A valid code_verifier has a minimum length of 43 characters and a maximum
	// length of 128 characters per https://tools.ietf.org/html/rfc7636#section-4.1.
	// Use 64 bytes of random data, which becomes 86 bytes after base64 encoding.
	b := make([]byte, 64)
	_, err := cryptorand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// initIndieAuth initializes the IndieAuth authorization endpoint.
// canonicalMe is the canonical IndieAuth 'me' user profile URL.
func initIndieAuth(usersService users.Service, canonicalMe *url.URL) {
	type authz struct {
		Expiry      time.Time
		ClientID    string
		RedirectURL string
		Challenge   string // The code_challenge value.
	}
	var authzsMu sync.Mutex
	var authzs = make(map[string]authz) // Code -> Authorization.
	http.Handle("/api/indieauth/authorization", cookieAuth{httputil.ErrorHandler(usersService,
		func(w http.ResponseWriter, req *http.Request) error {
			if err := httputil.AllowMethods(req, http.MethodGet, http.MethodPost); err != nil {
				return err
			}
			if err := req.ParseForm(); err != nil {
				return httperror.BadRequest{Err: err}
			}
			ru, err := url.Parse(req.Form.Get("redirect_uri"))
			if err != nil {
				return httperror.BadRequest{Err: err}
			}
			if q := ru.Query(); q.Get("code") != "" || q.Get("state") != "" {
				return httperror.BadRequest{Err: fmt.Errorf("redirect_uri contains an unexpected code or state query parameter")}
			}
			switch req.Method {
			case http.MethodGet:
				if typ := req.Form.Get("response_type"); typ != "code" {
					return httperror.BadRequest{Err: fmt.Errorf(`unexpected response_type type %q, want "code"`, typ)}
				}
				if me, ok := req.Form["me"]; ok && !(len(me) == 1 && me[0] == canonicalMe.String()) {
					return httperror.BadRequest{Err: fmt.Errorf("unexpected me value %q, want absent or %q", me, canonicalMe.String())}
				}
				if req.Form.Get("state") == "" {
					return httperror.BadRequest{Err: fmt.Errorf("mandatory state parameter is missing")}
				}
				if ccm := req.Form.Get("code_challenge_method"); ccm != "S256" {
					return httperror.BadRequest{Err: fmt.Errorf(`unsupported code_challenge_method value %q, only "S256" is supported`, ccm)}
				}
				if got, want := len(req.Form.Get("code_challenge")), base64.RawURLEncoding.EncodedLen(sha256.Size); got != want {
					return httperror.BadRequest{Err: fmt.Errorf("bad code_challenge length %d, want %d", got, want)}
				}
				clientID, err := indieauth.ParseClientID(req.Form.Get("client_id"))
				if err != nil {
					return httperror.BadRequest{Err: fmt.Errorf("bad client_id value: %v", err)}
				}
				if ru.Scheme != clientID.Scheme || ru.Host != clientID.Host {
					// Ensure the redirect_uri scheme, host and port match that of the client_id.
					//
					// TODO: support more advanced https://indieauth.spec.indieweb.org/#redirect-url cases:
					//       If the URL scheme, host or port of the redirect_uri in the request do not match
					//       that of the client_id, then the authorization endpoint SHOULD verify that the
					//       requested redirect_uri matches one of the redirect URLs published by the client,
					//       and SHOULD block the request from proceeding if not.
					return httperror.BadRequest{Err: fmt.Errorf("scheme+host of redirect_uri %q doesn't match scheme+host of client_id %q", ru.Scheme+"://"+ru.Host, clientID.Scheme+"://"+clientID.Host)}
				}
				if clientID.String() == canonicalMe.String() {
					// Redirect with an OAuth 2.0 error. See https://tools.ietf.org/html/rfc6749#section-4.1.2.1.
					q := ru.Query()
					q.Set("error", "access_denied")
					q.Set("error_description", "recursive sign in attempt; can't sign in into this site using this site")
					q.Set("state", req.Form.Get("state"))
					ru.RawQuery = q.Encode()
					return httperror.Redirect{URL: ru.String()}
				}
				if u, err := usersService.GetAuthenticatedSpec(req.Context()); err != nil {
					return err
				} else if u == (users.UserSpec{}) {
					loginURL := (&url.URL{
						Path:     "/login",
						RawQuery: url.Values{returnParameterName: {req.RequestURI}}.Encode(),
					}).String()
					return httperror.Redirect{URL: loginURL}
				} else if u != dmitshur {
					// Redirect with an OAuth 2.0 error. See https://tools.ietf.org/html/rfc6749#section-4.1.2.1.
					q := ru.Query()
					q.Set("error", "access_denied")
					q.Set("error_description", fmt.Sprintf("you are authenticated to %s as %d@%s, not as %d@%s", *siteNameFlag, u.ID, u.Domain, dmitshur.ID, dmitshur.Domain))
					q.Set("state", req.Form.Get("state"))
					ru.RawQuery = q.Encode()
					return httperror.Redirect{URL: ru.String()}
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				err = consentHTML.Execute(w, struct {
					ClientID    *url.URL
					Me          *url.URL
					RedirectURL *url.URL
				}{clientID, canonicalMe, ru})
				return err
			case http.MethodPost:
				switch authzCode := req.Form.Get("code"); {
				// Press of Allow button.
				case authzCode == "":
					if u, err := usersService.GetAuthenticatedSpec(req.Context()); err != nil {
						return err
					} else if u != dmitshur {
						return os.ErrPermission
					}

					// Add new authz code.
					authzCode := base64.RawURLEncoding.EncodeToString(cryptoRandBytes()) // OAuth 2.0 requires code to be printable ASCII, so use base64. See https://tools.ietf.org/html/rfc6749#appendix-A.11.
					expiry := time.Now().Add(time.Minute)
					authzsMu.Lock()
					for code, a := range authzs { // Clean up expired authorization codes.
						if time.Now().Before(a.Expiry) {
							continue
						}
						delete(authzs, code)
					}
					authzs[authzCode] = authz{
						Expiry:      expiry,
						ClientID:    req.Form.Get("client_id"),
						RedirectURL: req.Form.Get("redirect_uri"),
						Challenge:   req.Form.Get("code_challenge"),
					}
					authzsMu.Unlock()

					q := ru.Query()
					q.Set("code", authzCode)
					q.Set("state", req.Form.Get("state"))
					ru.RawQuery = q.Encode()
					return httperror.Redirect{URL: ru.String()}
				// Verification of authorization code.
				default:
					// Consume authz code.
					authzsMu.Lock()
					a, ok := authzs[authzCode]
					delete(authzs, authzCode)
					authzsMu.Unlock()

					if grant := req.Form.Get("grant_type"); grant == "" {
						// Respond with an OAuth 2.0 error. See https://tools.ietf.org/html/rfc6749#section-5.2.
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						return json.NewEncoder(w).Encode(struct {
							Error       string `json:"error"`
							Description string `json:"error_description"`
						}{"invalid_request", "mandatory grant_type parameter is missing"})
					} else if grant != "authorization_code" {
						// Respond with an OAuth 2.0 error. See https://tools.ietf.org/html/rfc6749#section-5.2.
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						return json.NewEncoder(w).Encode(struct {
							Error       string `json:"error"`
							Description string `json:"error_description"`
						}{"unsupported_grant_type", fmt.Sprintf("unexpected grant_type value %q, want %q", grant, "authorization_code")})
					}

					// Verify code, expiry, client_id, redirect_id, code_verifier match.
					if !ok || !time.Now().Before(a.Expiry) ||
						req.Form.Get("client_id") != a.ClientID ||
						req.Form.Get("redirect_uri") != a.RedirectURL ||
						!verifyPKCE(req.Form.Get("code_verifier"), a.Challenge) {

						// Respond with an OAuth 2.0 error. See https://tools.ietf.org/html/rfc6749#section-5.2.
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						return json.NewEncoder(w).Encode(struct {
							Error string `json:"error"`
						}{"invalid_grant"})
					}

					return httperror.JSONResponse{V: struct {
						Me string `json:"me"`
					}{canonicalMe.String()}}
				}
			default:
				panic("unreachable")
			}
		},
	)})
}

// verifyPKCE verifies the provided values according to the S256 code_challenge_method.
func verifyPKCE(verifier, challenge string) bool {
	if len(verifier) < 43 || 128 < len(verifier) {
		// A valid code_verifier has a minimum length of 43 characters and a maximum
		// length of 128 characters per https://tools.ietf.org/html/rfc7636#section-4.1.
		// Don't proceed if we see something outside that range.
		return false
	}
	s := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(s[:]) == challenge
}

// parseGitHubLogin parses a syntactically valid GitHub login from the path of a GitHub URL.
// The logic is derived from error messages on the GitHub sign up page, such as:
//
// 	• Username may only contain alphanumeric characters or single hyphens,
// 	  and cannot begin or end with a hyphen.
// 	• Username is too long (maximum is 39 characters).
//
func parseGitHubLogin(githubURLPath string) (string, bool) {
	if !strings.HasPrefix(githubURLPath, "/") {
		return "", false
	}
	login := githubURLPath[1:]
	if login == "" || len(login) > 39 {
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

// sanitizeReturn sanitizes a return URL. It must be
// a valid relative URL, otherwise "/" is returned.
func sanitizeReturn(returnURL string) *url.URL {
	u, err := url.Parse(returnURL)
	if err != nil ||
		u.Scheme != "" || u.Opaque != "" || u.User != nil || u.Host != "" ||
		u.Path == "" || u.RawPath != "" {
		return &url.URL{Path: "/"}
	}
	return &url.URL{Path: u.Path, RawQuery: u.RawQuery, Fragment: u.Fragment}
}

// displayURL returns the URL u in short form for display purposes.
// The scheme is omitted, and the "/" path isn't shown.
func displayURL(u url.URL) string {
	u.Scheme = ""
	if u.Path == "/" {
		u.Path = ""
	}
	return strings.TrimPrefix(u.String(), "//")
}

type signInPage struct {
	Logo template.HTML
}

func (p signInPage) Serve(w http.ResponseWriter, req *http.Request, continueTo, errorText string) error {
	// TODO: redirect to /login or some other friendlier URL and show the page there (via query params)?
	// TODO: consider using http.StatusUnauthorized rather than 200 OK status when errorText != ""?
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return signInHTML.Execute(w, struct {
		Logo       template.HTML
		SiteName   string
		ReturnURL  string
		ContinueTo string
		Error      string
	}{p.Logo, *siteNameFlag, req.FormValue(returnParameterName), continueTo, errorText})
}

var signInHTML = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Dmitri Shuralyov - Sign In</title>
		<link href="/icon.svg" rel="icon" type="image/svg+xml">
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
	white-space: pre-wrap;
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
strong {
	font-weight: 500;
}
small {
	font-size: 10px;
}
.gray, .gray a {
	color: gray;
}
@media (prefers-color-scheme: dark) {
	body, input {
		background-color: rgb(30, 30, 30);
		color: rgb(220, 220, 220);
	}
	button {
		background-color: rgb(40, 40, 40);
		color: rgb(220, 220, 220);
	}
	form, input, button {
		border-color: rgb(100, 100, 100);
	}
	div.error {
		background-color: hsl(353, 100%, 15%);
		border-color: hsl(358, 33%, 35%);
	}
	.gray, .gray a {
		color: rgb(180, 180, 180);
	}
}
		</style>
	</head>
	<body>
		<header>
			{{.Logo}}
			<h1>Sign in to {{.SiteName}}</h1>
			{{with .ContinueTo}}<h2>to continue to {{.}}</h2>{{end}}
		</header>
		{{with .Error}}<div class="error">{{.}}</div>{{end}}
		<form method="post" action="/login{{with .ReturnURL}}?return={{.}}{{end}}">
			<p>Enter your URL to sign in.</p>
			<p><input type="url" name="me" value="https://"></p>
			<p class="gray" style="font-size: 80%; margin-bottom: 8px;">Supported authentication methods:</p>
			<ul class="gray" style="font-size: 80%; margin-top: 8px; padding-left: 20px;">
				<li>https://example.com<small> — authenticate as <strong>example.com</strong> via <a href="https://indieauth.net">IndieAuth</a> or <a href="http://microformats.org/wiki/relmeauth">RelMeAuth</a></small></li>
				<li>https://github.com/example<small> — authenticate as <strong>example</strong> on GitHub</small></li>
			</ul>
			<p><button type="submit">Sign In</button></p>
		</form>
		<footer>
			<p class="gray" style="font-size: 80%;">Problem signing in?
			Please <a href="/about">let me know</a> and I'll fix it.</p>
		</footer>
	</body>
</html>
`))

var consentHTML = template.Must(template.New("").Funcs(template.FuncMap{
	"displayURL": displayURL,
}).Parse(`<!DOCTYPE html>
<html lang="en">
	<head>
		<title>Dmitri Shuralyov - Consent</title>
		<link href="/icon.svg" rel="icon" type="image/svg+xml">
		<meta name="viewport" content="width=device-width">
		<link href="/assets/fonts/fonts.css" rel="stylesheet" type="text/css">
		<style type="text/css">
body, input {
	font-family: Go;
	font-size: 80%;
}
.center {
	text-align: center;
}
.mt100 {
	margin-top: 100px;
}
.bold {
	font-weight: bold;
}
		</style>
	</head>
	<body>
		<div class="center mt100">
			<form class="center" method="post">
				<h1>Consent</h1>
				<p><a class="bold" href="{{.ClientID}}" title="{{.ClientID}}">{{displayURL .ClientID}}</a> would like to:</p>
				<p>• identify you as <abbr title="{{.Me}}">{{displayURL .Me}}</abbr></p>
				{{if ne .RedirectURL.Host .ClientID.Host}}
					<p>Authorizing will redirect to a different host:<br>
					<strong>{{.RedirectURL}}</strong></p>
				{{end}}
				<button type="submit" style="
font-family: inherit;
height: 18px;
border-radius: 4px;
box-shadow: 0 1px 1px rgba(0, 0, 0, .05);">Allow</button>
			</form>
		</div>
	</body>
</html>
`))
