---
type: journal
kind: decision
title: The maintenance rule applies to packages we choose directly
status: confirmed
author: Marek Kobylinski
created: 2026-10-08T17:38:53+0200
tags: [dependencies, svelte, vite, ui]
follows-up: 2026-10-08/decision-web-ui-is-built-with-svelte.md
---

# The maintenance rule applies to packages we choose directly

Decided by Marek on 8 October 2026, answering an operator question on the production UI task
(`tasks:4f6ecdb55a9238e9#q1`): the production web UI may ship built with the twelve transitive
packages that fail the six-month maintenance rule. He had allowed the same set for the design
prototype earlier in the day (`tasks:40a0d78a850976e8#q1`).

The rule — a commit within six months, or the module is not added — applies to packages chosen
directly (here Svelte, Vite and the Svelte plugin for Vite, all current). A transitive package
that fails it is tolerated when it arrives with a directly chosen one, and is listed by name with
its last commit date in `docs/dependencies.md`.

**Rejected:** keeping the rule strict for the whole dependency graph, which no Svelte and Vite
setup can satisfy and which would have meant rebuilding the UI as hand-written HTML and
JavaScript.

These packages are build-time only: the image contains the compiled JavaScript and CSS, no Node
and no `node_modules`.
