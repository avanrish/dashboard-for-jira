# 0007. Keep settings in the user's config directory and the API token in the password store

- Status: Accepted
- Date: 2026-11-08

## Context

[ADR 0002](0002-authenticate-with-jira-api-token.md) reads settings from a `.env` file in the current directory, with the API token in plain text.
That suits running the app from a clone of the repository.
Since [ADR 0006](0006-distribute-through-go-install.md), people install the binary with `go install` and run it from any directory, without the repository:

- There is no `.env` to copy from `.env.example`, and the app would read whichever `.env` is in the current directory, such as a web project's, whose `PORT` would move the dashboard.
- A token in a plain-text file leaks with anything that copies the file: backups, synced dotfiles, or a tool that reads files in the directory.

## Decision

- Read settings from `config.json` in a `dashboard-for-jira` directory under the user's config directory:
  `$XDG_CONFIG_HOME` or `~/.config` on macOS and Linux, and `%AppData%` on Windows.
  On macOS this deliberately differs from Go's `os.UserConfigDir`, which returns `~/Library/Application Support`, because command-line tools keep their settings in `~/.config`.
- The file is a JSON object with a key per setting, such as `site` and `boardId`, read with `encoding/json`.
  Unknown keys are an error, so a mistyped key is reported instead of ignored.
  Environment variables keep their names, such as `JIRA_SITE`, and still take precedence over the file.
- Stop reading `.env` from the current directory, and remove the `.env` parser.
- Read the API token from `JIRA_API_TOKEN`, or else from the system's password store: Keychain on macOS, Credential Manager on Windows, and the Secret Service on Linux.
  The entry's service is `dashboard-for-jira` and its account is the user's email address.
  Use `github.com/zalando/go-keyring`, which needs no cgo, so `go install` still needs no C compiler.
- Never keep the token in a file: `config.json` has no key for it, so the app refuses to start if the file holds one.
  Without a password store, the token comes from the environment only.
- `dashboard-for-jira start` asks in the terminal for any setting that is missing, instead of failing, and `dashboard-for-jira configure` asks for all of them, showing the current values.
  Both read the token without echoing it, using `golang.org/x/term`, check the settings against Jira, write `config.json` readable only by the user, and save the token in the password store.

This amends ADR 0002 for where settings and the token are kept; authenticating with a classic API token over basic auth stands.

## Alternatives considered

- **Keep reading `.env` from the current directory as well**: convenient in a clone of the repository, but surprising once the binary is installed, for the reason above.
- **Keep the `.env` format, `NAME=value` lines, for the settings file**: the existing parser reads it, but it has no types, no escaping, and no way to report a mistyped name, and the `JIRA_` names only make sense as environment variables.
- **TOML**: allows comments and is easier to edit by hand than JSON, but needs a dependency, and the app writes the file itself.
- **YAML**: needs a dependency, and its implicit typing turns some unquoted values into booleans or numbers.
- **`~/Library/Application Support` on macOS**: what `os.UserConfigDir` returns, but the path has a space and is not where command-line users look.
- **The token in `config.json`, readable only by the user**: simplest, but still plain text.
- **Fall back to the file when there is no password store**, like `gh auth login --insecure-storage`: brings back plain text for the machines most likely to be shared; the environment variable covers them.
- **Encrypt the file with a passphrase**: the user would type it on every start, or the app would need somewhere to keep it.
- **`github.com/99designs/keyring`**: supports more backends, but uses cgo for the macOS Keychain, so installing would need a C compiler.
- **A setup page in the web UI**: friendlier than a terminal prompt, but a form that accepts the token adds to what the local server must defend.

## Consequences

- There is no separate setup step: the first `start` asks for everything, and renewing an expired token is `configure`.
- The token is encrypted at rest and stays out of backups and dotfiles.
  It does not stop programs running as the user: on macOS, go-keyring stores the entry through `/usr/bin/security`, which can read it back without a prompt.
- Linux without a Secret Service, such as a server, a container, or WSL, needs `JIRA_API_TOKEN` in the environment.
- JSON allows no comments, so the README describes each key; a trailing comma or a missing quote stops the app with the parser's error.
- Tests must not touch the real password store; they call `keyring.MockInit()`.
- Development uses the same settings as an installed binary; environment variables override them, for example to try another board.
