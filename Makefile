build:
	CGO_ENABLED=1 go build -o bin/data ./cmd

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

.PHONY: build test lint
