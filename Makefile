VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -ldflags "-X main.version=$(VERSION)"

build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/data ./cmd

install:
	CGO_ENABLED=1 go build $(LDFLAGS) -o /usr/local/bin/data ./cmd

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

.PHONY: build install test lint
