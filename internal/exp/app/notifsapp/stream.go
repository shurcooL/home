// +build go1.14

package notifsapp

import (
	"context"
	"fmt"
	"html/template"
	"io"

	notifcomponent "github.com/shurcooL/home/internal/exp/app/notifsapp/component"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
)

// renderStreamBodyInnerHTML renders the inner HTML of
// the <body> element of the notifications stream view.
// It's safe for concurrent use.
func renderStreamBodyInnerHTML(ctx context.Context, w io.Writer, gopherbot bool, notificationService notification.Service, authenticatedUser users.User, bodyTop template.HTML) error {
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

	_, err := io.WriteString(w, bodyPre)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, string(bodyTop))
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

	_, err = io.WriteString(w, bodyPost)
	return err
}
