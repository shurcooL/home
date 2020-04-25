// Package httphandler contains an API handler for notification.Service.
package httphandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/httperror"
)

// Notification is an API handler for notification.Service.
// It returns errors compatible with httperror package.
type Notification struct {
	Notification notification.Service
}

func (h Notification) ListNotifications(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return httperror.Method{Allowed: []string{http.MethodGet}}
	}
	var opt notification.ListOptions // TODO: Automate this conversion process.
	opt.Namespace = req.URL.Query().Get("Namespace")
	opt.All, _ = strconv.ParseBool(req.URL.Query().Get("All"))
	notifs, err := h.Notification.ListNotifications(req.Context(), opt)
	return httperror.JSONResponse{V: struct {
		Notifs []notification.Notification
		Error  errorJSON
	}{notifs, errorJSON{err}}}
}

func (h Notification) StreamNotifications(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return httperror.Method{Allowed: []string{http.MethodGet}}
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("http.ResponseWriter %T is not a http.Flusher", w)
	}
	ch := make(chan []notification.Notification, 8)
	err := h.Notification.StreamNotifications(req.Context(), ch)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	for {
		select {
		case <-req.Context().Done():
			return req.Context().Err()
		case notifs := <-ch:
			err := enc.Encode(notifs)
			if err != nil {
				return err
			}
			fl.Flush()
		}
	}
}

func (h Notification) CountNotifications(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodGet {
		return httperror.Method{Allowed: []string{http.MethodGet}}
	}
	c, err := h.Notification.CountNotifications(req.Context())
	return httperror.JSONResponse{V: struct {
		Count uint64
		Error errorJSON
	}{c, errorJSON{err}}}
}

func (h Notification) MarkNotificationRead(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodPost {
		return httperror.Method{Allowed: []string{http.MethodPost}}
	}
	q := req.URL.Query() // TODO: Automate this conversion process.
	namespace := q.Get("Namespace")
	threadType := q.Get("ThreadType")
	threadID, err := strconv.ParseUint(q.Get("ThreadID"), 10, 64)
	if err != nil {
		return httperror.BadRequest{Err: fmt.Errorf("parsing ThreadID query parameter: %v", err)}
	}
	err = h.Notification.MarkNotificationRead(req.Context(), namespace, threadType, threadID)
	return err
}

// errorJSON marshals an error value into JSON.
//
// A nil Err value is encoded as the null JSON value, otherwise
// the string returned by the Error method is encoded as a JSON string.
type errorJSON struct {
	Err error
}

// MarshalJSON implements the json.Marshaler interface.
func (e errorJSON) MarshalJSON() ([]byte, error) {
	if e.Err == nil {
		return []byte("null"), nil
	}
	return json.Marshal(e.Err.Error())
}
