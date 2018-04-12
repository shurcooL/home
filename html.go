package main

import (
	"fmt"

	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// TODO: Dedup.

// tabnav is a left-aligned horizontal row of tabs Primer CSS component.
//
// http://primercss.io/nav/#tabnav
type tabnav struct {
	Tabs []tab
}

func (t tabnav) Render() []*html.Node {
	nav := &html.Node{
		Type: html.ElementNode, Data: atom.Nav.String(),
		Attr: []html.Attribute{{Key: atom.Class.String(), Val: "tabnav-tabs"}},
	}
	for _, t := range t.Tabs {
		htmlg.AppendChildren(nav, t.Render()...)
	}
	return []*html.Node{htmlg.DivClass("tabnav", nav)}
}

// tab is a single tab entry within a tabnav.
type tab struct {
	Content  htmlg.Component
	URL      string
	Selected bool
}

func (t tab) Render() []*html.Node {
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
	htmlg.AppendChildren(a, t.Content.Render()...)
	return []*html.Node{a}
}

type contentCounter struct {
	Content htmlg.Component
	Count   int
}

func (cc contentCounter) Render() []*html.Node {
	var ns []*html.Node
	ns = append(ns, cc.Content.Render()...)
	ns = append(ns, htmlg.SpanClass("counter", htmlg.Text(fmt.Sprint(cc.Count))))
	return ns
}

// iconText is an icon with text on the right.
// Icon must be not nil.
type iconText struct {
	Icon func() *html.Node // Must be not nil.
	Text string
}

func (it iconText) Render() []*html.Node {
	icon := htmlg.Span(it.Icon())
	icon.Attr = append(icon.Attr, html.Attribute{
		Key: atom.Style.String(), Val: "margin-right: 4px;",
	})
	text := htmlg.Text(it.Text)
	return []*html.Node{icon, text}
}
