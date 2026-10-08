# Sundew

**A catch-all SMS service for development and end-to-end tests — Mailpit, for text messages.**

Sundew stands in for the SMS providers your application talks to (Twilio, Vonage, MessageBird,
Plivo, …). Point the application's provider base URL at Sundew and every message it "sends" is
caught, stored and shown in a small web UI, and available to your E2E tests over a plain HTTP API.
Nothing ever reaches a phone.

## Why the name

The sundew (*Drosera*) is a small carnivorous plant whose leaves carry glistening sticky droplets.
Anything small that lands on them is held fast and kept, in plain view, for the plant to take its
time with. Sundew the service does the same with short messages: whatever your application sends
lands on it, sticks, and stays visible until you have looked. Small prey, small messages; a drop of
dew per text.

## Status

Foundation stage: configuration, in-memory message storage, event subscriptions and `/healthz`.
Provider façades, the query API and UI are being built. See the
[brief](docs/journal/2026-10-08/draft-sundew-brief.md).

## Run it

Build and run from this checkout:

```sh
docker build -t sundew:dev .
docker run --rm -p 8025:8025 sundew:dev
curl http://localhost:8025/healthz
```

The health endpoint returns `ok`. Stop with Ctrl-C; SIGTERM drains HTTP requests before exit.
The runtime image contains one static Go binary and CA certificates; it needs no Node runtime.

Messages live in memory and disappear when the process stops. No database or volume is needed.

Configuration is read once at startup. Empty variables use the defaults below; malformed
values stop startup with the variable name in the error.

| Variable | Default | Meaning |
| --- | --- | --- |
| `SUNDEW_ADDR` | `:8025` | HTTP listen address |
| `SUNDEW_CALLBACK_DELAY` | `0s` | Nonnegative Go duration between callbacks |
| `SUNDEW_CALLBACK_OUTCOME` | `delivered` | `delivered` or `failed` |
| `SUNDEW_BASE_URL` | empty | Absolute HTTP(S) base URL for provider response links |
| `SUNDEW_INBOUND_URL` | empty | Absolute HTTP(S) default inbound webhook target |

## Development checks

Run Go locally, and Docker acceptance against the selected local or remote engine:

```sh
go vet ./...
go test -race ./...
scripts/acceptance.sh
```

The acceptance runner builds a unique image, runs executable `scripts/acceptance/*.sh` steps
in name order and cleans up its containers, network, volumes and image on success or failure.
Clients run on the same Docker network as Sundew. No host ports or bind mounts are needed;
`DOCKER_HOST` and Docker contexts work for remote engines too. The health step measures
startup under one second; the runner requires an image smaller than 30 MB.
