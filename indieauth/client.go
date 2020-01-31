package indieauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// Client describes an IndieAuth client application that is
// configured to perform the IndieAuth authentication flow.
// (The IndieAuth authorization flow is not supported yet.)
//
// See https://indieauth.spec.indieweb.org/#authentication.
type Client struct {
	// ClientID is the URL that an IndieAuth client is identified by.
	//
	// It must be an absolute URL that follows rules described at
	// https://indieauth.spec.indieweb.org/#client-identifier.
	ClientID string

	// RedirectURL is the URL to redirect users going through the
	// IndieAuth authentication flow, after approving the request.
	//
	// The URL scheme, host and port should match that of the ClientID.
	// See https://indieauth.spec.indieweb.org/#authentication-request.
	RedirectURL string
}

// AuthnReqURL returns the authentication request URL for the given
// user profile and state.
//
// See https://indieauth.spec.indieweb.org/#authentication-request.
func (c *Client) AuthnReqURL(authzEndpoint *url.URL, me, state string) string {
	return authzEndpoint.ResolveReference(&url.URL{
		RawQuery: url.Values{
			"me":            {me},
			"client_id":     {c.ClientID},
			"redirect_uri":  {c.RedirectURL},
			"state":         {state},
			"response_type": {"id"},
		}.Encode(),
	}).String()
}

// Verify makes a POST request to the authorization endpoint to verify
// the authorization code and retrieve the final user profile URL.
//
// An error is returned if the final user profile URL has a host that
// does not equal enteredHost, the host of the entered user profile URL.
//
// See https://indieauth.spec.indieweb.org/#authorization-code-verification
// and https://indieauth.spec.indieweb.org/#differing-user-profile-urls.
func (c *Client) Verify(ctx context.Context, authzEndpoint, enteredHost, code string) (me *url.URL, _ error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authzEndpoint, strings.NewReader(url.Values{
		"code":         {code},
		"client_id":    {c.ClientID},
		"redirect_uri": {c.RedirectURL},
	}.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authorization endpoint verification request failed: %v", err)
	}
	defer resp.Body.Close()

	// Handle error response.
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			log.Printf("indieauth.Client.Verify: got error response with non-'application/json' Content-Type header %q\n", ct)
			if mediaType, _, err := mime.ParseMediaType(ct); err != nil {
				return nil, fmt.Errorf("authorization endpoint returned bad Content-Type header %q: %v", ct, err)
			} else if mediaType != "application/json" {
				return nil, fmt.Errorf("authorization endpoint returned media type %q, want %q", mediaType, "application/json")
			}
		}
		var v struct{ Error string }
		err = json.NewDecoder(resp.Body).Decode(&v)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(v.Error)
	} else if resp.StatusCode != http.StatusOK {
		// Neither 200 OK, nor 400 Bad Request or 401 Unauthorized.
		// This is not a valid OAuth 2.0 response status code.
		//
		// See https://tools.ietf.org/html/rfc6749#section-5.2.
		return nil, fmt.Errorf("authorization endpoint returned non-200/400/401 status code: %v", resp.Status)
	}

	// Successful 200 OK response.
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		log.Printf("indieauth.Client.Verify: got successful response with non-'application/json' Content-Type header %q\n", ct)
		if mediaType, _, err := mime.ParseMediaType(ct); err != nil {
			return nil, fmt.Errorf("authorization endpoint returned bad Content-Type header %q: %v", ct, err)
		} else if mediaType != "application/json" {
			return nil, fmt.Errorf("authorization endpoint returned media type %q, want %q", mediaType, "application/json")
		}
	}
	var v struct{ Me string }
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return nil, fmt.Errorf("authorization endpoint returned invalid JSON: %v", err)
	}
	me, err = ParseUserProfile(v.Me)
	if err != nil {
		return nil, fmt.Errorf("authorization endpoint returned a bad user profile URL %q: %v", v.Me, err)
	}

	// Verify the resulting profile URL has a matching domain of the initially-entered profile URL.
	//
	// See https://indieauth.spec.indieweb.org/#differing-user-profile-urls.
	if me.Host != enteredHost {
		return nil, fmt.Errorf("authorization endpoint authenticated you as %q, doesn't match entered host %q", v.Me, enteredHost)
	}

	return me, nil
}
