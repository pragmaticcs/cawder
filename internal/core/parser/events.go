package parser

type ParserEvent int

const (
	ParserEventText ParserEvent = iota
	ParserEventReasoning
	ParserEventToolCallStart
	ParserEventToolCall
	ParserEventToolCallError
	ParserEventDone
)

type ToolCallSource string

const (
	SourceNative ToolCallSource = "native"
	SourceText   ToolCallSource = "text"
)

type ToolCall struct {
	Index      int
	ID         string
	Name       string
	Arguments  string
	Source     ToolCallSource
	Convention string
}

type Event struct {
	Type         ParserEvent
	Text         string
	ToolCall     ToolCall
	FinishReason string
	Err          string
}
