---
type: journal
kind: decision
title: Go runs locally; the remote test host runs only containers
status: confirmed
author: Marek Kobylinski
created: 2026-10-08T05:30:59+0200
tags: [testing, docker, toolchain, environment]
---

# Go runs locally; the remote test host runs only containers

Decided by Marek on 8 October 2026.

- `go vet`, `go test ./...` and any other use of the Go toolchain run on the developer's own
  machine, in the lane's worktree.
- The remote Docker host is used only for what needs Docker itself: building the image and
  running the acceptance script against the built image.
- Go is never installed on the remote host. Anything that must compile there compiles inside a
  container (the multi-stage image build).

**Rejected:** installing a Go toolchain on the remote host so the whole proof could run in one
place. The host is a shared Docker machine with little disk; a host toolchain would be a second
environment to keep in step with the local one, and the image build already carries its own
compiler.

The host's address is private and is not recorded in this repository; the operator gives it to
the agent that needs it.
