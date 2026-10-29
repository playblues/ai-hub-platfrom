package config

import (
	"os"
	"strconv"
)

// Config holds configuration parameters for the API and Bot services.
type Config struct {
	// API Server settings
	Port     string
	LogLevel string

	// LLM settings
	LLMAPIKey string
	LLMAPIURL string
	LLMModel  string

	// Telegram Bot settings
	TelegramBotToken string
	APIURL           string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		LLMAPIKey:        getEnv("LLM_API_KEY", ""),
		LLMAPIURL:        getEnv("LLM_API_URL", "https://api.openai.com/v1/chat/completions"),
		LLMModel:         getEnv("LLM_MODEL", "gpt-4o-mini"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		APIURL:           getEnv("API_URL", "http://localhost:8080"),
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// GetEnvInt reads an int from environment variable or returns default.
func GetEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
