.PHONY: all build vet lint test-short test ci-check tidy deadcode vulncheck clean fmt test-integration load-test acceptance deploy-linux

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
	golangci-lint run ./... --timeout=3m 

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

# Integration tests (require ALMS_PG_DSN).
# Usage: ALMS_PG_DSN=... make test-integration
test-integration:
	go test -tags=integration -race -count=1 -v ./internal/integration/...

# Run all tests including integration.
# Usage: ALMS_PG_DSN=... make test-all
test-all:
	go test -race -count=1 -shuffle=on ./...
	go test -tags=integration -race -count=1 ./internal/integration/...

# Load test (requires ALMS_PG_DSN + docker-compose).
load-test:
	bash test/load-test.sh

# Phase 4 acceptance test.
acceptance:
	bash test/phase-4-acceptance.sh

# Cross-compile linux binary (verifies deploy.sh compatibility).
deploy-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/alms-linux ./cmd/alms/
	@echo "✅ Linux binary: bin/alms-linux ($$(ls -lh bin/alms-linux | awk '{print $$5}'))"

clean:
	rm -f coverage.out
	rm -rf bin/

