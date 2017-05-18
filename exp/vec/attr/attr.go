// Package attr defines functions to set attributes of an HTML node.
package attr

import (
	"github.com/shurcooL/home/exp/vec"
	"golang.org/x/net/html/atom"
)

func Class(class string) vec.Markup {
	return attr(atom.Class, class)
}

func Title(title string) vec.Markup {
	return attr(atom.Title, title)
}

func Width(width string) vec.Markup {
	return attr(atom.Width, width)
}

func Height(height string) vec.Markup {
	return attr(atom.Height, height)
}

func Src(src string) vec.Markup {
	return attr(atom.Src, src)
}

func Style(style string) vec.Markup {
	return attr(atom.Style, style)
}

func Href(href string) vec.Markup {
	return attr(atom.Href, href)
}

func attr(key atom.Atom, value string) vec.Markup {
	return func(h *vec.HTML) {
		if h.Attributes == nil {
			h.Attributes = make(map[atom.Atom]string)
		}
		h.Attributes[key] = value
	}
}
