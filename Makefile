.PHONY: help init build run run-dev migrate migrate-create test test-api test-db \
 version clean lint vet fmt docker-build docker-run docker-compose-up docker-compose-down \
 install-deps health-check update clean tidy bench load-test security-check dev db-init

# Конфигурация
APP_NAME = streaming-service
BIN_DIR = bin
BUILD_INFO = $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT_HASH = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
# Главная цель по умолчанию
.DEFAULT_GOAL := help

## 📚 Помощь
help:
	@echo "🚀 streaming-service - Makefile"
	@echo ""
	@echo "🏗️ Сборка и запуск:"
	@echo "  make build     - Сборка бинарника"
	@echo "  make run       - Сборка и запуск сервера"
	@echo "  make run-dev   - Запуск в режиме разработки"
	@echo "  make dev       - Запуск с hot reload (требуется air)"
	@echo "  make clean     - Очистка сборки"
	@echo ""
	@echo "🔧 Управление:"
	@echo "  make migrate        - Выполнить миграции БД"
	@echo "  make migrate-create - Создать новую миграцию"
	@echo "  make seed           - Применить сиды"
	@echo "  make db-init        - Миграции + сиды"
	@echo "  make health-check   - Проверить здоровье сервиса"
	@echo ""
	@echo "🧪 Тестирование и качество:"
	@echo "  make test           - Запуск всех тестов"
	@echo "  make test-api       - Тестирование API (curl health)"
	@echo "  make bench          - Бенчмарки"
	@echo "  make lint / vet / fmt / security-check"
	@echo ""
	@echo "🐳 Docker: make docker-build, docker-run, docker-compose-up"
	@echo ""

## 🏗️ Сборка и запуск
build:
	@echo "🔨 Building $(APP_NAME)..."
	mkdir -p $(BIN_DIR)
	go build -ldflags="-X 'main.Version=$(BUILD_INFO)' \
		-X 'main.Commit=$(COMMIT_HASH)' \
		-X 'main.BuildDate=$(BUILD_DATE)'" \
		-o $(BIN_DIR)/$(APP_NAME) ./cmd/streaming-service
	@echo "✅ Build complete: $(BIN_DIR)/$(APP_NAME)"

run: build
	@echo "🚀 Starting API server..."
	@echo "Server will be available at: http://localhost:8090"
	@echo "Health check: http://localhost:8090/health"
	@echo ""
	@cd $(BIN_DIR) && ./$(APP_NAME) api

run-dev:
	@echo "🚀 Starting in development mode..."
	@echo "For hot reload use: make dev"
	go run ./cmd/streaming-service api

dev:
	@echo "🔥 Starting API with hot reload..."
	@if command -v air > /dev/null; then \
		air -c .air.toml; \
	else \
		echo "⚠ air is not installed. Install: go install github.com/cosmtrek/air@latest"; \
		echo "Running without hot reload..."; \
		make run-dev; \
	fi

## 🔧 Управление
migrate: build
	@echo "🔄 Running migrations..."
	@cd $(BIN_DIR) && ./$(APP_NAME) migrate up

migrate-create: build
	@echo "📝 Creating migration..."
	@read -p "Enter migration name: " name; \
	cd $(BIN_DIR) && ./$(APP_NAME) command migrate-create $$name

seed: build
	@echo "🌱 Running seeds..."
	@cd $(BIN_DIR) && ./$(APP_NAME) seed

db-init: build
	@echo "🗄️ DB init (migrate + seed)..."
	@cd $(BIN_DIR) && ./$(APP_NAME) migrate up && ./$(APP_NAME) seed

health-check:
	@echo "❤️ Health checking service..."
	@if curl -s http://localhost:8090/health > /dev/null; then \
		echo "✅ Service is running"; \
	else \
		echo "❌ Service is not available"; \
	fi

## 🧪 Тестирование
test:
	@echo "🧪 Running all tests..."
	go test -v -race ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@echo "✅ Tests completed"

bench:
	@echo "📊 Running benchmarks..."
	go test -bench=. -benchmem ./...

load-test:
	@echo "⚡ Running load tests..."
	@if command -v k6 > /dev/null; then \
		k6 run scripts/loadtest.js; \
	else \
		echo "⚠ k6 is not installed. Install: https://k6.io/docs/getting-started/installation/"; \
	fi

## 🛠️ Code quality
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "⚠ golangci-lint is not installed"; \
	fi

vet:
	@echo "🔎 Checking code with vet..."
	go vet ./...
	@echo "✅ Vet completed"

fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	@echo "✅ Formatting completed"

security-check:
	@echo "🔒 Security checking..."
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "⚠ gosec is not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

## 📋 Утилиты
install-deps:
	@echo "📦 Installing dependencies..."
	go mod download
	@echo "✅ Dependencies installed"

update:
	@echo "🔄 Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✅ Dependencies updated"

init: install-deps
	@echo "✅ Project initialized"

clean:
	@echo "🧹 Cleaning..."
	rm -rf $(BIN_DIR) coverage.out
	go clean
	@echo "✅ Clean completed"

tidy:
	go mod tidy

docker-build:
	@echo "🐳 Building Docker image..."
	docker build -f deployments/Dockerfile -t streaming-service:latest .
	@echo "✅ Docker image built"

docker-run:
	@echo "🐳 Running Docker container..."
	docker run -p 8090:8090 streaming-service:latest

docker-compose-up:
	@echo "🐳 Starting with docker-compose..."
	docker compose -f deployments/docker-compose.yml up -d

docker-compose-down:
	@echo "🐳 Stopping docker-compose..."
	docker compose -f deployments/docker-compose.yml down

test-api:
	@echo "🧪 Testing API..."
	@curl -s http://localhost:8090/health | head -1

test-dual:
	@echo "🧪 Testing API..."
	@echo "1. Starting server..."
	@make run-dev &
	@SERVER_PID=$$!; sleep 3; echo ""; echo "2. Testing HTTP..."; curl -s http://localhost:8090/health; echo ""; echo "✅ Tests completed"; kill $$SERVER_PID 2>/dev/null || true
