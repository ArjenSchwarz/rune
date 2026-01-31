# Makefile for rune development
#
# This Makefile provides comprehensive developer tooling for the rune library.
#
# Usage:
#   make help          - Show this help message
#   make check         - Run complete validation (fmt, lint, tests)
#   make test          - Run unit tests
#   make test-all      - Run all tests including integration tests

.PHONY: help
help: ## Show available make targets and their descriptions
	@echo "Available make targets:"
	@echo ""
	@echo "Testing targets:"
	@awk 'BEGIN {FS = ":.*##"} /^test.*:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Code quality targets:"
	@awk 'BEGIN {FS = ":.*##"} /^(lint|fmt|modernize):.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Build targets:"
	@awk 'BEGIN {FS = ":.*##"} /^(build|install):.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Development targets:"
	@awk 'BEGIN {FS = ":.*##"} /^(mod-tidy|benchmark|clean):.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Composite targets:"
	@awk 'BEGIN {FS = ":.*##"} /^check:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Usage: make <target>"
	@echo ""

# Default target
.DEFAULT_GOAL := help

# Testing targets
.PHONY: test
test: ## Run unit tests
	@echo "Running unit tests..."
	@go test ./...

.PHONY: test-integration
test-integration: ## Run integration tests with INTEGRATION=1 environment variable
	@echo "Running integration tests..."
	@INTEGRATION=1 go test ./...

.PHONY: test-all
test-all: ## Run both unit and integration tests
	@echo "Running all tests..."
	@go test ./...
	@echo "Running integration tests..."
	@INTEGRATION=1 go test ./...

.PHONY: test-coverage
test-coverage: ## Generate test coverage report and open in browser
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "Opening coverage report in browser..."
	@go tool cover -html=coverage.out

# Code quality targets
.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@golangci-lint run

.PHONY: fmt
fmt: ## Format all Go code
	@echo "Formatting code..."
	@go fmt ./...

.PHONY: modernize
modernize: ## Apply modernize tool fixes
	@echo "Running modernize..."
	@modernize -fix ./...
	@echo "Formatting after modernization..."
	@$(MAKE) fmt

VERSION ?= dev
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags for version injection
LDFLAGS := -X github.com/arjenschwarz/rune/cmd.Version=$(VERSION) \
           -X github.com/arjenschwarz/rune/cmd.BuildTime=$(BUILD_TIME) \
           -X github.com/arjenschwarz/rune/cmd.GitCommit=$(GIT_COMMIT)

# Build the Rune application
build:
	@echo "Building rune binary..."
	go build -ldflags "$(LDFLAGS)" .

.PHONY: install
install: ## Install rune binary to $GOPATH/bin
	@echo "Installing rune..."
	@go install -ldflags "$(LDFLAGS)" .

# Development utility targets
.PHONY: mod-tidy
mod-tidy: ## Run go mod tidy
	@echo "Running go mod tidy..."
	@go mod tidy

.PHONY: benchmark
benchmark: ## Run performance benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

.PHONY: clean
clean: ## Remove generated files and test caches
	@echo "Cleaning generated files..."
	@rm -f coverage.out coverage.html
	@go clean -testcache

# Composite targets
.PHONY: check
check: fmt lint test ## Run complete validation: format, lint, and test
	@echo "All checks completed successfully!"
