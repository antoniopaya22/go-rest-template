# Project Layout

This template follows the common Go application layout patterns that are useful for a production-oriented service without turning the repository into ceremony-heavy scaffolding.

## Directories in Use

- `cmd/`: thin entrypoints for the API server and migration command.
- `internal/app/`: private application runtime wiring.
- `internal/app/api/`: API-specific auth, OpenAPI contract, transport, domain services and storage implementations.
- `internal/pkg/`: shared internal support code such as configuration loading and the application error model.
- `api/`: OpenAPI contract and API surface definition.
- `configs/`: environment template and future default configuration assets.
- `build/package/`: container packaging assets.
- `docs/`: design and repository documentation beyond package docs.
- `test/`: external integration tests.
- `old/`: preserved legacy implementation for migration reference.

## Intentional Exceptions

- `.github/workflows/ci.yml` stays where GitHub expects it; `build/ci/` is used for supporting CI assets and documentation.
- `Makefile` remains at the repository root because it is still the conventional entrypoint for developer tasks.
- `migrations/` stays top-level because both runtime code and operators interact with it directly as a deployment artifact.

## What This Avoids

- No `src/` directory.
- No public `pkg/` tree for code that is only meant to support this application.
- No extra architectural nesting unless the codebase actually grows enough to justify it.