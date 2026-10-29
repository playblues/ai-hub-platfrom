package config

import (
	"fmt"
	"net/url"
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

	// PostgreSQL settings
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	// Redis settings
	RedisAddr     string
	RedisPassword string

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
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:       getEnv("POSTGRES_DB", "aihub"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		APIURL:           getEnv("API_URL", "http://localhost:8080"),
	}
}

// PostgresDSN constructs connection string for pgx/v5.
func (c *Config) PostgresDSN() string {
	userInfo := url.UserPassword(c.PostgresUser, c.PostgresPassword)
	return fmt.Sprintf(
		"postgres://%s@%s:%s/%s?sslmode=%s",
		userInfo.String(),
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDB,
		c.PostgresSSLMode,
	)
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
