package component

import (
	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// BlankSlate is a blank slate.
type BlankSlate struct {
	Content htmlg.Component
}

func (bs BlankSlate) Render() []*html.Node {
	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: `border: 1px solid #ddd;
border-radius: 4px;
padding: 80px 0 80px 0;
text-align: center;`},
		},
	}
	htmlg.AppendChildren(div, bs.Content.Render()...)
	return []*html.Node{div}
}

// Flash is a flash message.
type Flash struct {
	Content htmlg.Component
}

func (f Flash) Render() []*html.Node {
	div := &html.Node{
		Type: html.ElementNode, Data: atom.Div.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: `background-color: rgb(255, 249, 236);
border: 1px solid rgb(224, 216, 196);
border-radius: 3px;
padding: 12px;
margin: 24px 0 24px 0;
text-align: center;`},
		},
	}
	htmlg.AppendChildren(div, f.Content.Render()...)
	return []*html.Node{div}
}
