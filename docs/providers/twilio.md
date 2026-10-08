# Twilio

Public references read **2026-10-08**; no Twilio account or real credential was used.

| Reference | Used for |
| --- | --- |
| [Messages resource](https://www.twilio.com/docs/messaging/api/message-resource) | Routes, request fields, response properties and status callbacks. |
| [API responses](https://www.twilio.com/docs/usage/twilios-response) | JSON errors, HTTP statuses and date format. |
| [Incoming webhook](https://www.twilio.com/docs/messaging/guides/webhook-request) | Incoming SMS fields and indexed media URLs. |
| [Security](https://www.twilio.com/docs/usage/security) | Form webhook signing algorithm and published test vector. |
| [SMS length](https://www.twilio.com/docs/glossary/what-sms-character-limit), [GSM-7](https://www.twilio.com/docs/glossary/what-is-gsm-7-character-encoding), [UCS-2](https://www.twilio.com/docs/glossary/what-is-ucs-2-character-encoding) | Segment limits and encoding. |
| [21604](https://www.twilio.com/docs/api/errors/21604), [21603](https://www.twilio.com/docs/api/errors/21603), [21602](https://www.twilio.com/docs/api/errors/21602) | Missing recipient, sender and content. |
| [20404](https://www.twilio.com/docs/api/errors/20404), [20004](https://www.twilio.com/docs/api/errors/20004) | Unknown resources and unsupported methods. |
| [Test credentials: SMS magic numbers](https://www.twilio.com/docs/iam/test-credentials#test-sending-an-sms) | Published From/To error triggers. |
| [30008](https://www.twilio.com/docs/api/errors/30008) | Default simulated failure: unknown delivery error. |

## Coverage

`twilio.New(config)` implements `core.Provider`. It serves form POSTs at
`/2010-04-01/Accounts/{account}/Messages.json` and JSON fetches at
`/2010-04-01/Accounts/{account}/Messages/{message}.json`.

The façade captures `To`, `From`, `Body`, repeated `MediaUrl`, `MessagingServiceSid`,
`StatusCallback` and `ValidityPeriod`. Other form fields are retained in Options but have no
behaviour. It stores a queued canonical message with a random `SM` + 32 hex SID. The queued
response is captured before the store publishes its creation event; the internal store ID does
not appear in the provider response. Fetch reads the current store status and checks the account namespace. Response links use `Config.BaseURL`, or relative paths when it is empty.

Missing recipient, sender, or content yields HTTP 400 and codes 21604, 21603, or 21602.
Missing resources yield HTTP 404/code 20404. Unsupported methods yield HTTP 405/code 20004
with Allow: POST for send, or GET, HEAD for fetch. All errors have `code`, `message`, `more_info`,
`status`. Every credential is accepted, including missing or malformed authorization. Basic
username is recorded as Message.Account when present; otherwise the account path is used.
The entire request headers and body are captured unchanged, including Authorization and Cookie,
per the [operator decision](../journal/2026-10-08/decision-any-credential-accepted-errors-by-magic-numbers.md).

The documented SMS magic inputs return HTTP 400 without storing a message:

| Parameter | Number | Code |
| --- | --- | --- |
| From | +15005550001 | 21212 |
| From | +15005550007 | 21606 |
| From | +15005550008 | 21611 |
| To | +15005550001 | 21211 |
| To | +15005550002 | 21612 |
| To | +15005550003 | 21408 |
| To | +15005550004 | 21610 |
| To | +15005550009 | 21614 |

From +15005550006 succeeds. Ordinary numbers succeed too: Sundew emulates the explicit SMS magic
inputs, not Twilio's account-bound restrictions or the unrelated number-purchase/voice tables.
Required-field validation runs first; when both sender and recipient trigger errors, sender wins.

Segments use GSM-7 septets (extensions take two), otherwise UTF-16 units. A single part holds
160/70 units; multipart parts hold 153/67. Escape sequences and surrogate pairs stay together.
Empty body has zero text segments, including media-only messages.

Status requests are form POSTs with MessageSid, MessageStatus, AccountSid, To, From and ApiVersion.
Failed requests carry ErrorCode, defaulting to 30008 when omitted; explicit codes are preserved.
No callback URL returns no request. Inbound requests contain the generated MessageSid and its
legacy aliases, account, sender, recipient, body, counts and indexed MediaUrl fields.

Status webhooks carry `X-Twilio-Signature` when the original send request used Basic auth. The
key is the presented token, read from the captured Authorization header. The signature is base64
HMAC-SHA1 over the full URL plus alphabetically ordered form names and values; the signed URL
keeps its port and query and excludes userinfo and the unsent fragment. Without a usable Basic token it is unsigned.
Inbound input has no originating credential; its webhook is unsigned. Request construction
performs no network I/O.

## Known gaps

- No account-backed reference traffic was captured. Exact validation precedence, error wording
  and undocumented headers remain unverified. Error codes and JSON shape follow public references;
  HTTP 400 is used for the SMS magic validation errors. No live parity claim is made.
- Only send and fetch are supported. No list/update/delete provider API, media resource hosting,
  feedback, templates, scheduling, sender pools or provider billing simulation.
- MessagingServiceSid is retained; Sundew has no sender pool to choose a From number. Its
  response uses a null From when omitted, remains queued, and reports computed text segments.
- Dates created/updated follow the store. `date_sent` stays null because the shared core model
  does not retain a separate sent timestamp; using the delivery update time would be misleading.
- The inbound builder generates a SID in the request. Its interface returns only an HTTP request;
  the API stores inbound messages with empty ProviderID by accepted first-release decision.
- Fetch on failed messages with an empty stored ErrorCode presents 30008, matching the webhook.
  Numeric explicit error codes are returned; an error description is supplied only for 30008.
- ValidityPeriod and unsupported options are captured, not enforced. No telephone-number, media
  reachability, sender ownership, carrier, destination, account-limit or opt-out validation.
- No smart encoding, national shift tables, toll-free-specific multipart limits or MMS accounting.
- Inbound MediaContentType, geography and carrier metadata cannot be inferred from the input;
  they are omitted. Sundew never fetches media to guess content types.
- Form bodies are limited to 1 MiB; malformed forms return a provider-shaped local validation
  error. These development-tool limits are not a claim about Twilio's production service.

## SDK acceptance

`scripts/acceptance/20-twilio-sdk.sh` uses separate Python and Node client containers on the
runner's Docker network, with no bind mounts or published ports. Each creates and fetches a
message through the official SDK and checks its parsed response. Each also sends to +15005550009
and checks that the SDK raises with HTTP 400/code 21614.

Pinned SDKs, checked 2026-10-08:

- Python `twilio==9.11.2`: [release](https://github.com/twilio/twilio-python/releases/tag/9.11.2),
  commit `2fd57cf8f344c472c6e3b14205ad266d1bd6babc` (2026-09-28).
- Node `twilio@6.1.2`: [release](https://github.com/twilio/twilio-node/releases/tag/6.1.2),
  release commit `de6edf89c937b87dd155b6f1ca9b659ae4dab54d` (2026-09-28);
  latest project commit `54049bffe48104324efd101e7c7180db1a16c27f` (2026-10-07).

These are test dependencies only. See `docs/dependencies.md` for maintenance checks.
