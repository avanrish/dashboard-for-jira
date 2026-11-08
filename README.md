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
The server in `internal/web/` renders pages from templ components in `internal/web/views/`, each reusable component in its own file.
They are styled with Tailwind classes, compiled from `internal/web/tailwind.css` into `internal/web/static/app.css`.
Use only the colour roles defined in `internal/web/tailwind.css`, such as `bg-card`, `text-fg-muted`, and `border-line`.
Tailwind's own palette is removed, so classes like `text-slate-500` produce nothing.
Each role has a light and a dark value, so templates need no `dark:` classes; when no role fits, add one with both.

- `mise run dev` — run the server with live reload at http://127.0.0.1:7331
- `mise run generate` — regenerate templ code and the Tailwind stylesheet. Run after editing `.templ` files or Tailwind classes, and commit the output; CI fails if it is stale.
<!-- LINT.ThenChange(//AGENTS.md:web_ui) -->

## Configuration

<!-- LINT.IfChange(configuration) -->
The server reads its settings from environment variables and from a `.env` file in the current directory; variables set in the environment take precedence.
Copy `.env.example` to `.env` and fill in your Jira site, email, API token, and board ID before running the server.
<!-- LINT.ThenChange(//AGENTS.md:configuration) -->

## License

[MIT](LICENSE)

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by Atlassian.
Jira is a trademark of Atlassian Pty Ltd.
