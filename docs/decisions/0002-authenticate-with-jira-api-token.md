# 0002. Authenticate to Jira Cloud with a personal API token kept in the password store

- Status: Accepted
- Date: 2026-11-07

## Context

The dashboard talks to Jira Cloud's REST API on behalf of one person, the developer running it on their own machine.
There are no other users, no hosted deployment, and no sign-in screen to build.
People install it with `go install` ([ADR 0006](0006-distribute-through-go-install.md)) and run it from any directory, without a clone of the repository.

Jira Cloud offers these ways for an external app to authenticate:

- **API token with basic auth**: the user creates a token in their Atlassian account and sends it with their email address.
  Classic (unscoped) tokens act with the user's full permissions against `https://<site>.atlassian.net`.
  Scoped tokens are limited to chosen scopes and are sent through the gateway at `https://api.atlassian.com/ex/jira/<cloudId>`.
  Tokens expire, at the latest one year after creation.
- **OAuth 2.0 (3LO)**: the app is registered in the Atlassian developer console, and each user signs in through a browser redirect.
  The app stores and refreshes access tokens.
- **Forge or Connect apps**: run inside or alongside Jira, and are meant for apps installed on a Jira site.

## Decision

- Authenticate with a classic (unscoped) API token and the user's Atlassian email address over HTTP basic auth, sent to the site URL `https://<site>.atlassian.net`.
  It is the simplest option: one token, no cloud ID, and no gateway.
  The app only reads from Jira, but the token itself is not limited to reading.
- Read settings from `config.json` in a `dashboard-for-jira` directory under the user's config directory: `$XDG_CONFIG_HOME` or `~/.config` on macOS and Linux, and `%AppData%` on Windows.
  On macOS this deliberately differs from Go's `os.UserConfigDir`, which returns `~/Library/Application Support`, because command-line tools keep their settings in `~/.config`.
  Unknown keys are an error, so a mistyped key is reported instead of ignored.
  Environment variables, such as `JIRA_SITE`, take precedence over the file.
- Read the API token from `JIRA_API_TOKEN`, or else from the system's password store: Keychain on macOS, Credential Manager on Windows, and the Secret Service on Linux.
  Use `github.com/zalando/go-keyring`, which needs no cgo, so `go install` needs no C compiler.
- Never keep the token in a file: the app refuses to start if `config.json` holds one.
  Without a password store, the token comes from the environment only.
- `dashboard-for-jira start` asks in the terminal for any missing setting instead of failing, and `dashboard-for-jira configure` asks for all of them.
  Both check the settings against Jira before saving them.
- Only the Go server holds the token.
  It never sends the token to the browser or writes it to logs.

## Alternatives considered

- **Scoped token with read-only scopes**: limits the damage if the token leaks, but adds a cloud ID, the `api.atlassian.com` gateway, and uncertainty about whether scoped tokens work with basic auth.
  Switching needs only a new token, the cloud ID, and a different base URL.
- **OAuth 2.0 (3LO)**: per-user sign-in and fine-grained scopes, but it needs a registered app, a client secret, a local callback handler, and token refresh logic.
  That is a lot of machinery for one person on one machine.
- **Forge app**: runs inside Jira on Atlassian's infrastructure, which contradicts running a small local tool with its own web interface.
- **A `.env` file in the current directory**, which the app first read: suits a clone of the repository, but an installed binary would read whichever `.env` it is started next to, such as a web project's.
- **The token in `config.json`, readable only by the user**, always or only without a password store: simplest, but plain text that leaks with backups and synced dotfiles.
- **TOML or YAML for the settings file**: easier to edit by hand, but each needs a dependency, and the app writes the file itself.
- **`github.com/99designs/keyring`**: supports more backends, but uses cgo for the macOS Keychain, so installing would need a C compiler.
- **A setup page in the web UI**: friendlier than a terminal prompt, but a form that accepts the token adds to what the local server must defend.

## Consequences

- There is no app registration and no separate setup step: the first `start` asks for everything.
- The token is encrypted at rest, but programs running as the user can still read it: on macOS, go-keyring reads it through `/usr/bin/security` without a prompt.
  It can do anything the user can do in their Atlassian products.
- The token expires, so the app should show a clear message when Jira returns `401`, telling the user to run `configure` with a new token.
- Linux without a Secret Service, such as a server, a container, or WSL, needs `JIRA_API_TOKEN` in the environment.
- A server on `127.0.0.1` holding a token can still be reached by malicious web pages in the same browser.
  The server must check the `Host` header against an allowlist, which blocks DNS rebinding.
  It must also reject cross-origin requests that change state.
- Hosting the dashboard for others would invalidate this decision and require OAuth 2.0.
