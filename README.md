# Pixels

[![CI](https://github.com/niflaot/pixels/actions/workflows/ci.yml/badge.svg)](https://github.com/niflaot/pixels/actions/workflows/ci.yml)

Pixels is a fast, idiomatic Go emulator for the pixel protocol. The project is intentionally small at the core, with reusable infrastructure in `pkg/`, realm behavior in `internal/`, packet logic in `networking/`, and controlled plugin-facing APIs in `sdk/`.

## Status

Pixels is being bootstrapped. The current module provides the first package boundaries, documentation rules, and CI checks that compile, vet, test, and enforce coverage.

## Layout

```text
pkg/         reusable global components
internal/    emulator-only realm features
networking/  pixel-protocol packet coding and decoding
sdk/         controlled plugin creation surface
```

## Development

Run the full local check:

```sh
go test ./...
```

Run the CI-equivalent coverage check:

```sh
go test -race -covermode=atomic -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Project rules for agents and contributors live in `AGENTS.md`.
