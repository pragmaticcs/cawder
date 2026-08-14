package picker

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticcs/cawder/internal/ui/keys"
)

func (m Model[T]) Update(msg tea.Msg) (Model[T], tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(k, keys.Keys.Quit):
			return m, func() tea.Msg { return CancelledMsg{} }
		case key.Matches(k, keys.Keys.Send):
			if e, ok := m.list.SelectedItem().(entry[T]); ok {
				v := e.value
				return m, func() tea.Msg {
					return SelectedMsg[T]{Value: v}
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model[T]) SetSize(w, h int) {
	if !m.ready {
		return
	}
	m.list.SetSize(w, h)
}
