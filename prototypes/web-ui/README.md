# Sundew web UI — design prototype

Task: `tasks:40a0d78a850976e8`. Designer: Daniel. This is a disposable Svelte application on fictional, in-memory fixtures; it makes **no API calls** and sends no SMS. No production application code belongs here.

**Review status:** Marek selected the split inbox (q2), accepted the inspector content/interactions (q3) and the dedicated Install page (q4), and requested both themes with automatic switching (q5). q2 also requested a more finished visual treatment; that refinement is now implemented and remains to be reviewed. The earlier choices are retained. The prototype-only dependency exception is approved in q1; see [the complete dependency audit](../../docs/dependencies.md).

## Run and inspect

```sh
cd prototypes/web-ui
npm ci
npm run dev
```

Local URL: <http://127.0.0.1:4279/>. Node is used only to build and serve this prototype. `npm run build` produces static assets in `dist/`; `npm run preview` serves that build on port 4280.

The selected split inbox is the only layout. [Open the prototype](http://127.0.0.1:4279/) to use the system theme; [force light](http://127.0.0.1:4279/?theme=light) or [force dark](http://127.0.0.1:4279/?theme=dark) for visual review. B/C and the layout switcher have been removed. The original three-way comparison is preserved in commit `f0ff096`.

The UI defaults to `prefers-color-scheme` and listens for changes while open. Both palettes are product requirements. A manual theme setting is not proposed: the `theme` override is only a prototype review aid.

Use [the optional development controls](http://127.0.0.1:4279/?controls=1) to select a state or simulate an arrival. They are hidden by default and excluded from the static build. `state`, `theme`, `view`, `q`, `message`, and `tab` are prototype-only URL parameters; they are not API parameters or the final application's route contract. Old `variant` links open the selected inbox. Reloading resets local deletions and arrivals. No local storage, cookies, accounts, or login is used.

## Binding versus illustrative

The split layout, inspector content/interactions, dedicated Install page and automatic light/dark themes are operator decisions on q2–q5. The following task requirements remain binding:

- Newest first; recipient, sender, complete body, provider, delivery status and time readable in the list. Long text wraps; it is not silently truncated.
- One search field matching a substring in recipient, sender, body or account, case-insensitively. Search and live updates do not move keyboard focus.
- Every message field is inspectable. **Raw** on a row opens the captured request in one action; **Response** shows exactly what Sundew originally returned, separate from current delivery status. Copy actions preserve the raw bytes shown.
- Delete-one and delete-all require a clearly scoped confirmation, with Cancel initially focused. Delete-all covers the entire store, including unloaded and filtered-out messages. There is no invented Undo API.
- No message composer, settings screen, accounts screen, or login. Installation help is always one navigation link away, including in empty/error states.
- Svelte compiles to static assets. The real UI talks to the query API only and is embedded in the Go binary. No Node server is required at runtime.
- Messages are held in memory only; there is no database/volume configuration. Any provider credential is accepted. The install page lists only `SUNDEW_ADDR`, `SUNDEW_BASE_URL`, `SUNDEW_CALLBACK_DELAY`, `SUNDEW_CALLBACK_OUTCOME`, and `SUNDEW_INBOUND_URL`.

The remaining review concerns the refined visual finish requested in q2. The revision narrows the scanning list, gives the inspector more room, groups the message body and metadata, strengthens typography and navigation, and adds deliberate hover, focus and status treatments. It preserves the accepted fields and actions.

Illustrative: all phone numbers, accounts, message IDs, content, provider responses, error code examples, media URLs and timings. The fixture generator is not a provider emulator or SMS segment calculator. Typography, spacing and colors are review candidates. The arrival control is a review aid, never a proposed product feature. Installation snippets use `<version>` as a release placeholder, not a claim that a release is already published.

## Screens and states

| State / screen | Reach it | Expected behavior |
| --- | --- | --- |
| Normal inbox | `/` | Eight fictional messages, newest first. Desktop shows the first message in the inspector; phone/tablet starts with the list. |
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
| Automatic theme | Default URL, without `theme` | Follows the system light/dark preference and changes with it. Review overrides: `?theme=light` or `?theme=dark`. |

All states use the selected split-inbox layout. Raw provider response bodies are preserved as strings and are never regenerated from current status. The inbound fixture deliberately contains an empty `Exchange`; the implementation must render the actual captured data it receives, without assuming every inbound message has a provider façade request.

## Responsive behavior

- **1366 px:** Separately scrollable list and inspector, in roughly 42% / 58% columns with an 18 px gap. The recipient/body stays readable in both. Installation uses an introduction alongside the steps.
- **768 px:** List → detail → back. The inspector uses the whole content width, and metadata is grouped into readable pairs. Installation becomes one column.
- **390 px:** Full-width message rows; metadata wraps; every body and raw string wraps without horizontal page scrolling. Detail occupies the content region. The optional review controls fit one bottom row with content padding.
- The split-layout breakpoint is 1000 px. General phone styles apply at 600 px. Layout reflow is CSS-driven, not inferred from user-agent strings.
- Installation code is wrapped for reading; Copy still returns the original unwrapped command. The app has no remote fonts, images, icon package, or decorative illustration.

## Keyboard and screen readers

- Skip-to-content link, named navigation, main landmark, one page heading, semantic sections/articles, visible focus outlines, descriptive button labels.
- `/` focuses search outside editable controls. Escape in a nonempty search clears it. Escape from an open detail returns to the selected row. Standard browser Back moves between Messages and Install.
- Opening detail places focus on its heading. Closing returns focus to its message row. Deleting, retrying a failed load, and explicitly choosing Show newest return focus to the inbox heading. Automatic arrivals preserve focus.
- Details use a tablist with one tab stop; Left/Right/Home/End select and focus tabs. Tab moves into the panel and its copy/actions.
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

See [VERIFICATION.md](VERIFICATION.md) for build results, screenshots and observed interaction checks. Screenshots and this README describe the candidate that was actually reviewed. Operator acceptance is recorded on the Rosemary task, one screen/decision per question. The rejected layouts and layout-switching controls are removed; state/theme preview controls are development-only and opt-in. Preserve the operator decisions above when implementing, and do not promote this disposable code directly into `web/`. No Rosemary gates are used, following the operator's instruction.
