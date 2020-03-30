package changesapp

import (
	"strings"

	homecomponent "github.com/shurcooL/home/component"
	"github.com/shurcooL/home/internal/exp/app/changesapp/component"
	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type commits struct {
	Commits []commit
}

func (cs commits) Render() []*html.Node {
	if len(cs.Commits) == 0 {
		// No commits. Let the user know via a blank slate.
		div := &html.Node{
			Type: html.ElementNode, Data: atom.Div.String(),
			Attr:       []html.Attribute{{Key: atom.Style.String(), Val: "text-align: center; margin-top: 80px; margin-bottom: 80px;"}},
			FirstChild: htmlg.Text("There are no commits."),
		}
		return []*html.Node{htmlg.DivClass("list-entry-border", div)}
	}

	var nodes []*html.Node
	for _, c := range cs.Commits {
		nodes = append(nodes, c.Render()...)
	}
	return []*html.Node{htmlg.DivClass("list-entry-border", nodes...)}
}

type commit struct {
	change.Commit
}

func (c commit) Render() []*html.Node {
	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "display: flex;"}},
	}

	avatarDiv := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "margin-right: 6px;"}},
	}
	htmlg.AppendChildren(avatarDiv, component.Avatar{User: c.Author, Size: 32}.Render()...)
	div.AppendChild(avatarDiv)

	titleAndByline := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "flex-grow: 1;"}},
	}
	{
		commitSubject, commitBody := splitCommitMessage(c.Message)

		title := htmlg.Div(
			&html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "black"},
					{Key: atom.Href.String(), Val: "files/" + c.SHA},
				},
				FirstChild: htmlg.Strong(commitSubject),
			},
		)
		if commitBody != "" {
			htmlg.AppendChildren(title, homecomponent.EllipsisButton{OnClick: "ToggleDetails(this);"}.Render()...)
		}
		titleAndByline.AppendChild(title)

		byline := htmlg.DivClass("gray tiny")
		byline.Attr = append(byline.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 2px;"})
		htmlg.AppendChildren(byline, component.User{User: c.Author}.Render()...)
		byline.AppendChild(htmlg.Text(" committed "))
		htmlg.AppendChildren(byline, component.Time{Time: c.AuthorTime}.Render()...)
		titleAndByline.AppendChild(byline)

		if commitBody != "" {
			pre := &html.Node{
				Type: html.ElementNode, Data: atom.Pre.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "commit-details"},
					{Key: atom.Style.String(), Val: `font-size: 13px;
font-family: Go;
color: #444;
margin-top: 10px;
margin-bottom: 0;
display: none;`}},
				FirstChild: htmlg.Text(commitBody),
			}
			titleAndByline.AppendChild(pre)
		}
	}
	div.AppendChild(titleAndByline)

	commitID := commitID{SHA: c.SHA}
	htmlg.AppendChildren(div, commitID.Render()...)

	listEntryDiv := htmlg.DivClass("list-entry-body multilist-entry commit-container", div)
	return []*html.Node{listEntryDiv}
}

// commitID is a component that displays a linked commit ID. E.g., "c0de1234".
type commitID struct {
	SHA     string
	HTMLURL string // Optional.
}

func (c commitID) Render() []*html.Node {
	sha := &html.Node{
		Type: html.ElementNode, Data: atom.Code.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "width: 8ch; overflow: hidden; display: inline-grid; white-space: nowrap;"},
			{Key: atom.Title.String(), Val: c.SHA},
		},
		FirstChild: htmlg.Text(c.SHA),
	}
	if c.HTMLURL != "" {
		sha = &html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Href.String(), Val: c.HTMLURL},
			},
			FirstChild: sha,
		}
	}
	return []*html.Node{sha}
}

// splitCommitMessage splits commit message s into subject and body, if any.
func splitCommitMessage(s string) (subject, body string) {
	i := strings.Index(s, "\n\n")
	if i == -1 {
		return s, ""
	}
	return s[:i], s[i+2:]
}
