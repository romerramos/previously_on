package termrender

import "charm.land/glamour/v2"

type MarkdownOptions struct {
	Style string
	Width int
}

func Markdown(input string, opts MarkdownOptions) (string, error) {
	style := opts.Style
	if style == "" {
		style = "tokyo-night"
	}
	width := opts.Width
	if width <= 0 {
		width = 100
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return renderer.Render(input)
}
