# Architecture

How the first release is cut into packages, so that lanes can be built side by side. The product
is described in the brief ([journal draft](journal/2026-10-08/draft-sundew-brief.md)); this file
says where each part lives and what it may depend on.

## Packages

| Package | Holds | Depends on |
| --- | --- | --- |
| `internal/core` | The seam: `Message`, `Filter`, `Store`, `Bus`, `Provider`, `Config`, `Deps`. Types and interfaces only. | nothing in this module |
| `internal/config` | Reads `SUNDEW_*` into `core.Config`. | `core` |
| `internal/bus` | The in-process `core.Bus`. | `core` |
| `internal/store` | The SQLite `core.Store` (file, or in-memory when `SUNDEW_DB` is empty). Publishes an event for every write. | `core` |
| `internal/server` | Builds the `http.Handler`: one `ServeMux`, `/healthz`, and the list of registrations. | `core` |
| `internal/provider/<name>` | One imitated provider: façade routes, auth, response shapes, status and inbound webhook payloads. | `core` |
| `internal/api` | The query API under `/api/v1` (messages, latest, stream, reset, inbound, `openapi.json`). | `core` |
| `internal/callback` | Delivers status webhooks after a message is created, and inbound webhooks on request. | `core` |
| `cmd/sundew` | `main`: config, store, bus, providers, server. | all of the above |
| `web/` | The Svelte UI; its build output is embedded and served at `/`. | the query API only |

A package depends on `core` and never on a sibling. `internal/api` does not import a provider;
`internal/provider/twilio` does not import the store. They meet in `cmd/sundew`.

## The seam

`internal/core/core.go` is on `main` before any lane starts and is the contract between lanes.
The JSON form of `core.Message` is the query API's message object, so the UI builds against it
too. Changing the file is a `decide` gate, not a line in a feature branch.

## How a message flows

1. The application calls a façade route. The provider module checks the credential (any is
   accepted unless `StrictAuth`), builds a `core.Message` with `Direction: Outbound`,
   `Status: queued` and the raw `Exchange`, calls `Store.Insert`, writes the provider-shaped
   response.
2. The store assigns `ID`, sets the timestamps, saves, publishes `message.created`.
3. `internal/api`'s stream endpoint forwards the event to connected clients.
4. `internal/callback` receives the same event. When the message carries
   `Options["status_callback"]`, it asks `Providers[m.Provider].StatusRequest` for each next
   status (`sent`, then `delivered` or `failed`), waits `CallbackDelay` between them, calls
   `Store.UpdateStatus`, sends the request.
5. `POST /api/v1/inbound` asks the chosen provider's `InboundRequest` for a webhook to
   `InboundURL` (or the `url` given in the request), sends it, and stores the message with
   `Direction: Inbound`, `Status: received`.

## Registration

`internal/server` exposes one function taking `core.Deps` and a list of
`func(*http.ServeMux, core.Deps)`. `cmd/sundew/main.go` holds the list, one line per package,
between the comments `// registrations:` and `// end registrations`. A lane that adds a package
adds its line there and nowhere else.

## Proof

- `go vet ./...` and `go test ./...` run on the developer's machine.
- The image and the acceptance run in Docker only. `scripts/acceptance.sh` builds the image and
  runs every `scripts/acceptance/*.sh` against it; each step runs its client in a container on
  the same Docker network as Sundew (no bind mounts, no published host ports, a compose project
  name unique to the run), so the script behaves the same against a local or a remote Docker
  engine selected by `DOCKER_HOST`.
- Go is not installed on a remote Docker host; anything that compiles there compiles in the
  image's build stage.
