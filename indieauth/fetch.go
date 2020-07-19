package indieauth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/peterhellberg/link"
	"golang.org/x/net/html"
)

// UserProfile is the parsed result of fetching
// a user profile URL with an HTTP GET request.
type UserProfile struct {
	CanonicalMe   *url.URL // Canonical user profile URL (taking redirects into account).
	AuthzEndpoint *url.URL // URL of IndieAuth authorization endpoint, or nil if there isn't one.
}

// FetchUserProfile fetches the user profile specified by me,
// which must be a valid user profile URL, by making an HTTP
// GET request to the URL. It returns an error if the request
// fails, or if the response status code is not 200 OK.
//
// As a matter of policy, it does not include raw bytes from
// the response body of the HTTP GET request in error messages.
//
// The caller is responsible for enforcing a timeout.
func FetchUserProfile(ctx context.Context, t http.RoundTripper, me *url.URL) (UserProfile, *html.Node, error) {
	// Make a GET request to the user profile URL.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, me.String(), nil)
	if err != nil {
		return UserProfile{}, nil, fmt.Errorf("internal error: http.NewRequestWithContext failed: %v", err)
	}
	resp, err := (&http.Client{Transport: t, CheckRedirect: httpsOnly}).Do(req)
	if err != nil {
		return UserProfile{}, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Do not include body in error.
		return UserProfile{}, nil, fmt.Errorf("non-200 OK status code: %v", resp.Status)
	}

	var u UserProfile

	// Determine the canonical user profile URL.
	//
	// Start with the URL of the final request.
	// Then iterate over all redirects (from end to start) and
	// check if any of them were temporary rather than permanent.
	u.CanonicalMe = resp.Request.URL
	for resp := resp.Request.Response; resp != nil; resp = resp.Request.Response {
		if code := resp.StatusCode; code != 301 && code != 308 {
			// The response status code wasn't a permament redirect,
			// so use the previous URL as the canonical profile URL.
			u.CanonicalMe = resp.Request.URL
		}
	}

	// Look for the authorization endpoint in the first of two places,
	// the HTTP Link headers.
	if authz, err := authzEndpointInHeader(resp.Header, u.CanonicalMe); err != nil {
		return UserProfile{}, nil, err
	} else {
		u.AuthzEndpoint = authz
	}

	// If the media type is not HTML,
	// stop the search at this point.
	if mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err != nil {
		return UserProfile{}, nil, err
	} else if mediaType != "text/html" {
		return u, nil, nil
	}

	// Parse the HTML document.
	doc, err := html.Parse(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		// Do not include error details, since it may involve body.
		return UserProfile{}, nil, errors.New("error parsing HTML")
	}

	// If the authorization endpoint wasn't found in HTTP Link headers,
	// look in the second of two places, the HTML <link> elements.
	if u.AuthzEndpoint == nil {
		if authz, err := authzEndpointInHTML(doc, u.CanonicalMe); err != nil {
			return UserProfile{}, nil, err
		} else {
			u.AuthzEndpoint = authz
		}
	}

	return u, doc, nil
}

func authzEndpointInHeader(header http.Header, me *url.URL) (*url.URL, error) {
	for _, v := range header["Link"] {
		if l, ok := link.Parse(v)["authorization_endpoint"]; ok {
			return parseAuthzEndpoint(l.URI, me)
		}
	}
	return nil, nil
}

func authzEndpointInHTML(doc *html.Node, me *url.URL) (*url.URL, error) {
	var authzEndpoint string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			if attr(n, "rel") == "authorization_endpoint" {
				authzEndpoint = attr(n, "href")
				f = nil // Break out.
			}
		}
		for c := n.FirstChild; f != nil && c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	if authzEndpoint == "" {
		return nil, nil
	}
	return parseAuthzEndpoint(authzEndpoint, me)
}

func parseAuthzEndpoint(authzEndpoint string, me *url.URL) (*url.URL, error) {
	authz, err := me.Parse(authzEndpoint)
	if err != nil {
		return nil, err
	}
	if q := authz.Query(); q.Get("me") != "" ||
		q.Get("client_id") != "" ||
		q.Get("redirect_uri") != "" ||
		q.Get("state") != "" ||
		q.Get("response_type") != "" {
		return nil, fmt.Errorf("authorization endpoint URL %q has unexpected query parameters", authz)
	}
	return authz, nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// httpsOnly is an http.Client CheckRedirect policy
// that requires redirect target scheme to be HTTPS.
func httpsOnly(req *http.Request, _ []*http.Request) error {
	if req.URL.Scheme != "https" {
		return fmt.Errorf("redirected to insecure URL %s", req.URL)
	}
	return nil
}
