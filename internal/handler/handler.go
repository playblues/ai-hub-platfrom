package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"ai-hub-platform/internal/model"
	"ai-hub-platform/internal/service"
)

// Handler holds dependencies for HTTP route handlers.
type Handler struct {
	summarizer service.Summarizer
}

// NewHandler creates a new Handler instance.
func NewHandler(summarizer service.Summarizer) *Handler {
	return &Handler{
		summarizer: summarizer,
	}
}

// Healthz responds with { "status": "ok" } for Kubernetes liveness/readiness probes.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// Summarize handles POST /api/summarize.
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

	resp, err := h.summarizer.Summarize(r.Context(), req.Text, req.Language)
	if err != nil {
		slog.Error("summarization error", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to process text")
		return
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

	// Structured JSON logging as required
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
		"status":  "recorded",
		"type":    req.Type,
		"time":    req.Timestamp,
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
