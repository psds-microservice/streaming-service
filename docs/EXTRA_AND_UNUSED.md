# Лишнее в streaming-service (удалено)

Сервис по факту: **REST (sessions) + WebSocket (stream relay)**. Ниже перечислено то, что было удалено как шаблонное/неиспользуемое.

---

## Удалено

| Что | Зачем было | Почему лишнее |
|-----|------------|----------------|
| **internal/grpc/** | gRPC-сервер (ApiService, Health RPC) | В спецификации streaming — только REST + WebSocket. Application не поднимает gRPC. |
| **internal/consumer/** | Очереди (RabbitMQ) | Заглушка. Worker-команда — stub. Никто не вызывает Consumer. |
| **internal/service/service.go** | Общий интерфейс `Service` и `New()` | Реально используются только `SessionService` и `StreamHub`. Этот файл нигде не импортируется. |
| **internal/command/command.go** | Агрегатор CLI-команд | `cmd/command.go` вызывает `database.MigrateUp` и `database.CreateMigration` напрямую. `command.Command` не используется. |
| **internal/dto/dto.go** | ExampleRequest, ExampleResponse | Нужны только для mapper/validator; сами mapper и validator не используются в handlers. |
| **internal/validator/validator.go** | ValidateExample(req) | Ни один handler не вызывает валидатор. Валидация — через Gin binding и сервис. |
| **internal/mapper/mapper.go** | SessionMapper, ToExampleResponse | Ни один handler не вызывает маппер. Ответы собираются в handler/service из model. |
| **internal/handler/healthcheck.go** | Health/Ready как `http.HandlerFunc` | Роутер использует `HealthHandler` из **health.go** (Gin). healthcheck.go — под старый net/http mux шаблона. |
| **pkg/proto/api.proto**, **pkg/gen/proto/** | ApiService (Health RPC) | Нужны только для gRPC-сервера, который мы не запускаем. |

---

Также удалены: **cmd/worker.go**, **cmd/scheduler.go**, **cmd/seeder.go**. Из Makefile убраны цели proto, worker, proto-openapi; test больше не зависит от proto.

---

## Итог

- **Используется:** config, database, errs, application, handler (health.go, session_handler, websocket_handler), model, router, service (session_service, stream_hub, ws_config), pkg/constants.
- **Лишнее для текущего домена:** grpc, consumer, service.go, command (internal), dto, validator, mapper, healthcheck.go, pkg/proto и pkg/gen/proto.

После удаления лишнего сборка и поведение API/WebSocket не меняются.
