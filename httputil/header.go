package httputil

import "net/http"

// HeaderWriter interface is used to construct an HTTP response header and trailer.
type HeaderWriter interface {
	// Header returns the header map that will be sent by
	// WriteHeader. The Header map also is the mechanism with which
	// Handlers can set HTTP trailers.
	//
	// Changing the header map after a call to WriteHeader (or
	// Write) has no effect unless the modified headers are
	// trailers.
	//
	// There are two ways to set Trailers. The preferred way is to
	// predeclare in the headers which trailers you will later
	// send by setting the "Trailer" header to the names of the
	// trailer keys which will come later. In this case, those
	// keys of the Header map are treated as if they were
	// trailers. See the example. The second way, for trailer
	// keys not known to the Handler until after the first Write,
	// is to prefix the Header map keys with the TrailerPrefix
	// constant value. See TrailerPrefix.
	//
	// To suppress implicit response headers (such as "Date"), set
	// their value to nil.
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
