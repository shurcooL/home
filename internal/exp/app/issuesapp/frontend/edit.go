// +build js,wasm,go1.14

package main

import (
	"bytes"
	"context"
	"log"
	"strconv"

	"github.com/shurcooL/github_flavored_markdown"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/markdownfmt/markdown"
	"honnef.co/go/js/dom/v2"
)

func (f *frontend) EditComment(action string, this dom.HTMLElement, evt dom.Event) {
	if evt.DefaultPrevented() {
		return
	}

	container := getAncestorByClassName(this, "comment-edit-container")
	// HACK: Currently the child nodes are [text, div, text, div, text], but that isn't reliable.
	commentView := container.ChildNodes()[1].(dom.HTMLElement)
	editView := container.ChildNodes()[3].(dom.HTMLElement)
	commentEditor := editView.QuerySelector(".comment-editor").(*dom.HTMLTextAreaElement)

	switch action {
	case "edit":
		commentEditor.SetValue(commentEditor.GetAttribute("data-raw"))

		commentView.Style().SetProperty("display", "none", "")
		editView.Style().SetProperty("display", "block", "")

		commentEditor.Focus()
	case "cancel", "update":
		switch action {
		case "cancel":
			if commentEditor.Value() != commentEditor.GetAttribute("data-raw") {
				if !dom.GetWindow().Confirm("Are you sure you want to discard your unsaved changes?") {
					return
				}
			}
			commentEditor.SetValue(commentEditor.GetAttribute("data-raw"))
		case "update":
			if commentEditor.Value() != commentEditor.GetAttribute("data-raw") {
				fmted, _ := markdown.Process("", []byte(commentEditor.Value()), nil)
				fmted = bytes.TrimSpace(fmted)
				if len(fmted) == 0 {
					// Empty body isn't allowed.
					// TODO: Unless it's an issue description (initial comment).
					// TODO: Display error? Disable "Update comment" button?
					return
				}
				commentID, err := strconv.ParseUint(commentEditor.GetAttribute("data-id"), 10, 64)
				if err != nil {
					panic(err)
				}

				go func() {
					body := string(fmted)
					cr := issues.CommentRequest{
						ID:   commentID,
						Body: &body,
					}
					_, err := f.is.EditComment(context.Background(), state.RepoSpec, state.IssueID, cr)
					if err != nil {
						// TODO: Handle failure more visibly in the UI.
						log.Println("EditComment:", err)
					}
				}()

				commentEditor.SetAttribute("data-raw", string(fmted))
				markdownBody := commentView.QuerySelector(".markdown-body").(*dom.HTMLDivElement)
				markdownBody.SetInnerHTML(string(github_flavored_markdown.Markdown(fmted)))
			}
		}

		commentView.Style().SetProperty("display", "block", "")
		editView.Style().SetProperty("display", "none", "")

		// TODO: switchWriteTab() (maybe without commentEditor.Focus() part).
		// TODO: Maybe without commentEditor.Focus() part?
		switchWriteTab(container, commentEditor)
	}
}

func getAncestorByClassName(el dom.Element, class string) dom.Element {
	for ; el != nil && !el.Class().Contains(class); el = el.ParentElement() {
	}
	return el
}
