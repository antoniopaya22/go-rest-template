# Changelog

All notable changes to this project are documented in this file. From 3.0.0
onwards it is maintained by [release-please](https://github.com/googleapis/release-please)
from [Conventional Commits](https://www.conventionalcommits.org/).

## 2.0.1 (2026-10-13)

Security release.

### Security

- The client IP is no longer taken from `X-Forwarded-For` or `X-Real-IP` headers that a client can forge ([GHSA-3fxj-6jh8-hvhx](https://github.com/advisories/GHSA-3fxj-6jh8-hvhx)). It comes from the TCP peer by default, or from the entry added by your own proxies when `HTTP_TRUSTED_PROXIES` is set.
- Direct dependencies with known vulnerabilities (chi, kin-openapi, pgx, x/text, x/crypto, x/sys) are upgraded.

### Tooling

- CI runs `govulncheck`. CodeQL, OpenSSF Scorecard and grouped Dependabot updates are enabled.

## 2.0.0 (2026-10-13)

A complete rewrite on modern Go: chi, `slog`, an OpenAPI 3.1 contract with
request and response validation, JWT authentication, PostgreSQL with SQLite
for local development, health endpoints and Prometheus metrics.

### Breaking changes

- Gin is replaced by chi; the HTTP layer, the project layout and the configuration are new.
- The module path is `github.com/antoniopaya22/go-rest-template`.

## 1.0.0 (2020-09-11)

The original Gin-based template.
