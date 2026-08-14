package input

import (
	"github.com/charmbracelet/bubbles/textinput"
)

type InputMode int

const (
	InputText InputMode = iota
	InputTerminal
)

type Model struct {
	input textinput.Model
	mode  InputMode
}

func New(placeholder string) Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 2000
	ti.Focus()
	return Model{
		input: ti,
		mode:  InputText,
	}
}

func (m Model) Value() string {
	return m.input.Value()
}
