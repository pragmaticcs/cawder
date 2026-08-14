package styles

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

//go:embed gruvbox.json
var gruvboxJSON []byte

func loadStyle() (ansi.StyleConfig, error) {
	var style ansi.StyleConfig
	if err := json.Unmarshal(gruvboxJSON, &style); err != nil {
		return ansi.StyleConfig{}, err
	}
	return style, nil
}

func tint(style ansi.StyleConfig, hex string) ansi.StyleConfig {
	c := hex
	style.Document.Color = &c
	style.Paragraph.Color = &c
	style.Text.Color = &c
	style.Item.Color = &c
	return style
}

type Renderers struct {
	User      *glamour.TermRenderer
	Agent     *glamour.TermRenderer
	Reasoning *glamour.TermRenderer
}

func newTintedRenderer(hex string, width int) (*glamour.TermRenderer, error) {
	base, err := loadStyle()
	if err != nil {
		return nil, fmt.Errorf("load base style: %w", err)
	}
	return glamour.NewTermRenderer(
		glamour.WithStyles(tint(base, hex)),
		glamour.WithWordWrap(width),
	)
}

func NewRenderers(width int) Renderers {
	user, _ := newTintedRenderer(gruvBlue.Dark, width)
	agent, _ := newTintedRenderer(gruvFg.Dark, width)
	reasoning, _ := newTintedRenderer(gruvGray.Dark, width)
	return Renderers{
		User:      user,
		Agent:     agent,
		Reasoning: reasoning,
	}
}
