.PHONY: all test test-coverage test-short test-race test-verbose bench lint fmt vet clean install build help

# Variables
GOBIN := $(shell go env GOPATH)/bin
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Default target
all: fmt vet lint test build

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE_FILE)
	@echo "Generating HTML coverage report..."
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report saved to $(COVERAGE_HTML)"

# Run short tests (exclude integration tests)
test-short:
	@echo "Running short tests..."
	@go test -short ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -race ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run --timeout=5m ./...; \
	else \
		echo "golangci-lint is not installed. Please install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Clean build artifacts and test cache
clean:
	@echo "Cleaning..."
	@go clean -testcache
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@rm -rf dist/

# Install the application
install:
	@echo "Installing soba..."
	@go install ./cmd/soba

# Build the application
build:
	@echo "Building soba..."
	@go build -o dist/soba ./cmd/soba

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# Verify dependencies
deps-verify:
	@echo "Verifying dependencies..."
	@go mod verify

# Check for security vulnerabilities
security:
	@echo "Checking for vulnerabilities..."
	@if command -v govulncheck > /dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck is not installed. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi

# Run CI pipeline locally
ci: fmt vet lint test-coverage security
	@echo "CI pipeline completed successfully!"

# Show help
help:
	@echo "Available targets:"
	@echo "  make test          - Run tests with race detector"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make test-short    - Run short tests (exclude integration tests)"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make test-verbose  - Run tests with verbose output"
	@echo "  make bench         - Run benchmarks"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make clean         - Clean build artifacts and test cache"
	@echo "  make install       - Install the application"
	@echo "  make build         - Build the application"
	@echo "  make deps          - Download dependencies"
	@echo "  make deps-update   - Update dependencies"
	@echo "  make deps-verify   - Verify dependencies"
	@echo "  make security      - Check for security vulnerabilities"
	@echo "  make ci            - Run full CI pipeline locally"
	@echo "  make help          - Show this help message"