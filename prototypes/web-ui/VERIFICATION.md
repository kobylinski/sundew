# Prototype verification

Task: `tasks:40a0d78a850976e8` · Daniel · 8 October 2026.

The selected split-inbox prototype builds and runs. The operator chose A, accepted the inspector and Install page content, and requested automatic light/dark themes in q2–q5. The requested visual refinement is implemented; its final appearance remains to be reviewed. B/C are removed. This is design evidence on fictional fixtures, not backend or end-to-end acceptance.

Marek approved the prototype-only dependency exception in `tasks:40a0d78a850976e8#q1` at 04:27 UTC. The complete lockfile audit is in [docs/dependencies.md](../../docs/dependencies.md); production dependencies are outside that approval.

## Build

Environment: macOS, Node `v24.13.1`, npm `11.8.0`. From the task worktree on 8 October 2026:

```sh
npm ci --no-audit --no-fund --prefix prototypes/web-ui
npm run build --prefix prototypes/web-ui
git diff --check
```

All exited **0**. The refined design builds with Vite 8.3.3 (119 modules), without compiler warnings. The static build was also opened through `npm run preview` on port 4280: eight fixture rows rendered and the development-only prototype controls were absent. Fresh preview loads produced no errors. The running development server briefly reported hot-reload failures while the rebase replaced component files; a full reload after the rebase rendered all eight rows with no new errors.

The development preview is <http://127.0.0.1:4279/>. Port 4179 was already occupied, so the prototype uses its own port. No existing service was stopped.

## Interaction checks

The [initial browser observations](interaction-checks.json) contain 30 checks of the comparison at `a42a337` (originally `f0ff096`). The [refinement observations](refinement-checks.json) add 13 focused checks after layout consolidation and visual changes: raw copy, tab navigation, system/default themes, responsive edge states, search, deletion focus, staged arrivals and hidden-by-default review controls. All 13 passed. The checks exercise the rendered UI through its controls; they are not a committed automated test suite.

| Area | Observed result |
| --- | --- |
| Search | Recipient/sender query `15559876543` returned two rows; uppercase `SIGN-IN` matched one body; the second account SID matched one row; a missing phrase showed a distinct no-results state. |
| Raw exchange | Row Raw opened the captured request; request and response clipboard contents exactly matched their displayed strings. The captured queued response stayed separate from the current delivered status. |
| Copy feedback | Changing Docker to Compose reset the previous Copied state; each format copied its own complete source string. |
| Detail navigation | Opening detail focused its heading; closing returned to the corresponding message row. B opened standalone detail; C expanded its request inline. |
| Keyboard | `/` focused search; Escape cleared a search while retaining focus. Arrow-key tab navigation selected and focused Response. Layout-switcher arrows did not intercept an editable search. |
| Delete | The native confirmation initially focused Cancel. Escape cancelled without removing rows. Delete-one removed the targeted filtered row. Delete-all cleared filtered-out rows too. Completion focused the inbox heading. |
| Paging | The many-message fixture initially rendered 50 rows; Load 50 more rendered 100 while retaining the first page. The badge indicated more available instead of claiming a server total. |
| Arrival | An arrival at the top added a row and preserved the selected inspector. While scrolled down, it stayed behind Show newest; choosing that control returned to the top with focus on the inbox heading. |
| Recovery | The disconnected state disabled search/destructive actions and exposed Try again and installation help. Retry restored the mock inbox and focused its heading. |
| Loading | Skeletons were absent from the accessibility snapshot; the loading region contained one loading status. Search and Delete all were disabled. |
| Message edge cases | Long text and an extended reference wrapped on the phone. Inbound records with no HTTP exchange showed an explicit empty state. Failed messages showed a text status and error code 30003. |
| Responsive | List, detail and installation layouts were inspected at the three required widths. Phone inbox, raw request, long body and install content had no horizontal page overflow. |

Independent fixture validation also passed: a Node assertion pass over 280 fixtures across eight states verified the exact `core.Message` JSON keys, newest-first ordering, timestamps, status/direction values and header arrays. `node --check` passed for `src/data.js` and `src/main.js`.

Issues found and corrected during verification: an empty CSS import prevented the first build; initial-value/accessibility compiler warnings; copy feedback carried into a changed snippet; missing spacing before installation defaults; focus fell to the document after Show newest and Retry. The successful build and final interaction checks above include the corrections.

## Responsive screenshots

These are actual browser captures of the refined design, not generated mockups. Review controls are hidden by default; the arrival example explicitly enables them to show how it was exercised. Full-page detail/install images are taller than the viewport listed in the column.

| Screen | 390 px | 768 px | 1366 px |
| --- | --- | --- | --- |
| A: split inbox | [Phone](screenshots/a-inbox-390.png) | [Tablet](screenshots/a-inbox-768.png) | [Desktop](screenshots/a-inbox-1366.png) |
| A: message detail | [Phone](screenshots/a-detail-390.png) | [Tablet](screenshots/a-detail-768.png) | [Desktop inspector](screenshots/a-inbox-1366.png) |
| Installation | [Phone](screenshots/install-390.png) | [Tablet](screenshots/install-768.png) | [Desktop](screenshots/install-1366.png) |

Additional evidence:

- Raw request: [phone](screenshots/a-request-390.png), [desktop](screenshots/a-request-1366.png); [captured response](screenshots/a-response-1366.png).
- [Delete-all confirmation while filtered](screenshots/delete-all-1366.png).
- [Empty](screenshots/empty-390.png), [loading](screenshots/loading-390.png), [disconnected](screenshots/offline-390.png), [long body](screenshots/long-390.png).
- [Inbound without an exchange](screenshots/inbound-1366.png), [failed delivery](screenshots/failed-1366.png), [staged arrival](screenshots/arrival-pending-1366.png).
- Dark theme: [phone](screenshots/a-dark-390.png), [desktop](screenshots/a-dark-1366.png).
- [Compose and expanded curl example on phone](screenshots/install-compose-390.png).

## Limits and handoff

No real provider, API, SSE connection or Docker command was exercised by this prototype. Retry, paging, arrival and deletion are local UI demonstrations. The README maps the implementation to the real API, including `q`, cursors, named SSE events, reconnection, null collections and 204 delete responses.

Semantic markup, focus and keyboard behavior were inspected in Chrome. No real screen reader, touch device, cross-browser run or formal accessibility certification was performed. Automatic theme selection matched the OS preference; light/dark overrides and returning to System were exercised. The OS setting itself was not changed during this session, so the live OS-change listener was code-reviewed rather than exercised. Browser-extension warnings appeared in the shared Chrome environment. The rebase-related hot-reload errors described above did not recur on a fresh page load.

All prototype code remains disposable and outside `web/` and `internal/`. q2–q5 are answered: layout A, inspector content/interactions, dedicated Install page, and both themes with automatic switching. Review of the visual refinement requested in q2 is the remaining task requirement. No Rosemary gate has been raised or claimed, following the operator's instruction.

The lane was rebased on `origin/main` at `70856c1`. The dependency-document overlap was resolved by retaining main's Go/Twilio records and appending the scoped Svelte audit. The refined build passed after the rebase.
