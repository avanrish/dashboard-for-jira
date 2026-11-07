<!-- LINT.IfChange(project_header) -->
# dashboard-for-jira

A web dashboard for Jira.
<!-- LINT.ThenChange(//README.md:project_header) -->

## Project structure

<!-- LINT.IfChange(project_structure) -->
- `cmd/<name>/` — one directory per executable; keep `main.go` thin and call into `internal/`
- `internal/` — application code, not importable by other modules
- `docs/decisions/` — architecture decision records explaining why the project is built the way it is
- Tests live next to the code they test as `*_test.go` files, with fixtures in a package-local `testdata/`
<!-- LINT.ThenChange(//README.md:project_structure) -->

## Commands

<!-- LINT.IfChange(commands) -->
Run `mise install` to provision Go and the development tools pinned in `mise.toml`, then use mise tasks (`mise tasks` lists them):

- `mise run fmt` — format Go code
- `mise run lint` — formatting, `go.mod` tidiness, govulncheck, and golangci-lint. Always run after making changes.
- `mise run test` — tests with the race detector
- `mise run doc` — browse the documentation locally in the pkg.go.dev style
<!-- LINT.ThenChange(//README.md:commands, //mise.toml:lint_task) -->

CI also runs checks that `mise run lint` does not: `typos`, `ifttt-lint`, and `gitleaks`. Their tools are pinned in `mise.ci.toml`; install them with `MISE_ENV=ci mise install`, then run `typos` and `ifttt-lint --diff 'main..@'` (`main...HEAD` with git) before pushing.
CI runs tests on Linux.

## Conventions

<!-- LINT.IfChange(commit_subject) -->
- Write commit messages as Conventional Commits with a capitalized subject of at most 50 characters (e.g. `feat: Add login`). Check them with `mise run check-commits`.
<!-- LINT.ThenChange(//committed.toml:commit_subject) -->
- Use narrow `LINT.IfChange` / `LINT.ThenChange` directives when duplicated cross-file content must stay in sync and cannot be eliminated.
  Write targets as repository-root paths starting with `//` (e.g. `//mise.toml:go_version`); ifttt-lint enforces them in CI.

## Decisions

Before changing architecture, dependencies, or how the app talks to Jira, read the records in `docs/decisions/`.
Do not go against an accepted record silently: propose a new record that supersedes it.
When you make a decision a reader could not work out from the code, add a record following `docs/decisions/README.md`.

## Version control

If a `.jj` directory exists in the repository root, default to using `jj` (Jujutsu) for version control; otherwise, use `git`.

- Every commit on `main` must pass CI. Pull requests land as a squash or a merge commit, so a PR's own commits need not each pass.
- Keep commits atomic: one responsibility each, with code, tests, and docs updated together. Put file renames in their own commit.
- Squash pull requests by default. Use a merge commit only when the history is worth keeping, such as what was tried or how a bug was tracked down.
