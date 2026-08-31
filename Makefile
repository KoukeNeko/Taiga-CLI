.PHONY: build test test-race lint test-integration verify

build:
	mkdir -p bin
	go build -o bin/taiga ./cmd/taiga

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
