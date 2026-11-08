# 0005. Fetch board issues from the Jira Software API

- Status: Accepted
- Date: 2026-11-08

## Context

[ADR 0003](0003-query-jira-live-without-local-storage.md) builds the board from the Agile API's board configuration and board issues endpoints.
The Agile API's board issues endpoint, `GET /rest/agile/1.0/board/{boardId}/issue`, pages by offset with `startAt`.
In [RFC-127](https://community.developer.atlassian.com/t/rfc-127-jira-software-cloud-api-changes-improving-scalability-for-software-space-issue-results/98871), Atlassian replaces it and the other Agile endpoints that list issues with versions paged by `nextPageToken`, and deprecates the old ones, to be removed after the deprecation period.
The replacement, `GET /rest/software/1.0/board/{boardId}/issue`, takes the same `jql` and `fields` parameters and returns the same issues.
The RFC leaves the board configuration endpoint alone.

## Decision

Fetch a board's issues from `GET /rest/software/1.0/board/{boardId}/issue`, following `nextPageToken` to the last page.
Keep reading the board's columns from the Agile API's `GET /rest/agile/1.0/board/{boardId}/configuration`.

This amends ADR 0003 for the board issues endpoint only; the rest of 0003 stands.

## Alternatives considered

- **Keep the Agile API's board issues endpoint**: it works today, but it is deprecated, and the dashboard would stop working when Atlassian removes it.
- **JQL search with the board's filter**: `/rest/api/3/search/jql` already pages by token, but the dashboard would have to rebuild the board's filter itself, and could drift from the issues the board in Jira shows, which 0003 avoids.

## Consequences

- Board issues and JQL search results page the same way, so the client follows pages with one loop for both.
- A board's data comes from two APIs: columns from the Agile API, issues from the Jira Software API.
  If Atlassian moves the board configuration to the Jira Software API as well, move it there too.
