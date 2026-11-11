# 0003. Query Jira live, without a local database

- Status: Accepted
- Date: 2026-11-07

## Context

The dashboard shows a Jira Kanban board: its columns and the issues in each.
It runs locally for one user, so the request volume is small.
Jira Cloud's REST API provides what a board view needs:

- The Agile API returns a board's configuration, including its columns and the statuses mapped to each.
- The Jira Software API returns the issues on a board, limited to the ones the board currently shows.
  It replaces the Agile API's board issues endpoint, which Atlassian has deprecated.
- `/rest/api/3/search/jql` runs JQL searches and pages through results with `nextPageToken`.
  It returns only issue IDs unless fields are requested.
  The older `/rest/api/3/search` endpoint has been removed.

Jira Cloud rate-limits API calls and returns `429` with a `Retry-After` header when a client goes over the limit.

## Decision

- Fetch the board's issues from Jira on each page load or htmx request.
  The dashboard keeps no copy of Jira data beyond a short-lived in-memory cache.
- Write a small client in `internal/` on top of `net/http`, covering only the endpoints the dashboard uses, with typed request and response structs.
- Build the board from Jira's board configuration and board issues endpoints, so the dashboard shows the same columns and issues as the board in Jira.
  Use JQL search only for views the board endpoints cannot provide.
- Request only the issue fields the view shows, follow pagination to the end, and honour `Retry-After` on `429` responses.
- Keep an open board current with htmx polling every 30 to 60 seconds, paused while the browser tab is hidden, plus a button to refresh now.
- Keep responses that change rarely, such as the board's columns and its epics' order and colours, in memory for a few minutes.
  Never cache the board's issues: they are what the board is for.

## Alternatives considered

- **Sync into a local database such as SQLite**: allows fast queries and history for metrics like cycle time or throughput.
  It costs a sync process, a schema, stale data to handle, and binary size, and a board view does not need any of that.
- **Third-party Go Jira client library**: saves writing request code, but the popular libraries lag behind Jira Cloud API changes like the new search endpoint.
  They would also add a large dependency for a handful of endpoints.
- **Manual refresh only**: fewest API calls, but a board left open goes stale.
- **Jira webhooks**: changes arrive instantly, but Jira Cloud would need to reach the local machine through a public tunnel, and registering webhooks needs admin rights.
- **Caching the board's issues too**: a repeated load would take milliseconds, but a card moved in Jira would keep its old place for minutes, which defeats a board.

## Consequences

- Cards show Jira's current state within one polling interval, with no sync to debug.
  Lanes and columns can lag by a few minutes after someone reorders epics or changes the board's settings.
- Each page load or poll costs a few API calls, which is far below Jira's rate limits for one user.
  Load time depends on Jira's response time, mostly for the board's issues.
- Fields to show must be listed explicitly, so new UI features need client changes.
- Features that need history, like charts over time, will need local storage.
  Write a new decision record when they come up.
