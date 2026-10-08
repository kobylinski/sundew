---
assignees:
- Marek Kobylinski
created_at: 2026-10-08T08:06:53.392590+00:00
id: 0771147d2c6234bb
labels: []
status: in_progress
title: 'Release workflow: build the image and publish it to Docker Hub from GitHub Actions'
updated_at: 2026-10-08T08:07:01.139455+00:00
---
## Outcome

Pushing a version tag (`v1.2.3`) to the repository builds the Sundew image for `linux/amd64` and `linux/arm64`, runs the acceptance against it, and publishes it to Docker Hub as `1.2.3`, `1.2` and `latest`. A push to a branch publishes nothing. The operator has three settings to make and a documented command to cut a release.

## Scope

In:
- `.github/workflows/release.yml`: runs on tags matching `v*.*.*` and on `workflow_dispatch` with a `dry_run` input (build and test everything, push nothing). Steps: checkout; `go vet`, `go test -race ./...`; `scripts/acceptance.sh` against the image built from the tag (linux/amd64); then a multi-platform build and push. The push step does not run unless every earlier step passed, nor on a dry run.
- Image name from the repository variable `DOCKERHUB_IMAGE` (for example `<namespace>/sundew`); login from the secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`. No account name in the file. A missing variable or secret fails early with a message naming it.
- Tags: the version without the `v`, `major.minor`, and `latest`; a pre-release tag (`v1.2.3-rc.1`) publishes only its own version, never `latest` or `major.minor`. OCI labels: source, revision, version, licence, created.
- `Dockerfile`: builds for the target platform by cross-compiling in the build stage (`--platform=$BUILDPLATFORM`, `GOOS`/`GOARCH` from `TARGETOS`/`TARGETARCH`), so the arm64 image needs no emulated compile. The version is compiled into the binary (`-ldflags -X`) and printed by `sundew --version` and at start-up; an untagged build says `dev`.
- Minimal permissions (`contents: read`); every third-party action pinned to a full commit SHA with the version in a comment; `concurrency` so two runs for one tag cannot race.
- `ci.yml`: unchanged in behaviour, but pin its actions the same way.
- `docs/releasing.md`: the three settings the operator makes once (where, what scope the Docker Hub token needs), how to cut a release (`git tag -a vX.Y.Z -m … && git push origin vX.Y.Z`), the dry run, how to check the published image (`docker run --rm <image>:<version> --version`), and what to do when a release run fails. README: the `docker run` and compose snippets use the published image name with `<version>`.
- `docs/dependencies.md`: each action used, its pinned version and the date of its last commit.

Out: publishing to any other registry; signing or SBOM attestation; a changelog generator; creating a GitHub Release page; pushing any tag or image yourself; the web UI.

## How to verify

On this machine, in the lane:
- `go vet ./...`, `go test -race ./...` exit 0; `scripts/acceptance.sh` exits 0 with the new Dockerfile.
- `docker buildx build --platform linux/amd64,linux/arm64 --build-arg VERSION=0.0.0-test .` exits 0 on the remote engine without pushing; the amd64 image runs `--version` and prints `0.0.0-test`. Remove what you built.
- A workflow linter (`actionlint`, run from its container image) exits 0 on both workflow files.
- The tag logic is proved by a table in `docs/releasing.md` and by a script or test that feeds it `v1.2.3`, `v1.2.3-rc.1`, `v0.1.0` and a non-version tag and checks the computed tags.
- You cannot run the real publish: say so in the Summary. The first real run is the operator's, as a `dry_run` dispatch, after he has made the three settings.

## Route

`design`: no. `review_code`: yes. `review_tests`: yes. e2e acceptance: no new step; the existing seven must stay green with the new Dockerfile.

## Decided and open

- Docker Hub, by GitHub Actions, on a version tag: `docs/journal/2026-10-08/decision-images-are-published-to-docker-hub-by-github-actions.md`.
- Never push a tag, an image, or anything to Docker Hub from this lane; never put a credential or account name in a file, commit, task or comment.
- Actions follow the dependency rule (a commit within six months, checked the day you add it). An action that fails it is a `decide`.
- The production UI lane (to be bound in parallel) also edits `Dockerfile` and `ci.yml`. This lane merges first; keep your `Dockerfile` change to the build-stage lines so its rebase is small.
- Go here, Docker on the remote engine by `DOCKER_HOST` and a private client config: as in the Foundation task and tasks:b099379fbbff6b1d#c22.

## Where to look

`Dockerfile`, `.github/workflows/ci.yml`, `scripts/acceptance.sh`, `cmd/sundew/main.go`, `AGENTS.md` ("Git"), `docs/dependencies.md`.