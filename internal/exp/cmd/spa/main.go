// +build js,wasm,go1.14

package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"syscall/js"

	"github.com/shurcooL/go/gopherjs_http/jsutil/v2"
	homecomponent "github.com/shurcooL/home/component"
	homehttp "github.com/shurcooL/home/http"
	notifhttpclient "github.com/shurcooL/home/internal/exp/service/notification/httpclient"
	"github.com/shurcooL/home/internal/exp/spa"
	"golang.org/x/oauth2"
	"honnef.co/go/js/dom/v2"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

func main() {
	homecomponent.RedLogo = js.Global().Get("RedLogo").Bool()

	httpClient := httpClient()

	notifService := notifhttpclient.NewNotification(httpClient, "", "", "/api/notificationv2")
	userService := homehttp.Users{}

	redirect := func(reqURL *url.URL) { openCh <- openRequest{URL: reqURL, PushState: true} }
	app = spa.NewApp(notifService, userService, redirect)

	// Start the scheduler loop.
	go scheduler(userService)

	js.Global().Set("Open", jsutil.Wrap(func(ev dom.Event, el dom.HTMLElement) {
		if me := ev.(*dom.MouseEvent); me.CtrlKey() || me.AltKey() || me.MetaKey() || me.ShiftKey() {
			return
		}
		ev.PreventDefault()
		reqURL := &url.URL{
			Path:     el.(*dom.HTMLAnchorElement).Pathname(),
			RawQuery: strings.TrimPrefix(el.(*dom.HTMLAnchorElement).Search(), "?"),
		}
		openCh <- openRequest{URL: reqURL, PushState: true}
	}))

	dom.GetWindow().AddEventListener("popstate", false, func(dom.Event) {
		reqURL := &url.URL{
			Path:     dom.GetWindow().Location().Pathname(),
			RawQuery: strings.TrimPrefix(dom.GetWindow().Location().Search(), "?"),
			// TODO: Preserve hash.
		}
		openCh <- openRequest{URL: reqURL}
	})

	openCh <- openRequest{URL: requestURL(), SetupOnly: true}

	select {}
}

var app spa.App

// httpClient gives an *http.Client for making API requests.
func httpClient() *http.Client {
	cookies := &http.Request{Header: http.Header{"Cookie": {document.Cookie()}}}
	if accessToken, err := cookies.Cookie("accessToken"); err == nil {
		// Authenticated client.
		src := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken.Value},
		)
		return oauth2.NewClient(context.Background(), src)
	}
	// Not authenticated client.
	return http.DefaultClient
}

// requestURL returns the effective HTTP request URL of the current page.
func requestURL() *url.URL {
	u, err := url.Parse(js.Global().Get("location").Get("href").String())
	if err != nil {
		log.Fatalln(err)
	}
	u.Scheme, u.Opaque, u.User, u.Host = "", "", nil, ""
	return u
}
