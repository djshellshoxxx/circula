# Circula build, test and release targets. Run `make help` for the list.
GO        ?= go
VERSION   ?= dev
COMMIT    := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
DATE      := $(shell git log -1 --format=%cI 2>/dev/null || echo unknown)
LDFLAGS   := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
BIN       := bin/circula
export CGO_ENABLED ?= 0
GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.1.4

.PHONY: help build test test-race vet fmt-check lint vuln actions-check check release-local clean

help:
	@echo "build          build $(BIN) for the host (CGO_ENABLED=0, -trimpath)"
	@echo "test           go test ./..."
	@echo "test-race      go test -race ./... (needs cgo)"
	@echo "vet            go vet ./..."
	@echo "fmt-check      fail if gofmt would change files"
	@echo "lint           golangci-lint run"
	@echo "vuln           govulncheck ./... (needs network for the vulnerability database)"
	@echo "actions-check  workflow action pin and permission checks"
	@echo "check          everything CI runs except the release build"
	@echo "release-local  package all OPS-003 targets into bin/dist (VERSION=v1.2.3)"
	@echo "clean          remove bin/"

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/circula

test:
	$(GO) test -count=1 ./...

test-race:
	$(GO) test -race -count=1 ./...

vet:
	$(GO) vet ./...

fmt-check:
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "needs gofmt:"; echo "$$unformatted"; exit 1; fi

lint:
	golangci-lint run

vuln:
	$(GO) run $(GOVULNCHECK) ./...

actions-check:
	scripts/check-action-pins.sh

check: fmt-check vet test-race lint vuln actions-check

release-local:
	VERSION=$(VERSION) OUT=bin/dist GO=$(GO) scripts/package-release.sh

clean:
	rm -rf bin/
