# streaming-service

Микросервис управления видеотрансляциями PSDS: жизненный цикл сессий, приём потока от клиента (Python/веб) по WebSocket и ретрансляция операторам.

## Назначение

- Создание и завершение сессии трансляции.
- Приём видео/аудио или данных от клиента по WebSocket.
- Ретрансляция потока всем подключённым операторам сессии.
- Состояния сессии: `waiting`, `active`, `finished`.

## API

### REST

- **POST /sessions** — создать сессию (тело: `{"client_id": "uuid"}`). Ответ: `session_id`, `stream_key`, `ws_url`, `status`.
- **DELETE /sessions/:id** — завершить сессию (204).
- **GET /sessions/:id/operators** — список операторов на сессии.

### WebSocket

- **GET /ws/stream/:session_id/:user_id** — подключение к сессии:
  - Если `user_id` совпадает с `client_id` сессии — это источник потока (клиент); все данные от него ретранслируются операторам.
  - Иначе — оператор (получатель потока). При первом подключении оператор добавляется в список участников.

### Health

- **GET /health** — health check.
- **GET /ready** — readiness (k8s).

## Конфигурация

Переменные окружения (см. `.env.example`):

- `APP_HOST`, `HTTP_PORT` (или `APP_PORT`) — хост и порт HTTP (по умолчанию 0.0.0.0:8090).
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_DATABASE`, `DB_SSLMODE` — PostgreSQL.
- `WS_MAX_MESSAGE_SIZE` — макс. размер сообщения WebSocket (по умолчанию 10MB).
- `SESSION_MAX_OPERATORS` — макс. операторов на сессию.
- `WS_BASE_URL` — базовый URL для поля `ws_url` в ответе CreateSession (например `wss://stream.example.com`).

При старте конфиг валидируется (`Validate()`); в production обязателен `DB_PASSWORD`.

## Запуск

Нужен запущенный PostgreSQL. Создайте БД (например `createdb streaming_service`) и задайте переменные в `.env`.

```bash
cp .env.example .env
make run
# или
go run ./cmd/streaming-service
# то же: go run ./cmd/streaming-service api
```

## Cobra-команды

- `streaming-service` / `streaming-service api` — запуск HTTP + WebSocket API (при старте выполняются миграции).
- `streaming-service migrate up` — только применить SQL-миграции из `database/migrations/`.
- `streaming-service seed` — миграции + выполнение `database/seeds/*.sql`.
- `streaming-service command migrate-create <name>` — создать заготовку миграции.

## Миграции

Версионированные SQL-миграции в `database/migrations/` (golang-migrate): `000001_streaming_sessions.up.sql` / `.down.sql`. При старте `api` выполняется `migrate up`.

## Docker

```bash
make docker-build
docker run -p 8090:8090 -e DB_HOST=host.docker.internal streaming-service:latest
```

По умолчанию контейнер запускает `./streaming-service` (т.е. API).

## Структура проекта

- `cmd/streaming-service/main.go` — точка входа (Cobra).
- `cmd/api.go`, `cmd/migrate.go`, `cmd/seed.go` — команды api, migrate, seed.
- `internal/config` — конфиг из env (вложенный DB), `Validate()`, `DSN()`, `DatabaseURL()`.
- `internal/database` — GORM Open(DSN), MigrateUp (golang-migrate), RunSeeds, CreateMigration.
- `internal/application` — NewAPI(cfg): миграции, БД, сервисы, роутер, HTTP-сервер; Run(ctx).
- `internal/model` — сущности GORM (StreamingSession, SessionOperator) и DTO.
- `internal/errs` — сентинель-ошибки (ErrSessionNotFound, ErrTooManyOperators).
- `internal/service` — SessionService, StreamHub.
- `internal/handler`, `internal/router` — REST, WebSocket, health; пути из `pkg/constants`.
