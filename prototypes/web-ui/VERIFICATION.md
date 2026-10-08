# Prototype verification

Task: `tasks:40a0d78a850976e8` · Daniel · 8 October 2026.

The q6 revision is implemented: one continuous message report with normal document scrolling, no inspector card or tabs, 14 px body text, one per-message delete location, search aligned with the list, exact phone autocomplete after `+`, and no sort label or inbox footer. The accepted Install content and automatic light/dark themes remain. Marek accepted that screen in q7 at 06:05 UTC on 8 October 2026 (`b36400e`). The c15 follow-up to make content links open in a new tab is now also implemented; HTTP-only rendering and the mixed-content fixture follow Ivan’s c17.

This is design evidence on fictional fixtures, not backend or end-to-end acceptance. All code remains under `prototypes/web-ui/`, outside `web/` and `internal/`. No new dependencies were added. Marek’s prototype-only exception in q1 still scopes the [existing dependency audit](../../docs/dependencies.md).

## Build

Environment: macOS, Node `v24.13.1`, npm `11.8.0`. Commands run from the task worktree:

```sh
npm ci --no-audit --no-fund --prefix prototypes/web-ui
npm run build --prefix prototypes/web-ui
git diff --check
```

All exited **0**. Vite 8.3.3 compiled 121 modules without warnings: CSS 25.23 kB, JavaScript 76.61 kB (28.63 kB gzip). The rebuilt static preview on port 4280 rendered eight messages and the continuous report, omitted development controls, and produced no browser error logs on the fresh load.

Development preview: <http://127.0.0.1:4279/>. Static preview: <http://127.0.0.1:4280/>. No existing service was stopped. This revision retains the previous dependency lockfile and base `70856c1`.

## Current interaction evidence

[report-revision-checks.json](report-revision-checks.json) records **33 focused browser observations at `b36400e`, all passed**, using visible controls and read-only DOM inspection. These are observed checks, not an automated test suite.

| Area | Observed result |
| --- | --- |
| Layout | Desktop search and list both measured 528.2734375 px. Their phone widths also matched. Report body text measured 14 px. No tabs, inner report scrolling, detail border, row delete controls, sort label or inbox footer remained. |
| Report | Message content, metadata, request and response were present together. Row Raw focused the request heading and scrolled the document to it; on phone its top was 24 px below the viewport edge. Back returned focus to the originating row. |
| Phone suggestions | Typing `+155598` exposed To and From suggestions. ArrowUp selected the last suggestion, ArrowDown the first; Enter applied it. Pointer selection also worked. Escape dismissed suggestions while preserving the query. |
| Exact filters | From `+15559876543` selected the inbound STOP message; To `+15551234567` selected two messages. Adding uppercase `PARCEL` narrowed the latter to one. Clear filters restored all eight. Reload retained the phone filter and selected a matching report. |
| Raw copying | Request and response clipboard text exactly matched the strings displayed in the report. |
| Delete | Exactly one per-message Delete action appeared in Message tools. Its dialog initially focused Cancel. Cancel retained eight rows. Confirming delete under a phone filter retained that filter and the next matching message; completion focused the inbox heading. Delete all also removed hidden messages. |
| Responsive | Report and raw strings had no horizontal overflow at 390 px, including the long-body fixture. Tablet and phone used document scrolling with no nested inspector scroll. Screenshots cover 390, 768 and 1366 px. |
| Edge messages | Inbound displayed both missing-exchange explanations in the same report. Failed delivery retained its text status and error code 30003. |
| Recovery | Retry restored eight rows and returned focus to the inbox heading. Empty, loading and disconnected states were recaptured. |
| Paging | The many-message fixture started at 50 rows; Load 50 more exposed 100. The list used document scrolling. |
| Arrivals | An arrival outside an exact phone filter preserved its selected report. A scrolled arrival waited behind Show newest without inserting rows. Show newest returned to the top and focused the inbox heading; the current 100-row page size was retained. |
| Static build and themes | The rebuilt static preview rendered the report without development controls. Default theme matched the OS. Both explicit palettes were visually inspected. |

The [initial 30 observations](interaction-checks.json) at `a42a337` and [13 earlier refinement observations](refinement-checks.json) at `f705a4e` are historical evidence. Their references to B/C layouts and detail tabs do not describe this candidate. The initial fixture validation covered 280 fixtures and exact `core.Message` fields; the new `links` scenario preserves the same message shape and is exercised in the follow-up checks below.

## Clickable-link follow-up

[link-checks.json](link-checks.json) records **16 focused checks, all passed**, of c15/c17: only the HTTP URL became a link in the mixed body, in both list and report; HTML stayed escaped; invalid callback/media schemes stayed inert. All active links had `_blank`, `noopener` and `noreferrer`. Clicking a body URL opened a separate browser tab and left the message selection unchanged; the temporary tab was closed afterwards. Row selection, Raw navigation, search highlighting within a URL, keyboard link access, return focus and original-text copying remained usable. Linked content wrapped at 390 px without page overflow. The static build also exercised the new fixture.

The row-selection button is separate from its body anchors, avoiding nested interactive elements. No additional dependencies or production code were added. The explicit link request is implemented; q7 is the recorded report-screen acceptance, not a claim that a later screenshot was separately accepted.

## Screenshots

These are actual browser captures of the q6 revision with the requested content links. Full reports are deliberately taller than the viewport because the document scrolls. The controls in the arrival example are development-only.

| Screen | 390 px | 768 px | 1366 px |
| --- | --- | --- | --- |
| Inbox | [Phone](screenshots/a-inbox-390.png) | [Tablet](screenshots/a-inbox-768.png) | [Desktop](screenshots/a-inbox-1366.png) |
| Complete report | [Phone](screenshots/a-detail-390.png) | [Tablet](screenshots/a-detail-768.png) | [Desktop](screenshots/a-report-1366.png) |
| Installation | [Phone](screenshots/install-390.png) | [Tablet](screenshots/install-768.png) | [Desktop](screenshots/install-1366.png) |
| Phone autocomplete | [Phone](screenshots/phone-autocomplete-390.png) | — | [Desktop](screenshots/phone-autocomplete-1366.png) |

Additional captures:

- Mixed content: [phone](screenshots/links-390.png), [desktop](screenshots/links-1366.png).
- Raw request jump: [phone](screenshots/a-request-390.png), [desktop](screenshots/a-request-1366.png); [response section](screenshots/a-response-1366.png).
- [Delete-all confirmation under a phone filter](screenshots/delete-all-1366.png).
- [Empty](screenshots/empty-390.png), [loading](screenshots/loading-390.png), [disconnected](screenshots/offline-390.png), [long report](screenshots/long-390.png).
- [Inbound without an exchange](screenshots/inbound-1366.png), [failed delivery](screenshots/failed-1366.png), [staged arrival](screenshots/arrival-pending-1366.png).
- Dark theme: [phone report](screenshots/a-dark-390.png), [desktop](screenshots/a-dark-1366.png).
- [Compose and expanded curl example on phone](screenshots/install-compose-390.png).

## Limits and handoff

No real provider, API, SSE connection or Docker command was exercised. Retry, paging, arrival and deletion are local UI demonstrations. The README maps the implementation to actual `q`, `to`, `from`, cursor and SSE contracts, null collections and 204 delete responses. Exact phone matching combined with `q` was checked against `internal/api/api.go` in this lane. Suggestions derive from loaded matching results; the API provides no complete phone-directory endpoint.

Semantic markup, focus and keyboard behavior were inspected in Chrome. No real screen reader, touch device, cross-browser run or formal accessibility certification was performed. The OS setting was not changed; initial automatic theme selection was exercised, while the existing live OS-change listener remains code-reviewed evidence.

The lane remains based on `origin/main` at `70856c1`. The earlier dependency-document overlap preserves main’s Go/Twilio records and the scoped Svelte audit. There is no Rosemary gate, following Marek’s instruction. q7 accepts the revised message screen; q4 accepts Install; q5 requests both automatic themes. c15’s link behavior is implemented and verified. No operator question is open. The README and evidence are ready for the Integrator’s handoff to implementation.
