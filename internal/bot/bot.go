package bot

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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"ai-hub-platform/internal/config"
	"ai-hub-platform/internal/model"
)

// Bot represents the Telegram bot service.
type Bot struct {
	cfg        *config.Config
	api        *tgbotapi.BotAPI
	httpClient *http.Client
}

// NewBot initializes a new Telegram bot instance.
func NewBot(cfg *config.Config) (*Bot, error) {
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	api, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize telegram bot: %w", err)
	}

	slog.Info("authorized on account", "username", api.Self.UserName)

	return &Bot{
		cfg: cfg,
		api: api,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Start begins processing updates until context is canceled.
func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	slog.Info("telegram bot started listening for updates")

	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping telegram bot update loop")
			b.api.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil {
				continue
			}

			go b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	if text == "" {
		reply := tgbotapi.NewMessage(chatID, "Пожалуйста, отправьте текстовое сообщение для суммаризации.")
		_, _ = b.api.Send(reply)
		return
	}

	// Handle commands
	if msg.IsCommand() {
		switch msg.Command() {
		case "start", "help":
			reply := tgbotapi.NewMessage(chatID,
				"👋 Привет! Я бот сервиса AI Hub Platform.\n\n"+
					"Отправь мне любой текст (статью, заметку, документ), и я составлю его краткую выжимку с помощью нашего API.")
			_, _ = b.api.Send(reply)
			return
		}
	}

	// Send typing action
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, _ = b.api.Send(action)

	// Call API /api/summarize
	resp, err := b.requestSummarize(text)
	if err != nil {
		slog.Error("failed to summarize text via API", "error", err, "chat_id", chatID)
		reply := tgbotapi.NewMessage(chatID, "❌ Произошла ошибка при обращении к API сервиса. Попробуйте позже.")
		_, _ = b.api.Send(reply)
		return
	}

	replyText := fmt.Sprintf(
		"📝 *Результат суммаризации* (провайдер: `%s`):\n\n%s\n\n_Символов: было %d ➔ стало %d_",
		resp.Provider,
		resp.Summary,
		resp.OriginalLength,
		resp.SummaryLength,
	)

	reply := tgbotapi.NewMessage(chatID, replyText)
	reply.ParseMode = "Markdown"
	if _, err := b.api.Send(reply); err != nil {
		// Fallback without markdown in case of formatting characters
		reply.ParseMode = ""
		reply.Text = fmt.Sprintf("Результат суммаризации:\n\n%s", resp.Summary)
		_, _ = b.api.Send(reply)
	}

	// Send analytical event to API asynchronously
	go b.sendAnalyticsEvent("bot_summarize_success", map[string]any{
		"chat_id":         chatID,
		"user":            msg.From.UserName,
		"original_length": resp.OriginalLength,
		"provider":        resp.Provider,
	})
}

func (b *Bot) requestSummarize(text string) (*model.SummarizeResponse, error) {
	url := fmt.Sprintf("%s/api/summarize", strings.TrimRight(b.cfg.APIURL, "/"))

	payload, err := json.Marshal(model.SummarizeRequest{
		Text:     text,
		Language: "ru",
	})
	if err != nil {
		return nil, fmt.Errorf("json marshal error: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("request create error: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request execution error: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("api status %d: %s", res.StatusCode, string(body))
	}

	var resp model.SummarizeResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	return &resp, nil
}

func (b *Bot) sendAnalyticsEvent(eventType string, metadata map[string]any) {
	url := fmt.Sprintf("%s/api/event", strings.TrimRight(b.cfg.APIURL, "/"))

	payload, err := json.Marshal(model.EventRequest{
		Type:      eventType,
		Source:    "telegram_bot",
		Timestamp: time.Now().UTC(),
		Metadata:  metadata,
	})
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := b.httpClient.Do(req)
	if err == nil {
		_ = res.Body.Close()
	}
}
