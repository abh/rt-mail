.PHONY: help build test test-coverage lint lint-fix fmt vet clean run install-tools

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the rt-mail binary
	@echo "Building rt-mail..."
	go build -v -o rt-mail .

install: ## Install rt-mail binary to $GOPATH/bin
	@echo "Installing rt-mail..."
	go install -v

run: ## Run rt-mail with sample config (requires rt-mail.json)
	@if [ ! -f rt-mail.json ]; then \
		echo "Error: rt-mail.json not found. Copy rt-mail.json.sample and configure it first."; \
		exit 1; \
	fi
	go run . -config=rt-mail.json -listen=:8081

test: ## Run all tests
	@echo "Running tests..."
	go test -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@echo "Coverage summary:"
	go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "To view detailed HTML coverage report, run: go tool cover -html=coverage.out"

test-coverage-html: test-coverage ## Generate and open HTML coverage report
	go tool cover -html=coverage.out

lint: ## Run linters (requires golangci-lint)
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run 'make install-tools' first." && exit 1)
	golangci-lint run ./...

lint-fix: ## Run linters and auto-fix issues where possible
	@echo "Running linters with auto-fix..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run 'make install-tools' first." && exit 1)
	golangci-lint run --fix ./...

fmt: ## Format code with gofumpt
	@echo "Formatting code..."
	gofumpt -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

clean: ## Clean build artifacts and test cache
	@echo "Cleaning..."
	rm -f rt-mail
	rm -f coverage.out
	go clean -testcache
	go clean -cache

install-tools: ## Install development tools (golangci-lint)
	@echo "Installing development tools..."
	@which golangci-lint > /dev/null || \
		(echo "Installing golangci-lint..." && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@echo "Tools installed successfully!"

check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo ""
	@echo "✅ All checks passed!"

ci: ## Run CI checks (used by GitHub Actions)
	@echo "Running CI checks..."
	go mod download
	go mod verify
	gofmt -s -l . | (! grep .) || (echo "Code not formatted. Run 'make fmt'" && exit 1)
	go vet ./...
	golangci-lint run ./...
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@echo "✅ CI checks passed!"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t rt-mail:latest .

docker-run: docker-build ## Build and run Docker image
	@echo "Running Docker container..."
	docker run -p 8081:8002 -v $(PWD)/rt-mail.json:/etc/rt-mail/config.json rt-mail:latest

.DEFAULT_GOAL := help
