COMPOSE_PROD := docker compose -f deployments/docker-compose.prod.yml
COMPOSE_DEV  := docker compose -f deployments/docker-compose.dev.yml

.PHONY: default help build lint test test-unit test-unit-cover coverage test-integration \
	run-api run-worker dev dev-api dev-worker swag tidy clean \
	docker-build docker-up docker-down docker-reup \
	docker-dev-up docker-dev-down docker-dev-check-elk docker-dev-logs-api docker-dev-logs-worker \
	test-elk

default: help

help:
	@echo "Usage: make [target]  (from repo root)"
	@echo ""
	@echo "  Local development (no full Compose stack)"
	@echo "  help             Show this help (default when running make with no target)"
	@echo "  build            Build api and worker binaries into ./bin/"
	@echo "  run-api          Run API: go run ./cmd/api"
	@echo "  run-worker       Run worker: go run ./cmd/worker"
	@echo "  dev              API + worker with Air (hot reload)"
	@echo "  dev-api          API only with Air"
	@echo "  dev-worker       Worker only with Air"
	@echo "  swag             Regenerate Swagger under docs/"
	@echo "  tidy             go mod tidy"
	@echo "  clean            Remove bin/ and coverage.out / coverage.html"
	@echo ""
	@echo "  Quality"
	@echo "  lint             golangci-lint"
	@echo "  test             test-unit + test-integration"
	@echo "  test-unit        Unit tests under ./internal/..."
	@echo "  test-unit-cover  Unit tests with coverage -> coverage.out"
	@echo "  coverage         Print total coverage % from coverage.out"
	@echo "  test-integration Integration tests (Docker / testcontainers)"
	@echo ""
	@echo "  Docker — production ($(COMPOSE_PROD))"
	@echo "  docker-build     Build api and worker images (no cache)"
	@echo "  docker-up        Up stack in background (-d): app, MongoDB, RabbitMQ, ELK, APM, Metricbeat"
	@echo "  docker-down      Stop and remove production Compose stack"
	@echo "  docker-reup      docker-build then docker-up"
	@echo ""
	@echo "  Docker — development ($(COMPOSE_DEV))"
	@echo "  docker-dev-up    Up dev stack with --build (ELK, APM, Metricbeat; logs app → Kibana via GELF)"
	@echo "  docker-dev-down  Stop development Compose stack"
	@echo "  docker-dev-check-elk  Lista índices rankmyapp* no ES + dicas se data view falhar"
	@echo "  docker-dev-logs-api    Follow api container logs (same as compose logs -f api)"
	@echo "  docker-dev-logs-worker Follow worker container logs"
	@echo ""
	@echo "  Observability"
	@echo "  test-elk         Log pipeline smoke test (needs API up, e.g. docker-dev-up in another terminal)"

build:
	@mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --timeout=5m ./...

test-unit:
	go test -v ./internal/...

test-unit-cover:
	go test -coverprofile=coverage.out -covermode=atomic -v ./internal/...

coverage: test-unit-cover
	@go tool cover -func=coverage.out | grep total

test-integration:
	go test -tags=integration -v ./test/integration/...

test: test-unit test-integration

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

dev dev-all:
	@trap 'kill 0' INT TERM; \
	go run github.com/air-verse/air@latest -c .air.toml & \
	go run github.com/air-verse/air@latest -c .air.worker.toml & \
	wait

dev-api:
	go run github.com/air-verse/air@latest -c .air.toml

dev-worker:
	go run github.com/air-verse/air@latest -c .air.worker.toml

swag:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

tidy:
	go mod tidy

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-build:
	$(COMPOSE_PROD) build --no-cache api worker

docker-up:
	$(COMPOSE_PROD) up -d

docker-down:
	$(COMPOSE_PROD) down

docker-reup: docker-build docker-up

docker-dev-up:
	$(COMPOSE_DEV) up --build

docker-dev-down:
	$(COMPOSE_DEV) down -v

docker-dev-logs-api:
	$(COMPOSE_DEV) logs -f api

docker-dev-logs-worker:
	$(COMPOSE_DEV) logs -f worker
