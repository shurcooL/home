package component

import (
	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// TODO: Dedup.

// TabNav is a left-aligned horizontal row of tabs Primer CSS component.
//
// http://primercss.io/nav/#tabnav
type TabNav struct {
	Tabs []Tab
}

func (t TabNav) Render() []*html.Node {
	nav := &html.Node{
		Type: html.ElementNode, Data: atom.Nav.String(),
		Attr: []html.Attribute{{Key: atom.Class.String(), Val: "tabnav-tabs"}},
	}
	for _, t := range t.Tabs {
		htmlg.AppendChildren(nav, t.Render()...)
	}
	return []*html.Node{htmlg.DivClass("tabnav", nav)}
}

// Tab is a single tab entry within a TabNav.
type Tab struct {
	Content  htmlg.Component
	URL      string
	OnClick  string
	Selected bool
}

func (t Tab) Render() []*html.Node {
	aClass := "tabnav-tab"
	if t.Selected {
		aClass += " selected"
	}
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Href.String(), Val: t.URL},
			{Key: atom.Class.String(), Val: aClass},
		},
	}
	if t.OnClick != "" {
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Onclick.String(), Val: t.OnClick})
	}
	htmlg.AppendChildren(a, t.Content.Render()...)
	return []*html.Node{a}
}
