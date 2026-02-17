BINARY_NAME=glab
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GO=go

.PHONY: all build clean test lint install fmt vet snapshot release

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
