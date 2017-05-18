// Package vec provides a vecty-like API for backend HTML rendering.
package vec

import (
	"fmt"
	"io"

	"github.com/shurcooL/htmlg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type HTML struct {
	Type       html.NodeType // html.ElementNode or html.TextNode.
	DataAtom   atom.Atom     // Used when Type is html.ElementNode.
	Data       string        // Used when Type is html.TextNode.
	Attributes map[atom.Atom]string

	Children  []*HTML
	Children2 []*html.Node // TODO: Generalize to/merge with all children. Currently, this is optional nodes after children.
}

type Component interface {
	Render() *HTML
}

func Render(w io.Writer, c Component) error {
	h := c.Render()
	err := renderHTML(w, h)
	return err
}

func RenderHTML(w io.Writer, hs ...*HTML) error {
	for _, h := range hs {
		err := renderHTML(w, h)
		if err != nil {
			return err
		}
	}
	return nil
}

func renderHTML(w io.Writer, h *HTML) error {
	switch h.Type {
	case html.ElementNode:
		_, err := io.WriteString(w, `<`)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, h.DataAtom.String())
		if err != nil {
			return err
		}

		for key, value := range h.Attributes {
			_, err = io.WriteString(w, ` `)
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, key.String())
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, `="`)
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, html.EscapeString(value))
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, `"`)
			if err != nil {
				return err
			}
		}

		_, err = io.WriteString(w, `>`)
		if err != nil {
			return err
		}

		for _, c := range h.Children {
			err = renderHTML(w, c)
			if err != nil {
				return err
			}
		}
		for _, c := range h.Children2 {
			err = html.Render(w, c)
			if err != nil {
				return err
			}
		}

		_, err = io.WriteString(w, `</`)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, h.DataAtom.String())
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, `>`)
		return err
	case html.TextNode:
		_, err := io.WriteString(w, html.EscapeString(h.Data))
		return err
	default:
		panic(fmt.Errorf("unknown node type %v", h.Type))
	}
}

type MarkupOrComponentOrHTML interface{}

type Markup func(h *HTML)

func Apply(h *HTML, m MarkupOrComponentOrHTML) {
	switch m := m.(type) {
	case Markup:
		m(h)
	case *HTML:
		h.Children = append(h.Children, m)
	case string:
		text := &HTML{Type: html.TextNode, Data: m}
		h.Children = append(h.Children, text)
	case *html.Node:
		panic(fmt.Errorf("*html.Node not supported"))
		//h.Children2 = append(h.Children2, m)
	case htmlg.Component:
		h.Children2 = append(h.Children2, m.Render()...)
	case Component:
		panic(fmt.Errorf("Component not supported"))
	default:
		panic(fmt.Errorf("invalid type %T does not match MarkupOrComponentOrHTML interface", m))
	}
}

//func Text(s string) *HTML {
//	return &HTML{
//		Type: html.TextNode,
//		Data: s,
//	}
//}

//func Attr(key atom.Atom, value string) Markup {
//	return func(h *HTML) {
//		if h.Attributes == nil {
//			h.Attributes = make(map[atom.Atom]string)
//		}
//		h.Attributes[key] = value
//	}
//}
