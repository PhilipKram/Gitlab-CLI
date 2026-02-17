BINARY_NAME=glab
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GO=go

.PHONY: all build clean test lint install

all: build

build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) .

install:
	$(GO) install $(LDFLAGS) .

clean:
	rm -rf bin/
	$(GO) clean

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...
