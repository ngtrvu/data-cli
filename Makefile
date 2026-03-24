VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -ldflags "-X main.version=$(VERSION)"

# ── Local build ───────────────────────────────────────────────────────────────

build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/data ./cmd

# ── Cross-platform (requires GoReleaser) ─────────────────────────────────────

# Build all platforms without publishing (output in dist/)
build-all:
	goreleaser build --snapshot --clean

# Full release — tags, archives, checksums (requires GITHUB_TOKEN)
release:
	goreleaser release --clean

# ── Dev ───────────────────────────────────────────────────────────────────────

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

install:
	CGO_ENABLED=1 go build $(LDFLAGS) -o /usr/local/bin/data ./cmd

.PHONY: build build-all release test lint install
