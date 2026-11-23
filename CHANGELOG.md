# Changelog

All notable changes to this project are documented in this file. From 3.0.0
onwards it is maintained by [release-please](https://github.com/googleapis/release-please)
from [Conventional Commits](https://www.conventionalcommits.org/).

## [3.0.1](https://github.com/antoniopaya22/go-rest-template/compare/v3.0.0...v3.0.1) (2026-11-23)


### Bug Fixes

* add the /v3 suffix to the module path ([#16](https://github.com/antoniopaya22/go-rest-template/issues/16)) ([b4bec0f](https://github.com/antoniopaya22/go-rest-template/commit/b4bec0f7272271f62091df778146ec12b6692428))

## [3.0.0](https://github.com/antoniopaya22/go-rest-template/compare/2.0.1...v3.0.0) (2026-11-16)


### ⚠ BREAKING CHANGES

* the HTTP API, the configuration variables and the database schema are not compatible with 2.x. See the migration guide.

### Features

* add the init tool and the resource generator ([122950b](https://github.com/antoniopaya22/go-rest-template/commit/122950b5bdc0fd6d91046eff68686228fbddaac9))
* rewrite the core around feature packages ([4247ac9](https://github.com/antoniopaya22/go-rest-template/commit/4247ac902b601918f446a293a019359f803fc6b4))


### Bug Fixes

* repair the Docker build and harden CI ([#12](https://github.com/antoniopaya22/go-rest-template/issues/12)) ([f06c837](https://github.com/antoniopaya22/go-rest-template/commit/f06c8378298dfb31f440b00a537a53d9bc850efe))
* return timestamps in UTC on every database driver ([b586f75](https://github.com/antoniopaya22/go-rest-template/commit/b586f752f82a04ac194f29101c549879178cbcca))


### Documentation

* add the project website ([0979ef7](https://github.com/antoniopaya22/go-rest-template/commit/0979ef7303b2a7a07ef1f093b19f4a8d72546300))
* rewrite the README for v3 and seed the changelog ([2fe3065](https://github.com/antoniopaya22/go-rest-template/commit/2fe306587f868f400f5a24dc7ca9e489fad741f8))

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
