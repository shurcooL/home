// +build js,wasm

package main

import (
	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type notificationTab uint8

const (
	noTab notificationTab = iota
	streamTab
	threadTab
)

func notificationTabnav(selected notificationTab) htmlg.Component {
	// TODO: pass in opt.BaseURL from backend
	return tabnav{
		Tabs: []tab{
			{
				Content:  htmlg.NodeComponent(*htmlg.Text("Stream")),
				URL:      "/notificationsv2", // TODO: "/notificationsv2" should be opt.BaseURL.
				Selected: selected == streamTab,
			},
			{
				Content:  htmlg.NodeComponent(*htmlg.Text("Threads")),
				URL:      "/notificationsv2/threads", // TODO: "/notificationsv2" should be opt.BaseURL.
				Selected: selected == threadTab,
			},
		},
	}
}

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
