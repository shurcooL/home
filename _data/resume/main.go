// resume renders the resume page entirely on the frontend.
// It is a Go package meant to be compiled with GOARCH=js
// and executed in a browser, where the DOM is available.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/shurcooL/frontend/reactionsmenu"
	homehttp "github.com/shurcooL/home/http"
	"github.com/shurcooL/home/resume"
	"github.com/shurcooL/notificationsapp/httpclient"
	"github.com/shurcooL/users"
	"golang.org/x/oauth2"
	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

func main() {
	switch readyState := document.ReadyState(); readyState {
	case "loading":
		document.AddEventListener("DOMContentLoaded", false, func(dom.Event) {
			go setup(context.Background())
		})
	case "interactive", "complete":
		setup(context.Background())
	default:
		panic(fmt.Errorf("internal error: unexpected document.ReadyState value: %v", readyState))
	}
}

func setup(ctx context.Context) {
	reactionsService := homehttp.Reactions{}
	authenticatedUser, err := homehttp.Users{}.GetAuthenticated(ctx)
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}

	query, _ := url.ParseQuery(strings.TrimPrefix(dom.GetWindow().Location().Search, "?"))
	prerender, _ := strconv.ParseBool(query.Get("prerender"))
	if !prerender {
		httpClient := httpClient()

		shurcool, err := homehttp.Users{}.Get(ctx, shurcool)
		if err != nil {
			log.Println(err)
			return
		}
		notificationsService := httpclient.NewNotifications(httpClient, "", "")
		returnURL := dom.GetWindow().Location().Pathname + dom.GetWindow().Location().Search

		var buf bytes.Buffer
		err = resume.RenderBodyInnerHTML(ctx, &buf, shurcool, reactionsService, notificationsService, authenticatedUser, returnURL)
		if err != nil {
			log.Println(err)
			return
		}
		document.Body().SetInnerHTML(buf.String())
	}

	reactionsmenu.Setup(resume.ReactableURL, reactionsService, authenticatedUser)
}

var shurcool = users.UserSpec{ID: 1924134, Domain: "github.com"}

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
