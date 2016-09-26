package idiomaticgo

import (
	"context"
	"fmt"
	"html/template"
	"io"

	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/notifications"
	resumecomponent "github.com/shurcooL/resume/component"
	"github.com/shurcooL/sanitized_anchor_name"
	"github.com/shurcooL/users"
)

const idiomaticGoURI = "dmitri.shuralyov.com/idiomatic-go"

// ReactableURL is the URL for reactionable items on the Idiomatic Go page.
const ReactableURL = idiomaticGoURI

// RenderBodyInnerHTML renders the inner HTML of the <body> element of the Idiomatic Go page.
// It's safe for concurrent use.
func RenderBodyInnerHTML(ctx context.Context, w io.Writer, issuesService issues.Service, notifications notifications.Service, authenticatedUser users.User, returnURL string) error {
	// Render the header.
	header := component.Header{
		CurrentUser:   authenticatedUser,
		ReturnURL:     returnURL,
		Notifications: notifications,
	}
	err := htmlg.RenderComponentsContext(ctx, w, header)
	if err != nil {
		return err
	}

	// TODO: This is messy rendering code, clean it up.
	fmt.Fprint(w, `<div style="margin: 0 auto 0 auto;  padding: 0 30px 0 30px; max-width: 800px;">`)
	{
		fmt.Fprint(w, `<div class="markdown-body markdown-header-anchor" style="margin-bottom: 60px;">`)
		w.Write(github_flavored_markdown.Markdown([]byte(`# Idiomatic Go

When reviewing Go code, if I run into a situation where I see an unnecessary deviation from
idiomatic Go style or best practice, I add an entry here complete with some rationale, and
link to it.

I can do this for the smallest and most subtle of details, since I care about Go a lot. I can
reuse this each time the same issue comes up, instead of having to re-write the rationale
multiple times, or skip explaining why I make a given suggestion.

You can view this as my supplement to https://github.com/golang/go/wiki/CodeReviewComments.`)))

		is, err := issuesService.List(ctx, issues.RepoSpec{URI: idiomaticGoURI}, issues.IssueListOptions{State: issues.StateFilter(issues.OpenState)})
		if err != nil {
			return err
		}
		fmt.Fprint(w, "<ul>")
		for _, issue := range is {
			fmt.Fprint(w, "<li>"+htmlg.Render(htmlg.A(issue.Title, template.URL("#"+sanitized_anchor_name.Create(issue.Title))))+"</li>")
		}
		fmt.Fprint(w, "</ul>")
		fmt.Fprint(w, `</div>`)

		for _, issue := range is {
			cs, err := issuesService.ListComments(ctx, issues.RepoSpec{URI: idiomaticGoURI}, issue.ID, nil)
			if err != nil {
				return err
			}
			const commentID = 0
			if commentID >= len(cs) {
				return fmt.Errorf("issue has no body")
			}
			comment := cs[commentID]

			fmt.Fprint(w, `<div class="markdown-body markdown-header-anchor" style="margin-bottom: 12px;">`)
			w.Write(github_flavored_markdown.Markdown([]byte("### " + issue.Title)))
			fmt.Fprint(w, `</div>`)
			fmt.Fprint(w, `<div class="markdown-body" style="padding: 10px; border: 1px solid #ddd; border-radius: 4px; margin-bottom: 6px;">`)
			w.Write(github_flavored_markdown.Markdown([]byte(comment.Body)))
			fmt.Fprint(w, `</div>`)

			fmt.Fprint(w, `<div class="reaction-bar-appear" style="margin-bottom: 6px;">`)
			err = htmlg.RenderComponentsContext(ctx, w, resumecomponent.ReactionsBar{
				Reactions:    IssuesReactions{Issues: issuesService},
				ReactableURL: ReactableURL,
				CurrentUser:  authenticatedUser,
				ID:           fmt.Sprintf("%v", issue.ID), // TODO: "/0"?
			})
			if err != nil {
				return err
			}
			fmt.Fprint(w, `</div>`)
			fmt.Fprint(w, `<div class="black-link markdown-body" style="margin-bottom: 60px;">`)
			fmt.Fprintf(w, `<a href="/issues/%v/%v" style="line-height: 20px;"><span class="octicon octicon-comment-discussion" style="margin-right: 6px;"></span>%v comments</a>`, idiomaticGoURI, issue.ID, issue.Replies)
			fmt.Fprint(w, `</div>`)
		}
	}
	fmt.Fprint(w, `</div>`)

	return nil
}
