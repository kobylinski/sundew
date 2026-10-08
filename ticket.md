---
assignees:
- Marek Kobylinski
created_at: 2026-10-08T03:37:43.387908+00:00
id: 40a0d78a850976e8
labels: []
status: in_progress
title: 'Design the web UI: message review and install information (Svelte prototype)'
updated_at: 2026-10-08T03:37:54.313072+00:00
---
## Outcome

A Svelte prototype of Sundew's whole web UI, accepted by the operator, and a README a Programmer builds the real UI from. The UI is simple and does two things:

1. **Review caught messages** — the list (newest first), search, the message view, delete one / delete all, live arrival of new messages.
2. **Tell the reader how to install Sundew** — how to run it and how to point an application at it.

## Scope

In:
- Message list: newest on top; destination number, sender, body, provider, status and time readable without a click; one search field (to / from / body / account); live update when a message arrives; delete one, delete all; empty state (no messages yet) that leads to the install information.
- Message view: all fields of the message, its status, the raw provider request and the response Sundew returned (one click away, copyable), provider options (status callback URL and others), media URLs.
- Install information page: `docker run`, the compose snippet, the `SUNDEW_*` environment variables, how to point a Twilio client at Sundew, how a test reads a message back (`GET /api/v1/messages/latest?to=…`). Take the facts from the brief; do not invent options.
- States: loading, empty, error (API unreachable), a long body, many messages, inbound vs outbound message, failed status.
- Responsive at 390, 768 and 1366 px; keyboard and screen-reader behaviour.

Out: application code under `web/` or `internal/`; login or accounts; settings screens; sending a message from the UI; branding beyond the name.

## How to verify

In the lane:
- `cd prototypes/web-ui && npm ci && npm run build` exits 0.
- `npm run dev` serves the prototype on mock data; every screen and state above is reachable.
- `prototypes/web-ui/README.md` states what is binding and what is illustrative, the states, responsive, keyboard and screen-reader behaviour, and the exact API calls each screen makes.
- `prototypes/web-ui/VERIFICATION.md` has screenshots at 390, 768 and 1366 px.
- The operator has accepted each screen through operator questions on this task.

## Route

- This task is yours end to end, @agent:Daniel: there is no Programmer on it. Cut the lane (`task/<id8>-web-ui-design`, worktree `.worktrees/<id8>-web-ui-design`), work as your contract's "The `design` gate" steps 2–3 describe, and when the operator has accepted, rebase on `origin/main` and raise `build` to the Integrator.
- `design`: this task is the design. `review_code`: no — a prototype, not application code. `review_tests`: no. e2e acceptance: no.

## Decided and open

- Svelte, compiled to static assets, client-side only, built with Vite, no SvelteKit server: `docs/journal/2026-10-08/decision-web-ui-is-built-with-svelte.md`.
- The message object is the JSON form of `core.Message` in `internal/core/core.go`; the API calls are those of the brief, section "What it is", item 4. Mock exactly that shape. A field or call you need and do not find is a `decide` to the Integrator, not an invention.
- Design values: your contract, last paragraph of "The `design` gate".
- Node runs on this machine only. Record Svelte, Vite and anything else you add in `docs/dependencies.md` with the date of its last commit (within six months, or it is not added).
- Open, for the operator through your questions: light/dark theme; whether install information is its own page or a panel.

## Where to look

`docs/journal/2026-10-08/draft-sundew-brief.md`, `docs/architecture.md`, `internal/core/core.go`, `docs/contracts/designer.md`.