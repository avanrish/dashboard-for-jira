# 0004. Run the templ CLI as a Go tool

- Status: Accepted
- Date: 2026-11-07

## Context

[ADR 0001](0001-server-rendered-go-with-htmx.md) renders HTML with templ and pins its CLI in `mise.toml`, like the project's other tools.
templ has two parts: the CLI that generates Go code from `.templ` files, and the `github.com/a-h/templ` library that the generated code calls.
They must be the same version: templ warns about a mismatch, and newer generated code can call library functions that an older library lacks.
With the CLI in `mise.toml` and the library in `go.mod`, the version lives in two places, and Renovate updates them in separate pull requests.

## Decision

Add the CLI to `go.mod` with `go get -tool github.com/a-h/templ/cmd/templ` and run it as `go tool templ`.
The CLI and the library come from the same module, so one version in `go.mod` pins both.

This amends ADR 0001 for templ only; the Tailwind CLI stays in `mise.toml`.

## Alternatives considered

- **Pin the CLI in `mise.toml` with the `go:` backend**, as ADR 0001 said and as govulncheck is pinned: keeps the CLI's dependencies out of `go.mod`, but the two versions can drift apart.

## Consequences

- Updating templ is a single `go get` or a single Renovate pull request.
- The CLI's dependencies, about 14 modules, appear in `go.mod` as indirect requirements.
  They are not linked into the app binary, but they share one version of each module with the app, so the CLI can raise the versions the app links.
- templ is the only development tool not listed in `mise.toml`.
  Add a tool to `go.mod` only when it must match the version of a library in `go.mod`; other Go tools, such as golangci-lint and govulncheck, stay in `mise.toml`.
