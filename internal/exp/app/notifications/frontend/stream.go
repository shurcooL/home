// +build js,wasm

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"

	homecomponent "github.com/shurcooL/home/component"
	notifcomponent "github.com/shurcooL/home/internal/exp/app/notifications/component"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
)

// renderStreamBodyInnerHTML renders the inner HTML of
// the <body> element of the notifications stream view.
// It's safe for concurrent use.
func renderStreamBodyInnerHTML(ctx context.Context, w io.Writer, reqURL *url.URL, gopherbot bool, notificationService notification.Service, authenticatedUser users.User) error {
	notifs, notifsError := notificationService.ListNotifications(ctx, notification.ListOptions{
		All: true,
	})
	var error string
	if notifsError != nil {
		error = "There was a problem getting latest notifications."
		if authenticatedUser.SiteAdmin {
			error += "\n\n" + notifsError.Error()
		}
	}

	_, err := io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	var nc uint64
	if authenticatedUser.UserSpec != (users.UserSpec{}) {
		nc, err = notificationService.CountNotifications(ctx)
		if err != nil {
			log.Println("notificationService.CountNotifications:", err)
		}
	}

	// Render the header.
	header := homecomponent.Header{
		CurrentUser:       authenticatedUser,
		NotificationCount: nc,
		ReturnURL:         reqURL.String(),
	}
	err = htmlg.RenderComponents(w, header)
	if err != nil {
		return err
	}

	// Render the tabnav.
	err = htmlg.RenderComponents(w, notificationTabnav(streamTab))
	if err != nil {
		return fmt.Errorf("htmlg.RenderComponents: %v", err)
	}

	_, err = io.WriteString(w, `<div class="showNotifications"><label><input id="show" type="checkbox" checked>Show Notifications</label></div>`)
	if err != nil {
		return err
	}

	// Render the notification stream.
	err = htmlg.RenderComponents(w, notifcomponent.Stream{
		Notifications: notifs,
		Error:         error,
		GopherBot:     gopherbot,
	})
	if err != nil {
		return fmt.Errorf("htmlg.RenderComponents: %v", err)
	}

	_, err = io.WriteString(w, `</div>`)
	return err
}
