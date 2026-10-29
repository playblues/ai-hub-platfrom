package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-hub-platform/internal/config"
	"ai-hub-platform/internal/model"
	"ai-hub-platform/internal/service"
)

type mockDB struct {
	pingErr error
	saved   bool
}

func (m *mockDB) InitSchema(ctx context.Context) error { return nil }
func (m *mockDB) SaveSummary(ctx context.Context, text, result, language string) (int64, error) {
	m.saved = true
	return 1, nil
}
func (m *mockDB) Ping(ctx context.Context) error { return m.pingErr }
func (m *mockDB) Close()                         {}

type mockCache struct {
	cached *model.SummarizeResponse
	stored bool
}

func (m *mockCache) GetSummary(ctx context.Context, text, lang string) (*model.SummarizeResponse, error) {
	return m.cached, nil
}
func (m *mockCache) SetSummary(ctx context.Context, text, lang string, resp *model.SummarizeResponse, ttl time.Duration) error {
	m.stored = true
	return nil
}
func (m *mockCache) Ping(ctx context.Context) error { return nil }
func (m *mockCache) Close() error                   { return nil }

func TestHandler(t *testing.T) {
	cfg := &config.Config{}
	summarizer := service.NewSummarizer(cfg)

	t.Run("GET /healthz returns ok status when DB is healthy", func(t *testing.T) {
		db := &mockDB{pingErr: nil}
		h := NewHandler(summarizer, db, nil)

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

	t.Run("GET /healthz returns 503 when DB is down", func(t *testing.T) {
		db := &mockDB{pingErr: errors.New("connection refused")}
		h := NewHandler(summarizer, db, nil)

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		h.Healthz(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", rec.Code)
		}

		var body map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body["status"] != "error" {
			t.Errorf("expected status 'error', got %q", body["status"])
		}
	})

	t.Run("POST /api/summarize saves to DB and cache", func(t *testing.T) {
		db := &mockDB{}
		cache := &mockCache{}
		h := NewHandler(summarizer, db, cache)

		payload := []byte(`{"text":"Testing summarization payload", "language":"ru"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/summarize", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		h.Summarize(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res model.SummarizeResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.Summary == "" {
			t.Errorf("expected non-empty summary")
		}

		if !db.saved {
			t.Error("expected summary to be saved in postgres")
		}
		if !cache.stored {
			t.Error("expected summary to be saved in redis cache")
		}
	})

	t.Run("POST /api/summarize serves from Redis cache on cache hit", func(t *testing.T) {
		db := &mockDB{}
		cache := &mockCache{
			cached: &model.SummarizeResponse{
				Summary:        "Cached summary result",
				OriginalLength: 100,
				SummaryLength:  22,
				Provider:       "cache",
			},
		}
		h := NewHandler(summarizer, db, cache)

		payload := []byte(`{"text":"Already cached text", "language":"ru"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/summarize", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		h.Summarize(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res model.SummarizeResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		if res.Provider != "cache" {
			t.Errorf("expected provider 'cache', got %s", res.Provider)
		}
		if res.Summary != "Cached summary result" {
			t.Errorf("unexpected summary: %s", res.Summary)
		}
		if db.saved {
			t.Error("expected DB save to be skipped on cache hit")
		}
	})

	t.Run("POST /api/event records structured event", func(t *testing.T) {
		h := NewHandler(summarizer, nil, nil)
		payload := []byte(`{"type":"page_view", "source":"landing", "metadata":{"browser":"chrome"}}`)
		req := httptest.NewRequest(http.MethodPost, "/api/event", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		h.Event(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected status 202, got %d", rec.Code)
		}
	})
}
