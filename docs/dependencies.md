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
