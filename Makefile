.PHONY: test test-unit test-integration test-all build clean help

# Default target
.DEFAULT_GOAL := help

# Load .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Build variables
BINARY_NAME=helm-gcs
BUILD_DIR=bin
CMD_PATH=./cmd/helm-gcs

# Test variables
TEST_TIMEOUT=5m
COVERAGE_DIR=coverage

## help: Display this help message
help:
	@echo "Available targets:"
	@echo "  make setup             - Setup development environment (.env file)"
	@echo "  make test              - Run unit tests"
	@echo "  make test-unit         - Run unit tests with coverage"
	@echo "  make test-integration  - Run integration tests (requires GCS_TEST_BUCKET)"
	@echo "  make test-all          - Run all tests (unit + integration)"
	@echo "  make test-coverage     - Run tests and generate coverage report"
	@echo "  make build             - Build the binary"
	@echo "  make clean             - Clean build artifacts"
	@echo "  make lint              - Run linters"
	@echo "  make fmt               - Format code"
	@echo "  make vet               - Run go vet"
	@echo ""
	@echo "Environment:"
	@if [ -f .env ]; then \
		echo "  ✓ .env file found"; \
		if [ -n "$$GCS_TEST_BUCKET" ]; then \
			echo "  ✓ GCS_TEST_BUCKET: $$GCS_TEST_BUCKET"; \
		else \
			echo "  ✗ GCS_TEST_BUCKET not set in .env"; \
		fi; \
	else \
		echo "  ✗ .env file not found (run 'make setup')"; \
	fi

## setup: Create .env file from .env.example
setup:
	@if [ -f .env ]; then \
		echo ".env file already exists. Remove it first to recreate."; \
	else \
		cp .env.example .env; \
		echo "Created .env file from .env.example"; \
		echo "Please edit .env and fill in your configuration"; \
	fi

## test: Run unit tests
test: test-unit

## test-unit: Run unit tests with race detector
test-unit:
	@echo "Running unit tests..."
	@go test -v -race -timeout $(TEST_TIMEOUT) ./pkg/...

## test-integration: Run integration tests (requires GCS_TEST_BUCKET env var)
test-integration:
	@echo "Running integration tests..."
	@if [ -z "$(GCS_TEST_BUCKET)" ]; then \
		echo "ERROR: GCS_TEST_BUCKET environment variable not set"; \
		echo "Example: make test-integration GCS_TEST_BUCKET=gs://my-bucket/test-path"; \
		exit 1; \
	fi
	@go test -v -race -timeout $(TEST_TIMEOUT) -tags=integration ./pkg/repo

## test-all: Run all tests (unit + integration)
test-all: test-unit test-integration

## test-coverage: Generate test coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -v -race -timeout $(TEST_TIMEOUT) -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./pkg/...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print "Total coverage: " $$3}'

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Clean build artifacts and test cache
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@go clean -testcache
	@echo "Clean complete"

## lint: Run golangci-lint
lint:
	@echo "Running linters..."
	@golangci-lint run ./...

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .
	@echo "Code formatted"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test-unit
	@echo "All checks passed!"
