// blog sets up reactions menu for /blog page.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/shurcooL/frontend/reactionsmenu"
	"github.com/shurcooL/home/http"
	"github.com/shurcooL/home/idiomaticgo"
	"github.com/shurcooL/issuesapp/httpclient"
	"github.com/shurcooL/users"
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
	issuesService := httpclient.NewIssues("", "")
	reactionsService := idiomaticgo.IssuesReactions{Issues: issuesService}
	authenticatedUser, err := http.Users{}.GetAuthenticated(context.TODO())
	if err != nil {
		log.Println(err)
		authenticatedUser = users.User{} // THINK: Should it be a fatal error or not? What about on frontend vs backend?
	}

	const blogURI = "dmitri.shuralyov.com/blog"
	reactionsmenu.Setup(blogURI, reactionsService, authenticatedUser)
}
