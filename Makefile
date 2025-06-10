# GitHub Activity CLI Makefile

# Variables
BINARY_NAME=github-activity
GO=go
GOFLAGS=-v
LDFLAGS=-s -w

# Default target
.DEFAULT_GOAL := build

# Build the application
.PHONY: build
build:
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) .

# Run the application
.PHONY: run
run: build
	./$(BINARY_NAME) $(ARGS)

# Clean build artifacts
.PHONY: clean
clean:
	$(GO) clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*

# Format code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Run linter
.PHONY: lint
lint:
	golangci-lint run

# Run tests
.PHONY: test
test:
	$(GO) test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

# Build for Linux
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-arm64 .

# Build for macOS
.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-darwin-arm64 .

# Build for Windows
.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-windows-amd64.exe .

# Install globally
.PHONY: install
install: build
	$(GO) install

# Demo run
.PHONY: demo
demo: build
	@echo "=== GitHub Activity CLI Demo ==="
	@echo ""
	@echo "Fetching activity for torvalds..."
	./$(BINARY_NAME) torvalds --limit 5
	@echo ""
	@echo "Filtering by PushEvent..."
	./$(BINARY_NAME) torvalds --type PushEvent --limit 3
	@echo ""
	@echo "Showing detailed view..."
	./$(BINARY_NAME) torvalds --detailed --limit 2

# Help
.PHONY: help
help:
	@echo "GitHub Activity CLI - Makefile Commands"
	@echo ""
	@echo "Usage: make [command]"
	@echo ""
	@echo "Commands:"
	@echo "  build       	  Build the application for current platform"
	@echo "  run             Build and run the application (use ARGS='username' to pass arguments)"
	@echo "  clean           Remove build artifacts"
	@echo "  fmt             Format Go code"
	@echo "  lint            Run linter (requires golangci-lint)"
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  build-all       Build for all platforms"
	@echo "  build-linux     Build for Linux (amd64, arm64)"
	@echo "  build-darwin    Build for macOS (amd64, arm64)"
	@echo "  build-windows   Build for Windows (amd64)"
	@echo "  install         Install globally"
	@echo "  demo            Run a demo"
	@echo "  help            Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make run ARGS='kamranahmedse'"
	@echo "  make run ARGS='--type PushEvent --limit 5 torvalds'"
