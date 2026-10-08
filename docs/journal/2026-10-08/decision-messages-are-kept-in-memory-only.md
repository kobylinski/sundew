---
type: journal
kind: decision
title: Messages are kept in memory only
status: confirmed
author: Marek Kobylinski
created: 2026-10-08T05:58:13+0200
tags: [store, persistence, dependencies]
follows-up: 2026-10-08/draft-sundew-brief.md
---

# Messages are kept in memory only

Decided by Marek on 8 October 2026: Sundew retains nothing. Caught messages live in memory for
the life of the process and are gone when the container stops. There is no database file, no
`SUNDEW_DB`, no volume to mount.

**Rejected:** SQLite through a pure-Go driver, in memory by default with an optional file, as the
brief first proposed. The question came up because both maintained pure-Go drivers
(`modernc.org/sqlite`, `github.com/ncruces/go-sqlite3`) pull in a transitive module with no
commit in over six months, against the dependency rule. Dropping persistence removes the driver
and the question with it: the store needs no third-party module at all.

Consequence: the store is a plain in-memory implementation of `core.Store`; the Go side of the
first release has no third-party dependency.
