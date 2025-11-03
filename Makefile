.PHONY: build install clean test run

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	@echo "Building vervids..."
	@go build -ldflags "$(LDFLAGS)" -o bin/vervids .
	@echo "✓ Build complete: bin/vervids"

install: build
	@echo "Installing to /usr/local/bin..."
	@cp bin/vervids /usr/local/bin/
	@echo "✓ vervids installed successfully"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "✓ Clean complete"

test:
	@echo "Running tests..."
	@go test -v ./...

run: build
	@./bin/vervids

