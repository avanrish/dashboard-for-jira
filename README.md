<!-- LINT.IfChange(project_header) -->
# dashboard-for-jira

A web dashboard for Jira.
<!-- LINT.ThenChange(//AGENTS.md:project_header) -->

## Project structure

<!-- LINT.IfChange(project_structure) -->
- `cmd/<name>/` — one directory per executable; keep `main.go` thin and call into `internal/`
- `internal/` — application code, not importable by other modules
- Tests live next to the code they test as `*_test.go` files, with fixtures in a package-local `testdata/`
<!-- LINT.ThenChange(//AGENTS.md:project_structure) -->

## Commands

<!-- LINT.IfChange(commands) -->
Run `mise install` to provision Go and the development tools pinned in `mise.toml`, then use mise tasks (`mise tasks` lists them):

- `mise run fmt` — format Go code
- `mise run lint` — formatting, `go.mod` tidiness, govulncheck, and golangci-lint. Always run after making changes.
- `mise run test` — tests with the race detector
- `mise run doc` — browse the documentation locally in the pkg.go.dev style
<!-- LINT.ThenChange(//AGENTS.md:commands, //mise.toml:lint_task) -->
