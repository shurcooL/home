package component

import (
	"fmt"

	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/htmlg"
	issuescomponent "github.com/shurcooL/issuesapp/component"
	"github.com/shurcooL/octicon"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Changes is a component that displays a page of changes,
// with a navigation bar on top.
type Changes struct {
	ChangesNav ChangesNav
	Filter     change.StateFilter
	Entries    []ChangeEntry
}

func (i Changes) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <div class="list-entry list-entry-border">
	// 	{{render .ChangesNav}}
	// 	{{with .Entries}}{{range .}}
	// 		{{render .}}
	// 	{{end}}{{else}}
	// 		<div style="text-align: center; margin-top: 80px; margin-bottom: 80px;">There are no {{.Filter}} changes.</div>
	// 	{{end}}
	// </div>

	var ns []*html.Node
	ns = append(ns, i.ChangesNav.Render()...)
	for _, e := range i.Entries {
		ns = append(ns, e.Render()...)
	}
	if len(i.Entries) == 0 {
		// No changes with this filter. Let the user know via a blank slate.
		div := &html.Node{
			Type: html.ElementNode, Data: atom.Div.String(),
			Attr: []html.Attribute{{Key: atom.Style.String(), Val: "text-align: center; margin-top: 80px; margin-bottom: 80px;"}},
		}
		switch i.Filter {
		case change.FilterOpen:
			div.AppendChild(htmlg.Text("There are no open changes."))
		case change.FilterClosedMerged:
			div.AppendChild(htmlg.Text("There are no closed/merged changes."))
		case change.FilterAll:
			div.AppendChild(htmlg.Text("There are no changes."))
		}
		ns = append(ns, div)
	}

	div := htmlg.DivClass("list-entry list-entry-border", ns...)
	return []*html.Node{div}
}

// ChangeEntry is an entry within the list of changes.
type ChangeEntry struct {
	Change change.Change
	Unread bool // Unread indicates whether the change contains unread notifications for authenticated user.

	// TODO, THINK: This is router details, can it be factored out or cleaned up?
	BaseURL string
}

func (i ChangeEntry) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <div class="list-entry-body multilist-entry"{{if .Unread}} style="box-shadow: 2px 0 0 #4183c4 inset;"{{end}}>
	// 	<div style="display: flex;">
	// 		{{render (issueIcon .State)}}
	// 		<div style="flex-grow: 1;">
	// 			<div>
	// 				<a class="black" href="{{state.BaseURL}}/{{.ID}}"><strong>{{.Title}}</strong></a>
	// 				{{range .Labels}}{{render (label .)}}{{end}}
	// 			</div>
	// 			<div class="gray tiny">#{{.ID}} opened {{render (time .CreatedAt)}} by {{.User.Login}}</div>
	// 		</div>
	// 		<span title="{{.Replies}} replies" class="tiny {{if .Replies}}gray{{else}}lightgray{{end}}">{{octicon "comment"}} {{.Replies}}</span>
	// 	</div>
	// </div>

	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "display: flex;"}},
	}
	htmlg.AppendChildren(div, ChangeIcon{State: i.Change.State}.Render()...)

	titleAndByline := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: atom.Style.String(), Val: "flex-grow: 1;"}},
	}
	{
		title := htmlg.Div(
			&html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "black"},
					{Key: atom.Href.String(), Val: fmt.Sprintf("%s/%d", i.BaseURL, i.Change.ID)},
					{Key: atom.Onclick.String(), Val: "Open(event, this)"},
				},
				FirstChild: htmlg.Strong(i.Change.Title),
			},
		)
		for _, l := range i.Change.Labels {
			span := &html.Node{
				Type: html.ElementNode, Data: atom.Span.String(),
				Attr: []html.Attribute{{Key: atom.Style.String(), Val: "margin-left: 4px;"}},
			}
			htmlg.AppendChildren(span, issuescomponent.Label{Label: l}.Render()...)
			title.AppendChild(span)
		}
		titleAndByline.AppendChild(title)

		byline := htmlg.DivClass("gray tiny")
		byline.Attr = append(byline.Attr, html.Attribute{Key: atom.Style.String(), Val: "margin-top: 2px;"})
		byline.AppendChild(htmlg.Text(fmt.Sprintf("#%d opened ", i.Change.ID)))
		htmlg.AppendChildren(byline, Time{Time: i.Change.CreatedAt}.Render()...)
		byline.AppendChild(htmlg.Text(fmt.Sprintf(" by %s", i.Change.Author.Login)))
		titleAndByline.AppendChild(byline)
	}
	div.AppendChild(titleAndByline)

	spanClass := "tiny"
	switch i.Change.Replies {
	default:
		spanClass += " gray"
	case 0:
		spanClass += " lightgray"
	}
	span := &html.Node{
		Type: html.ElementNode, Data: atom.Span.String(),
		Attr: []html.Attribute{
			{Key: atom.Title.String(), Val: fmt.Sprintf("%d replies", i.Change.Replies)},
			{Key: atom.Class.String(), Val: spanClass},
		},
	}
	span.AppendChild(octicon.Comment())
	span.AppendChild(htmlg.Text(fmt.Sprintf(" %d", i.Change.Replies)))
	div.AppendChild(span)

	listEntryDiv := htmlg.DivClass("list-entry-body multilist-entry", div)
	if i.Unread {
		listEntryDiv.Attr = append(listEntryDiv.Attr,
			html.Attribute{Key: atom.Style.String(), Val: "box-shadow: 2px 0 0 #4183c4 inset;"},
		)
	}
	return []*html.Node{listEntryDiv}
}
