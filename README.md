# dashboard-for-jira

A web dashboard for Jira.

## Install

With [Go](https://go.dev/dl/) installed, run:

```sh
go install github.com/avanrish/dashboard-for-jira@latest
```

This builds `dashboard-for-jira` into `$(go env GOPATH)/bin`, usually `~/go/bin`; add that directory to your `PATH` to run it from anywhere.
Replace `@latest` with a release tag, such as `@v0.1.0`, to install that version, and run the command again to update.

Then start the dashboard:

```sh
dashboard-for-jira start
```

The first time, it asks for your Jira site, email address, API token, and board, checks them with Jira, and saves them as described under [Configuration](#configuration); after that, it asks only for a setting that is missing.
The dashboard is then at <http://127.0.0.1:8080>.

To change a setting, or to replace an expired token, run `dashboard-for-jira configure`; press Enter to keep a value.
Run `dashboard-for-jira` on its own to list the commands.

## Project structure

- `main.go` — the executable, at the repository root so `go install github.com/avanrish/dashboard-for-jira@latest` installs it; keep it thin and call into `internal/`
- `internal/` — application code, not importable by other modules
- `docs/decisions/` — architecture decision records explaining why the project is built the way it is
- Tests live next to the code they test as `*_test.go` files, with fixtures in a package-local `testdata/`

## Commands

<!-- LINT.IfChange(commands) -->
Run `mise install` to provision Go and the development tools pinned in `mise.toml`, then use mise tasks (`mise tasks` lists them):

- `mise run fmt` — format Go code
- `mise run lint` — formatting, `go.mod` tidiness, govulncheck, and golangci-lint. Always run after making changes.
- `mise run test` — tests with the race detector
- `mise run doc` — browse the documentation locally in the pkg.go.dev style
<!-- LINT.ThenChange(//mise.toml:lint_task) -->

## Web UI

The server in `internal/web/` renders pages from templ components in `internal/web/views/`, each reusable component in its own file.
They are styled with Tailwind classes, compiled from `internal/web/tailwind.css` into `internal/web/static/app.css`.
Use only the colour roles defined in `internal/web/tailwind.css`, such as `bg-card`, `text-fg-muted`, and `border-line`.
Tailwind's own palette is removed, so classes like `text-slate-500` produce nothing.
Each role has a light and a dark value, so templates need no `dark:` classes; when no role fits, add one with both.
Interactions that need no server round trip use Alpine.js, vendored in `internal/web/static/vendor/` with its version in the file name.
Put JavaScript that only one page uses in a `<script>` element in that page's templ component, not in a file under `internal/web/static/`, which holds only the compiled stylesheet and vendored libraries.

- `mise run dev` — run the server with live reload at http://127.0.0.1:7331
- `mise run generate` — regenerate templ code and the Tailwind stylesheet. Run after editing `.templ` files or Tailwind classes, and commit the output; CI fails if it is stale.
  Never edit `*_templ.go` or `internal/web/static/app.css` by hand, including through a search-and-replace across the repository: change the `.templ` or Tailwind source and run `mise run generate`.

## Configuration

`dashboard-for-jira start` asks for missing settings and `dashboard-for-jira configure` changes them; you can also edit the file yourself.
The app reads them from environment variables and from a settings file, `config.json`:

- macOS and Linux: `~/.config/dashboard-for-jira/config.json`, or under `$XDG_CONFIG_HOME` when it is set
- Windows: `%AppData%\dashboard-for-jira\config.json`

```json
{
  "site": "example.atlassian.net",
  "email": "ada@example.com",
  "boardId": 12
}
```

Variables set in the environment take precedence over the file:

| Key | Variable | Description |
| --- | --- | --- |
| `site` | `JIRA_SITE` | Your Jira Cloud site, without `https://`, such as `example.atlassian.net` |
| `email` | `JIRA_EMAIL` | The email address of your Atlassian account |
| `boardId` | `JIRA_BOARD_ID` | The number in the board's address: `.../boards/<id>` |
| `port` | `PORT` | Optional. The port on `127.0.0.1` to serve the dashboard on; `mise run dev` expects the default, 8080 |

The API token never goes in the file.
Create one at <https://id.atlassian.com/manage-profile/security/api-tokens>; it expires within a year.
The app reads it from the `JIRA_API_TOKEN` environment variable, or else from the system's password store: Keychain on macOS, Credential Manager on Windows, or the Secret Service, such as GNOME Keyring or KWallet, on Linux.
There it is saved under the service `dashboard-for-jira` and your email address.
On a machine without a password store, such as a server without a desktop session, set `JIRA_API_TOKEN` in the environment.

## License

[MIT](LICENSE)

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by Atlassian.
Jira is a trademark of Atlassian Pty Ltd.
