# Sundew — brief

Written 8 October 2026 by Adam (Merchant Caddy's Architect) for Marek, who asked for a reusable
Docker service that catches SMS the way Mailpit catches email, implements the public APIs of
popular SMS providers so applications need no code change to use it, and offers an API that E2E
frameworks can read messages from. First consumer: Merchant Caddy's public forms on demand (send a
form-request link by SMS; `local:7af88e884f141fd1` in that project). Nothing in this brief is
Merchant Caddy specific.

## What it is

One container, one binary, no external dependencies. It listens on one port and offers:

1. **Provider façades** — HTTP endpoints that imitate the *sending* side of real SMS providers
   closely enough that an unmodified client SDK or a hand-written client works when its base URL
   is pointed at Sundew. Authentication is imitated too (Basic auth with account SID and token,
   API-key headers, bearer tokens): by default any credential is accepted and recorded; a strict
   mode accepts only configured credentials and answers the provider's own 401 shape.
2. **A catch store** — every accepted message is stored with: provider, account/credential used,
   from, to, body, segments, media URLs, provider-shaped message id, status, timestamps, the raw
   request, and any provider options (status callback URL, validity, sender id).
3. **A web UI** — a list of caught messages, newest first, search by to/from/body/account, a
   message view with the raw request and the provider response Sundew returned, delete one/all,
   a live update when new messages arrive. Clean, small, no login (it is a dev tool behind a
   dev gateway), responsive enough to glance at on a phone.
4. **A query API for tests** — `GET /api/v1/messages` with filters (`to`, `from`, `body`
   contains, `account`, `since`, `provider`), paged; `GET /api/v1/messages/{id}`;
   `DELETE /api/v1/messages`, `DELETE /api/v1/messages/{id}`; `GET /api/v1/messages/latest?to=…`
   (the common E2E case: "the text that just went to this number"); long-poll or SSE
   `GET /api/v1/messages/stream` so a test can wait for a message without sleeping. All
   JSON; stable; documented by a generated OpenAPI file served at `/api/v1/openapi.json`.
5. **Callback simulation** — when a façade request carries a status-callback URL, Sundew can
   deliver the provider-shaped status webhooks (queued → sent → delivered, or failed) to the
   application, with a configurable delay and outcome, so the application's inbound webhook
   handling is testable. Also an **inbound message simulator**: `POST /api/v1/inbound` makes
   Sundew call the application's inbound-SMS webhook in the chosen provider's shape ("the
   merchant replied STOP").
6. **Health and reset** — `GET /healthz`; `POST /api/v1/reset` empties the store (tests call it
   in setup).

## Providers to imitate, in order

| Provider | Façade (sending) | Status callback shape | Inbound shape |
| --- | --- | --- | --- |
| Twilio | `POST /2010-04-01/Accounts/{sid}/Messages.json` (form-encoded), `GET …/Messages/{id}.json` | `StatusCallback` POST, `MessageStatus` | Twilio webhook form fields |
| Vonage (Nexmo) | `POST /sms/json` and the Messages API `POST /v1/messages` | delivery receipt webhook | inbound webhook JSON |
| MessageBird / Bird | `POST /messages` (legacy) and Bird's conversations send | status webhook | inbound webhook |
| Plivo | `POST /v1/Account/{id}/Message/` | `url` callback | inbound |
| Sinch | `POST /xms/v1/{plan}/batches` | delivery report | inbound |

Twilio first, alone, end to end (façade, UI, query API, callbacks); then one provider at a time.
Each provider is one module implementing a small interface (parse request → canonical message;
canonical message → provider response; status → provider callback payload; inbound → provider
webhook payload) so adding a provider never touches the core. Exactness is measured against the
provider's public API reference, cited by URL and date in the module's header; where a provider
behaviour is unknowable without an account, say so in the module rather than guessing.

## Non-goals

No real sending, ever — there is no provider credential anywhere in Sundew. No MMS media
hosting beyond storing the URLs given. No multi-user accounts or login in the UI. No persistence
guarantees beyond the container's lifetime unless a volume is mounted (SQLite file; in-memory by
default). No email (Mailpit does that).

## Shape

- **Language and runtime:** a single static binary. Go is the recommendation (as `fouroclock`):
  one image of a few megabytes, trivial cross-compilation, a standard library HTTP server, SQLite
  through a pure-Go driver, embedded UI assets. The UI is plain HTML + a small amount of
  JavaScript, embedded; no Node at runtime.
- **Image:** `ghcr.io/<owner>/sundew:<version>`, `FROM scratch` or distroless, one port (default
  `8025`-style single port for UI, façades and API, with a second optional port for the façades
  only if a consumer needs them separated). Configuration by environment variables:
  `SUNDEW_STRICT_AUTH`, `SUNDEW_ACCOUNTS` (sid:token pairs), `SUNDEW_CALLBACK_DELAY`,
  `SUNDEW_CALLBACK_OUTCOME`, `SUNDEW_DB` (path; empty = memory), `SUNDEW_BASE_URL` (what the
  façades put in their own response links).
- **Compose snippet** in the README for the common case (an app whose `TWILIO_API_BASE` points
  at `http://sundew:8025`), and a gateway label example for stacks behind a dev gateway.
- **E2E helpers:** the HTTP API is the contract; thin optional helpers for Playwright and Vitest
  (`waitForSms({to})`) may live in `clients/` later, published separately, never required.
- **Dependency policy:** every third-party module must show a commit within the last six months
  at the time it is added; record the check in `docs/dependencies.md`.
- **Licence:** MIT, so other projects and third parties can adopt it without asking.

## Acceptance of the first release (Twilio only)

1. `docker run -p 8025:8025 sundew` starts in under a second; `/healthz` answers.
2. The official Twilio Python and Node SDKs, with `TWILIO_…` credentials of any value and the
   base URL pointed at Sundew, send a message without error and receive a response the SDK parses
   (SID, status `queued`, the echoed fields).
3. The UI shows the message within a second; search by the destination number finds it.
4. `GET /api/v1/messages/latest?to=%2B15551234567` returns it; `GET …/stream` delivers it to a
   client that connected before it was sent.
5. With a `StatusCallback`, the application receives `queued`, `sent`, `delivered` POSTs in
   Twilio's shape with the configured delay; `SUNDEW_CALLBACK_OUTCOME=failed` yields `failed`
   with an error code.
6. `POST /api/v1/inbound` makes Sundew POST a Twilio-shaped inbound message to the configured
   application webhook.
7. Strict auth refuses an unknown SID/token with Twilio's 401 body; default mode accepts and
   records anything.
8. `POST /api/v1/reset` empties the store; a test suite's setup can rely on it.
9. The repository has: README with the compose snippet, `docs/providers/twilio.md` with the
   cited reference and the known gaps, the OpenAPI file, a CI that builds the image and runs the
   acceptance above against it.

## How Merchant Caddy will use it

Its SMS adapter interface gets a provider adapter (Twilio first) whose base URL is configurable;
dev lanes point it at a `sundew` service in the dev compose, and the E2E journey that sends a
form-request link by SMS reads the link back from `GET /api/v1/messages/latest`. Production points
the same adapter at the real provider. Until Sundew exists, Merchant Caddy's SMS send is behind
the adapter with no default sink; the "Outbox page" alternative was declined by Marek on 8 October
in favour of this service.

## Open for Marek

- ~~The GitHub owner and visibility~~ — decided 8 October: public repository `kobylinski/sundew`, MIT.
- Whether the first release should also carry Vonage (Caddy Pay or another project may need it).
- Whether a tiny Polish-market provider (SMSAPI, SerwerSMS) belongs in the provider list.
