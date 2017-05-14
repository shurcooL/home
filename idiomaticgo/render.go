// Package idiomaticgo contains functionality for rendering /idiomatic-go page.
package idiomaticgo

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	resumecomponent "github.com/shurcooL/resume/component"
	"github.com/shurcooL/sanitized_anchor_name"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const idiomaticGoURI = "dmitri.shuralyov.com/idiomatic-go"

// ReactableURL is the URL for reactionable items on the Idiomatic Go page.
const ReactableURL = idiomaticGoURI

// RenderBodyInnerHTML renders the inner HTML of the <body> element of the Idiomatic Go page.
// It's safe for concurrent use.
func RenderBodyInnerHTML(ctx context.Context, w io.Writer, issuesService issues.Service, notifications notifications.Service, authenticatedUser users.User, returnURL string) error {
	var nc uint64
	if authenticatedUser.ID != 0 {
		var err error
		nc, err = notifications.Count(ctx, nil)
		if err != nil {
			return err
		}
	}

	_, err := io.WriteString(w, `<div style="max-width: 800px; margin: 0 auto 100px auto;">`)
	if err != nil {
		return err
	}

	// Render the header.
	header := component.Header{
		CurrentUser:       authenticatedUser,
		NotificationCount: nc,
		ReturnURL:         returnURL,
	}
	err = htmlg.RenderComponents(w, header)
	if err != nil {
		return err
	}

	// TODO: This is messy rendering code, clean it up.
	io.WriteString(w, `<div class="markdown-body markdown-header-anchor" style="margin-bottom: 60px;">`)
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
	var lis []*html.Node
	for _, issue := range is {
		if issue.State != issues.OpenState || !accepted(issue) {
			continue
		}
		lis = append(lis, htmlg.LI(
			htmlg.A(issue.Title, "#"+sanitized_anchor_name.Create(issue.Title)),
		))
	}
	html.Render(w, htmlg.UL(lis...))

	openProposals := 0
	for _, issue := range is {
		if issue.State == issues.OpenState && !accepted(issue) {
			openProposals++
		}
	}
	if openProposals > 0 {
		html.Render(w, htmlg.P(
			htmlg.Text("There are also "),
			htmlg.A(fmt.Sprintf("%d open proposals", openProposals), "/issues/dmitri.shuralyov.com/idiomatic-go"),
			htmlg.Text(" being considered."),
		))
	}
	io.WriteString(w, `</div>`)

	for _, issue := range is {
		if issue.State != issues.OpenState || !accepted(issue) {
			continue
		}
		const commentID = 0
		cs, err := issuesService.ListComments(ctx, issues.RepoSpec{URI: idiomaticGoURI}, issue.ID, &issues.ListOptions{Start: commentID, Length: 1})
		if err != nil {
			return err
		}
		if commentID >= len(cs) {
			return fmt.Errorf("issue has no body")
		}
		comment := cs[commentID]

		io.WriteString(w, `<div class="markdown-body markdown-header-anchor" style="margin-bottom: 16px;">`)
		html.Render(w, github_flavored_markdown.Heading(atom.H3, issue.Title))
		io.WriteString(w, `</div>`)
		io.WriteString(w, `<div class="markdown-body" style="padding-bottom: 10px; border-bottom: 1px solid #eee; margin-bottom: 8px;">`)
		w.Write(github_flavored_markdown.Markdown([]byte(comment.Body)))
		io.WriteString(w, `</div>`)

		io.WriteString(w, `<div class="reaction-bar-appear" style="display: flex; justify-content: space-between; margin-bottom: 60px;">`)
		err = htmlg.RenderComponents(w, resumecomponent.ReactionsBar{
			Reactions:    comment.Reactions,
			ReactableURL: ReactableURL,
			CurrentUser:  authenticatedUser,
			ID:           fmt.Sprintf("%d", issue.ID), // TODO: "/0"?
		})
		if err != nil {
			return err
		}
		// TODO: Use iconText or similar component here?
		io.WriteString(w, `<span class="black-link markdown-body" style="display: inline-block; margin-top: 4px; min-width: 150px; text-align: right;">`)
		fmt.Fprintf(w, `<a href="/issues/%s/%d" style="line-height: 30px;"><span style="margin-right: 6px; position: relative; top: 7px;">%s</span>%d comments</a>`, idiomaticGoURI, issue.ID, octiconsCommentDiscussion, issue.Replies)
		io.WriteString(w, `</span>`)
		io.WriteString(w, `</div>`)
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

var octiconsCommentDiscussion = func() string {
	var buf bytes.Buffer
	err := html.Render(&buf, octiconssvg.CommentDiscussion())
	if err != nil {
		panic(err)
	}
	return buf.String()
}()
