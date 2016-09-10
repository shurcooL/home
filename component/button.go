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
			{Key: atom.Style.String(), Val: `font-size: 11px;
line-height: 11px;
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
