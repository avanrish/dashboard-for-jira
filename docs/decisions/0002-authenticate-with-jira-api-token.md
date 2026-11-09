# 0002. Authenticate to Jira Cloud with a personal API token

- Status: Accepted, amended by [0007](0007-keep-settings-in-config-dir-and-token-in-password-store.md)
- Date: 2026-11-07

## Context

The dashboard talks to Jira Cloud's REST API on behalf of one person, the developer running it on their own machine.
There are no other users, no hosted deployment, and no sign-in screen to build.

Jira Cloud offers these ways for an external app to authenticate:

- **API token with basic auth**: the user creates a token in their Atlassian account and sends it with their email address.
  Classic (unscoped) tokens act with the user's full permissions against `https://<site>.atlassian.net`.
  Scoped tokens are limited to chosen scopes and are sent through the gateway at `https://api.atlassian.com/ex/jira/<cloudId>`.
  Tokens expire, at the latest one year after creation.
- **OAuth 2.0 (3LO)**: the app is registered in the Atlassian developer console, and each user signs in through a browser redirect.
  The app stores and refreshes access tokens.
- **Forge or Connect apps**: run inside or alongside Jira, and are meant for apps installed on a Jira site.

## Decision

- Authenticate with an API token and the user's Atlassian email address over HTTP basic auth.
- Use a classic (unscoped) API token, sent to the site URL `https://<site>.atlassian.net`.
  It is the simplest option: one token, no cloud ID, and no gateway.
  The app only reads from Jira, but the token itself is not limited to reading.
- Configuration comes from environment variables: `JIRA_SITE` (for example `example.atlassian.net`), `JIRA_EMAIL`, and `JIRA_API_TOKEN`.
  The app builds the API base URL from `JIRA_SITE`.
- At startup the app loads a `.env` file from the current directory, if present, using a small built-in parser.
  Variables already set in the environment take precedence over the file.
  `.env` is git-ignored, and a committed `.env.example` lists every variable with placeholder values.
- Only the Go server holds the token.
  It never sends the token to the browser or writes it to logs.

## Alternatives considered

- **Scoped token with read-only scopes**: limits the damage if the token leaks, but adds a cloud ID, the `api.atlassian.com` gateway, and uncertainty about whether scoped tokens work with basic auth.
  Planned as a later change; switching needs only a new token, the cloud ID, and a different base URL.
- **OAuth 2.0 (3LO)**: per-user sign-in and fine-grained scopes, but it needs a registered app, a client secret, a local callback handler, and token refresh logic.
  That is a lot of machinery for one person on one machine.
- **Forge app**: runs inside Jira on Atlassian's infrastructure, which contradicts running a small local tool with its own web interface.
- **Loading `.env` through mise**: no app code, but the built binary would see no configuration when run outside mise.
- **macOS Keychain or a config file in `~/.config`**: the Keychain encrypts the token at rest but is macOS-specific, and a config file is still plain text and adds a parser.

## Consequences

- Setup is a single token in `.env`.
  There is no app registration and no sign-in flow.
- The token is stored in plain text and can do anything the user can do in their Atlassian products, so `.env` must stay git-ignored and readable only by the user.
- The token expires, so the app should show a clear message when Jira returns `401`, telling the user to create a new token.
- Write features, such as moving cards between columns, need no new token.
- A server on `127.0.0.1` holding a token can still be reached by malicious web pages in the same browser.
  The server must check the `Host` header against an allowlist, which blocks DNS rebinding.
  It must also reject cross-origin requests that change state.
- Hosting the dashboard for others would invalidate this decision and require OAuth 2.0.
