# 0004. Distribute the app through `go install` only

- Status: Accepted
- Date: 2026-11-08

## Context

People should be able to run the dashboard without cloning the repository: install one binary, put it on their `PATH`, and run it.
The app is a single Go binary with its stylesheet embedded, the generated templ code committed, and no cgo, so it builds anywhere Go runs.

The usual ways to ship a Go binary:

- **`go install <module>@<version>`**: builds from source on the user's machine.
  The command names the package path, so with `main.go` in `cmd/dashboard-for-jira/` it would be `go install github.com/avanrish/dashboard-for-jira/cmd/dashboard-for-jira@latest`.
- **Prebuilt binaries on GitHub Releases**, built by a tool such as GoReleaser when a tag is pushed.
  macOS blocks an unsigned binary downloaded through a browser ("Apple could not verify…") until the user clears the quarantine attribute or allows it in System Settings.
  Avoiding that needs a Developer ID certificate and notarization, which need the Apple Developer Program at $99 a year.
  Windows may show a SmartScreen warning for an unsigned `.exe` too.
- **A package manager**, such as a Homebrew tap, which needs a second repository and its upkeep.

## Decision

- Distribute the app only through `go install`.
- Keep the `main` package at the repository root, so the command is `go install github.com/avanrish/dashboard-for-jira@latest`, and the binary is named `dashboard-for-jira`.
- A release is a semantic version tag on `main`, such as `v0.1.0`.
  `@latest` resolves to the highest release tag, and `@v0.1.0` installs that version.
- Go records the version in the binary, so the app reads it from its build information instead of having it set at build time.

## Alternatives considered

- **Prebuilt binaries on GitHub Releases**: works without Go, but each platform needs its own workaround or a paid signing setup to install without warnings, and the release pipeline needs upkeep.
- **Release binaries for Linux and Windows only, with `go install` for macOS**: two ways to install, documented and tested separately, for little gain.
- **A Homebrew tap**: an extra repository and token to maintain.
- **`main` in `cmd/dashboard-for-jira/`**, the common layout for repositories with several executables: this repository has one, and the layout only makes the install command longer.

## Consequences

- Users need Go installed.
  A Go older than the version in `go.mod` downloads the right toolchain itself.
- Nothing to build, sign, or upload per release: releasing is pushing a tag.
- Each install builds from source, and the Go checksum database verifies that the source matches what everyone else gets for that version.
- The Go module proxy caches every version it has served, so a pushed tag cannot be changed or withdrawn.
  A broken release is fixed by a new one, and `retract` in `go.mod` can steer `@latest` away from it.
- A new executable would go in its own directory, with an install command of its own.
- If people without Go need the app, revisit prebuilt binaries, with signing and notarization for macOS.
