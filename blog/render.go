// Package blog contains functionality for rendering /blog page.
package blog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/home/component"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	reactionscomponent "github.com/shurcooL/reactions/component"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var shurcool = users.UserSpec{ID: 1924134, Domain: "github.com"}

// RenderBodyInnerHTML renders the inner HTML of the <body> element of the Blog page.
// It's safe for concurrent use.
func RenderBodyInnerHTML(ctx context.Context, w io.Writer, issuesService issues.Service, blogURI issues.RepoSpec, notifications notifications.Service, authenticatedUser users.User, returnURL string) error {
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

	// New Blog Post button for shurcooL.
	if authenticatedUser.UserSpec == shurcool {
		io.WriteString(w, `<div style="text-align: right;"><button style="font-size: 11px; line-height: 11px; border-radius: 4px; border: solid #d2d2d2 1px; background-color: #fff; box-shadow: 0 1px 1px rgba(0, 0, 0, .05);" onclick="window.location = '/blog/new';">New Blog Post</button></div>`)
	}

	is, err := issuesService.List(ctx, blogURI, issues.IssueListOptions{State: issues.StateFilter(issues.OpenState)})
	if err != nil {
		return err
	}
	sort.Slice(is, func(i, j int) bool { return is[i].CreatedAt.After(is[j].CreatedAt) })
	for _, issue := range is {
		if issue.State != issues.OpenState {
			continue
		}
		const commentID = 0
		cs, err := issuesService.ListComments(ctx, blogURI, issue.ID, &issues.ListOptions{Start: commentID, Length: 1})
		if err != nil {
			return err
		}
		if commentID >= len(cs) {
			return fmt.Errorf("issue has no body")
		}
		comment := cs[commentID]

		// Header.
		io.WriteString(w, `<div class="markdown-body">`)
		html.Render(w, htmlg.H3(htmlg.A(issue.Title, fmt.Sprintf("/blog/%d", issue.ID))))
		io.WriteString(w, `</div>`)

		// Post meta information.
		var lis = []*html.Node{
			htmlg.LIClass("post-meta", iconText{
				Icon:    octiconssvg.Calendar,
				Text:    comment.CreatedAt.Format("January 2, 2006"),
				Tooltip: humanize.Time(comment.CreatedAt) + " – " + comment.CreatedAt.Local().Format("Jan 2, 2006, 3:04 PM MST"), // TODO: Use local time of page viewer, not server.
			}.Render()...),
			htmlg.LIClass("post-meta", imageText{ImageURL: comment.User.AvatarURL, Text: comment.User.Login}.Render()...),
		}
		if labels := labelNames(issue.Labels); len(labels) != 0 {
			lis = append(lis, htmlg.LIClass("post-meta", iconText{Icon: octiconssvg.Tag, Text: strings.Join(labels, ", ")}.Render()...))
		}
		html.Render(w, htmlg.ULClass("post-meta", lis...))

		// Contents.
		io.WriteString(w, `<div class="markdown-body" style="padding-bottom: 10px; border-bottom: 1px solid #eee; margin-bottom: 8px;">`)
		w.Write(github_flavored_markdown.Markdown([]byte(comment.Body)))
		io.WriteString(w, `</div>`)

		io.WriteString(w, `<div class="reaction-bar-appear" style="display: flex; justify-content: space-between; margin-bottom: 60px;">`)
		err = htmlg.RenderComponents(w, reactionscomponent.ReactionsBar{
			Reactions:   comment.Reactions,
			CurrentUser: authenticatedUser,
			ID:          fmt.Sprintf("%d", issue.ID), // TODO: "/0"?
		})
		if err != nil {
			return err
		}
		// TODO: Use iconText or similar component here?
		io.WriteString(w, `<span class="black-link markdown-body" style="display: inline-block; margin-top: 4px; min-width: 150px; text-align: right;">`)
		fmt.Fprintf(w, `<a href="/blog/%d" style="line-height: 30px;"><span style="margin-right: 6px; position: relative; top: 7px;">%s</span>%d comments</a>`, issue.ID, octiconsCommentDiscussion, issue.Replies)
		io.WriteString(w, `</span>`)
		io.WriteString(w, `</div>`)
	}

	_, err = io.WriteString(w, `</div>`)
	return err
}

func labelNames(labels []issues.Label) (names []string) {
	for _, l := range labels {
		names = append(names, l.Name)
	}
	return names
}

// iconText is an icon with text on the right.
// Icon must be not nil.
type iconText struct {
	Icon    func() *html.Node // Must be not nil.
	Text    string
	Tooltip string // Optional tooltip.
}

func (it iconText) Render() []*html.Node {
	icon := htmlg.Span(it.Icon())
	icon.Attr = append(icon.Attr, html.Attribute{
		Key: atom.Style.String(), Val: "margin-right: 4px;",
	})
	text := htmlg.Text(it.Text)
	span := htmlg.Span(icon, text)
	if it.Tooltip != "" {
		span.Attr = append(span.Attr, html.Attribute{Key: atom.Title.String(), Val: it.Tooltip})
	}
	return []*html.Node{span}
}

// imageText is an image with text on the right.
// ImageURL must be not empty.
type imageText struct {
	ImageURL string // Must be not empty.
	Text     string
}

func (it imageText) Render() []*html.Node {
	image := &html.Node{
		Type: html.ElementNode, Data: atom.Img.String(),
		Attr: []html.Attribute{
			{Key: atom.Src.String(), Val: it.ImageURL},
			{Key: atom.Style.String(), Val: "width: 18px; height: 18px; border-radius: 3px; vertical-align: bottom; margin-right: 4px;"},
		},
	}
	text := htmlg.Text(it.Text)
	return []*html.Node{image, text}
}

var octiconsCommentDiscussion = func() string {
	var buf bytes.Buffer
	err := html.Render(&buf, octiconssvg.CommentDiscussion())
	if err != nil {
		panic(err)
	}
	return buf.String()
}()
