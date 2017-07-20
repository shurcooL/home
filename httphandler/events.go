package httphandler

import (
	"net/http"

	"github.com/shurcooL/events"
	"github.com/shurcooL/events/event"
	"github.com/shurcooL/httperror"
)

// Events is an API handler for events.Service.
type Events struct {
	Events events.Service
}

func (h Events) List(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return httperror.Method{Allowed: []string{http.MethodGet}}
	}
	var v struct {
		Events []event.Event
		Error  *string
	}
	events, err := h.Events.List(req.Context())
	if err != nil {
		error := err.Error()
		v.Error = &error
	}
	v.Events = events
	return httperror.JSONResponse{V: v}
}
