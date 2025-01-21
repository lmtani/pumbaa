.PHONY: all format build

# Use shell find to list all Go files in the project (excluding vendor if desired).
GO_FILES := $(shell find . -type f -name '*.go')

all: format build

format:
	@echo "Running goimports..."
	@goimports -w $(GO_FILES)
	@echo "Running gofmt..."
	@gofmt -s -w $(GO_FILES)

build:
	@echo "Building Go project..."
	go build -o dist/pumbaa cmd/cli/main.go
