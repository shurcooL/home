// idiomaticgo sets up reactions menu for /idiomatic-go page.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/shurcooL/frontend/reactionsmenu"
	homehttp "github.com/shurcooL/home/http"
	"github.com/shurcooL/home/internal/page/idiomaticgo"
	"github.com/shurcooL/issuesapp/httpclient"
	"github.com/shurcooL/users"
	"golang.org/x/oauth2"
	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

func main() {
	switch readyState := document.ReadyState(); readyState {
	case "loading":
		document.AddEventListener("DOMContentLoaded", false, func(dom.Event) {
			go setup()
		})
	case "interactive", "complete":
		setup()
	default:
		panic(fmt.Errorf("internal error: unexpected document.ReadyState value: %v", readyState))
	}
}

func setup() {
	httpClient := httpClient()

	issuesService := httpclient.NewIssues(httpClient, "", "")
	reactionsService := idiomaticgo.IssuesReactions{Issues: issuesService}
	authenticatedUser, err := homehttp.Users{}.GetAuthenticated(context.TODO())
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}

	// THINK: Consider rendering on frontend.
	/*if !document.Body().HasChildNodes() {
		var buf bytes.Buffer
		returnURL := dom.GetWindow().Location().Pathname + dom.GetWindow().Location().Search
		err = idiomaticgo.RenderBodyInnerHTML(context.TODO(), &buf, issuesService, http.Notifications{}, authenticatedUser, returnURL)
		if err != nil {
			log.Println(err)
			return
		}
		document.Body().SetInnerHTML(buf.String())
	}*/

	reactionsmenu.Setup(idiomaticgo.ReactableURL, reactionsService, authenticatedUser)
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
