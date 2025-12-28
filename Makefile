.PHONY: help init test test-coverage test-watch dev build run clean migrate-up migrate-down docker-build docker-run

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

init: ## Initialize project (install dependencies, setup DB)
	@echo "Installing dependencies..."
	go mod download
	@echo "Installing tools..."
	go install github.com/air-verse/air@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Running migrations..."
	make migrate-up

test: ## Run all tests with race detection
	go test -race -coverprofile=coverage.out ./...

test-coverage: test ## Generate coverage report (HTML)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-watch: ## Watch mode for TDD (requires air)
	air -c .air.test.toml

dev: ## Run server with hot reload (requires air)
	air

build: ## Build production binary
	@echo "Building..."
	mkdir -p bin
	go build -ldflags="-w -s" -o bin/momlaunchpad cmd/server/main.go
	@echo "Binary created: bin/momlaunchpad"

run: build ## Run the built binary
	./bin/momlaunchpad

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

migrate-up: ## Apply database migrations
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Rollback last migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create: ## Create new migration (use NAME=your_migration_name)
	migrate create -ext sql -dir migrations -seq $(NAME)

docker-build: ## Build Docker image
	docker build -t momlaunchpad-be:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env momlaunchpad-be:latest

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

ci: fmt vet lint test ## Run all CI checks
