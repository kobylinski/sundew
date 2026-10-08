# Sundew web UI — design prototype

Task: `tasks:40a0d78a850976e8`. Designer: Daniel. This is a disposable Svelte application on fictional, in-memory fixtures; it makes **no API calls** and sends no SMS. No production application code belongs here.

**Review status:** not yet accepted. The operator must accept the concrete screens and theme before this becomes the implementation handoff. Marek approved the prototype-only dependency exception in `tasks:40a0d78a850976e8#q1`; see [the complete dependency audit](../../docs/dependencies.md).

## Run and compare

```sh
cd prototypes/web-ui
npm ci
npm run dev
```

Local URL: <http://127.0.0.1:4279/>. Node is used only to build and serve this prototype. `npm run build` produces static assets in `dist/`; `npm run preview` serves that build on port 4280.

| Candidate | URL | Structural choice |
| --- | --- | --- |
| A — split inbox (recommended) | <http://127.0.0.1:4279/?variant=A> | List and inspector together on desktop; a focused detail screen on smaller devices. |
| B — compact list | <http://127.0.0.1:4279/?variant=B> | Full-width rows for scanning many messages; details open as a separate content view. |
| C — message stream | <http://127.0.0.1:4279/?variant=C> | A narrow reading stream, larger bodies, and an inspector expanded below the selected message. |

Use the dark floating **Prototype · mock data** bar to change layout, state, or theme and to simulate an incoming arrival. Left/right arrows also change variants outside editable controls and tab lists. These controls exist only in the development server; they are excluded from the production build. Append `controls=0` for clean screenshots.

All variants use the same fixture shape and local state. `variant`, `state`, `theme`, `view`, `q`, `message`, and `tab` are prototype-only URL parameters; they are not API parameters or the final application's route contract. Reloading resets local deletions and arrivals. No local storage, cookies, accounts, or login is used.

## Binding versus illustrative

Nothing here is operator-accepted yet. The following requirements already come from the task and product decisions and remain binding regardless of which candidate is chosen:

- Newest first; recipient, sender, complete body, provider, delivery status and time readable in the list. Long text wraps; it is not silently truncated.
- One search field matching a substring in recipient, sender, body or account, case-insensitively. Search and live updates do not move keyboard focus.
- Every message field is inspectable. **Raw** on a row opens the captured request in one action; **Response** shows exactly what Sundew originally returned, separate from current delivery status. Copy actions preserve the raw bytes shown.
- Delete-one and delete-all require a clearly scoped confirmation, with Cancel initially focused. Delete-all covers the entire store, including unloaded and filtered-out messages. There is no invented Undo API.
- No message composer, settings screen, accounts screen, or login. Installation help is always one navigation link away, including in empty/error states.
- Svelte compiles to static assets. The real UI talks to the query API only and is embedded in the Go binary. No Node server is required at runtime.
- Messages are held in memory only; there is no database/volume configuration. Any provider credential is accepted. The install page lists only `SUNDEW_ADDR`, `SUNDEW_BASE_URL`, `SUNDEW_CALLBACK_DELAY`, `SUNDEW_CALLBACK_OUTCOME`, and `SUNDEW_INBOUND_URL`.

Pending visual decisions: the selected layout, light/dark theme, and whether installation guidance remains a dedicated page. Candidate A is my recommendation: it keeps the message body and debugging context visible together without turning the inbox into a dashboard.

Illustrative: all phone numbers, accounts, message IDs, content, provider responses, error code examples, media URLs and timings. The fixture generator is not a provider emulator or SMS segment calculator. Typography, spacing and colors are review candidates. The arrival control is a review aid, never a proposed product feature. Installation snippets use `<version>` as a release placeholder, not a claim that a release is already published.

## Screens and states

| State / screen | Reach it | Expected behavior |
| --- | --- | --- |
| Normal inbox | `?variant=A` | Eight fictional messages, newest first. Desktop A shows the first message in the inspector; phone/tablet starts with the list. |
| Message detail | Click the row, or `?message=msg_0001` | Full text; metadata; provider options; media URLs; delete action. |
| Raw request | Click **Raw**, or `?message=msg_0001&tab=request` | Method, path/query, all headers, and original body; one Copy request action. |
| Raw response | Choose Response, or `?message=msg_0001&tab=response` | Captured HTTP status, headers and body; queued creation response can coexist with a later delivered status. |
| Loading | `?state=loading` | Non-interactive skeletons; one screen-reader loading announcement; destructive actions disabled. |
| Empty store | `?state=empty` | Ready-for-first-message copy and a link to the install page; still listening. |
| No search results | Search for `not-a-match` | Clear-search recovery; the search is not confused with an empty store. |
| API unreachable | `?state=offline` | Disconnected label, explanatory error, Try again, installation guide. Retry returns to mock normal state. |
| Long text | `?state=long` | Paragraphs and an unbroken reference wrap inside the row and detail. |
| Many messages | `?state=many` | 240 fixtures; 50 initially visible, then Load 50 more. Count is `50+`, never a fabricated total from the API. |
| Inbound | `?state=inbound` | Received status and inbound direction. Missing captured exchange has an explicit empty state. |
| Failed | `?state=failed` | Text status plus an error-code note in detail; failure does not rely on color alone. |
| Live arrival | **+ Arrival** in prototype controls | Adds a queued message at the top; preserves selection/focus. When scrolled away, accumulates a Show newest banner until requested. |
| Delete one / all | Row/detail delete or Delete all | Native dialog, explicit scope, safe initial focus, Escape/Cancel, success announcement. |
| Installation | Install navigation, or `?view=install` | Docker/Compose, Twilio adapter base URLs, curl example, query API example, five supported environment variables. |
| Dark theme | Theme button, or `?theme=dark` | All surfaces and status text switch together; illustrative until accepted. |

All states can be combined with A/B/C. Raw provider response bodies are preserved as strings and are never regenerated from current status. The inbound fixture deliberately contains an empty `Exchange`; the implementation must render the actual captured data it receives, without assuming every inbound message has a provider façade request.

## Responsive behavior

- **1366 px:** A has independently scrollable list and inspector with the destination/body visible on both. B has a full-width, column-aligned list and standalone detail. C is a narrow stream with inline expansion. Installation uses an introduction alongside the steps.
- **768 px:** A becomes list → detail → back, rather than squeezing two unreadable columns. B retains compact rows with metadata rearranged. Installation becomes one column. C keeps inline expansion.
- **390 px:** Full-width message rows; metadata wraps; every body and raw string wraps without horizontal page scrolling. A/B details occupy the content region. Buttons and native form controls remain usable by touch. The prototype-only bar occupies two compact rows, with bottom padding so it does not hide content.
- The A breakpoint is 1000 px. General phone styles apply at 600 px. Layout reflow is CSS-driven, not inferred from user-agent strings.
- Installation code is wrapped for reading; Copy still returns the original unwrapped command. The app has no remote fonts, images, icon package, or decorative illustration.

## Keyboard and screen readers

- Skip-to-content link, named navigation, main landmark, one page heading, semantic sections/articles, visible focus outlines, descriptive button labels.
- `/` focuses search outside editable controls. Escape in a nonempty search clears it. Escape from an open detail returns to the selected row. Standard browser Back moves between Messages and Install.
- Opening detail places focus on its heading. Closing returns focus to its message row. Deleting, retrying a failed load, and explicitly choosing Show newest return focus to the inbox heading. Automatic arrivals preserve focus.
- Details use a tablist with one tab stop; Left/Right/Home/End select and focus tabs. Tab moves into the panel and its copy/actions. The prototype's layout arrows never intercept this tablist.
- Native `<dialog>` traps focus for delete confirmation. Cancel receives initial focus; Escape dismisses; confirmation announces the result. A changed search is never silently used as the scope of Delete all.
- Status/direction have text labels as well as color/icons. Icons are hidden from the accessibility tree; Copy announces completion/failure through a polite live region. New arrivals are politely announced without moving focus.
- Loading skeletons are aria-hidden; their containing state is busy and has one loading announcement. Search results have a separate polite count announcement.
- Reduced-motion preference disables cosmetic animation and transitions. The prototype has not been certified with a real screen reader; automated DOM inspection is not a substitute for that test.

## Exact API handoff

These are **implementation calls**, not network requests made by the prototype. Search, paging and SSE were confirmed by Ivan on `tasks:40a0d78a850976e8#c2` and introduced on `main` in `f4c098e`. The canonical message JSON comes from `internal/core/core.go`. This mapping was also checked against Patryk's API implementation at `2473de7` (`internal/api/api.go`).

| UI action | HTTP contract |
| --- | --- |
| Open inbox | `GET /api/v1/messages?limit=50` |
| Search | `GET /api/v1/messages?limit=50&q=<URL-encoded text>`; omit `q` when empty. |
| Load another page | Same `q` and `limit`, plus `cursor=<URL-encoded next_cursor>`. |
| Open/revalidate detail | `GET /api/v1/messages/{id}` with an encoded ID. |
| Listen for changes | `GET /api/v1/messages/stream` via EventSource; subscribe without `q` so updates that remove a search match are still observed. |
| Confirm delete one | `DELETE /api/v1/messages/{id}` |
| Confirm delete all | `DELETE /api/v1/messages` with no search filters. |
| Install page | No calls. Docker/Twilio/curl snippets are copyable text, never executed in the UI. |
| Test example shown on Install | `GET /api/v1/messages/latest?to=%2B15551234567` |
| Other documented test actions | `POST /api/v1/reset`, `GET /api/v1/messages/stream`, `GET /api/v1/openapi.json`; shown as documentation only. |

List response: `{ "items": [<core.Message>], "next_cursor": "..." }`. An empty `next_cursor` is the end. The allowed limit is 1–500 and the default is 50. There is no total field: the visible badge counts loaded rows and adds `+` when another page exists. On a new search, discard the previous cursor and start a new first page. Debounce user typing (~200 ms), cancel stale requests, and apply only the latest response. The prototype filters synchronously to keep the comparison fast.

Open the stream before loading the initial snapshot; buffer events while that snapshot loads, then reconcile them by ID. Refetch after each successful connection/reconnection, so an arrival between a list response and the stream opening is not lost.

SSE event names are `message.created`, `message.updated`, `message.deleted`, and `store.reset`. The data is `core.Event`: `{ "type": "message.created", "message": <core.Message> }`; reset has no message. Comment heartbeats arrive every 15 seconds. Treat a reconnect as a reason to re-fetch the list and selected message: the in-memory bus has no replay guarantee and a slow subscriber can lose events. Reconcile by Sundew `id`, not provider ID; sort by `created_at`. Current delivery status comes from the current message, not `exchange.response_body`.

Do not steal focus, change selection, or jump the reader to the top on arrival. Insert immediately when at the top; otherwise stage new rows behind the new-message banner. Apply status changes to existing rows/detail in place. Remove a deleted row; reset clears the list and selection. If an update no longer matches the current `q`, remove it from those results. A lost stream should show a reconnecting/disconnected state, not an unconditional Live badge.

Error envelope: `{ "error": { "code": "...", "message": "..." } }`. Successful delete-one, delete-all and reset return `204 No Content`; do not try to parse a JSON body. Destructive controls are disabled while the request is pending. Remove rows only after successful deletion; on failure keep the data, report the failure, and allow retry. A detail 404 means the message was deleted elsewhere: explain that and return to the list. On failure after a successful load, retain existing rows with a disconnected notice; the initial-load error mock demonstrates recovery controls, not a requirement to discard useful cached data.

Fields consumed (no additions to the backend model): `id`, `provider`, `provider_id`, `account`, `direction`, `from`, `to`, `body`, `segments`, `media_urls`, `status`, optional `error_code`, `options`, `exchange`, `created_at`, `updated_at`. Exchange fields: `method`, `path`, `query`, `header`, `body`, `response_status`, `response_header`, `response_body`. Headers are maps of string arrays. Empty/null media, option and header collections should be normalized when reading JSON. Inbound records can have an empty provider ID, zero segments, null options and an empty exchange; show what is recorded without inventing a provider ID or a segment estimate. Times should use the operator's local time zone; fixture screenshots use Europe/Warsaw explicitly for repeatability.

## Installation facts and boundaries

The install content follows the brief and the accepted Svelte, in-memory-only and accept-any-credential decisions. `<version>` is an explicit release placeholder. `TWILIO_API_BASE` is an example application's adapter setting; the page does not claim that Twilio's SDK reads that environment variable itself. The shown defaults (address `:8025`, outcome `delivered`) were verified against `internal/config/config.go` on `main` at `cadafbf`. Any further defaults must come from the actual config implementation before publishing the help in the real application.

## Review and delivery

See [VERIFICATION.md](VERIFICATION.md) for build results, screenshots and observed interaction checks. Screenshots and this README describe the candidate that was actually reviewed. Operator acceptance is recorded on the Rosemary task, one screen/decision per question. After selection, remove the discarded variants and review controls before preparing the final implementation reference; do not promote this disposable code directly into `web/`.
