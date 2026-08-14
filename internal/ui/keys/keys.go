package keys

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Send          key.Binding
	Quit          key.Binding
	SelectModel   key.Binding
	SelectSession key.Binding
}

var Keys = KeyMap{
	Send: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "send"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("ctrl+c/esc", "quit"),
	),
	SelectModel: key.NewBinding(
		key.WithKeys("alt+m"),
		key.WithHelp("alt+m", "select model"),
	),
	SelectSession: key.NewBinding(
		key.WithKeys("alt+s"),
		key.WithHelp("alt+s", "select session"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Send,
		k.Quit,
		k.SelectModel,
		k.SelectSession,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			k.Send,
			k.Quit,
			k.SelectModel,
			k.SelectSession,
		},
	}
}
