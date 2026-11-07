<!-- LINT.IfChange(project_header) -->
# dashboard-for-jira

A web dashboard for Jira.
<!-- LINT.ThenChange(//AGENTS.md:project_header) -->

## Project structure

<!-- LINT.IfChange(project_structure) -->
- `cmd/<name>/` — one directory per executable; keep `main.go` thin and call into `internal/`
- `internal/` — application code, not importable by other modules
- `docs/decisions/` — architecture decision records explaining why the project is built the way it is
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

## Web UI

<!-- LINT.IfChange(web_ui) -->
The UI is rendered on the server from templ templates in `internal/web/`, styled with Tailwind classes compiled into `internal/web/static/app.css`.

- `mise run dev` — run the server with live reload at http://127.0.0.1:7331
- `mise run generate` — regenerate templ code and the Tailwind stylesheet. Run after editing `.templ` files or Tailwind classes, and commit the output; CI fails if it is stale.
<!-- LINT.ThenChange(//AGENTS.md:web_ui) -->

## License

[MIT](LICENSE)

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by Atlassian.
Jira is a trademark of Atlassian Pty Ltd.
