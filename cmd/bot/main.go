package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ai-hub-platform/internal/bot"
	"ai-hub-platform/internal/config"
)

func main() {
	// Initialize structured JSON logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()
	slog.Info("starting telegram bot service", "api_url", cfg.APIURL)

	if cfg.TelegramBotToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN environment variable is not set. Please set it before running the bot.")
		os.Exit(1)
	}

	tgBot, err := bot.NewBot(cfg)
	if err != nil {
		slog.Error("failed to initialize telegram bot", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := tgBot.Start(ctx); err != nil {
		slog.Error("telegram bot stopped with error", "error", err)
		os.Exit(1)
	}

	slog.Info("telegram bot stopped successfully")
}
