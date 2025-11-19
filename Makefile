.PHONY: help build test lint clean install

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -o bin/gocreator ./cmd/gocreator

install: ## Install the binary
	go install ./cmd/gocreator

test: ## Run tests
	go test -v -race -cover -short ./...

test-integration: ## Run integration tests
	go test -v -race -tags=integration ./tests/integration/...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run ./...

lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix ./...

sec: ## Run security scanner
	gosec ./...

fmt: ## Format code
	gofmt -s -w .
	goimports -w .

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
