.PHONY: all build vet lint test-short test ci-check tidy deadcode vulncheck clean fmt

BINARY    := bin/alms
VERSION   := $(shell git describe --tags 2>/dev/null || echo dev)
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS   := -ldflags="-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

all: build test lint

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/alms/

vet:
	go vet ./...

fmt:
	goimports -w -local github.com/ghassan/alms .

tidy:
	go mod tidy

lint:
	golangci-lint run ./... --timeout=3m

lint-ci:
	golangci-lint run ./... --timeout=3m --out-format=github-actions

test:
	go test -race -count=1 -shuffle=on -coverprofile=coverage.out ./... && \
	go tool cover -func=coverage.out | grep total

test-short:
	go test -short -race -count=1 -shuffle=on ./...

ci-check: tidy build vet lint-ci test-short
	@echo "✅ CI check passed"

deadcode:
	deadcode ./...

vulncheck:
	govulncheck ./...

clean:
	rm -f coverage.out
	rm -rf bin/
