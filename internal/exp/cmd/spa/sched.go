// +build js,wasm,go1.14

package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"syscall/js"

	"github.com/shurcooL/home/internal/exp/spa"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
)

var (
	openCh = make(chan openRequest, 8)

	rendReqCh  = make(chan renderRequest, 8)
	rendRespCh = make(chan renderResponse)
)

type openRequest struct {
	URL       *url.URL
	PushState bool
	SetupOnly bool
}

type renderRequest struct {
	Ctx       context.Context
	URL       *url.URL
	PushState bool
	SetupOnly bool
}

type renderResponse struct {
	Req   renderRequest
	State interface{}
	Body  string
	Error error
}

func scheduler(userService users.Service) {
	// TODO: Think about best place for this.
	//       On backend, it needs to be done for each request (could be serving different users).
	//       On frontend, it needs to be done only once (given a user can't sign in or out completely on frontend).
	//       Can optimize the frontend query by embedding information in the HTML (like RedLogo).
	authenticatedUser, err := userService.GetAuthenticatedSpec(context.Background())
	if err != nil {
		log.Println(err)
		authenticatedUser = users.UserSpec{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}

	go renderer()

	var prevCancel context.CancelFunc

	for {
		select {
		case req := <-openCh:
			if prevCancel != nil {
				prevCancel()
			}
			var ctx context.Context
			ctx, prevCancel = context.WithCancel(context.Background())
			rendReqCh <- renderRequest{ctx, req.URL, req.PushState, req.SetupOnly}

		case resp := <-rendRespCh:
			// TODO: Is there a better place to factor out the os.IsPermission(err) && u == nil check?
			if os.IsPermission(resp.Error) && authenticatedUser == (users.UserSpec{}) {
				loginURL := (&url.URL{
					Path:     "/login",
					RawQuery: url.Values{"return": {resp.Req.URL.String()}}.Encode(),
				}).String()
				handleError(httperror.Redirect{URL: loginURL})
				continue // Skip PushState. TODO: Do it better.
			}
			if resp.Error != nil {
				// TODO: Factor out error handling into renderer?
				handleError(resp.Error)
			} else if resp.Req.SetupOnly {
				// TODO: Better place/way.
				app.SetupPage(resp.Req.Ctx, resp.State)
			} else {
				js.Global().Get("document").Get("body").Set("innerHTML", resp.Body)
				app.SetupPage(resp.Req.Ctx, resp.State)
			}
			if resp.Req.PushState {
				// TODO: dom.GetWindow().History().PushState(...)
				// TODO: Use existing dom.GetWindow().Location().Search, just change "tab" query.
				// #TODO: If query.Encode() is blank, don't include "?" prefix. Hmm, apparently I might not be able to do it here because History.PushState interprets that as doing nothing... Or maybe if I specifully absolute path.
				// TODO: Verify the "." thing works in general case, e.g., for files, different subfolders, etc.
				js.Global().Get("window").Get("history").Call("pushState", nil, nil, resp.Req.URL.String())
			}
			recordPageView(resp.Req.URL)
		}
	}
}

func renderer() {
	var buf bytes.Buffer

	for {
		req := <-rendReqCh
		if req.Ctx.Err() != nil {
			continue
		}

		if req.SetupOnly {
			// TODO: Refactor so there isn't a need to
			//       render HTML into a discard writer.
			st, err := app.ServePage(req.Ctx, ioutil.Discard, req.URL)
			if req.Ctx.Err() != nil {
				continue
			}
			rendRespCh <- renderResponse{
				Req:   req,
				State: st,
				Error: err,
			}
			continue
		}

		buf.Reset()
		st, err := app.ServePage(req.Ctx, &buf, req.URL)
		if req.Ctx.Err() != nil {
			continue
		}
		rendRespCh <- renderResponse{
			Req:   req,
			State: st,
			Body:  buf.String(),
			Error: err,
		}
	}
}

// handleError handles error err, which must be non-nil.
func handleError(err error) {
	if e, ok := spa.IsOutOfScope(err); ok {
		js.Global().Get("location").Set("href", e.URL.String())
		select {} // TODO, THINK: Decide whether pausing execution here is the best thing to do.
	}
	if err, ok := httperror.IsRedirect(err); ok {
		u, _ := url.Parse(err.URL)
		openCh <- openRequest{URL: u} // Redirect.
		return
	}
	if err, ok := httperror.IsBadRequest(err); ok {
		error := "400 Bad Request\n\n" + err.Error()
		js.Global().Get("document").Get("body").Set("innerHTML", `<pre style="word-wrap: break-word; white-space: pre-wrap;">`+html.EscapeString(error)+"</pre>")
		return
	}
	if os.IsNotExist(err) {
		log.Println(err)
		error := "404 Not Found\n\n" + err.Error()
		js.Global().Get("document").Get("body").Set("innerHTML", `<pre style="word-wrap: break-word; white-space: pre-wrap;">`+html.EscapeString(error)+"</pre>")
		return
	}
	if os.IsPermission(err) {
		// TODO: Factor in a GetAuthenticatedSpec.ID == 0 check out here, maybe?
		log.Println(err)
		error := "403 Forbidden\n\n" + err.Error()
		js.Global().Get("document").Get("body").Set("innerHTML", `<pre style="word-wrap: break-word; white-space: pre-wrap;">`+html.EscapeString(error)+"</pre>")
		return
	}

	log.Println(err)
	error := "500 Internal Server Error\n\n" + err.Error()
	js.Global().Get("document").Get("body").Set("innerHTML", `<pre style="word-wrap: break-word; white-space: pre-wrap;">`+html.EscapeString(error)+"</pre>")
}

// recordPageView records a virtual page view using
// the global "gtag" object, if one is defined.
// It uses the global "GAID" value as the property ID.
//
// See https://developers.google.com/analytics/devguides/collection/gtagjs/single-page-applications#measure_virtual_pageviews.
func recordPageView(reqURL *url.URL) {
	gtag := js.Global().Get("gtag")
	if gtag.IsUndefined() {
		return
	}
	gtag.Invoke("config", js.Global().Get("GAID"), map[string]interface{}{"page_path": reqURL.String()})
}
