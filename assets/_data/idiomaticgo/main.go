package main

import (
	"context"
	"fmt"
	"log"

	"github.com/shurcooL/frontend/reactionsmenu"
	"github.com/shurcooL/home/http"
	"github.com/shurcooL/home/idiomaticgo"
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
	issuesService := http.NewIssues("", "")
	reactionsService := idiomaticgo.IssuesReactions{Issues: issuesService}
	authenticatedUser, err := http.Users{}.GetAuthenticated(context.TODO())
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
