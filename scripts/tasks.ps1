param(
    [Parameter(Mandatory = $true)]
    [string]$Task,
    [string]$Go = "go",
    [string]$Binary = "./bin/api",
    [string]$ComposeFile = "deployments/docker-compose.yml",
    [string]$GolangCILint = ""
)

$ErrorActionPreference = "Stop"

switch ($Task) {
    "tidy" {
        & $Go mod tidy
    }
    "fmt" {
        & $Go fmt ./...
    }
    "vet" {
        & $Go vet ./...
    }
    "lint" {
        if ($GolangCILint -ne "") {
            & $GolangCILint run
        } else {
            & $Go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run
        }
    }
    "test" {
        & $Go test ./...
    }
    "integration" {
        & $Go test ./test -run TestAPIIntegration -count=1
    }
    "test-race" {
        & $Go test -race ./...
    }
    "cover" {
        & $Go test -coverprofile=coverage.out ./...
        & $Go tool cover -func=coverage.out
    }
    "build" {
        & $Go build -trimpath -ldflags "-s -w" -o $Binary ./cmd/api
    }
    "run" {
        & $Go run ./cmd/api
    }
    "migrate" {
        & $Go run ./cmd/migrate
    }
    "compose-up" {
        docker compose -f $ComposeFile up --build -d
    }
    "compose-down" {
        docker compose -f $ComposeFile down --remove-orphans
    }
    default {
        throw "unsupported task: $Task"
    }
}