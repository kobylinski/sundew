---
type: journal
kind: decision
title: Any credential is accepted; errors come from magic numbers
status: confirmed
author: Marek Kobylinski
created: 2026-10-08T05:58:13+0200
tags: [auth, providers, twilio, errors]
follows-up: 2026-10-08/draft-sundew-brief.md
---

# Any credential is accepted; errors come from magic numbers

Decided by Marek on 8 October 2026:

- A façade accepts whatever credential it is given. There is no strict mode and no list of
  configured accounts (`SUNDEW_STRICT_AUTH` and `SUNDEW_ACCOUNTS` are gone).
- The caught request is stored as sent, credential included; nothing is redacted.
- An application tests its error handling by sending to or from functional ("magic") phone
  numbers, each of which makes the façade answer a specific provider error.

**Rejected:** a strict mode that accepts only configured SID/token pairs and answers the
provider's 401 otherwise, as the brief first proposed; and redacting the secret in the stored
request.

The Integrator's reading, not separately decided: each provider module uses the magic numbers
that provider documents publicly (for Twilio, the test-credential numbers in its API reference)
and cites them in `docs/providers/<name>.md`; Sundew invents numbers only where the provider
documents none. Whether an authentication error (401) also needs a trigger is open.
