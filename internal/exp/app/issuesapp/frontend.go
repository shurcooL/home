// +build js,wasm,go1.14

package issuesapp

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"strings"
	"syscall/js"

	statepkg "dmitri.shuralyov.com/state"
	"github.com/shurcooL/frontend/reactionsmenu/v2"
	"github.com/shurcooL/frontend/tabsupport/v2"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go/gopherjs_http/jsutil/v2"
	"github.com/shurcooL/home/internal/exp/app/issuesapp/component"
	issues "github.com/shurcooL/home/internal/exp/service/issue"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/markdownfmt/markdown"
	"github.com/shurcooL/reactions"
	reactionscomponent "github.com/shurcooL/reactions/component"
	"honnef.co/go/js/dom/v2"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

func (a *app) SetupPage(ctx context.Context, state interface{}) {
	as := appAndState{
		app:   a,
		State: state.(State),
	}

	js.Global().Set("MarkdownPreview", jsutil.Wrap(MarkdownPreview))
	js.Global().Set("SwitchWriteTab", jsutil.Wrap(SwitchWriteTab))
	js.Global().Set("PasteHandler", jsutil.Wrap(PasteHandler))
	js.Global().Set("CreateNewIssue", funcOf(as.CreateNewIssue))
	js.Global().Set("ToggleIssueState", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		st := statepkg.Issue(args[0].String())
		as.ToggleIssueState(st)
		return nil
	}))
	js.Global().Set("PostComment", funcOf(as.PostComment))
	js.Global().Set("EditComment", jsutil.Wrap(as.EditComment))
	js.Global().Set("TabSupportKeyDownHandler", jsutil.Wrap(tabsupport.KeyDownHandler))

	setupIssueToggleButton()
	setupScroll()

	if createIssueButton, ok := document.GetElementByID("create-issue-button").(dom.HTMLElement); ok {
		titleEditor := document.GetElementByID("title-editor").(*dom.HTMLInputElement)
		titleEditor.AddEventListener("input", false, func(_ dom.Event) {
			if strings.TrimSpace(titleEditor.Value()) == "" {
				createIssueButton.SetAttribute("disabled", "disabled")
			} else {
				createIssueButton.RemoveAttribute("disabled")
			}
		})
	}

	// TODO: Make this work better across page navigation.
	reactionsService := IssuesReactions{Issues: as.is}
	reactionsmenu.Setup(as.State.RepoSpec.URI, reactionsService, as.State.CurrentUser)
}

type appAndState struct {
	*app
	State State
}

func (a *appAndState) CreateNewIssue() {
	titleEditor := document.GetElementByID("title-editor").(*dom.HTMLInputElement)
	commentEditor := document.QuerySelector(".comment-editor").(*dom.HTMLTextAreaElement)

	title := strings.TrimSpace(titleEditor.Value())
	if title == "" {
		log.Println("cannot create issue with empty title")
		return
	}
	fmted, _ := markdown.Process("", []byte(commentEditor.Value()), nil)
	newIssue := issues.Issue{
		Title: title,
		Comment: issues.Comment{
			Body: string(bytes.TrimSpace(fmted)),
		},
	}

	go func() {
		issue, err := a.is.Create(context.Background(), a.State.RepoSpec, newIssue)
		if err != nil {
			// TODO: Display error in the UI, so it is more visible.
			log.Println("creating issue failed:", err)
			return
		}

		// Redirect.
		a.redirect(&url.URL{Path: fmt.Sprintf("%s/%d", a.State.BaseURL, issue.ID)})
	}()
}

func (a *appAndState) ToggleIssueState(issueState statepkg.Issue) {
	go func() {
		// Post comment first if there's text entered, and we're closing.
		if strings.TrimSpace(document.QuerySelector("#new-comment-container .comment-editor").(*dom.HTMLTextAreaElement).Value()) != "" &&
			issueState == statepkg.IssueClosed {
			err := a.postComment()
			if err != nil {
				log.Println(err)
				return
			}
		}

		issue, events, err := a.is.Edit(context.Background(), a.State.RepoSpec, a.State.IssueID, issues.IssueRequest{
			State: &issueState,
		})
		if err != nil {
			log.Println("a.is.Edit:", err)
			return
		}

		{
			// State badge.
			var buf bytes.Buffer
			err = htmlg.RenderComponents(&buf, component.IssueStateBadge{Issue: issue})
			if err != nil {
				log.Println(fmt.Errorf("render state badge: %v", err))
				return
			}
			document.GetElementByID("issue-state-badge").SetInnerHTML(buf.String())

			// Toggle button.
			buf.Reset()
			tt, err := t.Clone()
			if err != nil {
				log.Println(fmt.Errorf("t.Clone: %v", err))
				return
			}
			err = tt.ExecuteTemplate(&buf, "toggle-button", issue.State)
			if err != nil {
				log.Println(fmt.Errorf("render toggle button: %v", err))
				return
			}
			document.GetElementByID("issue-toggle-button").SetOuterHTML(buf.String())
			setupIssueToggleButton()

			// Events.
			for _, e := range events {
				buf.Reset()
				err := htmlg.RenderComponents(&buf, component.Event{Event: e})
				if err != nil {
					log.Println(fmt.Errorf("render event: %v", err))
					return
				}

				// Create event.
				newEvent := document.CreateElement("div").(*dom.HTMLDivElement)
				newItemMarker := document.GetElementByID("new-item-marker")
				newItemMarker.ParentNode().InsertBefore(newEvent, newItemMarker)
				newEvent.SetOuterHTML(buf.String())
			}
		}

		// Post comment after if there's text entered, and we're reopening.
		if strings.TrimSpace(document.QuerySelector("#new-comment-container .comment-editor").(*dom.HTMLTextAreaElement).Value()) != "" &&
			issueState == statepkg.IssueOpen {
			err := a.postComment()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()
}

func (a *appAndState) PostComment() {
	go func() {
		err := a.postComment()
		if err != nil {
			log.Println(err)
		}
	}()
}

// postComment posts the comment to the remote API.
func (a *appAndState) postComment() error {
	commentEditor := document.QuerySelector("#new-comment-container .comment-editor").(*dom.HTMLTextAreaElement)

	fmted, _ := markdown.Process("", []byte(commentEditor.Value()), nil)
	if len(fmted) == 0 {
		return fmt.Errorf("cannot post empty comment")
	}
	body := string(bytes.TrimSpace(fmted))

	comment := issues.Comment{
		Body: body,
	}
	comment, err := a.is.CreateComment(context.Background(), a.State.RepoSpec, a.State.IssueID, comment)
	if err != nil {
		// TODO: Handle failure more visibly in the UI.
		return fmt.Errorf("CreateComment: %v", err)
	}

	// TODO: Dedup.
	// Re-parse "comment" template with updated reactionsBar and reactableID template functions.
	tt, err := t.Clone()
	if err != nil {
		return fmt.Errorf("t.Clone: %v", err)
	}
	tt, err = tt.Funcs(template.FuncMap{
		"reactableID": func(commentID uint64) string {
			return fmt.Sprintf("%d/%d", a.State.IssueID, commentID)
		},
		"reactionsBar": func(reactions []reactions.Reaction, reactableID string) htmlg.Component {
			return reactionscomponent.ReactionsBar{
				Reactions:   reactions,
				CurrentUser: a.State.CurrentUser,
				ID:          reactableID,
			}
		},
	}).Parse(`
{{/* Dot is an issues.Comment. */}}
{{define "comment"}}
<div class="comment-edit-container">
	<div>` /* The comment view div. Visible initially. */ + `
		<div style="display: flex;" class="list-entry">
			<div style="margin-right: 10px;">{{render (avatar .User)}}</div>
			<div id="comment-{{.ID}}" style="flex-grow: 1; display: flex;">
				<div class="list-entry-container list-entry-border">
					<div class="list-entry-header" style="display: flex;">
						<span style="flex-grow: 1;">{{render (user .User)}} commented <a class="black" href="#comment-{{.ID}}" onclick="AnchorScroll(this, event);">{{render (time .CreatedAt)}}</a>
							{{with .Edited}} · <span style="cursor: default;" title="{{.By.Login}} edited this comment {{reltime .At}}.">edited{{if not (equalUsers $.User .By)}} by {{.By.Login}}{{end}}</span>{{end}}
						</span>
						<span class="right-icon">{{render (newReaction (reactableID .ID))}}</span>
						{{if .Editable}}<span class="right-icon"><a href="javascript:" title="Edit" onclick="EditComment({{` + "`edit`" + ` | json}}, this, event);">{{octicon "pencil"}}</a></span>{{end}}
					</div>
					<div class="list-entry-body">
						<div class="markdown-body">
							{{with .Body}}
								{{. | gfm}}
							{{else}}
								<i class="gray">No description.</i>
							{{end}}
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	<div style="display: none;">` /* The edit view div. Hidden initially. */ + `
		{{template "edit-comment" .}}
	</div>
	{{render (reactionsBar .Reactions (reactableID .ID))}}
</div>
{{end}}
`)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = tt.ExecuteTemplate(&buf, "comment", comment)
	if err != nil {
		return fmt.Errorf("t.ExecuteTemplate: %v", err)
	}

	// Create comment.
	newComment := document.CreateElement("div").(*dom.HTMLDivElement)

	newItemMarker := document.GetElementByID("new-item-marker")
	newItemMarker.ParentNode().InsertBefore(newComment, newItemMarker)

	newComment.SetOuterHTML(buf.String())

	// Reset new-comment component.
	commentEditor.SetValue("")
	commentEditor.Underlying().Call("dispatchEvent", js.Global().Get("CustomEvent").New("input")) // Trigger "input" event listeners.
	switchWriteTab(document.GetElementByID("new-comment-container"), commentEditor)

	return nil
}

func setupIssueToggleButton() {
	if issueToggleButton := document.GetElementByID("issue-toggle-button"); issueToggleButton != nil {
		commentEditor := document.QuerySelector("#new-comment-container .comment-editor").(*dom.HTMLTextAreaElement)
		commentEditor.AddEventListener("input", false, func(_ dom.Event) {
			if strings.TrimSpace(commentEditor.Value()) == "" {
				issueToggleButton.SetTextContent(issueToggleButton.GetAttribute("data-1-action"))
			} else {
				issueToggleButton.SetTextContent(issueToggleButton.GetAttribute("data-2-actions"))
			}
		})
	}
}

func MarkdownPreview(this dom.HTMLElement) {
	container := getAncestorByClassName(this, "edit-container")

	if container.QuerySelector(".preview-tab-link").(dom.Element).Class().Contains("active") {
		return
	}

	commentEditor := container.QuerySelector(".comment-editor").(*dom.HTMLTextAreaElement)
	commentPreview := container.QuerySelector(".comment-preview").(*dom.HTMLDivElement)

	fmted, _ := markdown.Process("", []byte(commentEditor.Value()), nil)
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

func funcOf(f func()) js.Func {
	return js.FuncOf(func(js.Value, []js.Value) interface{} { f(); return nil })
}
