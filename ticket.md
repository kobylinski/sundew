---
assignees: []
created_at: 2026-10-08T08:06:54.928681+00:00
id: 4f6ecdb55a9238e9
labels: []
status: open
title: 'Production web UI: the accepted design, built in Svelte and embedded in the binary'
updated_at: 2026-10-08T08:06:54.928681+00:00
---
## Outcome

`docker run -p 8025:8025 <image>` and a browser at `http://localhost:8025/` show Sundew's real UI: the inbox of caught messages updating live, search with the phone picker, the continuous message report with the raw exchange, delete one and delete all, and the Install page — exactly the design the operator accepted, on real data from the query API. One binary, no Node at runtime.

## Scope

In — you own `web/` and `internal/ui`:
- `web/`: the Svelte + Vite application. Start from `prototypes/web-ui/src` (components, styles, both automatic themes) and replace the fixtures with the real API. Remove every prototype-only aid (state/theme/variant URL controls, arrival simulator, fixtures) from the production build.
- Everything listed as binding in `prototypes/web-ui/README.md` ("Binding versus illustrative", the screens and states, responsive, keyboard and screen-reader sections, and its API mapping). That document is the specification; where it and this task differ, raise `explain`.
- API use, and only these calls: `GET /api/v1/messages` (`q`, `to`, `from`, `limit`, `cursor`) with "load more" paging; `GET /api/v1/messages/{id}`; `DELETE /api/v1/messages/{id}`; `DELETE /api/v1/messages`; `GET /api/v1/messages/stream` (SSE) for live created / updated / deleted / reset, reconnecting with backoff and re-fetching the list after a reconnect; an "API unreachable" state when calls fail. Phone suggestions come from the To/From values of messages already loaded — no new endpoint.
- Links in bodies, media URLs and the status-callback URL: http/https only, `target="_blank"`, `rel="noopener noreferrer"`; a body is never rendered as HTML (no `{@html}`); the raw exchange stays literal.
- Client-side routes (inbox, a message, Install) that survive a reload: `internal/ui` serves the embedded build at `/`, with unknown non-API paths falling back to `index.html`; it never shadows `/api/`, `/healthz` or a provider route. Hashed assets are served with long cache headers, `index.html` with `no-cache`. A `Content-Security-Policy` that allows only same-origin scripts, styles and connections.
- Embedding: `internal/ui` embeds the build output with `go:embed` and registers itself with one line in `cmd/sundew/main.go`. `go build ./...`, `go vet ./...` and `go test ./...` must work on a fresh clone with no Node installed: commit a small placeholder page for that case (it says the UI is not built), and have the image build replace it.
- `Dockerfile`: a Node build stage (`npm ci`, `npm run build`) feeding the Go build stage; the final image stays `FROM scratch` with one binary. `ci.yml`: the web build and its tests.
- Tests: unit tests for the link parser, the search/filter-to-query mapping and the stream reconciliation (created, updated, deleted, reset, reconnect); Go tests for `internal/ui` (index, asset, fallback, API paths not shadowed, headers).
- `scripts/acceptance/50-ui.sh`: against the image — `/` returns the real UI (not the placeholder), every asset it references returns 200 with the right content type, a deep link falls back to the app, `/api/v1/messages` and `/healthz` still answer as before.
- `docs/dependencies.md`: the production set, audited as the prototype's was. README: a "Web UI" section and one screenshot.

Out: any change to the query API, `internal/core`, the store or a provider (a gap is a `decide`); login, settings, sending a message from the UI; a manual theme switch; server-side rendering or a Node server.

## How to verify

On this machine, in the lane:
- `go vet ./...` and `go test -race ./...` exit 0, both before and after `npm run build`.
- `npm ci`, `npm test` and `npm run build` in `web/` exit 0 with no compiler warnings.
- `scripts/acceptance.sh` exits 0 with all eight steps.
- Run the binary locally, send three messages through the Twilio façade (curl is enough), and record in the task what you saw in a browser at 390, 768 and 1366 px: live arrival without reload, search, the phone picker, the report, the raw exchange, delete one, delete all, the Install page, light and dark. Screenshots go in `web/docs/` or the README.
- Image size stated in the Summary (it was 10.1 MB before the UI).

## Route

`design`: no — the design is accepted; build it as specified. A screen question the handoff does not answer is an `explain` to the Integrator, who asks the Designer. `review_code`: yes. `review_tests`: yes. e2e acceptance: yes — `50-ui.sh`; the Reviewer extends it with a browser-driven step.

## Decided and open

- Svelte, client-side only, compiled and embedded: `docs/journal/2026-10-08/decision-web-ui-is-built-with-svelte.md`. The design: operator answers q2–q7 on tasks:40a0d78a850976e8.
- **Dependencies — open, and it gates the merge, not the work.** The operator's exception for the twelve transitive packages that fail the six-month rule (tasks:40a0d78a850976e8#q1) was given for the prototype only. Build with the same pinned set, add nothing that is not needed, and audit it. I will not pass `build` until the operator has said in so many words that the exception covers the production UI; I am asking him.
- Any new direct dependency (a test runner, for instance) must itself pass the rule; prefer what Vite and Svelte already bring.
- The release-workflow lane also edits `Dockerfile` and `ci.yml` and merges first; rebase onto it.
- Go and Node run on this machine; Docker on the remote engine by `DOCKER_HOST` with a private client config (tasks:b099379fbbff6b1d#c22). Node is never installed on the remote host: the image's Node stage is the only Node there.

## Where to look

`prototypes/web-ui/README.md`, `prototypes/web-ui/VERIFICATION.md`, `prototypes/web-ui/src`, `internal/api/api.go` and `internal/api/openapi.go`, `internal/core/core.go`, `internal/server/server.go`, `cmd/sundew/main.go`, `docs/architecture.md`.