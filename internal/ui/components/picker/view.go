package picker

func (m Model[T]) View() string {
	return m.list.View()
}
