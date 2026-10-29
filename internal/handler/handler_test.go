package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-hub-platform/internal/config"
	"ai-hub-platform/internal/service"
)

func TestHandler(t *testing.T) {
	cfg := &config.Config{}
	summarizer := service.NewSummarizer(cfg)
	h := NewHandler(summarizer)

	t.Run("GET /healthz returns ok status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		h.Healthz(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("expected status 'ok', got %q", body["status"])
		}
	})

	t.Run("POST /api/summarize success", func(t *testing.T) {
		payload := []byte(`{"text":"Testing summarization payload", "language":"ru"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/summarize", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		h.Summarize(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res["summary"] == "" {
			t.Errorf("expected non-empty summary")
		}
	})

	t.Run("POST /api/event records structured event", func(t *testing.T) {
		payload := []byte(`{"type":"page_view", "source":"landing", "metadata":{"browser":"chrome"}}`)
		req := httptest.NewRequest(http.MethodPost, "/api/event", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		h.Event(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected status 202, got %d", rec.Code)
		}
	})
}
