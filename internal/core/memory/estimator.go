package memory

import (
	"sync"
	"unicode/utf8"
)

type TokenEstimator struct {
	charsPerToken float64
	mu            sync.RWMutex
}

func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{
		charsPerToken: 3.0, // Initial estimate for English
	}
}

func (e *TokenEstimator) Estimate(chars int) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return int(float64(chars) / e.charsPerToken)
}

func (e *TokenEstimator) EstimateMessage(msg Message) int {
	return e.Estimate(msg.CharCount())
}

func (e *TokenEstimator) EstimateTurn(turn Turn) int {
	total := 0
	for _, msg := range turn.Messages {
		total += e.EstimateMessage(msg)
	}
	return total
}

func (e *TokenEstimator) EstimateString(s string) int {
	return e.Estimate(utf8.RuneCountInString(s))
}

func (e *TokenEstimator) Update(chars int, promptTokens int) {
	if promptTokens <= 0 || chars <= 0 {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	currentRatio := float64(chars) / float64(promptTokens)
	// EMA
	emaRatio := (e.charsPerToken * 0.8) + (currentRatio * 0.2)
	// Clamp
	e.charsPerToken = min(max(emaRatio, 1.5), 8)
}
