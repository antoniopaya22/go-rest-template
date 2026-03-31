GO ?= go
BINARY ?= ./bin/api
GOLANGCI_LINT ?= $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: tidy fmt vet lint test integration test-race cover build run migrate

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	$(GOLANGCI_LINT) run

test:
	$(GO) test ./...

integration:
	$(GO) test ./test -run TestAPIIntegration -count=1

test-race:
	$(GO) test -race ./...

cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

build:
	$(GO) build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/api

run:
	$(GO) run ./cmd/api

migrate:
	$(GO) run ./cmd/migrate
