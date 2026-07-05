# AGENTS.md

This repository contains Pixels, a fast and idiomatic Go emulator for the pixel protocol. Agents working here should keep the codebase simple, reusable, and easy to extend without hiding behavior behind premature abstractions.

## Project Layout

- `pkg/` contains reusable global components such as storage, Redis, WebSocket helpers, logging, and process utilities.
- `internal/` contains pixel-protocol realm features that are private to this emulator.
- `networking/` contains pixel-protocol packet coding, decoding, framing, and transport logic.
- `sdk/` contains controlled reusable implementations for plugin creation and extension points.

## Package Rules

- Prefer small, single-name packages with nested paths over long package names.
- Use `networking/session/ping/packet.go` and `networking/session/ping/packet_test.go` instead of names like `networking/session/pingpacket.go`.
- Keep each package focused on one responsibility.
- Keep each file focused on one responsibility.
- Keep every file at or below 250 lines.
- Keep each package to a maximum of six file pairs, where `hello.go` plus `hello_test.go` counts as one pair.
- If a package needs more tests after six file pairs, create a `tests/` folder inside that package.

## Go Style

- Write Go the Go way: clear data flow, small functions, explicit errors, goroutines where concurrency is natural, and channels only where they clarify ownership.
- Document every package, function, method, struct, interface, type, const, var, and test helper in Go doc style.
- Do not add comments inside function bodies.
- Avoid unnecessary interfaces. Introduce an interface only when it decouples a real boundary, supports multiple implementations, or enables focused tests.
- Prefer composition over inheritance-like hierarchies.
- Keep public APIs conservative and stable.
- Keep private APIs readable enough that new contributors can follow them quickly.

## Testing

- All code must maintain more than 80% test coverage.
- Add tests with every behavioral change.
- Keep tests focused on behavior, not implementation details.
- Prefer table-driven tests when cases share the same setup.
- CI must compile and test the full module before changes are considered ready.

## SDK Rules

- Treat `sdk/` as a controlled extension surface.
- Ask before adding new SDK APIs, exported types, extension hooks, or compatibility promises.
- Keep SDK additions decoupled from realm internals.
- Prefer explicit capability objects and small contracts over broad plugin access.

## Change Discipline

- Keep changes scoped to the requested behavior.
- Do not refactor unrelated code while implementing a feature.
- Split responsibilities before files or packages become hard to scan.
- Preserve the legacy tree unless the task explicitly targets it.
