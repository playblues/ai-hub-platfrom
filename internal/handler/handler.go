package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"ai-hub-platform/internal/cache"
	"ai-hub-platform/internal/database"
	"ai-hub-platform/internal/model"
	"ai-hub-platform/internal/service"
)

// Handler holds dependencies for HTTP route handlers.
type Handler struct {
	summarizer service.Summarizer
	db         database.DB
	cache      cache.Cache
}

// NewHandler creates a new Handler instance with db and cache dependencies.
func NewHandler(summarizer service.Summarizer, db database.DB, cache cache.Cache) *Handler {
	return &Handler{
		summarizer: summarizer,
		db:         db,
		cache:      cache,
	}
}

// Healthz responds with { "status": "ok" } only when database is healthy.
// Critical for Kubernetes liveness/readiness probes.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.db != nil {
		if err := h.db.Ping(r.Context()); err != nil {
			slog.Error("healthz check failed: database unreachable", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  "database connection failed",
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// Summarize handles POST /api/summarize with Redis caching and PostgreSQL persistence.
func (h *Handler) Summarize(w http.ResponseWriter, r *http.Request) {
	var req model.SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("failed to decode summarize request", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if req.Text == "" {
		writeJSONError(w, http.StatusBadRequest, "field 'text' is required")
		return
	}

	if req.Language == "" {
		req.Language = "ru"
	}

	// 1. Check Redis cache first
	if h.cache != nil {
		if cached, err := h.cache.GetSummary(r.Context(), req.Text, req.Language); err == nil && cached != nil {
			slog.Info("cache hit for summary request", "text_len", len([]rune(req.Text)))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(cached)
			return
		}
	}

	// 2. Generate summary via service
	resp, err := h.summarizer.Summarize(r.Context(), req.Text, req.Language)
	if err != nil {
		slog.Error("summarization error", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to process text")
		return
	}

	// 3. Persist summary to PostgreSQL
	if h.db != nil {
		if id, dbErr := h.db.SaveSummary(r.Context(), req.Text, resp.Summary, req.Language); dbErr != nil {
			slog.Error("failed to save summary in database", "error", dbErr)
		} else {
			slog.Info("summary saved to postgres", "id", id)
		}
	}

	// 4. Save to Redis cache for future requests (24 hour TTL)
	if h.cache != nil {
		if cacheErr := h.cache.SetSummary(r.Context(), req.Text, req.Language, resp, 24*time.Hour); cacheErr != nil {
			slog.Warn("failed to cache summary in redis", "error", cacheErr)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Event handles POST /api/event and logs the event as structured JSON.
func (h *Handler) Event(w http.ResponseWriter, r *http.Request) {
	var req model.EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("failed to decode event payload", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid event JSON payload")
		return
	}

	if req.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "field 'type' is required")
		return
	}

	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now().UTC()
	}

	// Structured JSON logging
	slog.Info("analytics_event",
		"event_type", req.Type,
		"source", req.Source,
		"timestamp", req.Timestamp.Format(time.RFC3339),
		"metadata", req.Metadata,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "recorded",
		"type":   req.Type,
		"time":   req.Timestamp,
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
