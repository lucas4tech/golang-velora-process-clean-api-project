.PHONY: help swag lint test test-unit test-unit-cover coverage test-integration build run-api run-worker dev dev-api dev-worker dev-all tidy clean \
	docker-build docker-up docker-down docker-reup docker-dev-build docker-dev-up docker-dev-down

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "  help             Show this help (default)"
	@echo "  build            Build api and worker binaries in ./bin/"
	@echo "  dev              Run API + Worker with hot reload (both restart on code change)"
	@echo "  dev-api          Run only API with hot reload"
	@echo "  dev-worker       Run only Worker with hot reload"
	@echo "  lint             Run golangci-lint (via go run, no install needed)"
	@echo "  test             Run unit tests and integration tests"
	@echo "  test-unit        Run unit tests only (internal/)"
	@echo "  test-unit-cover  Run unit tests with coverage, writes coverage.out"
	@echo "  coverage         Run unit tests with coverage and show total %"
	@echo "  test-integration Run integration tests only"
	@echo "  run-api          Run API locally (go run ./cmd/api)"
	@echo "  run-worker       Run worker locally (go run ./cmd/worker)"
	@echo "  swag             Regenerate Swagger docs (docs/)"
	@echo "  tidy             go mod tidy"
	@echo "  clean            Remove bin/ and coverage files"
	@echo "  docker-build     Rebuild api and worker images (production)"
	@echo "  docker-up        Start all containers (production)"
	@echo "  docker-down      Stop all containers"
	@echo "  docker-reup      docker-build + docker-up"
	@echo "  docker-dev-build Build dev images (Go + Air)"
	@echo "  docker-dev-up    Start stack with hot reload in containers (Air)"
	@echo "  docker-dev-down  Stop dev stack"

build:
	@mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --timeout=5m ./cmd/... ./configs/... ./internal/... ./pkg/...

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
	docker compose -f deployments/docker-compose.prod.yml build --no-cache api worker

docker-up:
	docker compose -f deployments/docker-compose.prod.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.prod.yml down

docker-reup: docker-build docker-up

docker-dev-build:
	docker compose -f deployments/docker-compose.dev.yml build --no-cache api worker

docker-dev-up:
	docker compose -f deployments/docker-compose.dev.yml up --build

docker-dev-down:
	docker compose -f deployments/docker-compose.dev.yml down
