package component

import (
	"fmt"

	"github.com/shurcooL/home/internal/route"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/octicon"
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

type RepositoryTab uint8

const (
	NoTab RepositoryTab = iota
	PackagesTab
	HistoryTab
	IssuesTab
	ChangesTab
)

func RepositoryTabNav(selected RepositoryTab, repoPath string, packages int, openIssues, openChanges uint64) htmlg.Component {
	return TabNav{
		Tabs: []Tab{
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.Package, Text: "Packages"},
					Count:   packages,
				},
				URL:      route.RepoIndex(repoPath),
				Selected: selected == PackagesTab,
			},
			{
				Content:  iconText{Icon: octicon.History, Text: "History"},
				URL:      route.RepoHistory(repoPath),
				Selected: selected == HistoryTab,
			},
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.IssueOpened, Text: "Issues"},
					Count:   int(openIssues),
				},
				URL: route.RepoIssues(repoPath), OnClick: "Open(event, this)",
				Selected: selected == IssuesTab,
			},
			{
				Content: contentCounter{
					Content: iconText{Icon: octicon.GitPullRequest, Text: "Changes"},
					Count:   int(openChanges),
				},
				URL: route.RepoChanges(repoPath), OnClick: "Open(event, this)",
				Selected: selected == ChangesTab,
			},
		},
	}
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
