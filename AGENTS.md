# Instructions for AI agents

`README.md` is written for people and is part of these instructions: read it before starting a task.
It covers installing and configuring the app, the project structure, the development commands and which to run after a change, and how the web UI is built and styled.
This file adds only what agents need on top of it.

## CI

CI also runs checks that `mise run lint` does not: `typos`, `ifttt-lint`, and `gitleaks`. Their tools are pinned in `mise.ci.toml`; install them with `MISE_ENV=ci mise install`, then run `typos` and `ifttt-lint --diff 'main..@'` (`main...HEAD` with git) before pushing.
CI runs tests on Linux.

## Configuration

`internal/configure/` holds the prompts of `dashboard-for-jira configure` and `start`; both check the settings with Jira before saving them.
The API token is read and saved through `github.com/zalando/go-keyring`.
Tests must not touch the real password store: call `keyring.MockInit()` in any test that reads or saves a token.

## Conventions

<!-- LINT.IfChange(commit_subject) -->
- Write commit messages as Conventional Commits with a capitalized subject of at most 50 characters (e.g. `feat: Add login`). Check them with `mise run check-commits`.
<!-- LINT.ThenChange(//committed.toml:commit_subject) -->
- Use narrow `LINT.IfChange` / `LINT.ThenChange` directives when duplicated cross-file content must stay in sync and cannot be eliminated.
  Write targets as repository-root paths starting with `//` (e.g. `//mise.toml:go_version`); ifttt-lint enforces them in CI.
  Put them inside the declaration they guard, around only the lines that must stay in sync; placed directly above a declaration, Go reads them as its doc comment.
- Prefer building strings from parts with `fmt.Sprintf`, and errors with `fmt.Errorf`, over joining them with `+`; this includes expressions in `.templ` files.
  Appending a piece to a string with `+=` is fine.
  Prefer building URLs with `net/url` (`url.URL`, `JoinPath`, `url.Values`).
- Take optional per-call settings as an exported options struct whose zero value means the defaults (e.g. `RequestOptions{}`), not as variadic functional options.
  The zero value should be the safe choice: `RequestOptions{}` always asks Jira, and a caller gets a cached response only by setting `CacheTTL`.
- Ask before adding a `//nolint` directive, and only for a finding that `mise run lint` actually reports and that cannot be fixed.
- Separate the steps of a long function with blank lines.
- Write a comment only where a reader would otherwise misread the code; most choices need no explanation.
- Change only what the task asks for; propose extra tests, checks, or safeguards instead of adding them.

## Decisions

Before changing architecture, dependencies, or how the app talks to Jira, read the records in `docs/decisions/`.
Do not go against an accepted record silently: propose a new record that supersedes it.
Add a record only for a decision that shapes the whole project and is costly to reverse, such as the stack, how the app authenticates, where it keeps data, or how it is distributed; most changes need none.
Write records following `docs/decisions/README.md`, and never rewrite one that has been pushed: amend or supersede it with a new record.

## Version control

If a `.jj` directory exists in the repository root, default to using `jj` (Jujutsu) for version control; otherwise, use `git`.

- Every commit on `main` must pass CI. Pull requests land as a squash or a merge commit, so a PR's own commits need not each pass.
- Keep commits atomic: one responsibility each, with code, tests, and docs updated together. Put file renames in their own commit.
- Squash pull requests by default. Use a merge commit only when the history is worth keeping, such as what was tried or how a bug was tracked down.
- A release is a semantic version tag on `main`, such as `v0.1.0`, which `go install` resolves. Never move or delete a pushed tag: the Go module proxy keeps the first version it saw.
