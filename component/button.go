package component

import (
	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

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
	button := &html.Node{
		Type: html.ElementNode, Data: atom.Button.String(),
		Attr: []html.Attribute{
			{Key: atom.Type.String(), Val: "submit"},
			{Key: atom.Style.String(), Val: `font-family: inherit;
font-size: 11px;
line-height: 11px;
height: 18px;
border-radius: 4px;
border: solid #d2d2d2 1px;
background-color: #fff;
box-shadow: 0 1px 1px rgba(0, 0, 0, .05);`},
		},
	}
	button.AppendChild(htmlg.Text(b.Text))
	form.AppendChild(button)
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

// EllipsisButton is a button with a horizontal ellipsis.
// It can be used to expand/collapse additional details.
type EllipsisButton struct {
	// OnClick is the value that the onclick handler gets set to.
	OnClick string
}

func (b EllipsisButton) Render() []*html.Node {
	// TODO: Find a way to embed the CSS into this component. Currently, it's in _data/commits/style.css only.
	svg := &html.Node{
		Type: html.ElementNode, Data: atom.Svg.String(),
		Attr: []html.Attribute{
			{Key: "viewBox", Val: "0 0 16 16"},
			{Key: atom.Style.String(), Val: "fill: currentColor; height: 12px;"},
		},
	}
	svg.AppendChild(&html.Node{
		Type: html.ElementNode, Data: "circle",
		Attr: []html.Attribute{{Key: "cx", Val: "3"}, {Key: "cy", Val: "8"}, {Key: "r", Val: "1.5"}},
	})
	svg.AppendChild(&html.Node{
		Type: html.ElementNode, Data: "circle",
		Attr: []html.Attribute{{Key: "cx", Val: "8"}, {Key: "cy", Val: "8"}, {Key: "r", Val: "1.5"}},
	})
	svg.AppendChild(&html.Node{
		Type: html.ElementNode, Data: "circle",
		Attr: []html.Attribute{{Key: "cx", Val: "13"}, {Key: "cy", Val: "8"}, {Key: "r", Val: "1.5"}},
	})
	button := &html.Node{
		Type: html.ElementNode, Data: atom.Button.String(),
		Attr: []html.Attribute{
			{Key: atom.Class.String(), Val: "ellipsis-button"},
			{Key: atom.Type.String(), Val: "button"}, // For accessibility.
			{Key: atom.Onclick.String(), Val: b.OnClick},
		},
		FirstChild: svg,
	}
	return []*html.Node{button}
}
