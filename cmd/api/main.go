package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"ai-hub-platform/internal/cache"
	"ai-hub-platform/internal/config"
	"ai-hub-platform/internal/database"
	"ai-hub-platform/internal/handler"
	"ai-hub-platform/internal/service"
)

func main() {
	// Initialize structured JSON logging (cloud-native standard for Kubernetes)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()
	slog.Info("starting api server",
		"port", cfg.Port,
		"llm_enabled", cfg.LLMAPIKey != "",
		"postgres_host", cfg.PostgresHost,
		"redis_addr", cfg.RedisAddr,
	)

	// Initialize PostgreSQL pool
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	db, err := database.NewPostgres(initCtx, cfg.PostgresDSN())
	if err != nil {
		slog.Error("failed to connect to postgresql", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Automatically create 'summaries' table if it does not exist
	if err := db.InitSchema(initCtx); err != nil {
		slog.Error("failed to initialize postgres schema", "error", err)
		os.Exit(1)
	}

	// Initialize Redis cache
	redisCache := cache.NewRedis(cfg.RedisAddr, cfg.RedisPassword)
	defer func() {
		if err := redisCache.Close(); err != nil {
			slog.Warn("error closing redis connection", "error", err)
		}
	}()

	// Initialize domain services and handlers
	summarizer := service.NewSummarizer(cfg)
	h := handler.NewHandler(summarizer, db, redisCache)

	// Router setup
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS support
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Core & DevOps endpoints
	r.Get("/healthz", h.Healthz)
	r.Handle("/metrics", promhttp.Handler())

	// API routes (API is solely responsible for /api/*, /healthz and /metrics)
	r.Route("/api", func(r chi.Router) {
		r.Post("/summarize", h.Summarize)
		r.Post("/event", h.Event)
	})

	// Server with graceful shutdown
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Goroutine for listening
	go func() {
		slog.Info(fmt.Sprintf("HTTP server listening on http://0.0.0.0:%s", cfg.Port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed to listen", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT or SIGTERM (K8s pod lifecycle)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutting down server...", "signal", sig.String())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("server gracefully stopped")
	}
}
