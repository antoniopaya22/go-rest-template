#!/usr/bin/env sh

set -eu

task="${1:-}"
go_bin="${2:-go}"
binary="${3:-./bin/api}"
compose_file="${4:-deployments/docker-compose.yml}"
golangci_lint="${5:-}"

if [ -z "$task" ]; then
	echo "task name is required" >&2
	exit 1
fi

run_lint() {
	if [ -n "$golangci_lint" ]; then
		exec "$golangci_lint" run
	fi

	exec "$go_bin" run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run
}

case "$task" in
	tidy)
		exec "$go_bin" mod tidy
		;;
	fmt)
		exec "$go_bin" fmt ./...
		;;
	vet)
		exec "$go_bin" vet ./...
		;;
	lint)
		run_lint
		;;
	test)
		exec "$go_bin" test ./...
		;;
	integration)
		exec "$go_bin" test ./test -run TestAPIIntegration -count=1
		;;
	test-race)
		exec "$go_bin" test -race ./...
		;;
	cover)
		"$go_bin" test -coverprofile=coverage.out ./...
		exec "$go_bin" tool cover -func=coverage.out
		;;
	build)
		exec "$go_bin" build -trimpath -ldflags="-s -w" -o "$binary" ./cmd/api
		;;
	run)
		exec "$go_bin" run ./cmd/api
		;;
	migrate)
		exec "$go_bin" run ./cmd/migrate
		;;
	compose-up)
		exec docker compose -f "$compose_file" up --build -d
		;;
	compose-down)
		exec docker compose -f "$compose_file" down --remove-orphans
		;;
	*)
		echo "unsupported task: $task" >&2
		exit 1
		;;
esac