// Package resume contains functionality for rendering /resume page.
package resume

import (
	"context"
	"io"
	"log"

	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/resume"
	"github.com/shurcooL/users"
)

// ReactableURL is the URL for reactionable items on this resume.
const ReactableURL = "dmitri.shuralyov.com/resume"

// RenderBodyInnerHTML renders the inner HTML of the <body> element of the page that displays the resume.
// It's safe for concurrent use.
func RenderBodyInnerHTML(ctx context.Context, w io.Writer, shurcool users.User, reactionsService reactions.Service, notifications notifications.Service, authenticatedUser users.User, returnURL string) error {
	var nc uint64
	if authenticatedUser.ID != 0 {
		var err error
		nc, err = notifications.Count(ctx, nil)
		if err != nil {
			// THINK: Should it be a fatal error or not? What about on frontend vs backend?
			log.Println(err)
			nc = 0
		}
	}

	reactions, err := reactionsService.List(ctx, ReactableURL)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	// Render the header.
	header := homecomponent.Header{
		CurrentUser:       authenticatedUser,
		NotificationCount: nc,
		ReturnURL:         returnURL,
	}
	err = htmlg.RenderComponents(w, header)
	if err != nil {
		return err
	}

	// Render the resume contents.
	resume := resume.DmitriShuralyov(shurcool, reactions, authenticatedUser)
	err = htmlg.RenderComponents(w, resume)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, `</div>`)
	return err
}
