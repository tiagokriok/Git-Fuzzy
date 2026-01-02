.PHONY: help build run test test-verbose test-coverage fmt lint vet clean install

# Variables
BINARY_NAME=gitf
BINARY_PATH=./$(BINARY_NAME)
VERSION?=0.1.0
BUILD_FLAGS=-ldflags "-s -w"

# Default target
help:
	@echo "🔧 GitF (Git Fuzzy) - Makefile targets:"
	@echo ""
	@echo "  build              Build the gitf binary"
	@echo "  run                Run gitf directly (development)"
	@echo "  test               Run all tests"
	@echo "  test-verbose       Run tests with verbose output"
	@echo "  test-coverage      Run tests with coverage report"
	@echo "  fmt                Format code with gofmt"
	@echo "  lint               Run go vet for linting"
	@echo "  clean              Remove build artifacts and temp files"
	@echo "  install            Install gitf to \$$GOPATH/bin"
	@echo "  help               Show this help message"
	@echo ""

# Build the binary
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@go build -o $(BINARY_PATH) ./cmd/gf
	@echo "✅ Build complete: $(BINARY_PATH)"
	@du -h $(BINARY_PATH)

# Build optimized binary (smaller size)
build-optimized:
	@echo "🔨 Building optimized $(BINARY_NAME)..."
	@go build $(BUILD_FLAGS) -o $(BINARY_PATH) ./cmd/gf
	@echo "✅ Optimized build complete: $(BINARY_PATH)"
	@du -h $(BINARY_PATH)

# Run gitf directly
run: build
	@./$(BINARY_PATH)

# Run all tests
test:
	@echo "🧪 Running tests..."
	@go test ./...

# Run tests with verbose output
test-verbose:
	@echo "🧪 Running tests (verbose)..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	@go test -v -cover ./...
	@echo ""
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Format code
fmt:
	@echo "📝 Formatting code..."
	@go fmt ./...
	@gofmt -l -w .
	@echo "✅ Code formatted"

# Lint code
lint: fmt
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✅ Linting passed"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning up..."
	@rm -f $(BINARY_PATH)
	@rm -f coverage.out coverage.html
	@go clean
	@echo "✅ Cleanup complete"

# Install to GOPATH/bin
install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	@cp $(BINARY_PATH) $$(go env GOPATH)/bin/$(BINARY_NAME)
	@echo "✅ Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

# Download dependencies
deps:
	@echo "📚 Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies ready"

# Remove config and binary for fresh start
reset-local:
	@echo "🔄 Resetting local state..."
	@rm -f $(BINARY_PATH)
	@rm -rf ~/.config/gf/
	@echo "✅ Reset complete. Run 'make run' to start fresh"

# Run all checks (fmt + lint + test)
check: fmt lint test
	@echo "✅ All checks passed!"

# Development workflow
dev: clean fmt lint test build
	@echo "✅ Development build ready!"
	@echo "Run './gitf' to test"
