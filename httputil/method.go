package httputil

import (
	"net/http"

	"github.com/shurcooL/httperror"
)

// AllowMethods returns nil if req.Method is one of allowed methods,
// or an error of type httperror.Method otherwise.
func AllowMethods(req *http.Request, allowed ...string) error {
	for _, method := range allowed {
		if req.Method == method {
			return nil
		}
	}
	return httperror.Method{Allowed: allowed}
}
