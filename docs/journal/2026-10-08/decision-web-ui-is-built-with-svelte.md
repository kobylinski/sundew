---
type: journal
kind: decision
title: The web UI is built with Svelte
status: confirmed
author: Marek Kobylinski
created: 2026-10-08T05:34:09+0200
tags: [ui, svelte, frontend, build]
follows-up: 2026-10-08/draft-sundew-brief.md
---

# The web UI is built with Svelte

Decided by Marek on 8 October 2026: Sundew's web UI is a Svelte application. It stays simple —
a place to review caught messages, plus information on how to install Sundew and point an
application at it.

**Rejected:** plain HTML with a small amount of hand-written JavaScript, as the brief first
proposed. The brief's constraint that mattered — no Node at runtime, one static binary — does not
require it.

What follows from it (the Integrator's reading, not separately decided):

- Svelte is compiled at build time to static assets, which are embedded in the Go binary. Node
  exists only in a build stage of the image and on a developer's machine; the released image
  still contains one binary and no Node.
- A client-side application built with Vite, talking to `/api/v1`. No SvelteKit server and no
  server-side rendering, since nothing but the Go binary runs.
- Svelte and Vite are third-party dependencies and go into `docs/dependencies.md` when added.
  Checked 8 October 2026: `sveltejs/svelte` last commit 7 October 2026 (5.57.2);
  `vitejs/vite` last commit 8 October 2026 (8.3.3).
