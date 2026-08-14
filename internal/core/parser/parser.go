package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
)

type Convention struct {
	Name  string
	Start string
	End   string
}

var DefaultConventions = []Convention{
	{Name: "hermes", Start: "<tool_call>", End: "</tool_call>"},
	{Name: "mistral", Start: "[TOOL_CALLS]", End: ""},
}

const nativeConventionName = "native"
const DefaultMaxCaptureBytes = 8 * 1024 * 1024

func longestStartPrefixSuffix(s, marker string) int {
	n := min(len(s), len(marker))
	for l := n; l > 0; l-- {
		if s[len(s)-l:] == marker[:l] {
			return l
		}
	}
	return 0
}

type jsonBalanceScanner struct {
	depth   int
	inStr   bool
	escape  bool
	started bool
}

func (b *jsonBalanceScanner) feed(s string) int {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if b.inStr {
			switch {
			case b.escape:
				b.escape = false
			case c == '\\':
				b.escape = true
			case c == '"':
				b.inStr = false
			}
			continue
		}
		switch c {
		case '"':
			b.inStr = true
		case '{', '[':
			b.depth++
			b.started = true
		case '}', ']':
			if b.depth > 0 {
				b.depth--
				if b.started && b.depth == 0 {
					return i
				}
			}
		}
	}
	return -1
}

type ToolCallDeltaRaw struct {
	Index     int64
	ID        string
	Name      string
	Arguments string
}

type scanState int

const (
	stateScanning scanState = iota
	stateCapturing
)

type nativeCall struct {
	id, name, args string
	started        bool
	done           bool
}

type Parser struct {
	conventions     []Convention
	MaxCaptureBytes int

	// text-scanning state
	state          scanState
	pending        string
	activeConv     *Convention
	capture        strings.Builder
	captureScanned int
	balancer       jsonBalanceScanner
	textSeq        int

	// native tool_calls state
	nativeCalls   map[int64]*nativeCall
	nativeOrder   []int64
	haveNative    bool
	lastNativeIdx int64

	finished bool
}

func NewParser(conventions ...Convention) *Parser {
	if conventions == nil {
		conventions = DefaultConventions
	}
	filtered := make([]Convention, 0, len(conventions))
	for _, c := range conventions {
		if c.Start != "" {
			filtered = append(filtered, c)
		}
	}
	return &Parser{
		conventions:     filtered,
		MaxCaptureBytes: DefaultMaxCaptureBytes,
		nativeCalls:     make(map[int64]*nativeCall),
	}
}

func (p *Parser) Reset() {
	convs := p.conventions
	maxBytes := p.MaxCaptureBytes
	*p = Parser{conventions: convs, MaxCaptureBytes: maxBytes, nativeCalls: make(map[int64]*nativeCall)}
}

func (p *Parser) Feed(chunk openai.ChatCompletionChunk) []Event {
	if len(chunk.Choices) == 0 {
		return nil
	}
	choice := chunk.Choices[0]
	delta := choice.Delta

	var toolCalls []ToolCallDeltaRaw
	for _, tc := range delta.ToolCalls {
		toolCalls = append(toolCalls, ToolCallDeltaRaw{
			Index:     tc.Index,
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return p.FeedDelta(delta.Content, extractReasoning(delta), toolCalls, choice.FinishReason)
}

func extractReasoning(delta openai.ChatCompletionChunkChoiceDelta) string {
	for _, key := range []string{"reasoning_content", "reasoning"} {
		f, ok := delta.JSON.ExtraFields[key]
		if !ok {
			continue
		}
		raw := f.Raw()
		if raw == "" || raw == "null" {
			continue
		}
		var s string
		if err := json.Unmarshal([]byte(raw), &s); err == nil && s != "" {
			return s
		}
	}
	return ""
}

func (p *Parser) FeedDelta(content, reasoning string, toolCalls []ToolCallDeltaRaw, finishReason string) []Event {
	var events []Event

	if reasoning != "" {
		events = append(events, Event{Type: ParserEventReasoning, Text: reasoning})
	}
	if content != "" {
		events = append(events, p.scanText(content)...)
	}
	for _, tc := range toolCalls {
		events = append(events, p.feedNativeToolCall(tc)...)
	}
	if finishReason != "" {
		events = append(events, p.finalizeTurn(finishReason)...)
	}
	return events
}

func (p *Parser) Finalize() []Event {
	return p.finalizeTurn("")
}

func (p *Parser) scanText(s string) []Event {
	var events []Event
	for len(s) > 0 {
		if p.state == stateScanning {
			p.pending += s
			s = ""

			startIdx, conv := p.findEarliestStart(p.pending)
			if conv != nil {
				if startIdx > 0 {
					events = append(events, Event{Type: ParserEventText, Text: p.pending[:startIdx]})
				}
				s = p.pending[startIdx+len(conv.Start):]
				p.pending = ""
				p.state = stateCapturing
				p.activeConv = conv
				p.capture.Reset()
				p.captureScanned = 0
				p.balancer = jsonBalanceScanner{}
				continue
			}

			holdBack := p.longestPendingMarkerPrefix()
			if flush := p.pending[:len(p.pending)-holdBack]; flush != "" {
				events = append(events, Event{Type: ParserEventText, Text: flush})
			}
			p.pending = p.pending[len(p.pending)-holdBack:]
			continue
		}

		conv := p.activeConv
		if conv.End != "" {
			p.capture.WriteString(s)
			s = ""
			captured := p.capture.String()
			searchFrom := p.captureScanned - len(conv.End) + 1
			if searchFrom < 0 {
				searchFrom = 0
			}
			if idx := strings.Index(captured[searchFrom:], conv.End); idx >= 0 {
				closeAt := searchFrom + idx
				body, leftover := captured[:closeAt], captured[closeAt+len(conv.End):]
				events = append(events, p.closeCapture(conv, body)...)
				s = leftover
				continue
			}
			p.captureScanned = len(captured)
			if len(captured) > p.MaxCaptureBytes {
				events = append(events, p.abortCapture())
			}
			continue
		}

		if closeAt := p.balancer.feed(s); closeAt >= 0 {
			p.capture.WriteString(s[:closeAt+1])
			leftover := s[closeAt+1:]
			events = append(events, p.closeCapture(conv, p.capture.String())...)
			s = leftover
			continue
		}
		p.capture.WriteString(s)
		s = ""
		if p.capture.Len() > p.MaxCaptureBytes {
			events = append(events, p.abortCapture())
		}
	}
	return events
}

func (p *Parser) abortCapture() Event {
	conv := p.activeConv
	text := conv.Start + p.capture.String()
	p.state = stateScanning
	p.activeConv = nil
	p.capture.Reset()
	p.captureScanned = 0
	return Event{
		Type: ParserEventToolCallError,
		Text: text,
		Err: fmt.Sprintf("parser: %q capture exceeded %d bytes with no closing marker found",
			conv.Name, p.MaxCaptureBytes),
		ToolCall: ToolCall{Convention: conv.Name},
	}
}

func (p *Parser) findEarliestStart(buf string) (int, *Convention) {
	bestIdx := -1
	var best *Convention
	for i := range p.conventions {
		c := &p.conventions[i]
		idx := strings.Index(buf, c.Start)
		if idx < 0 {
			continue
		}
		if bestIdx == -1 || idx < bestIdx || (idx == bestIdx && len(c.Start) > len(best.Start)) {
			bestIdx, best = idx, c
		}
	}
	if best == nil {
		return -1, nil
	}
	tail := buf[bestIdx:]
	for i := range p.conventions {
		o := &p.conventions[i]
		if o == best || len(o.Start) <= len(best.Start) {
			continue
		}
		if len(tail) < len(o.Start) && strings.HasPrefix(o.Start, tail) {
			return -1, nil
		}
	}
	return bestIdx, best
}

func (p *Parser) longestPendingMarkerPrefix() int {
	holdBack := 0
	for i := range p.conventions {
		if n := longestStartPrefixSuffix(p.pending, p.conventions[i].Start); n > holdBack {
			holdBack = n
		}
	}
	return holdBack
}

func (p *Parser) closeCapture(conv *Convention, body string) []Event {
	p.state = stateScanning
	p.activeConv = nil
	p.capture.Reset()
	p.captureScanned = 0

	calls, err := parseToolCallBody(strings.TrimSpace(body))
	if err == nil && len(calls) == 0 {
		err = fmt.Errorf("parser: %q body contained no tool calls", conv.Name)
	}
	if err != nil {
		return []Event{{
			Type:     ParserEventToolCallError,
			Text:     conv.Start + body + conv.End,
			Err:      err.Error(),
			ToolCall: ToolCall{Convention: conv.Name},
		}}
	}

	events := make([]Event, 0, len(calls)*2)
	for _, c := range calls {
		tc := ToolCall{
			Index:      p.textSeq,
			Name:       c.Name,
			Arguments:  c.Arguments,
			Source:     SourceText,
			Convention: conv.Name,
		}
		p.textSeq++
		events = append(events,
			Event{Type: ParserEventToolCallStart, ToolCall: tc},
			Event{Type: ParserEventToolCall, ToolCall: tc},
		)
	}
	return events
}

func (p *Parser) feedNativeToolCall(d ToolCallDeltaRaw) []Event {
	var events []Event

	if p.haveNative && p.lastNativeIdx != d.Index {
		if e := p.finalizeNativeCall(p.lastNativeIdx); e != nil {
			events = append(events, *e)
		}
	}

	c, ok := p.nativeCalls[d.Index]
	if !ok {
		c = &nativeCall{}
		p.nativeCalls[d.Index] = c
		p.nativeOrder = append(p.nativeOrder, d.Index)
	} else if c.done {
		*c = nativeCall{}
	}

	isNew := !c.started
	c.started = true
	if d.ID != "" {
		c.id = d.ID
	}
	c.name += d.Name
	if len(c.args) < p.MaxCaptureBytes {
		c.args += d.Arguments
	}

	p.haveNative = true
	p.lastNativeIdx = d.Index

	if isNew {
		events = append(events, Event{Type: ParserEventToolCallStart, ToolCall: ToolCall{
			Index: int(d.Index), ID: c.id, Name: c.name, Arguments: c.args,
			Source: SourceNative, Convention: nativeConventionName,
		}})
	}
	return events
}

func (p *Parser) finalizeNativeCall(idx int64) *Event {
	c, ok := p.nativeCalls[idx]
	if !ok || c.done || !c.started {
		return nil
	}
	c.done = true
	return &Event{Type: ParserEventToolCall, ToolCall: ToolCall{
		Index: int(idx), ID: c.id, Name: c.name, Arguments: c.args,
		Source: SourceNative, Convention: nativeConventionName,
	}}
}

func (p *Parser) finalizeTurn(finishReason string) []Event {
	if p.finished {
		return nil
	}
	p.finished = true

	var events []Event
	for _, idx := range p.nativeOrder {
		if e := p.finalizeNativeCall(idx); e != nil {
			events = append(events, *e)
		}
	}
	events = append(events, p.flushIncomplete()...)
	events = append(events, Event{Type: ParserEventDone, FinishReason: finishReason})
	return events
}

func (p *Parser) flushIncomplete() []Event {
	var events []Event
	if p.state == stateCapturing {
		conv := p.activeConv
		events = append(events, Event{
			Type:     ParserEventToolCallError,
			Text:     conv.Start + p.capture.String(),
			Err:      fmt.Sprintf("parser: stream ended mid %q tool call", conv.Name),
			ToolCall: ToolCall{Convention: conv.Name},
		})
		p.capture.Reset()
		p.captureScanned = 0
		p.activeConv = nil
		p.state = stateScanning
	}
	if p.pending != "" {
		events = append(events, Event{Type: ParserEventText, Text: p.pending})
		p.pending = ""
	}
	return events
}

type rawCall struct {
	Name      string
	Arguments string
}

func parseToolCallBody(body string) ([]rawCall, error) {
	var generic any
	if err := json.Unmarshal([]byte(body), &generic); err != nil {
		return nil, err
	}
	switch v := generic.(type) {
	case []any:
		calls := make([]rawCall, 0, len(v))
		for _, item := range v {
			c, err := rawCallFromValue(item)
			if err != nil {
				return nil, err
			}
			calls = append(calls, c)
		}
		return calls, nil
	case map[string]any:
		c, err := rawCallFromValue(v)
		if err != nil {
			return nil, err
		}
		return []rawCall{c}, nil
	default:
		return nil, fmt.Errorf("parser: tool call body is a %T, want object or array", v)
	}
}

func rawCallFromValue(v any) (rawCall, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return rawCall{}, fmt.Errorf("parser: tool call entry is a %T, want object", v)
	}
	name, _ := m["name"].(string)
	if name == "" {
		return rawCall{}, fmt.Errorf(`parser: tool call entry missing "name"`)
	}

	argsVal, ok := m["arguments"]
	if !ok {
		argsVal = m["parameters"]
	}
	switch a := argsVal.(type) {
	case nil:
		return rawCall{Name: name, Arguments: "{}"}, nil
	case string:
		return rawCall{Name: name, Arguments: a}, nil
	default:
		b, err := json.Marshal(a)
		if err != nil {
			return rawCall{}, err
		}
		return rawCall{Name: name, Arguments: string(b)}, nil
	}
}
