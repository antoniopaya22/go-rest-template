# Tooling versions. Tools run through `go run pkg@version`, so nothing needs
# to be installed and everyone, CI included, uses the same release.
GOLANGCI_LINT_VERSION ?= v2.15.0
GOVULNCHECK_VERSION   ?= v1.9.0
AIR_VERSION           ?= v1.67.4

GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOVULNCHECK   := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
AIR           := go run github.com/air-verse/air@$(AIR_VERSION)

MODULE  := $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/platform/version.Version=$(VERSION) \
	-X $(MODULE)/internal/platform/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/platform/version.Date=$(DATE)

IMAGE   ?= go-rest-template:dev
COMPOSE := docker compose -f deployments/compose.yaml # init:compose

.DEFAULT_GOAL := help

##@ Development

.PHONY: run
run: ## Run the API
	go run ./cmd/api

.PHONY: dev
dev: ## Run the API and restart it when a file changes
	$(AIR) --build.cmd "go build -o bin/api ./cmd/api" --build.bin "bin/api" --build.exclude_dir "bin,data,site,deployments"

.PHONY: build
build: ## Build the api and migrate binaries into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/api ./cmd/api
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/migrate ./cmd/migrate

.PHONY: generate
generate: ## Regenerate configs/env.example and the configuration reference
	go run ./cmd/api -print-config env > configs/env.example
	go run ./cmd/api -print-config markdown > docs/reference/configuration.md # init:site

# init:example:begin
.PHONY: resource
resource: ## Scaffold a resource from the tasks example: make resource name=project
	@test -n "$(name)" || (echo "usage: make resource name=<singular>" && exit 1)
	go run ./tools/scaffold -name $(name) $(if $(plural),-plural $(plural))
# init:example:end

##@ Quality

.PHONY: fmt
fmt: ## Format the code
	$(GOLANGCI_LINT) fmt

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: vuln
vuln: ## Scan dependencies and code for known vulnerabilities
	$(GOVULNCHECK) ./...

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	go mod tidy

.PHONY: test
test: ## Run the tests
	go test ./...

.PHONY: test-race
test-race: ## Run the tests with the race detector
	go test -race ./...

.PHONY: cover
cover: ## Run the tests with the race detector and write coverage.out
	go test -race -covermode=atomic -coverpkg=./internal/... -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -n 1

.PHONY: check
check: tidy vet lint test-race ## Run everything CI runs on the code

##@ Database

.PHONY: migrate-up
migrate-up: ## Apply pending migrations
	go run ./cmd/migrate up

.PHONY: migrate-down
migrate-down: ## Roll back the most recent migration
	go run ./cmd/migrate down

.PHONY: migrate-status
migrate-status: ## Show which migrations are applied
	go run ./cmd/migrate status

##@ Containers

.PHONY: docker-build
docker-build: ## Build the container image
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg DATE=$(DATE) -t $(IMAGE) .

# init:compose:begin
.PHONY: compose-up
compose-up: ## Start the API with PostgreSQL
	$(COMPOSE) up --build -d

# init:observability:begin
.PHONY: obs-up
obs-up: ## Start the API with PostgreSQL, Prometheus, Grafana and Tempo
	$(COMPOSE) --profile observability up --build -d
# init:observability:end

.PHONY: compose-down
compose-down: ## Stop and remove the compose stack
	$(COMPOSE) --profile "*" down --remove-orphans
# init:compose:end

# init:template:begin
##@ Template

.PHONY: init
init: ## Turn this template into your project (interactive)
	go run ./tools/init

.PHONY: test-init
test-init: ## Check that the init tool produces working projects for every option
	sh scripts/test-init.sh

.PHONY: docs-serve
docs-serve: ## Serve the documentation site at http://localhost:8000
	mkdocs serve

.PHONY: docs-build
docs-build: ## Build the documentation site into site/
	mkdocs build --strict
# init:template:end

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} \
		/^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} \
		/^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
