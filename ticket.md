---
assignees:
- Marek Kobylinski
created_at: 2026-10-08T03:37:44.368737+00:00
id: 71faaaf3455634f5
labels: []
status: in_progress
title: 'Twilio façade: send, fetch, auth, status and inbound webhook shapes'
updated_at: 2026-10-08T03:37:56.031368+00:00
---
## Outcome

The official Twilio Python and Node SDKs, with credentials of any value and their base URL pointed at Sundew, send a message without error and get back a response they parse (SID, status `queued`, the echoed fields). The message is in the store with the raw exchange. Strict auth refuses an unknown SID/token with Twilio's own 401 body.

## Scope

In — you own `internal/provider/twilio` and `docs/providers/twilio.md`:
- `core.Provider` named `twilio`.
- `POST /2010-04-01/Accounts/{sid}/Messages.json` (form-encoded): `To`, `From` or `MessagingServiceSid`, `Body`, `MediaUrl` (repeatable), `StatusCallback`, `ValidityPeriod`. Twilio's validation errors in Twilio's shape and status for a missing `To`, a missing sender, and neither `Body` nor `MediaUrl`.
- `GET /2010-04-01/Accounts/{sid}/Messages/{sid}.json` returning the message with its current status; Twilio's 404 shape when unknown.
- The response resource with the fields the SDKs read: `sid` (`SM` + 32 hex), `account_sid`, `to`, `from`, `body`, `status`, `num_segments`, `num_media`, `direction`, `date_created`, `date_updated`, `date_sent`, `uri`, `subresource_uris`, `error_code`, `error_message`, `price`, `price_unit`, `api_version`, `messaging_service_sid`. Links built from `Config.BaseURL`.
- Segment count: GSM-7 (160 / 153) and UCS-2 (70 / 67), with the GSM-7 extension characters counted as two.
- Auth: HTTP Basic, account SID and token. Default: anything accepted, the SID recorded in `Message.Account`. `StrictAuth`: only `Config.Accounts`; otherwise Twilio's 401 body and headers. The SID in the path must match the credential's.
- `StatusRequest`: Twilio's `StatusCallback` POST (form fields `MessageSid`, `MessageStatus`, `AccountSid`, `To`, `From`, `ApiVersion`, and `ErrorCode` when failed). `nil, nil` when the message has no `status_callback`.
- `InboundRequest`: Twilio's inbound-message webhook form fields.
- `docs/providers/twilio.md`: the public reference pages used (URL and the date you read them), what the façade covers, the known gaps. The same citation in the package's header comment. A behaviour you cannot know without an account is stated as unknown, not guessed. Request signing (`X-Twilio-Signature`) on outgoing webhooks: implement it if the public reference specifies it fully, signing with the account's token from `Config.Accounts` when there is one; list it as a gap otherwise.
- `scripts/acceptance/20-twilio-sdk.sh`: the Python and the Node SDK, each in its own container on Sundew's network, send one message and assert on the parsed response; a second case under strict auth asserts the 401.

Out: delivering webhooks (the callback engine does it; you only build the request), the query API, the store, other providers, MMS media hosting. Do not change `internal/core`; a change you need is a `decide`.

## How to verify

On this machine, in the lane:
- `go vet ./...` exits 0; `go test -race ./...` exits 0. Handler tests use `httptest` and a fake `core.Store` in the test files.
- Tests cover: a successful send and the stored `core.Message` (every field, the `Exchange` included); each validation error; fetch found / not found; default auth; strict auth accepted / refused / path SID mismatch; segment counts at 160, 161, 306, 307 GSM-7 characters, at 70 and 71 UCS-2 characters, and with extension characters; `StatusRequest` for each status and for no callback; `InboundRequest`.
- After the Foundation lane has merged and you have rebased: your one registration line is in `cmd/sundew/main.go`, and `scripts/acceptance.sh` exits 0 with your step in it.

## Route

`design`: no. `review_code`: yes. `review_tests`: yes. e2e acceptance: yes — `20-twilio-sdk.sh`; the Reviewer extends it.

## Decided and open

- Start now, against `internal/core` and fakes; you do not need the Foundation lane to write and test the package. You need it merged before `build`: rebase, add your registration line, run the acceptance. Conflicts in `go.mod`, `go.sum`, `docs/dependencies.md` and the registration block are yours to resolve on rebase.
- Package layout, ownership and proof rules: `docs/architecture.md`. Go here, Docker possibly remote, containers only: `docs/journal/2026-10-08/decision-go-runs-locally-remote-host-runs-only-containers.md`.
- Standard library only is expected for this package; any module you add follows the six-month rule.
- The SDK versions used in the acceptance are pinned and recorded in `docs/providers/twilio.md`.

## Where to look

`internal/core/core.go` (`Provider`, `Message`, `Exchange`, `Options` keys), `docs/architecture.md`, the brief (`docs/journal/2026-10-08/draft-sundew-brief.md`: "Providers to imitate", acceptance items 2 and 7), Twilio's public Messages API reference.