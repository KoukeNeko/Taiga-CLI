.PHONY: build install test test-race lint test-integration verify

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || printf unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
LDFLAGS := -s -w \
	-X github.com/KoukeNeko/taiga-cli/internal/version.Version=$(VERSION) \
	-X github.com/KoukeNeko/taiga-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/KoukeNeko/taiga-cli/internal/version.BuildDate=$(BUILD_DATE)

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/taiga ./cmd/taiga

install: build
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 0755 bin/taiga "$(DESTDIR)$(BINDIR)/taiga"

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	go vet ./...
	golangci-lint run

test-integration:
	./scripts/test-integration.sh

verify: test test-race lint build
