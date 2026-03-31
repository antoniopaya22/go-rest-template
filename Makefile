GO ?= go
BINARY ?= ./bin/api
GOLANGCI_LINT ?=
COMPOSE_FILE ?= deployments/docker-compose.yml

ifeq ($(OS),Windows_NT)
SCRIPT_RUNNER := powershell -NoProfile -ExecutionPolicy Bypass -File scripts/tasks.ps1
else
SCRIPT_RUNNER := sh scripts/tasks.sh
endif

.PHONY: tidy fmt vet lint test integration test-race cover build run migrate compose-up compose-down

tidy:
	$(SCRIPT_RUNNER) tidy "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

fmt:
	$(SCRIPT_RUNNER) fmt "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

vet:
	$(SCRIPT_RUNNER) vet "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

lint:
	$(SCRIPT_RUNNER) lint "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

test:
	$(SCRIPT_RUNNER) test "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

integration:
	$(SCRIPT_RUNNER) integration "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

test-race:
	$(SCRIPT_RUNNER) test-race "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

cover:
	$(SCRIPT_RUNNER) cover "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

build:
	$(SCRIPT_RUNNER) build "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

run:
	$(SCRIPT_RUNNER) run "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

migrate:
	$(SCRIPT_RUNNER) migrate "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

compose-up:
	$(SCRIPT_RUNNER) compose-up "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"

compose-down:
	$(SCRIPT_RUNNER) compose-down "$(GO)" "$(BINARY)" "$(COMPOSE_FILE)" "$(GOLANGCI_LINT)"
