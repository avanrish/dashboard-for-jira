# 0001. Serve server-rendered HTML from Go with templ, htmx, Alpine.js, and Tailwind

- Status: Accepted
- Date: 2026-11-07

## Context

The dashboard runs on the developer's own machine and shows a Jira Kanban board in the browser.
It is not hosted or shared.
It should be small, simple, and fast, should not need a separate frontend project or a Node.js toolchain, and should keep the binary small.

## Decision

- A single Go binary serves the web interface with `net/http`, listening on `127.0.0.1` only.
- Pages and fragments are rendered on the server with [templ](https://templ.guide/), which compiles templates into type-checked Go functions.
- [htmx](https://htmx.org/) handles partial page updates: links and forms request HTML fragments from the server and swap them into the page.
- [Alpine.js](https://alpinejs.dev/) is added only when some interaction is purely client-side, such as toggling a panel or filtering visible cards, and is not worth a server round trip.
- Styling uses [Tailwind CSS](https://tailwindcss.com/) utility classes in templ files, compiled by Tailwind's standalone CLI.
  Its base styles (Preflight) remove most differences between browsers' default styles.
- htmx and Alpine.js are vendored into the repository at pinned versions.
  All static assets are embedded into the binary with `embed`; there is no CDN.
- The templ and Tailwind CLIs are pinned in `mise.toml`.
- Generated files are committed: templ's `*_templ.go` files and the compiled Tailwind stylesheet.
  A plain `go build` then works without either CLI, and CI fails if regenerating them changes anything.
- During development, `templ generate --watch --proxy` regenerates templates, restarts the server, and reloads the browser.
  The Tailwind CLI runs in watch mode alongside it, and a mise task starts both.

## Alternatives considered

- **Standard library `html/template`**: no code generation and no extra dependency, but templates are checked only at runtime.
  It also makes the binary much larger: `text/template` looks up methods by name through reflection, which stops the linker from removing unused methods.
  In a test with Go 1.27.3, a minimal `net/http` server was 8.3 MB without templates, 8.9 MB with templ v0.3.1100, and 12.8 MB with `html/template`.
- **React, Vue, or Svelte single-page app**: richer client-side state, but needs a separate project, a Node.js build, a JSON API, and two languages for one small tool.
- **Plain JavaScript with `fetch`**: no dependency, but would re-implement what htmx already does.
- **Hand-written CSS or a classless framework such as Pico CSS**: no build step, but slower to style, and a classless framework still needs custom CSS for the board itself.
- **Loading htmx and Alpine.js from a CDN**: nothing to vendor, but every page load depends on a third party and the app stops working offline.
- **Git-ignoring generated files**: cleaner diffs, but every build, including `go install`, would need both CLIs, and a forgotten step ships broken pages.

## Consequences

- `go build` produces one self-contained, small binary.
- Template mistakes, such as a wrong field or type, fail at compile time.
- templ drops the indentation and line breaks written in templates, so responses carry little extra whitespace.
- Two generators, templ and Tailwind, must run after editing templates.
  The watch task handles this during development, and CI catches stale output.
- Generated code appears in diffs.
  Review the source files, not the output.
- templ is pre-1.0, so upgrades may need small template changes.
- The size benefit holds only while no reachable code calls `reflect.Value.Method` or `MethodByName`, which running `text/template` or `html/template` templates does.
  Importing those packages is not enough: templ imports `html/template` without calling those paths.
  To check, run `go build -ldflags=-dumpdep -o /dev/null ./cmd/dashboard-for-jira 2>&1 | grep -F '<ReflectMethod>'`; any output means the linker has stopped removing unused methods.
- Most behaviour lives in Go handlers and is testable with `net/http/httptest`.
- Handlers return HTML, not JSON.
  Adding another client, such as a CLI, would need a separate API.
- Updating htmx or Alpine.js means replacing the vendored files by hand.
