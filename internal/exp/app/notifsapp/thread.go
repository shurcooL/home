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
)

// renderThreadBodyInnerHTML renders the inner HTML of
// the <body> element of the notifications thread view.
// It's safe for concurrent use.
func renderThreadBodyInnerHTML(ctx context.Context, w io.Writer, all bool, notificationService notification.Service, bodyTop template.HTML) error {
	notifs, err := notificationService.ListNotifications(ctx, notification.ListOptions{
		All: all,
	})
	if err != nil {
		return fmt.Errorf("notificationService.ListNotifications: %v", err)
	}

	_, err = io.WriteString(w, bodyPre)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, string(bodyTop))
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

	_, err = io.WriteString(w, bodyPost)
	return err
}
