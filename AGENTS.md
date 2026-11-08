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

## Web UI

<!-- LINT.IfChange(web_ui) -->
The server in `internal/web/` renders pages from templ components in `internal/web/views/`, each reusable component in its own file.
They are styled with Tailwind classes, compiled from `internal/web/tailwind.css` into `internal/web/static/app.css`.
Use only the colour roles defined in `internal/web/tailwind.css`, such as `bg-card`, `text-fg-muted`, and `border-line`.
Tailwind's own palette is removed, so classes like `text-slate-500` produce nothing.
Each role has a light and a dark value, so templates need no `dark:` classes; when no role fits, add one with both.

- `mise run dev` — run the server with live reload at http://127.0.0.1:7331
- `mise run generate` — regenerate templ code and the Tailwind stylesheet. Run after editing `.templ` files or Tailwind classes, and commit the output; CI fails if it is stale.
<!-- LINT.ThenChange(//README.md:web_ui) -->

CI also runs checks that `mise run lint` does not: `typos`, `ifttt-lint`, and `gitleaks`. Their tools are pinned in `mise.ci.toml`; install them with `MISE_ENV=ci mise install`, then run `typos` and `ifttt-lint --diff 'main..@'` (`main...HEAD` with git) before pushing.
CI runs tests on Linux.

## Configuration

<!-- LINT.IfChange(configuration) -->
The server reads its settings from environment variables and from a `.env` file in the current directory; variables set in the environment take precedence.
Copy `.env.example` to `.env` and fill in your Jira site, email, API token, and board ID before running the server.
<!-- LINT.ThenChange(//README.md:configuration) -->

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
