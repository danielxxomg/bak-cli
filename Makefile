# bak-cli Makefile

# Build variables
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X github.com/danielxxomg/bak-cli/cmd.Version=$(VERSION) -X github.com/danielxxomg/bak-cli/cmd.Commit=$(COMMIT) -X github.com/danielxxomg/bak-cli/cmd.Date=$(DATE)

# Go variables
GO := go
BINARY := bak

.PHONY: all build test test-verbose lint vet clean run help

all: vet test build

## build: Build the bak binary
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: Run all tests
test:
	$(GO) test ./...

## test-verbose: Run all tests with verbose output
test-verbose:
	$(GO) test -v ./...

## test-cover: Run tests with coverage report
test-cover:
	$(GO) test -cover ./...

## lint: Run golangci-lint (requires golangci-lint installed)
lint:
	golangci-lint run

## vet: Run go vet
vet:
	$(GO) vet ./...

## tidy: Tidy and verify module dependencies
tidy:
	$(GO) mod tidy

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -f $(BINARY).exe
	rm -rf dist/

## run: Build and run bak with arguments (use: make run ARGS="backup --preset full")
run: build
	./$(BINARY) $(ARGS)

## release: Create a release with goreleaser (requires goreleaser installed)
release:
	goreleaser release --clean

## release-snapshot: Create a snapshot release (for testing)
release-snapshot:
	goreleaser release --snapshot --clean

## help: Show this help message
help:
	@echo "bak-cli development targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Example: make run ARGS='backup --preset full'"
