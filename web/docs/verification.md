# Production UI verification

Task `tasks:4f6ecdb55a9238e9`; author Paula, 8 October 2026. The accepted design
is `prototypes/web-ui/README.md`. Screenshots show the real embedded application,
using only disposable messages sent through the local Twilio façade. The browser's
actual theme was **light**; its system preference was not changed.

Observed in the local browser:

- Three messages arrived without reload; desktop selection and keyboard focus stayed put.
- Long text and raw HTTP strings wrapped at 390, 768 and 1366 px without horizontal overflow.
- Phone autocomplete distinguished To/From; pointer selection retained search focus. Exact
  phone filtering combined with a body substring and produced the expected single result.
- Raw opened the report and focused the request heading. Clipboard contents exactly equaled
  the displayed request. Original queued response remained literal alongside delivered status.
- HTTP(S) body/media/callback links had `_blank` plus `noopener noreferrer`; unsafe schemes
  and HTML fragments were text. Raw sections contained no links.
- Message and Install deep links survived reload; browser Back returned to the message report.
- Delete-one's dialog named its target and focused Cancel. Cancel kept the message; confirming
  deleted it and focused the inbox heading. Delete-all named the entire store, including hidden
  and unloaded messages; clearing the filter afterward showed an empty store.
- Fifty rows showed `50+`; Load more returned the remaining two and showed `52`. An arrival
  while scrolled away kept 52 rows and displayed the new-message banner; Show newest then
  displayed 53, scrolled to the top and focused the inbox heading.

Evidence: [desktop](inbox-1366.png), [tablet](inbox-768.png), [phone](inbox-390.png),
[phone raw report](raw-390.png), [phone Install](install-390.png).

Per Ivan's task comment c10, the opposite theme is verified **without a fresh browser capture**.
`npm test` builds the production CSS and compares both palettes and the dark status colors
with the accepted prototype, checks token completeness, and requires the dark palette inside
`prefers-color-scheme: dark`. Production has no JavaScript theme state or theme override:
CSS handles initial preference and live changes. The accepted prototype's dark screenshots
are in `prototypes/web-ui/screenshots/a-dark-390.png` and `a-dark-1366.png`.

`npm test` also covers safe link parsing, query/phone/cursor mapping, nullable inbound records,
created/updated/deleted/reset reconciliation, stream-before-snapshot ordering, buffered races,
reconnect revalidation, stale search cancellation, pagination deletion races, 204 deletion and
failed-delete retention, and superseded-page request/buffer cleanup. Go checks run with only
the committed fallback page and again with compiled assets. Image acceptance covers the
existing provider/API/callback suite plus real UI assets, routes, HEAD, CSP and cache headers.
