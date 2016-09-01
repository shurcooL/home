package main

import (
	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Notifications is an icon for displaying if user has unread notifications.
type Notifications struct {
	// Unread is whether the user has unread notifications.
	Unread bool
}

func (n Notifications) Render() []*html.Node {
	a := &html.Node{
		Type: html.ElementNode, Data: atom.A.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "notifications"}, // TODO: Factor in that CSS class's declaration block, and :hover selector.
			{Key: atom.Href.String(), Val: "/notifications"},
		},
	}
	a.AppendChild(Octicon("bell"))
	if n.Unread {
		a.Attr = append(a.Attr, html.Attribute{Key: atom.Title.String(), Val: "You have unread notifications."})
		a.AppendChild(htmlg.SpanClass("notifications-unread")) // TODO: Factor in that CSS class's declaration block.
	}
	return []*html.Node{a}
}

// PostButton is a button that performs a POST action.
type PostButton struct {
	Action    string
	Text      string
	ReturnURL string
}

func (b PostButton) Render() []*html.Node {
	// TODO: Make this much nicer.
	/*
		<form method="post" action="{{.Action}}" style="display: inline-block; margin-bottom: 0;">
			<input type="submit" value="{{.Text}}" style=...>
			<input type="hidden" name="return" value="{{.ReturnURL}}">
		</form>
	*/
	form := &html.Node{
		Type: html.ElementNode, Data: atom.Form.String(),
		Attr: []html.Attribute{
			{Key: atom.Method.String(), Val: "post"},
			{Key: atom.Action.String(), Val: b.Action},
			{Key: atom.Style.String(), Val: `display: inline-block; margin-bottom: 0;`},
		},
	}
	form.AppendChild(&html.Node{
		Type: html.ElementNode, Data: atom.Input.String(),
		Attr: []html.Attribute{
			{Key: atom.Type.String(), Val: "submit"},
			{Key: atom.Value.String(), Val: b.Text},
			{Key: atom.Style.String(), Val: `font-size: 11px;
line-height: 11px;
border-radius: 4px;
border: solid #d2d2d2 1px;
background-color: #fff;
box-shadow: 0 1px 1px rgba(0, 0, 0, .05);`},
		},
	})
	form.AppendChild(&html.Node{
		Type: html.ElementNode, Data: atom.Input.String(),
		Attr: []html.Attribute{
			{Key: atom.Type.String(), Val: "hidden"},
			{Key: atom.Name.String(), Val: "return"},
			{Key: atom.Value.String(), Val: b.ReturnURL},
		},
	})
	return []*html.Node{form}
}

// Octicon returns an Octicons SVG node for symbol.
//
// TODO: Factor this out.
func Octicon(symbol string) *html.Node {
	switch symbol {
	case "bell":
		parent := (*html.Node)(&html.Node{
			Parent:      (*html.Node)(nil),
			PrevSibling: (*html.Node)(nil),
			NextSibling: (*html.Node)(nil),
			Type:        (html.NodeType)(html.ElementNode),
			DataAtom:    (atom.Atom)(atom.Svg),
			Data:        (string)("svg"),
			Namespace:   (string)("svg"),
			Attr: ([]html.Attribute)([]html.Attribute{
				(html.Attribute)(html.Attribute{
					Namespace: (string)(""),
					Key:       (string)("xmlns"),
					Val:       (string)("http://www.w3.org/2000/svg"),
				}),
				(html.Attribute)(html.Attribute{
					Namespace: (string)(""),
					Key:       (string)("width"),
					Val:       (string)("16"),
				}),
				(html.Attribute)(html.Attribute{
					Namespace: (string)(""),
					Key:       (string)("height"),
					Val:       (string)("16"),
				}),
				(html.Attribute)(html.Attribute{
					Namespace: (string)(""),
					Key:       (string)("viewBox"),
					Val:       (string)("0 0 14 16"),
				}),
				(html.Attribute)(html.Attribute{
					Namespace: (string)(""),
					Key:       (string)("style"),
					Val:       (string)("vertical-align: top;"),
				}),
			}),
		})
		child := (*html.Node)(&html.Node{
			Parent:      (*html.Node)(parent),
			FirstChild:  (*html.Node)(nil),
			LastChild:   (*html.Node)(nil),
			PrevSibling: (*html.Node)(nil),
			NextSibling: (*html.Node)(nil),
			Type:        (html.NodeType)(html.ElementNode),
			DataAtom:    (atom.Atom)(0),
			Data:        (string)("path"),
			Namespace:   (string)("svg"),
			Attr: ([]html.Attribute)([]html.Attribute{
				(html.Attribute)(html.Attribute{
					Namespace: (string)(""),
					Key:       (string)("d"),
					Val:       (string)("M14 12v1H0v-1l.73-.58c.77-.77.81-2.55 1.19-4.42C2.69 3.23 6 2 6 2c0-.55.45-1 1-1s1 .45 1 1c0 0 3.39 1.23 4.16 5 .38 1.88.42 3.66 1.19 4.42l.66.58H14zm-7 4c1.11 0 2-.89 2-2H5c0 1.11.89 2 2 2z"),
				}),
			}),
		})
		parent.FirstChild = child
		parent.LastChild = child
		return parent
	default:
		return nil
	}
}
