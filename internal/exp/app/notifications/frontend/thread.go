// +build js,wasm

package main

import (
	"context"
	"fmt"
	"io"
	"net/url"

	homecomponent "github.com/shurcooL/home/component"
	notifcomponent "github.com/shurcooL/home/internal/exp/app/notifications/component"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
)

// renderThreadBodyInnerHTML renders the inner HTML of
// the <body> element of the notifications thread view.
// It's safe for concurrent use.
func renderThreadBodyInnerHTML(ctx context.Context, w io.Writer, reqURL *url.URL, all bool, notificationService notification.Service, authenticatedUser users.User) error {
	notifs, err := notificationService.ListNotifications(ctx, notification.ListOptions{
		All: all,
	})
	if err != nil {
		return fmt.Errorf("notificationService.ListNotifications: %v", err)
	}

	_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	var nc uint64
	if authenticatedUser.UserSpec != (users.UserSpec{}) {
		nc, err = notificationService.CountNotifications(ctx)
		if err != nil {
			return err
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
	err = htmlg.RenderComponents(w, notificationTabnav(threadTab))
	if err != nil {
		return fmt.Errorf("htmlg.RenderComponents: %v", err)
	}

	// Render the notifications contents.
	err = htmlg.RenderComponents(w, notifcomponent.NotificationsByRepo{Notifications: notifs})
	if err != nil {
		return fmt.Errorf("htmlg.RenderComponents: %v", err)
	}

	_, err = io.WriteString(w, `</div>`)
	return err
}
