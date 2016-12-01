// Package idiomaticgo contains functionality for rendering /idiomatic-go page.
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
	_, err := io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	// Render the header.
	header := component.Header{
		CurrentUser:   authenticatedUser,
		ReturnURL:     returnURL,
		Notifications: notifications,
	}
	err = htmlg.RenderComponentsContext(ctx, w, header)
	if err != nil {
		return err
	}

	// TODO: This is messy rendering code, clean it up.
	fmt.Fprint(w, `<div class="markdown-body markdown-header-anchor" style="margin-bottom: 60px;">`)
	w.Write(github_flavored_markdown.Markdown([]byte(`# Idiomatic Go

When reviewing Go code, if I run into a situation where I see an unnecessary deviation from
idiomatic Go style or best practice, I add an entry here complete with some rationale, and
link to it.

I can do this for the smallest and most subtle of details, since I care about Go a lot. I can
reuse this each time the same issue comes up, instead of having to re-write the rationale
multiple times, or skip explaining why I make a given suggestion.

You can view this as my supplement to https://github.com/golang/go/wiki/CodeReviewComments.

This page is generated from the list of issues with label "Accepted" [here](/issues/dmitri.shuralyov.com/idiomatic-go).
If you'd like to add a new suggestion, please provide convincing rationale and references
(e.g., links to places in Go project that support your suggestion), and open a new issue.
It'll show up here when I add an "Accepted" label.`)))

	is, err := issuesService.List(ctx, issues.RepoSpec{URI: idiomaticGoURI}, issues.IssueListOptions{State: issues.StateFilter(issues.OpenState)})
	if err != nil {
		return err
	}
	fmt.Fprint(w, "<ul>")
	for _, issue := range is {
		if issue.State != issues.OpenState || !accepted(issue) {
			continue
		}
		fmt.Fprint(w, "<li>"+htmlg.Render(htmlg.A(issue.Title, template.URL("#"+sanitized_anchor_name.Create(issue.Title))))+"</li>")
	}
	fmt.Fprint(w, "</ul>")

	openProposals := 0
	for _, issue := range is {
		if issue.State == issues.OpenState && !accepted(issue) {
			openProposals++
		}
	}
	if openProposals > 0 {
		w.Write(github_flavored_markdown.Markdown([]byte(fmt.Sprintf("There are also [%d open proposals](/issues/dmitri.shuralyov.com/idiomatic-go) being considered.", openProposals))))
	}
	fmt.Fprint(w, `</div>`)

	for _, issue := range is {
		if issue.State != issues.OpenState || !accepted(issue) {
			continue
		}
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

		fmt.Fprint(w, `<div class="reaction-bar-appear" style="display: flex; justify-content: space-between; margin-bottom: 60px;">`)
		err = htmlg.RenderComponentsContext(ctx, w, resumecomponent.ReactionsBar{
			Reactions:    IssuesReactions{Issues: issuesService},
			ReactableURL: ReactableURL,
			CurrentUser:  authenticatedUser,
			ID:           fmt.Sprintf("%v", issue.ID), // TODO: "/0"?
		})
		if err != nil {
			return err
		}
		fmt.Fprint(w, `<span class="black-link markdown-body" style="display: inline-block; margin-top: 4px; min-width: 150px; text-align: right;">`)
		fmt.Fprintf(w, `<a href="/issues/%v/%v" style="line-height: 30px;"><span class="octicon octicon-comment-discussion" style="margin-right: 6px;"></span>%v comments</a>`, idiomaticGoURI, issue.ID, issue.Replies)
		fmt.Fprint(w, `</span>`)
		fmt.Fprint(w, `</div>`)
	}

	_, err = io.WriteString(w, `</div>`)
	return err
}

// accepted reports if issue has an "Accepted" label.
func accepted(issue issues.Issue) bool {
	for _, l := range issue.Labels {
		if l.Name == "Accepted" {
			return true
		}
	}
	return false
}
