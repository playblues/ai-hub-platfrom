# AI Hub Platform — Архитектурный прототип

Микросервисный каркас на **Go (Golang)**, подготовленный для деплоя в Kubernetes и настройки DevOps-пайплайна.

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
│   ├── config/               # Загрузка и типизация переменных окружения
│   ├── handler/              # HTTP-хэндлеры (healthz, metrics, summarize, event)
│   ├── model/                # DTO структуры запросов и ответов
│   └── service/              # Бизнес-логика (Mock и реальный LLM вызов)
├── web/
│   └── index.html            # Интерактивный лендинг на Tailwind CDN
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
| `LLM_API_KEY` | *пусто* | API-ключ к LLM (если пуст, отдается мок-ответ) |
| `LLM_API_URL` | `https://api.openai.com/v1/chat/completions` | Эндпоинт OpenAI-совместимого API |
| `LLM_MODEL` | `gpt-4o-mini` | Название используемой модели |
| `TELEGRAM_BOT_TOKEN` | *пусто* | Токен бота от [@BotFather](https://t.me/BotFather) |
| `API_URL` | `http://localhost:8080` | URL API-сервиса, используемый ботом |

---

## 🚀 Инструкция по запуску

### 1. Запуск HTTP API сервера

```bash
# Из корня проекта:
go run ./cmd/api
```

Сервер запустится на `http://localhost:8080`.
При открытии `http://localhost:8080` в браузере откроется интерактивный лендинг.

### 2. Запуск Telegram-бота

В отдельном терминале:

```bash
# Linux/macOS:
export TELEGRAM_BOT_TOKEN="ВАШ_ТОКЕН_БОТА"
export API_URL="http://localhost:8080"
go run ./cmd/bot

# Windows (PowerShell):
$env:TELEGRAM_BOT_TOKEN="ВАШ_ТОКЕН_БОТА"
$env:API_URL="http://localhost:8080"
go run ./cmd/bot
```

### 3. Запуск тестов

```bash
go test -v ./...
```

---

## 📡 Эндпоинты API

### `GET /healthz`
Используется Kubernetes для **Liveness & Readiness Probes**.
```bash
curl http://localhost:8080/healthz
```
Ответ:
```json
{
  "status": "ok"
}
```

### `GET /metrics`
Экспорт стандартных метрик приложения в формате **Prometheus**.
```bash
curl http://localhost:8080/metrics
```

### `POST /api/summarize`
Суммаризация переданного текста. При наличии `LLM_API_KEY` обращается к LLM, иначе возвращает структурированный мок.
```bash
curl -X POST http://localhost:8080/api/summarize \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Kubernetes — это платформа с открытым исходным кодом для управления контейнеризированными приложениями.",
    "language": "ru"
  }'
```
Ответ:
```json
{
  "summary": "[Мок-суммаризация ИИ] Главная мысль: «Kubernetes — это платформа...». Исходный текст содержал 105 символов и был успешно обработан.",
  "original_length": 105,
  "summary_length": 139,
  "provider": "mock"
}
```

### `POST /api/event`
Принимает событие аналитики с лендинга или бота и пишет его в stdout в виде структурированного JSON (`log/slog`).
```bash
curl -X POST http://localhost:8080/api/event \
  -H "Content-Type: application/json" \
  -d '{
    "type": "page_view",
    "source": "landing",
    "metadata": { "browser": "Chrome" }
  }'
```

---

## ☸️ Готовность к Kubernetes и DevOps

1. **Structured JSON Logs**: логирование выполнено через стандартный `log/slog` в формате JSON в stdout (идеально парсится Fluentbit/Vector/Promtail).
2. **Graceful Shutdown**: обработка сигналов `SIGINT` и `SIGTERM` для плавного вывода подов из балансировки без потери запросов.
3. **12-Factor App**: конфигурация строго изолирована в переменных окружения.
4. **Легковесность**: компилируемые Go-бинарники без внешних runtime-зависимостей, готовые для упаковки в образы `scratch` или `alpine`.
