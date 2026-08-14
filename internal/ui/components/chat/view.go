package chat

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}
	return m.viewport.View()
}
