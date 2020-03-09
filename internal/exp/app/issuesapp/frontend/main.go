// frontend script for issuesapp.
//
// It's a Go package meant to be compiled with GOARCH=js
// and executed in a browser, where the DOM is available.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/frontend/reactionsmenu"
	"github.com/shurcooL/frontend/tabsupport"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go/gopherjs_http/jsutil"
	"github.com/shurcooL/home/internal/exp/app/issuesapp/common"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/home/internal/exp/service/issue/httpclient"
	"github.com/shurcooL/markdownfmt/markdown"
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

	f := &frontend{is: httpclient.NewIssues(httpClient, "", "")}

	js.Global.Set("MarkdownPreview", jsutil.Wrap(MarkdownPreview))
	js.Global.Set("SwitchWriteTab", jsutil.Wrap(SwitchWriteTab))
	js.Global.Set("PasteHandler", jsutil.Wrap(PasteHandler))
	js.Global.Set("CreateNewIssue", f.CreateNewIssue)
	js.Global.Set("ToggleIssueState", ToggleIssueState)
	js.Global.Set("PostComment", PostComment)
	js.Global.Set("EditComment", jsutil.Wrap(f.EditComment))
	js.Global.Set("TabSupportKeyDownHandler", jsutil.Wrap(tabsupport.KeyDownHandler))

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
	setupIssueToggleButton()
	setupScroll()

	if createIssueButton, ok := document.GetElementByID("create-issue-button").(dom.HTMLElement); ok {
		titleEditor := document.GetElementByID("title-editor").(*dom.HTMLInputElement)
		titleEditor.AddEventListener("input", false, func(_ dom.Event) {
			if strings.TrimSpace(titleEditor.Value) == "" {
				createIssueButton.SetAttribute("disabled", "disabled")
			} else {
				createIssueButton.RemoveAttribute("disabled")
			}
		})
	}

	if !state.DisableReactions {
		reactionsService := IssuesReactions{Issues: f.is}
		reactionsmenu.Setup(state.RepoSpec.URI, reactionsService, state.CurrentUser)
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
	is issues.Service
}

func setupIssueToggleButton() {
	if issueToggleButton := document.GetElementByID("issue-toggle-button"); issueToggleButton != nil {
		commentEditor := document.QuerySelector("#new-comment-container .comment-editor").(*dom.HTMLTextAreaElement)
		commentEditor.AddEventListener("input", false, func(_ dom.Event) {
			if strings.TrimSpace(commentEditor.Value) == "" {
				issueToggleButton.SetTextContent(issueToggleButton.GetAttribute("data-1-action"))
			} else {
				issueToggleButton.SetTextContent(issueToggleButton.GetAttribute("data-2-actions"))
			}
		})
	}
}

func (f *frontend) CreateNewIssue() {
	titleEditor := document.GetElementByID("title-editor").(*dom.HTMLInputElement)
	commentEditor := document.QuerySelector(".comment-editor").(*dom.HTMLTextAreaElement)

	title := strings.TrimSpace(titleEditor.Value)
	if title == "" {
		log.Println("cannot create issue with empty title")
		return
	}
	fmted, _ := markdown.Process("", []byte(commentEditor.Value), nil)
	newIssue := issues.Issue{
		Title: title,
		Comment: issues.Comment{
			Body: string(bytes.TrimSpace(fmted)),
		},
	}

	go func() {
		issue, err := f.is.Create(context.Background(), state.RepoSpec, newIssue)
		if err != nil {
			// TODO: Display error in the UI, so it is more visible.
			log.Println("creating issue failed:", err)
			return
		}

		// Redirect.
		dom.GetWindow().Location().Href = fmt.Sprintf("%s/%d", state.BaseURI, issue.ID)
	}()
}

func ToggleIssueState(issueState issues.State) {
	go func() {
		// Post comment first if there's text entered, and we're closing.
		if strings.TrimSpace(document.QuerySelector("#new-comment-container .comment-editor").(*dom.HTMLTextAreaElement).Value) != "" &&
			issueState == issues.ClosedState {
			err := postComment()
			if err != nil {
				log.Println(err)
				return
			}
		}

		ir := issues.IssueRequest{
			State: &issueState,
		}
		value, err := json.Marshal(ir)
		if err != nil {
			panic(err)
		}

		resp, err := http.PostForm(state.BaseURI+state.ReqPath+"/edit", url.Values{"value": {string(value)}})
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		data, err := url.ParseQuery(string(body))
		if err != nil {
			log.Println(err)
			return
		}

		switch resp.StatusCode {
		case http.StatusOK:
			issueStateBadge := document.GetElementByID("issue-state-badge")
			issueStateBadge.SetInnerHTML(data.Get("issue-state-badge"))

			issueToggleButton := document.GetElementByID("issue-toggle-button")
			issueToggleButton.SetOuterHTML(data.Get("issue-toggle-button"))
			setupIssueToggleButton()

			for _, newEventData := range data["new-event"] {
				// Create event.
				newEvent := document.CreateElement("div").(*dom.HTMLDivElement)
				newItemMarker := document.GetElementByID("new-item-marker")
				newItemMarker.ParentNode().InsertBefore(newEvent, newItemMarker)
				newEvent.SetOuterHTML(newEventData)
			}
		}

		// Post comment after if there's text entered, and we're reopening.
		if strings.TrimSpace(document.QuerySelector("#new-comment-container .comment-editor").(*dom.HTMLTextAreaElement).Value) != "" &&
			issueState == issues.OpenState {
			err := postComment()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()
}

func PostComment() {
	go func() {
		err := postComment()
		if err != nil {
			log.Println(err)
		}
	}()
}

// postComment posts the comment to the remote API.
func postComment() error {
	commentEditor := document.QuerySelector("#new-comment-container .comment-editor").(*dom.HTMLTextAreaElement)

	fmted, _ := markdown.Process("", []byte(commentEditor.Value), nil)
	if len(fmted) == 0 {
		return fmt.Errorf("cannot post empty comment")
	}
	value := string(bytes.TrimSpace(fmted))

	resp, err := http.PostForm(state.BaseURI+state.ReqPath+"/comment", url.Values{"value": {value}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("got reply: %v\n", resp.Status)

	switch resp.StatusCode {
	case http.StatusOK:
		// Create comment.
		newComment := document.CreateElement("div").(*dom.HTMLDivElement)

		newItemMarker := document.GetElementByID("new-item-marker")
		newItemMarker.ParentNode().InsertBefore(newComment, newItemMarker)

		newComment.SetOuterHTML(string(body))

		// Reset new-comment component.
		commentEditor.Value = ""
		commentEditor.Underlying().Call("dispatchEvent", js.Global.Get("CustomEvent").New("input")) // Trigger "input" event listeners.
		switchWriteTab(document.GetElementByID("new-comment-container"), commentEditor)

		return nil
	default:
		return fmt.Errorf("did not get acceptable status code: %v", resp.Status)
	}
}

func MarkdownPreview(this dom.HTMLElement) {
	container := getAncestorByClassName(this, "edit-container")

	if container.QuerySelector(".preview-tab-link").(dom.Element).Class().Contains("active") {
		return
	}

	commentEditor := container.QuerySelector(".comment-editor").(*dom.HTMLTextAreaElement)
	commentPreview := container.QuerySelector(".comment-preview").(*dom.HTMLDivElement)

	fmted, _ := markdown.Process("", []byte(commentEditor.Value), nil)
	value := bytes.TrimSpace(fmted)

	if len(value) != 0 {
		commentPreview.SetInnerHTML(string(github_flavored_markdown.Markdown(value)))
	} else {
		commentPreview.SetInnerHTML(`<i class="gray">Nothing to preview.</i>`)
	}

	container.QuerySelector(".write-tab-link").(dom.Element).Class().Remove("active")
	container.QuerySelector(".preview-tab-link").(dom.Element).Class().Add("active")
	commentEditor.Style().SetProperty("display", "none", "")
	commentPreview.Style().SetProperty("display", "block", "")
}

func SwitchWriteTab(this dom.HTMLElement) {
	container := getAncestorByClassName(this, "edit-container")
	commentEditor := container.QuerySelector(".comment-editor").(*dom.HTMLTextAreaElement)
	switchWriteTab(container, commentEditor)
}

func switchWriteTab(container dom.Element, commentEditor *dom.HTMLTextAreaElement) {
	if container.QuerySelector(".preview-tab-link").(dom.Element).Class().Contains("active") {
		commentPreview := container.QuerySelector(".comment-preview").(*dom.HTMLDivElement)

		container.QuerySelector(".write-tab-link").(dom.Element).Class().Add("active")
		container.QuerySelector(".preview-tab-link").(dom.Element).Class().Remove("active")
		commentEditor.Style().SetProperty("display", "block", "")
		commentPreview.Style().SetProperty("display", "none", "")
	}

	commentEditor.Focus()
}
