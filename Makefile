VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -ldflags "-s -w -X main.version=$(VERSION)"

# ── Local build ───────────────────────────────────────────────────────────────

build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/data ./cmd

install:
	CGO_ENABLED=1 go build $(LDFLAGS) -o /usr/local/bin/data ./cmd

# ── Local release (requires: gh, Docker) ─────────────────────────────────────

release-local: _build-darwin _build-linux _package
	git push origin $(VERSION)
	gh release create $(VERSION) dist/*.tar.gz dist/checksums.txt \
		--title "$(VERSION)" \
		--generate-notes

_build-darwin:
	@mkdir -p dist
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		go build $(LDFLAGS) -o dist/data-darwin-arm64 ./cmd
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		go build $(LDFLAGS) -o dist/data-darwin-amd64 ./cmd

_build-linux:
	@mkdir -p dist
	# Run each container on its native platform — no cross-compiler needed
	docker run --rm --platform linux/amd64 \
		-v $(PWD):/src -w /src \
		-v $(HOME)/go/pkg/mod:/go/pkg/mod \
		-e CGO_ENABLED=1 \
		golang:1.23 \
		go build $(LDFLAGS) -o dist/data-linux-amd64 ./cmd
	docker run --rm --platform linux/arm64 \
		-v $(PWD):/src -w /src \
		-v $(HOME)/go/pkg/mod:/go/pkg/mod \
		-e CGO_ENABLED=1 \
		golang:1.23 \
		go build $(LDFLAGS) -o dist/data-linux-arm64 ./cmd

_package:
	@cd dist && for f in data-*; do \
		OS_ARCH=$${f#data-}; \
		OS=$$(echo $$OS_ARCH | cut -d- -f1); \
		ARCH=$$(echo $$OS_ARCH | cut -d- -f2); \
		VER=$(VERSION); VER=$${VER#v}; \
		NAME="data-cli_$${VER}_$${OS}_$${ARCH}"; \
		mkdir -p "$$NAME"; \
		cp "$$f" "$$NAME/data"; \
		cp ../README.md ../LICENSE ../config.example.toml "$$NAME/"; \
		tar -czf "$${NAME}.tar.gz" "$$NAME"; \
		rm -rf "$$NAME" "$$f"; \
	done
	cd dist && shasum -a 256 *.tar.gz > checksums.txt

clean:
	rm -rf dist/ bin/

# ── Dev ───────────────────────────────────────────────────────────────────────

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

.PHONY: build install release-local _build-darwin _build-linux _package clean test lint
