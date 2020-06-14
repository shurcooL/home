package assets

import (
	"github.com/shurcooL/github_flavored_markdown/gfmstyle"
	"github.com/shurcooL/gofontwoff"
	"github.com/shurcooL/reactions/emojis"
)

var (
	// Fonts contains the Go font family WOFF data.
	Fonts = gofontwoff.Assets

	// Emojis contains emojis image data.
	Emojis = emojis.Assets

	// GFMStyle contains CSS styles for rendering GitHub Flavored Markdown.
	GFMStyle = gfmstyle.Assets
)
