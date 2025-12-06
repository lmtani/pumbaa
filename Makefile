# Build variables
BINARY_NAME := pumbaa
BUILD_DIR := build
CMD_DIR := ./cmd/cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(BUILD_TIME)"

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet
GOLINT := golangci-lint

# Platforms for cross-compilation
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build build-all clean test test-verbose test-coverage fmt tidy vet lint help install run dev deps antlr

# Default target
all: fmt vet test build

## Build targets

# Build for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary created at $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}$(if $(findstring windows,$${platform}),.exe,) $(CMD_DIR); \
		echo "Built: $(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Run the application
run:
	$(GOCMD) run $(CMD_DIR) $(ARGS)

# Development build with race detector
dev:
	$(GOBUILD) -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-dev $(CMD_DIR)

## Code quality targets

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@echo "Done formatting"

# Check formatting (useful for CI)
fmt-check:
	@echo "Checking code formatting..."
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "Done tidying"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Done downloading"

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify
	@echo "Done verifying"

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Done vetting"

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

# Create a release build
release: clean fmt vet test build-all
	@echo "Release builds created in $(BUILD_DIR)/"

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
	@echo "  release      Create release builds for all platforms"
	@echo "  docker-build Build Docker image"
	@echo "  help         Show this help"
