package picker

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticcs/cawder/internal/ui/styles"
)

type SelectedMsg[T any] struct{ Value T }
type CancelledMsg struct{}

type entry[T any] struct {
	value T
	title string
	desc  string
}

func (e entry[T]) Title() string {
	return e.title
}
func (e entry[T]) Description() string {
	return e.desc
}
func (e entry[T]) FilterValue() string {
	return e.title
}

type Model[T any] struct {
	list  list.Model
	ready bool
}

func (m Model[T]) Init() tea.Cmd {
	return nil
}

func New[T any](title string, values []T, titleFn, descFn func(T) string, width, height int) Model[T] {
	items := make([]list.Item, len(values))
	for i, v := range values {
		items[i] = entry[T]{
			value: v,
			title: titleFn(v),
			desc:  descFn(v),
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles = styles.ListItemStyles

	l := list.New(items, delegate, width, height)
	l.Title = title
	l.Styles = styles.ListStyles
	l.Help.Styles = styles.ListHelpStyles
	l.SetShowStatusBar(false)

	return Model[T]{
		list:  l,
		ready: true,
	}
}
