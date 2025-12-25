# Build variables
BINARY_NAME := pumbaa
BUILD_DIR := dist
CMD_DIR := ./cmd/cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
SENTRY_DSN ?= ""
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(BUILD_TIME) -X github.com/lmtani/pumbaa/internal/infrastructure/telemetry.DSN=$(SENTRY_DSN)"

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet
GOLINT := golangci-lint
GOIMPORTS := goimports

# Platforms for cross-compilation
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test test-verbose test-coverage fmt goimports goimports-check fmt-check tidy vet lint help run dev deps antlr release-dry-run release-check

# Default target
all: fmt vet test build

## Build targets

# Build for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary created at $(BUILD_DIR)/$(BINARY_NAME)"


## Code quality targets

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@echo "Done formatting"

# Format imports
goimports:
	@echo "Formatting imports..."
	@which $(GOIMPORTS) > /dev/null || (echo "goimports not installed. Run: go install golang.org/x/tools/cmd/goimports@latest" && exit 1)
	$(GOIMPORTS) -w -local github.com/lmtani/pumbaa .
	@echo "Done formatting imports"

# Run go vet
vet:
	@echo "Running go vet (excluding generated code)..."
	$(GOCMD) list ./... | grep -v "pkg/wdl/parser" | xargs $(GOVET)
	@echo "Done vetting"

# Check if code is formatted
fmt-check:
	@echo "Checking formatting..."
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "The following files are not formatted:"; \
		$(GOFMT) -l .; \
		exit 1; \
	fi
	@echo "Formatting is correct"

# Check if imports are formatted
goimports-check:
	@echo "Checking imports..."
	@which $(GOIMPORTS) > /dev/null || (echo "goimports not installed. Run: go install golang.org/x/tools/cmd/goimports@latest" && exit 1)
	@if [ -n "$$($(GOIMPORTS) -l -local github.com/lmtani/pumbaa .)" ]; then \
		echo "The following files have incorrect imports:"; \
		$(GOIMPORTS) -l -local github.com/lmtani/pumbaa .; \
		exit 1; \
	fi
	@echo "Imports are correct"

# Run golangci-lint (must be installed separately)
lint:
	@echo "Running golangci-lint..."
	@which $(GOLINT) > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	$(GOLINT) run ./...
	@echo "Done linting"

## Test targets

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) ./... -v
	@echo "Done testing"

# Run tests with short flag
test-short:
	@echo "Running short tests..."
	$(GOTEST) ./... -short
	@echo "Done testing"

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	$(GOTEST) ./... -v
	@echo "Done testing"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) ./... -coverprofile=$(BUILD_DIR)/coverage.out
	$(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report: $(BUILD_DIR)/coverage.html"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) ./... -race
	@echo "Done testing"

## ANTLR targets

# Regenerate ANTLR parser (requires Java and ANTLR jar)
ANTLR_JAR := /tmp/antlr-4.13.1-complete.jar
ANTLR_URL := https://www.antlr.org/download/antlr-4.13.1-complete.jar
PARSER_DIR := pkg/wdl/parser

antlr-download:
	@if [ ! -f $(ANTLR_JAR) ]; then \
		echo "Downloading ANTLR..."; \
		curl -o $(ANTLR_JAR) $(ANTLR_URL); \
	fi

antlr: antlr-download
	@echo "Regenerating ANTLR parser..."
	@which java > /dev/null || (echo "Java is required for ANTLR" && exit 1)
	cd $(PARSER_DIR) && java -jar $(ANTLR_JAR) -Dlanguage=Go -visitor -no-listener WdlV1_1Lexer.g4 WdlV1_1Parser.g4
	@echo "Done regenerating parser"

## Clean targets

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	@echo "Done cleaning"

# Clean everything including caches
clean-all: clean
	@echo "Cleaning caches..."
	$(GOCMD) clean -cache -testcache -modcache
	@echo "Done cleaning caches"

## Release targets

# Preview release locally with goreleaser (no publish)
release-dry-run:
	@echo "Running goreleaser in dry-run mode..."
	@which goreleaser > /dev/null || (echo "goreleaser not installed. Run: go install github.com/goreleaser/goreleaser@latest" && exit 1)
	goreleaser release --snapshot --clean
	@echo "Preview release created in dist/"

# Preview changelog based on conventional commits (mimics goreleaser groups)
release-changelog:
	@echo "=== CHANGELOG PREVIEW (since last tag) ==="
	@echo ""
	@LAST_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo ""); \
	if [ -z "$$LAST_TAG" ]; then \
		echo "No previous tag found"; \
	else \
		echo "Changes since $$LAST_TAG:"; \
		echo ""; \
		echo "🚀 New Features"; \
		git log $$LAST_TAG..HEAD --oneline --grep="^feat" 2>/dev/null | sed 's/^/  /' || true; \
		echo ""; \
		echo "🐛 Bug Fixes"; \
		git log $$LAST_TAG..HEAD --oneline --grep="^fix" 2>/dev/null | sed 's/^/  /' || true; \
		echo ""; \
		echo "⚡ Performance Improvements"; \
		git log $$LAST_TAG..HEAD --oneline --grep="^perf" 2>/dev/null | sed 's/^/  /' || true; \
		echo ""; \
		echo "♻️ Refactoring"; \
		git log $$LAST_TAG..HEAD --oneline --grep="^refactor" 2>/dev/null | sed 's/^/  /' || true; \
		echo ""; \
		echo "(Excluded from changelog: docs, chore, test, ci, build, style)"; \
	fi

# Check goreleaser configuration
release-check:
	@echo "Checking goreleaser configuration..."
	@which goreleaser > /dev/null || (echo "goreleaser not installed. Run: go install github.com/goreleaser/goreleaser@latest" && exit 1)
	goreleaser check
	@echo "Configuration is valid"

## Documentation

# Serve documentation locally with live reload
docs-serve:
	@echo "Starting MkDocs development server..."
	mkdocs serve

# Build documentation for production
docs-build:
	@echo "Building documentation..."
	mkdocs build

## Help

help:
	@echo "Pumbaa - Cromwell CLI Tool"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build        Build for current platform"
	@echo "  build-all    Build for all platforms (linux, darwin, windows)"
	@echo "  install      Install binary to GOPATH/bin"
	@echo "  run          Run the application (use ARGS= to pass arguments)"
	@echo "  dev          Build with race detector"
	@echo ""
	@echo "Code quality targets:"
	@echo "  fmt          Format code with gofmt"
	@echo "  fmt-check    Check if code is formatted"
	@echo "  goimports    Format imports with goimports"
	@echo "  goimports-check Check if imports are formatted"
	@echo "  tidy         Tidy go.mod dependencies"
	@echo "  deps         Download dependencies"
	@echo "  verify       Verify dependencies"
	@echo "  vet          Run go vet"
	@echo "  lint         Run golangci-lint"
	@echo ""
	@echo "Test targets:"
	@echo "  test         Run tests"
	@echo "  test-short   Run short tests"
	@echo "  test-verbose Run tests with verbose output"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  test-race    Run tests with race detector"
	@echo ""
	@echo "ANTLR targets:"
	@echo "  antlr        Regenerate WDL parser from grammar"
	@echo ""
	@echo "Clean targets:"
	@echo "  clean        Clean build artifacts"
	@echo "  clean-all    Clean everything including caches"
	@echo ""
	@echo "Other targets:"
	@echo "  release-dry-run   Preview release with goreleaser (no publish)"
	@echo "  release-changelog Preview changelog that will be generated"
	@echo "  release-check     Validate goreleaser configuration"
	@echo "  docker-build      Build Docker image"
	@echo "  help              Show this help"
