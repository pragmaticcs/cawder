package message

import (
	"fmt"
	"strings"

	"github.com/pragmaticcs/cawder/internal/ui/styles"
)

func (m Model) View(renderers styles.Renderers) string {
	switch m.Type {
	case MessageUserResponse:
		text, _ := renderers.User.Render(m.Content)
		return styles.UserResponseStyle.Render(strings.TrimRight(text, "\n"))
	case MessageAgentResponse:
		text, _ := renderers.Agent.Render(m.Content)
		return styles.AgentResponseStyle.Render(strings.TrimRight(text, "\n"))
	case MessageAgentReasoning:
		text, _ := renderers.Reasoning.Render(m.Content)
		return styles.AgentReasoningStyle.Render(strings.TrimRight(text, "\n"))
	case MessageToolCall:
		return m.Tool.View()
	case MessageError:
		text := fmt.Sprintf("Error: %s", m.Content)
		return styles.ErrorStyle.Render(text)
	default:
		return "loading..."
	}
}
