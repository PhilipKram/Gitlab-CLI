BINARY_NAME=glab
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GO=go

.PHONY: all build clean test test-integration test-e2e test-all test-coverage test-coverage-all lint install fmt vet snapshot release

all: build

build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) .

install:
	$(GO) install $(LDFLAGS) .

clean:
	rm -rf bin/ dist/
	$(GO) clean

test:
	$(GO) test ./...

test-coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -coverprofile=coverage.out ./...
	@echo "\n=== Coverage Summary ==="
	@$(GO) tool cover -func=coverage.out | tail -n 1
	@echo "\n=== cmd/ Package Coverage (Top 10 Files) ==="
	@$(GO) tool cover -func=coverage.out | grep "/cmd/" | head -n 10 || echo "No cmd/ coverage data found"
	@echo "\n=== Overall cmd/ Package Coverage ==="
	@$(GO) tool cover -func=coverage.out | grep "/cmd/" | awk '{gsub(/%/, "", $$NF); sum += $$NF; count++} END {if (count > 0) printf "Average: %.1f%% (%d functions)\n", sum/count, count; else print "No coverage data found"}'
	@echo "\nDetailed coverage report saved to coverage.out"
	@echo "Run 'go tool cover -html=coverage.out' to view HTML report"

test-integration:
	@echo "Running integration tests..."
	$(GO) test -v ./tests/integration/...

test-e2e:
	@echo "Running E2E tests..."
	$(GO) test -v ./tests/e2e/...

test-all:
	@echo "Running all tests (unit + integration)..."
	$(GO) test -v ./...

test-coverage-all:
	@echo "Running all tests with coverage..."
	@$(GO) test -coverprofile=coverage-all.out ./...
	@echo "\n=== Coverage Summary ==="
	@$(GO) tool cover -func=coverage-all.out | tail -n 1
	@echo "\nDetailed coverage report saved to coverage-all.out"
	@echo "Run 'go tool cover -html=coverage-all.out' to view HTML report"

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

# Build cross-platform snapshot (no publish)
snapshot:
	goreleaser release --snapshot --clean

# Full release (requires GITHUB_TOKEN)
release:
	goreleaser release --clean
