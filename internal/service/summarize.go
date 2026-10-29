package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"ai-hub-platform/internal/config"
	"ai-hub-platform/internal/model"
)

// Summarizer defines the interface for text summarization.
type Summarizer interface {
	Summarize(ctx context.Context, text, lang string) (*model.SummarizeResponse, error)
}

type summarizeService struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewSummarizer creates a new Summarizer service.
func NewSummarizer(cfg *config.Config) Summarizer {
	return &summarizeService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *summarizeService) Summarize(ctx context.Context, text, lang string) (*model.SummarizeResponse, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	if lang == "" {
		lang = "ru"
	}

	// If LLM_API_KEY is configured, attempt real LLM call
	if s.cfg.LLMAPIKey != "" {
		resp, err := s.callLLM(ctx, trimmed, lang)
		if err == nil {
			return resp, nil
		}
		slog.Warn("LLM call failed, falling back to mock", "error", err)
	}

	// Mock response
	return s.mockSummarize(trimmed, lang), nil
}

func (s *summarizeService) mockSummarize(text, lang string) *model.SummarizeResponse {
	runes := []rune(text)
	origLen := len(runes)

	var preview string
	if origLen > 150 {
		preview = string(runes[:150]) + "..."
	} else {
		preview = text
	}

	var summary string
	if lang == "en" {
		summary = fmt.Sprintf(
			"[Mock AI Summary] Key insight: \"%s\". Input text contained %d characters and was successfully analyzed.",
			preview, origLen,
		)
	} else {
		summary = fmt.Sprintf(
			"[Мок-суммаризация ИИ] Главная мысль: «%s». Исходный текст содержал %d символов и был успешно обработан.",
			preview, origLen,
		)
	}

	return &model.SummarizeResponse{
		Summary:        summary,
		OriginalLength: origLen,
		SummaryLength:  len([]rune(summary)),
		Provider:       "mock",
	}
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float32             `json:"temperature,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *summarizeService) callLLM(ctx context.Context, text, lang string) (*model.SummarizeResponse, error) {
	prompt := fmt.Sprintf("Summarize the following text briefly in language '%s':\n\n%s", lang, text)

	reqBody := openAIChatRequest{
		Model: s.cfg.LLMModel,
		Messages: []openAIChatMessage{
			{Role: "system", Content: "You are a concise text summarization assistant."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   300,
		Temperature: 0.3,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.LLMAPIURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.LLMAPIKey)

	res, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d: %s", res.StatusCode, string(bodyBytes))
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return nil, fmt.Errorf("llm error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices from llm")
	}

	resultText := strings.TrimSpace(chatResp.Choices[0].Message.Content)

	return &model.SummarizeResponse{
		Summary:        resultText,
		OriginalLength: len([]rune(text)),
		SummaryLength:  len([]rune(resultText)),
		Provider:       "llm",
	}, nil
}
