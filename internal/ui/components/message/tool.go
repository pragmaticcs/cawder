package message

import (
	"fmt"
	"strings"

	"github.com/pragmaticcs/cawder/internal/ui/styles"
)

type ToolInfo struct {
	CallID  string
	Name    string
	MainArg string
	Result  string
	Status  ToolStatus
}

func NewToolCall(id, name string) Model {
	return Model{
		Type: MessageToolCall,
		Tool: ToolInfo{
			CallID: id,
			Name:   name,
			Status: ToolStatusRunning,
		},
	}
}

func (m *Model) CompleteToolCall(mainArg, result string, isError bool) {
	if mainArg != "" {
		m.Tool.MainArg = mainArg
	}
	m.Tool.Result = result
	if isError {
		m.Tool.Status = ToolStatusError
	} else {
		m.Tool.Status = ToolStatusDone
	}
}

func (t ToolInfo) View() string {
	header := fmt.Sprintf("%s(%s)", t.Name, t.MainArg)
	switch t.Status {
	case ToolStatusRunning:
		return styles.ToolCallPendingStyle.Render(header)

	case ToolStatusError:
		head := styles.ToolCallErrorStyle.Render(header)
		if strings.TrimSpace(t.Result) == "" {
			return head
		}
		return head + "\n" + styles.ToolErrorResultStyle.Render(t.Result)

	default:
		head := styles.ToolCallStyle.Render(header)
		if strings.TrimSpace(t.Result) == "" {
			return head
		}
		return head + "\n" + styles.ToolResultStyle.Render(t.Result)
	}
}
