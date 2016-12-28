package httphandler

import (
	"net/http"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/notifications"
)

// Notifications is an API handler for notifications.Service.
type Notifications struct {
	Notifications notifications.Service
}

func (h Notifications) Count(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return httputil.MethodError{Allowed: []string{"GET"}}
	}
	n, err := h.Notifications.Count(req.Context(), nil)
	if err != nil {
		return err
	}
	return httputil.JSONResponse{V: n}
}
