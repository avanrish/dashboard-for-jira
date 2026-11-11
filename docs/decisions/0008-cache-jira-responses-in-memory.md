# 0008. Cache Jira responses in memory

- Status: Accepted
- Date: 2026-11-11

## Context

[ADR 0003](0003-query-jira-live-without-local-storage.md) fetches board data from Jira on every page load, and says to add a short-lived in-memory cache in front of the client if pages become slow.
A page load makes up to five Jira calls in four rounds, each waiting for the one before.
On a board with 294 issues, a load took about 2 seconds: about 1 second for the board's issues, 0.6 to 1 second for the two searches that follow (subtasks' parents, then the epics' order and colours), and 0.25 seconds for the board configuration and the site's fields, fetched in parallel.
The issues change often, and they are what the board is for; the rest changes rarely.

## Decision

- The client in `internal/jira/` keeps successful responses in memory, keyed by request URL.
- Each request takes `RequestOptions`, whose `CacheTTL` sets how old a reused response may be and how long the response it gets is kept, much like HTTP's `max-age`.
  There is no default: a request without a TTL always asks Jira, so a caller gets older data only when it says how old.
- The board fetches its issues without a TTL, so on every load, and reuses its other responses for five minutes.
  The searches' URLs list the IDs they look up, so a new epic or parent on the board still goes to Jira.
- The client caches a paged list of issues as one entry, so a result never mixes pages fetched at different times.
- Errors are not cached.
  Expired entries are dropped whenever a response is stored, which bounds memory use.

This amends ADR 0003, which fetched everything from Jira on every page load; the rest of 0003 stands.

## Alternatives considered

- **Cache the board's issues too**: a repeated load would take milliseconds, but a card moved in Jira would keep its old place for up to five minutes, which defeats a board.
- **A default TTL of five minutes**: shorter calls, but a request that says nothing about caching gets data up to five minutes old; a credentials check, for one, would keep passing for five minutes after the token is revoked.
- **Cache only the board configuration and the fields**: saves only 0.25 seconds, because they are fetched in parallel; the searches after the issues cost more.
- **Cache the rendered board in the web layer**: simpler, but it allows one TTL for the whole board and none for individual requests.
- **Cache each page of a paged list separately**: pages would expire at slightly different times, so one load could combine an old page with a new one and show an issue twice or not at all after a rank change.
- **A cache library**: a dependency for what a map and a mutex do.
- **A persistent cache on disk**: survives restarts, but only the first load after a restart is slower, and 0003 asks for its own record for persistent storage.

## Consequences

- A page load makes one Jira call while the rest is cached, which took the board above from about 2 seconds to 0.8 to 1 second.
  The issues request is most of what remains; making it faster means requesting less, not caching.
- Cards are always current, but lanes can lag by up to five minutes after someone reorders epics, changes an epic's colour, or moves an issue with subtasks to another epic; columns lag as long after the board's settings change.
- Restarting the server clears the cache.
- Requests that miss the cache at the same moment each call Jira, which is harmless at one user's request volume.
