# Dependencies

Checked on 2026-10-08. The Go application uses only the standard library: no third-party Go
module is added. `go list -m all` reports only `github.com/kobylinski/sundew`.

Messages are kept in memory, as decided in
[the memory-only decision](journal/2026-10-08/decision-messages-are-kept-in-memory-only.md).
No SQLite driver is required.

Build/test infrastructure uses the official `golang:1.26-alpine` build image,
`python:3.14-alpine` for isolated HTTP acceptance clients, `actions/checkout@v4`, and
`actions/setup-go@v5`. None ships in the scratch runtime image. Future third-party Go modules
must have an upstream commit within six months when added, with its date and source recorded here.

## Twilio SDK acceptance

Maintenance checked 2026-10-08; these SDKs run only in acceptance containers.

| Dependency | Purpose and pin | Maintenance evidence |
| --- | --- | --- |
| [twilio-python](https://github.com/twilio/twilio-python) | Container acceptance only; `twilio==9.11.2` | [Commit 2fd57cf](https://github.com/twilio/twilio-python/commit/2fd57cf8f344c472c6e3b14205ad266d1bd6babc), 2026-09-28; within six months. |
| [twilio-node](https://github.com/twilio/twilio-node) | Container acceptance only; `twilio@6.1.2` | [Commit 54049bf](https://github.com/twilio/twilio-node/commit/54049bffe48104324efd101e7c7180db1a16c27f), 2026-10-07; within six months. |
