package httputil

import "net/http"

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
