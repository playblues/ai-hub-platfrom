package service

import (
	"context"
	"testing"

	"ai-hub-platform/internal/config"
)

func TestMockSummarizer(t *testing.T) {
	cfg := &config.Config{
		LLMAPIKey: "", // triggers mock
	}
	s := NewSummarizer(cfg)

	t.Run("empty text returns error", func(t *testing.T) {
		_, err := s.Summarize(context.Background(), "   ", "ru")
		if err == nil {
			t.Fatal("expected error on empty text, got nil")
		}
	})

	t.Run("valid russian mock summarization", func(t *testing.T) {
		res, err := s.Summarize(context.Background(), "Длинный текст о микросервисах и Kubernetes", "ru")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Provider != "mock" {
			t.Errorf("expected provider 'mock', got %s", res.Provider)
		}
		if res.OriginalLength == 0 || res.SummaryLength == 0 {
			t.Errorf("invalid lengths: orig=%d, sum=%d", res.OriginalLength, res.SummaryLength)
		}
		if res.Summary == "" {
			t.Error("summary is empty")
		}
	})

	t.Run("valid english mock summarization", func(t *testing.T) {
		res, err := s.Summarize(context.Background(), "Long text about microservices and Kubernetes cluster", "en")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Provider != "mock" {
			t.Errorf("expected provider 'mock', got %s", res.Provider)
		}
	})
}
