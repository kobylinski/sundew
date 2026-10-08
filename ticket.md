---
assignees:
- Marek Kobylinski
created_at: 2026-10-08T03:37:43.945391+00:00
id: b099379fbbff6b1d
labels: []
status: closed
title: 'Foundation: binary, config, store, bus, server, image and acceptance runner'
updated_at: 2026-10-08T04:23:04.052444+00:00
---
## Outcome

`docker run -p 8025:8025 sundew` starts in under a second and `GET /healthz` answers 200. Behind it: configuration from the environment, the SQLite store, the event bus, the HTTP server other packages register on, the image, CI, and the acceptance runner the other lanes add steps to. This lane merges first; two other lanes rebase onto it.

## Scope

In — you own these paths:
- `cmd/sundew/main.go`: config → bus → store → providers map → server; graceful shutdown on SIGTERM. Registrations between the comments `// registrations:` and `// end registrations`.
- `internal/config`: `SUNDEW_*` → `core.Config`, exactly the fields and defaults in `internal/core/core.go`; an invalid value is a start-up error naming the variable.
- `internal/bus`: `core.Bus`. Publish never blocks; a slow subscriber loses events; the channel closes when its context ends.
- `internal/store`: `core.Store` on SQLite through a pure-Go driver; in-memory when `SUNDEW_DB` is empty, a file otherwise. `Insert` assigns `ID` when empty and sets both timestamps. Newest first. `Filter` semantics as documented on the type. Opaque cursor paging. Every successful write publishes its event. Safe for concurrent use.
- `internal/server`: one function taking `core.Deps` and a list of `func(*http.ServeMux, core.Deps)`, returning the `http.Handler`; `GET /healthz`.
- `Dockerfile`: multi-stage, Go build stage, final stage `FROM scratch` or distroless, one static binary, port 8025.
- `scripts/acceptance.sh`: builds the image, then runs every executable `scripts/acceptance/*.sh` in name order, stops at the first failure, always tears down. Rules in `docs/architecture.md`, "Proof". Your own step: `scripts/acceptance/10-health.sh` (starts, `/healthz` 200, start-up under one second).
- `.github/workflows/ci.yml`: `go vet`, `go test ./...`, `scripts/acceptance.sh`.
- `docs/dependencies.md`: the SQLite driver and anything else, each with the date of its last commit.
- README: the status line and a "Run it" section with `docker run`.

Out: any route under `/api/v1`, any provider route, callbacks, the UI, `web/`. Do not change `internal/core`; a change you need is a `decide`.

## How to verify

On this machine, in the lane:
- `go vet ./...` exits 0; `go test -race ./...` exits 0.
- Store tests cover: insert/get, `GetByProviderID`, each `Filter` field, `Since`, paging through more than one page with no duplicate and no gap, `Latest`, `UpdateStatus`, `Delete`, `Reset`, `ErrNotFound`, concurrent inserts, a file database surviving reopen, and an event published for each write.
- `scripts/acceptance.sh` exits 0.
- `docker image inspect` shows an image under 30 MB.

## Route

`design`: no. `review_code`: yes. `review_tests`: yes. e2e acceptance: your `10-health.sh`; the Reviewer extends it when the façade and API lanes land.

## Decided and open

- Package layout, ownership and the proof rules: `docs/architecture.md`.
- Go runs on this machine; Docker work may be sent to a remote engine through `DOCKER_HOST`; Go is never installed on a remote host: `docs/journal/2026-10-08/decision-go-runs-locally-remote-host-runs-only-containers.md`. So: no bind mounts, no published host ports in the acceptance, a compose project name unique to the run, and remove images and volumes the run created.
- Dependency rule: a commit within six months or it is not added (`AGENTS.md`). The choice of pure-Go SQLite driver is yours; say why in `docs/dependencies.md`.
- Default page size: 50, maximum 500.

## Where to look

`internal/core/core.go`, `docs/architecture.md`, `docs/journal/2026-10-08/draft-sundew-brief.md` (sections "Shape" and "Acceptance of the first release", items 1 and 9), `docs/contracts/programmer.md`.