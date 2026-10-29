package model

import "time"

// SummarizeRequest represents the incoming payload for text summarization.
type SummarizeRequest struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

// SummarizeResponse represents the response containing the summarized text.
type SummarizeResponse struct {
	Summary        string `json:"summary"`
	OriginalLength int    `json:"original_length"`
	SummaryLength  int    `json:"summary_length"`
	Provider       string `json:"provider"` // "mock" or "llm"
}

// EventRequest represents analytics events received from landing page or bot.
type EventRequest struct {
	Type      string         `json:"type"`                // e.g. "page_view", "summarize_click"
	Source    string         `json:"source"`              // e.g. "landing", "telegram_bot"
	Timestamp time.Time      `json:"timestamp,omitempty"` // Event timestamp
	Metadata  map[string]any `json:"metadata,omitempty"`  // Arbitrary context
}

// HealthResponse represents the healthcheck response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}
