.PHONY: build install clean test run

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	@echo "Building vervideos..."
	@go build -ldflags "$(LDFLAGS)" -o bin/vervideos .
	@echo "✓ Build complete: bin/vervideos"

install: build
	@echo "Installing to /usr/local/bin..."
	@cp bin/vervideos /usr/local/bin/
	@echo "✓ vervideos installed successfully"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "✓ Clean complete"

test:
	@echo "Running tests..."
	@go test -v ./...

run: build
	@./bin/vervideos

