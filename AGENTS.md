# Sundew

A catch-all SMS service for development and E2E tests — Mailpit, for text messages. Read
[README.md](README.md) for the name and [docs/journal/2026-10-08/draft-sundew-brief.md](docs/journal/2026-10-08/draft-sundew-brief.md) for what is being built;
the brief is the product direction until a spec replaces it.

`CLAUDE.md` is a symlink to this file; keep shared guidance here.

## Documentation

- `docs/journal/2026-10-08/draft-sundew-brief.md` — the brief (Marek, 8 October 2026).
- `docs/journal/YYYY-MM-DD/<kind>-<slug>.md` — decisions, research, lessons; the `journal` skill's
  format (frontmatter: type, kind, title, status, author, created, tags). An accepted decision is a
  `decision` entry; provisional work is `draft`.
- `docs/providers/<name>.md` — one per imitated provider: the cited public reference (URL and date
  read), the façade's coverage, known gaps.
- `docs/dependencies.md` — every third-party module with the date its maintenance was checked
  (commit within six months, or it is not added).

## Rosemary

Rosemary indexes this repository and holds its tasks. `rosemary.yml` is committed; `.mcp.json`
wires the daemon for Claude and Codex. Tasks are git-native (`tasks:<id>`), the same from every
branch and worktree. Join only on the operator's instruction; contracts live in `docs/contracts/`
when the team is formed.

## Git

`main` is the integration branch. One task, one branch, one worktree under `.worktrees/`
(ignored). Nothing deploys from a push; releases are tagged images.

## Stack (when scaffolded)

Go, standard library HTTP, an in-memory store; the web UI is a Svelte application
compiled to static assets and embedded; a single static binary in a `FROM scratch` or distroless
image. Node only at build time, never at runtime.
