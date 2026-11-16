#!/usr/bin/env sh
# Generates projects from the template with different options and checks that
# each one builds, is formatted and passes its tests. Run it with `make test-init`.
set -eu

root=$(pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# name | flags passed to the init tool
cases='
defaults|
postgres-only|-db postgres
sqlite-only|-db sqlite
bare|-db sqlite -example=false -metrics=false -tracing=false -ratelimit=false -devcontainer=false
no-example|-example=false
no-metrics|-metrics=false
no-tracing|-tracing=false
no-ratelimit|-ratelimit=false
no-extras|-helm=false -observability=false -devcontainer=false
'

echo "$cases" | while IFS='|' read -r name flags; do
	[ -n "$name" ] || continue
	echo "==> $name"

	project="$work/$name"
	mkdir -p "$project"
	# Copy the tracked and untracked files, without build output or history.
	git ls-files -co --exclude-standard | while read -r file; do
		[ -e "$file" ] || continue
		mkdir -p "$project/$(dirname "$file")"
		cp "$file" "$project/$file"
	done

	# shellcheck disable=SC2086 # the flags are meant to be split.
	go run ./tools/init -yes -dir "$project" -module "example.com/acme/$name" $flags > "$project/init.log" || {
		cat "$project/init.log"
		exit 1
	}

	cd "$project"

	if grep -rIl --exclude-dir=.git -e 'antoniopaya22/go-rest-template' -e 'init:[a-z]*' . | grep -v -e '^./README.md$' -e '^./init.log$'; then
		echo "the files above still mention the template or carry init markers" >&2
		exit 1
	fi

	unformatted=$(gofmt -l .)
	if [ -n "$unformatted" ]; then
		echo "not formatted: $unformatted" >&2
		exit 1
	fi

	go build ./...
	go vet ./...
	if [ "${INIT_LINT:-0}" = 1 ]; then
		make -s lint
	fi
	go test ./... > test.log 2>&1 || {
		cat test.log
		exit 1
	}

	# A generated project keeps the resource generator when it keeps the example.
	if [ -d tools/scaffold ]; then
		go run ./tools/scaffold -name project > /dev/null
		go build ./...
		go test ./internal/... > test.log 2>&1 || {
			cat test.log
			exit 1
		}
	fi

	cd "$root"
done

echo "every generated project builds and passes its tests"
