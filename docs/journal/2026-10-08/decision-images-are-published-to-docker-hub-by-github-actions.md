---
type: journal
kind: decision
title: Images are published to Docker Hub by GitHub Actions
status: confirmed
author: Marek Kobylinski
created: 2026-10-08T10:05:58+0200
tags: [release, docker-hub, github-actions, image]
follows-up: 2026-10-08/draft-sundew-brief.md
---

# Images are published to Docker Hub by GitHub Actions

Decided by Marek on 8 October 2026: the Sundew image is built and published to Docker Hub by a
GitHub Actions workflow in this repository.

**Rejected:** `ghcr.io/kobylinski/sundew`, the registry the brief first named.

The Integrator's reading, not separately decided:

- A release is a version tag (`v1.2.3`) pushed to the repository; the workflow builds the image
  from that tag, runs the acceptance against it, and only then pushes it. A push to a branch
  publishes nothing ("nothing deploys from a push", `AGENTS.md`).
- Tags published: the version (`1.2.3`), the minor line (`1.2`) and `latest`.
- Platforms: `linux/amd64` and `linux/arm64`.
- The Docker Hub repository name is a repository variable and the credentials are repository
  secrets (a user name and an access token), set by the operator; nothing about the account is
  written in the workflow file.
