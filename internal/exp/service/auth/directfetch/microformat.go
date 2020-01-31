package directfetch

import (
	"fmt"
	"net/url"
	"strings"

	"willnorris.com/go/microformats"
)

// photoURL returns the URL of the first h-card.photo property entry,
// or empty string if an h-card.photo property doesn't exist.
func photoURL(data *microformats.Data) (string, error) {
	h := hCard(data.Items)
	if h == nil {
		// There isn't a h-card.photo property.
		return "", nil
	}
	photos := h.Properties["photo"]
	if len(photos) == 0 {
		return "", fmt.Errorf("h-card.photo property exists but is empty, want non-empty")
	}
	u, ok := photos[0].(string)
	if !ok {
		return "", fmt.Errorf("h-card.photo[0] type is %T, want string", photos[0])
	} else if _, err := url.Parse(u); err != nil {
		return "", fmt.Errorf("error parsing photo URL %q: %v", u, err)
	}
	return u, nil
}

// hCard returns the first h-card microformat element,
// or nil if an h-card microformat element doesn't exist.
func hCard(items []*microformats.Microformat) *microformats.Microformat {
	for _, m := range items {
		if len(m.Type) == 1 && m.Type[0] == "h-card" {
			return m
		}
	}
	return nil
}

// githubLogin scans data for a rel="me" link pointing to a GitHub profile
// with a valid GitHub login (for example, "https://github.com/example" or
// "https://www.github.com/example"), and returns the login if it exists.
func githubLogin(data *microformats.Data) (string, bool) {
	for _, me := range data.Rels["me"] {
		switch {
		case strings.HasPrefix(me, "https://github.com/"):
			login := strings.TrimSuffix(me[len("https://github.com/"):], "/") // Trim trailing slash, if any.
			if !verifyGitHubLogin(login) {
				continue
			}
			return login, true
		case strings.HasPrefix(me, "https://www.github.com/"):
			login := strings.TrimSuffix(me[len("https://www.github.com/"):], "/") // Trim trailing slash, if any.
			if !verifyGitHubLogin(login) {
				continue
			}
			return login, true
		}
	}
	return "", false
}

// verifyGitHubLogin reports whether login is a syntactically valid GitHub login.
// The logic is derived from error messages on the GitHub sign up page, such as:
//
// 	• Username may only contain alphanumeric characters or single hyphens,
// 	  and cannot begin or end with a hyphen.
// 	• Username is too long (maximum is 39 characters).
//
func verifyGitHubLogin(login string) bool {
	if login == "" || len(login) > 39 {
		return false
	}
	for _, b := range []byte(login) {
		ok := ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') || b == '-'
		if !ok {
			return false
		}
	}
	if strings.HasPrefix(login, "-") || strings.HasSuffix(login, "-") || strings.Contains(login, "--") {
		return false
	}
	return true
}
