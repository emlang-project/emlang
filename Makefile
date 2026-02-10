# emlang Makefile

BINARY_NAME=emlang
VERSION=1.0.0
BUILD_DIR=bin
GO=go

# Build flags
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all build build-all build-dev build-wasm test lint fmt vet clean install help

## all: Build optimized binary (default)
all: build

## build: Build optimized binary
build:
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/emlang

## build-all: Build for all platforms (linux, macOS, Windows)
build-all:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/emlang
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/emlang
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/emlang
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/emlang

## build-wasm: Build WebAssembly module
build-wasm:
	GOOS=js GOARCH=wasm $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/emlang.wasm ./cmd/wasm
	@GOROOT="$$($(GO) env GOROOT)"; \
	if [ -f "$$GOROOT/misc/wasm/wasm_exec.js" ]; then \
		cp "$$GOROOT/misc/wasm/wasm_exec.js" $(BUILD_DIR)/wasm_exec.js; \
	elif [ -f "$$GOROOT/lib/wasm/wasm_exec.js" ]; then \
		cp "$$GOROOT/lib/wasm/wasm_exec.js" $(BUILD_DIR)/wasm_exec.js; \
	else \
		echo "warning: wasm_exec.js not found in GOROOT"; \
	fi
	@ls -lh $(BUILD_DIR)/emlang.wasm
	@if command -v gzip >/dev/null 2>&1; then \
		gzip -k -f $(BUILD_DIR)/emlang.wasm; \
		ls -lh $(BUILD_DIR)/emlang.wasm.gz; \
	fi

## build-dev: Build with debug symbols
build-dev:
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/emlang

## test: Run all tests
test:
	$(GO) test -v ./...

## test-cover: Run tests with coverage
test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GO) fmt ./...

## vet: Run go vet
vet:
	$(GO) vet ./...

## clean: Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## install: Install binary to GOPATH/bin
install:
	$(GO) install $(LDFLAGS) ./cmd/emlang

## run: Run with a file (usage: make run FILE=path/to/file.yaml)
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) lint $(FILE)

## help: Show this help
help:
	@echo "emlang v$(VERSION) - The Emlang toolchain (https://emlang-project.github.io/)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
