// frontend script for changesapp.
//
// It's a Go package meant to be compiled with GOARCH=js
// and executed in a browser, where the DOM is available.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/frontend/reactionsmenu"
	"github.com/shurcooL/go/gopherjs_http/jsutil"
	"github.com/shurcooL/home/internal/exp/app/changesapp/common"
	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/home/internal/exp/service/change/httpclient"
	"golang.org/x/oauth2"
	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

var state common.State

func main() {
	stateJSON := js.Global.Get("State").String()
	err := json.Unmarshal([]byte(stateJSON), &state)
	if err != nil {
		panic(err)
	}

	httpClient := httpClient()

	f := &frontend{cs: httpclient.NewChange(httpClient, "", "", "/api/change")}

	js.Global.Set("ToggleDetails", jsutil.Wrap(ToggleDetails))

	switch readyState := document.ReadyState(); readyState {
	case "loading":
		document.AddEventListener("DOMContentLoaded", false, func(dom.Event) {
			go setup(f)
		})
	case "interactive", "complete":
		setup(f)
	default:
		panic(fmt.Errorf("internal error: unexpected document.ReadyState value: %v", readyState))
	}
}

func setup(f *frontend) {
	setupScroll()

	if !state.DisableReactions {
		reactionsService := ChangeReactions{Change: f.cs}
		reactionsmenu.Setup(state.RepoSpec, reactionsService, state.CurrentUser)
	}
}

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

type frontend struct {
	cs change.Service
}

func ToggleDetails(el dom.HTMLElement) {
	container := getAncestorByClassName(el, "commit-container").(dom.HTMLElement)
	details := container.QuerySelector("pre.commit-details").(dom.HTMLElement)

	switch details.Style().GetPropertyValue("display") {
	default:
		details.Style().SetProperty("display", "none", "")
	case "none":
		details.Style().SetProperty("display", "block", "")
	}
}

func getAncestorByClassName(el dom.Element, class string) dom.Element {
	for ; el != nil && !el.Class().Contains(class); el = el.ParentElement() {
	}
	return el
}
