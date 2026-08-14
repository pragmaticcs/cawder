package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	"github.com/pragmaticcs/cawder/internal/ui/keys"
	"github.com/pragmaticcs/cawder/internal/ui/styles"
)

func getGreeting() string {
	hour := time.Now().Hour()

	switch {
	case hour < 12:
		return "Good morning!"
	case hour < 18:
		return "Good afternoon!"
	default:
		return "Good evening!"
	}
}

func (m Model) View() string {
	header := styles.HeaderStyle.Render(fmt.Sprintf("Cawder says '%s'", getGreeting()))
	chat := m.chat.View()

	inputWidth := max(m.width-4, 0)
	input := styles.InputStyle.Width(inputWidth).Render(m.input.View())

	switch m.mode {
	case TUIModeChat:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			chat,
			input,
			m.footerView(),
		)
	case TUIModeModelSelect:
		return m.modelPicker.View()
	case TUIModeSessionSelect:
		return m.sessionPicker.View()
	}
	return ""
}

func (m Model) footerView() string {
	helpView := help.New().View(keys.Keys)

	limit := m.selectedModel.Context
	limitStr := formatContextLimit(limit)
	pct := 0.0
	if limit > 0 {
		pct = (float64(m.tokenCount) / float64(limit)) * 100
	}

	statusModel := fmt.Sprintf("model: %s", m.selectedModel.Name)
	statusContext := fmt.Sprintf("ctx: %d/%s (%.0f%%)", m.tokenCount, limitStr, pct)
	statusText := fmt.Sprintf("%s  •  %s", statusModel, statusContext)

	var statusRight string
	if m.responding {
		timeStr := fmt.Sprintf("%.1fs", m.elapsed.Seconds())
		statusText = fmt.Sprintf("generating... (%s)  •  %s", timeStr, statusText)
		styledText := styles.StatusStyle.Render(statusText)
		statusRight = m.spinner.View() + " " + styledText
	} else {
		statusRight = styles.StatusStyle.Render(statusText)
	}

	footer := lipgloss.JoinVertical(lipgloss.Left, statusRight, helpView)

	return styles.FooterStyle.Render(footer)
}

func formatContextLimit(limit int) string {
	if limit >= 1000 {
		if limit%1000 == 0 {
			return fmt.Sprintf("%dk", limit/1000)
		}
		return fmt.Sprintf("%.1fk", float64(limit)/1000.0)
	}
	return fmt.Sprintf("%d", limit)
}
