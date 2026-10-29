# AI Hub Platform — Архитектурный прототип

Микросервисный каркас на **Go (Golang)** с поддержкой **PostgreSQL** и **Redis**, подготовленный для деплоя в Kubernetes и настройки DevOps-пайплайна.

## 📁 Структура проекта

```text
ai-hub-platform/
├── cmd/
│   ├── api/
│   │   └── main.go           # Точка входа HTTP API сервера
│   └── bot/
│       └── main.go           # Точка входа Telegram-бота
├── internal/
│   ├── bot/                  # Логика работы Telegram-бота и интеграции с API
│   ├── cache/                # Кэширование запросов в Redis (go-redis/v9)
│   ├── config/               # Загрузка и типизация переменных окружения
│   ├── database/             # PostgreSQL пул коннектов (pgx/v5) и авто-миграция
│   ├── handler/              # HTTP-хэндлеры (healthz, metrics, summarize, event)
│   ├── model/                # DTO структуры запросов и ответов
│   └── service/              # Бизнес-логика (Mock и реальный LLM вызов)
├── web/
│   ├── index.html            # Интерактивный лендинг на Tailwind CDN
│   └── nginx.conf            # Конфигурация Nginx для раздачи статики и проксирования API
├── .env.example              # Пример переменных окружения
├── go.mod                    # Модуль ai-hub-platform и зафиксированные зависимости
├── go.sum
└── README.md
```

---

## ⚙️ Переменные окружения

Конфигурация сервисов осуществляется строго через переменные окружения:

| Переменная | По умолчанию | Описание |
| :--- | :--- | :--- |
| `PORT` | `8080` | Порт HTTP API сервера |
| `LOG_LEVEL` | `info` | Уровень логирования (`info`, `warn`, `error`) |
| `POSTGRES_HOST` | `localhost` | Хост PostgreSQL |
| `POSTGRES_PORT` | `5432` | Порт PostgreSQL |
| `POSTGRES_USER` | `postgres` | Пользователь PostgreSQL |
| `POSTGRES_PASSWORD` | `postgres` | Пароль к PostgreSQL |
| `POSTGRES_DB` | `aihub` | Название базы данных PostgreSQL |
| `POSTGRES_SSLMODE` | `disable` | SSL режим PostgreSQL |
| `REDIS_ADDR` | `localhost:6379` | Адрес сервера Redis |
| `REDIS_PASSWORD` | *пусто* | Пароль к Redis |
| `LLM_API_KEY` | *пусто* | API-ключ к LLM (если пуст, отдается мок-ответ) |
| `LLM_API_URL` | `https://api.openai.com/v1/chat/completions` | Эндпоинт OpenAI-совместимого API |
| `LLM_MODEL` | `gpt-4o-mini` | Название используемой модели |
| `TELEGRAM_BOT_TOKEN` | *пусто* | Токен бота от [@BotFather](https://t.me/BotFather) |
| `API_URL` | `http://localhost:8080` | URL API-сервиса, используемый ботом |

---

## 🗄️ Базы данных и кэш

### PostgreSQL (`pgx/v5`)
- При старте API автоматически создает таблицу `summaries`, если её еще нет:
  ```sql
  CREATE TABLE IF NOT EXISTS summaries (
      id BIGSERIAL PRIMARY KEY,
      text TEXT NOT NULL,
      result TEXT NOT NULL,
      language VARCHAR(10) NOT NULL,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
  );
  ```
- Каждый успешный запрос на суммаризацию сохраняется в таблицу `summaries`.
- Эндпоинт `GET /healthz` выполняет `db.Ping()`. Если база недоступна, возвращается `503 Service Unavailable`, что сигнализирует Kubernetes Liveness/Readiness probe о неготовности пода.

### Redis (`go-redis/v9`)
- Перед отправкой запроса на суммаризацию проверяется наличие результата в Redis (ключ `summary:<sha256>`).
- При **cache hit** результат мгновенно возвращается клиенту с признаком `"provider": "cache"`.
- При **cache miss** результат генерируется, сохраняется в PostgreSQL и кэшируется в Redis со сроком жизни (TTL) 24 часа.

---

## 🌐 Фронтенд и Nginx (`web/nginx.conf`)
- Раздача статики полностью вынесена из Go API. API отвечает исключительно за `/api/*`, `/healthz` и `/metrics`.
- В папке `web/` расположен `nginx.conf`:
  - Раздает `index.html` на порту 80.
  - Проксирует `/api/` на `http://api:8080/api/`.

---

## 🚀 Инструкция по сборке и запуску

```bash
# Проверка сборки API:
go build ./cmd/api

# Проверка сборки бота:
go build ./cmd/bot

# Запуск тестов:
go test -v ./...
```
