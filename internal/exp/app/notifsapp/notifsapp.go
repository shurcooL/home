// +build go1.14

// Package notifsapp is a notification tracking web app.
package notifsapp

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/shurcooL/home/internal/exp/service/notification"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/users"
)

func New(ns notification.Service, us users.Service, opt Options) *app {
	return &app{
		ns:  ns,
		us:  us,
		opt: opt,
	}
}

// Options for configuring notifsapp.
type Options struct {
	// BodyTop provides components to include at the top of the <body> element. It can be nil.
	BodyTop func(context.Context, State) ([]htmlg.Component, error)
}

type State struct {
	ReqURL      *url.URL
	CurrentUser users.User
}

func (s State) RequestURL() *url.URL { return s.ReqURL }

type app struct {
	ns notification.Service
	us users.Service

	opt Options
}

func (a *app) ServePage(ctx context.Context, w io.Writer, reqURL *url.URL) (interface{}, error) {
	// TODO: Think about best place for this.
	//       On backend, it needs to be done for each request (could be serving different users).
	//       On frontend, it needs to be done only once (given a user can't sign in or out completely on frontend).
	//       Can optimize the frontend query by embedding information in the HTML (like RedLogo).
	authenticatedUser, err := a.us.GetAuthenticated(ctx)
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}

	// TODO: Think about best place?
	if authenticatedUser.UserSpec == (users.UserSpec{}) {
		return nil, os.ErrPermission
	}

	st := State{
		ReqURL:      reqURL,
		CurrentUser: authenticatedUser,
	}

	// TODO: Factor out the prefix maybe?
	route := reqURL.Path[len("/notificationsv2"):]
	if route == "" {
		route = "/"
	}

	switch route {
	case "/":
		return st, a.serveStream(ctx, w, st)
	case "/threads":
		return st, a.serveThread(ctx, w, st)
	default:
		return nil, os.ErrNotExist
	}
}

func (a *app) serveStream(ctx context.Context, w io.Writer, st State) error {
	gopherbot, _ := strconv.ParseBool(st.ReqURL.Query().Get("gopherbot"))

	// TODO: Is it okay to write title in <body>?
	_, err := io.WriteString(w, `<title>Notifications - Stream</title>`)
	if err != nil {
		return err
	}

	// TODO: {{.BaseURL}} ...
	_, err = io.WriteString(w, `<link href="/assets/notifications/stream.css" rel="stylesheet" type="text/css">`)
	if err != nil {
		return err
	}

	bodyTop, err := a.bodyTop(ctx, st)
	if err != nil {
		return fmt.Errorf("bodyTop: %w", err)
	}

	// Initial page render.
	err = renderStreamBodyInnerHTML(ctx, w, gopherbot, a.ns, st.CurrentUser, bodyTop)
	return err
}

func (a *app) serveThread(ctx context.Context, w io.Writer, st State) error {
	all, _ := strconv.ParseBool(st.ReqURL.Query().Get("all"))

	// TODO: Is it okay to write title in <body>?
	_, err := io.WriteString(w, `<title>Notifications - Threads</title>`)
	if err != nil {
		return err
	}

	// TODO: {{.BaseURL}} ...
	_, err = io.WriteString(w, `<link href="/assets/notifications/thread.css" rel="stylesheet" type="text/css">`)
	if err != nil {
		return err
	}

	bodyTop, err := a.bodyTop(ctx, st)
	if err != nil {
		return fmt.Errorf("bodyTop: %w", err)
	}

	// Initial page render.
	err = renderThreadBodyInnerHTML(ctx, w, all, a.ns, bodyTop)
	return err
}

func (a *app) bodyTop(ctx context.Context, st State) (template.HTML, error) {
	if a.opt.BodyTop == nil {
		return "", nil
	}
	c, err := a.opt.BodyTop(ctx, st)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = htmlg.RenderComponents(&buf, c...)
	if err != nil {
		return "", fmt.Errorf("htmlg.RenderComponents: %v", err)
	}
	return template.HTML(buf.String()), nil
}

const (
	bodyPre  = `<div style="max-width: 800px; margin: 0 auto 100px auto;">`
	bodyPost = `</div>`
)
