// +build go1.14

// Package spa implements a single-page application
// used on the dmitri.shuralyov.com website.
//
// It is capable of
// serving page HTML on the frontend and backend, and
// setting page state on the frontend.
package spa

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/exp/app/notifsapp"
	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
)

// An App is a single-page application.
type App interface {
	// ServePage renders the page HTML for reqURL to w,
	// and returns the state it computed.
	//
	// It returns an error of OutOfScopeError type
	// if reqURL is out of scope for the app.
	//
	// The returned state must implement the PageState interface.
	ServePage(ctx context.Context, w io.Writer, reqURL *url.URL) (interface{}, error)

	// SetupPage sets up the frontend page state,
	// using state returned from ServePage.
	SetupPage(ctx context.Context, state interface{})
}

type PageState interface {
	// RequestURL returns the request URL for the page.
	RequestURL() *url.URL
}

func NewApp(
	notifService notification.Service,
	userService users.Service,
	redirect func(*url.URL), // Only needed on frontend.
) *app {
	notifsApp := notifsapp.New(
		notifService,
		userService,
		notifsapp.Options{
			BodyTop: func(ctx context.Context, st notifsapp.State) ([]htmlg.Component, error) {
				var nc uint64
				if st.CurrentUser.UserSpec != (users.UserSpec{}) {
					var err error
					nc, err = notifService.CountNotifications(ctx)
					if err != nil {
						log.Println("notifService.CountNotifications:", err)
					}
				}
				header := homecomponent.Header{
					CurrentUser:       st.CurrentUser,
					NotificationCount: nc,
					ReturnURL:         st.ReqURL.String(),
				}
				return []htmlg.Component{header}, nil
			},
		},
	)
	return &app{
		NotifsApp: notifsApp,
	}
}

type app struct {
	NotifsApp App
}

func (a *app) ServePage(ctx context.Context, w io.Writer, reqURL *url.URL) (interface{}, error) {
	switch {
	case reqURL.Path == "/notificationsv2", strings.HasPrefix(reqURL.Path, "/notificationsv2/"):
		return a.NotifsApp.ServePage(ctx, w, reqURL)
	default:
		return nil, OutOfScopeError{URL: reqURL}
	}
}

func (a *app) SetupPage(ctx context.Context, state interface{}) {
	// TODO: Make this safer and better.
	switch reqURL := state.(PageState).RequestURL(); {
	case reqURL.Path == "/notificationsv2", strings.HasPrefix(reqURL.Path, "/notificationsv2/"):
		a.NotifsApp.SetupPage(ctx, state)
	}
}

// OutOfScopeError is an error returned when the requested page
// is out of scope for the single-page application and therefore
// cannot be served by it directly.
type OutOfScopeError struct {
	// URL is the URL of the requested page.
	URL *url.URL
}

func (o OutOfScopeError) Error() string { return fmt.Sprintf("%s is out of scope", o.URL) }

// IsOutOfScope reports whether err is an OutOfScopeError error.
func IsOutOfScope(err error) (OutOfScopeError, bool) {
	e, ok := err.(OutOfScopeError)
	return e, ok
}
