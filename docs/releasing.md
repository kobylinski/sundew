# Releasing Sundew

Only a pushed version tag publishes an image to Docker Hub. Branch pushes and pull
requests run CI; manual release dispatches build and test without publishing.
Images support `linux/amd64` and `linux/arm64` and contain one static binary.

## Set up once

Use the operator-selected Docker Hub repository `kobylinski/sundew`. In the GitHub repository,
open **Settings → Secrets and variables → Actions** and set these three values:

| Tab | Name | Value |
| --- | --- | --- |
| Variables | `DOCKERHUB_IMAGE` | `kobylinski/sundew`, without a registry prefix or tag |
| Secrets | `DOCKERHUB_USERNAME` | Docker Hub login with write access to that repository |
| Secrets | `DOCKERHUB_TOKEN` | A Docker Hub personal access token for that login with **Read & Write** access; Delete is unnecessary |

Create the token in Docker account settings → Personal access tokens. Keep its
value in GitHub Secrets. [Docker's token guide](https://docs.docker.com/security/access-tokens/personal-access-tokens/)
describes permissions and expiry. Both release and dry-run jobs fail early if any
setting is missing, naming the missing setting without printing its value.

After the workflow is on the default branch and the settings exist, run the first
verification yourself: **Actions → Release image → Run workflow**, select `main`,
and leave `dry_run` checked. This runs vet, race tests, tag-logic tests, all eight
container acceptance steps, and the two-platform build. It never logs in or pushes.
For a tag's exact candidate, select that tag instead of `main`. A branch dry run
embeds `dev`; a tag dry run embeds the validated version without `v`.
Manual dispatch with `dry_run=false` fails; publication always requires a tag push.

## Cut a release

From the intended, tested integration commit, replace `X.Y.Z` with its version:

```sh
git tag -a vX.Y.Z -m 'Release X.Y.Z'
git push origin vX.Y.Z
```

The release job checks out the tag, runs `go vet ./...`, `go test -race ./...`,
`scripts/test-release-tags.sh`, and `scripts/acceptance.sh` against a versioned
`linux/amd64` build from that checkout. It then builds both platforms before
logging in and publishing. Builds cross-compile Go from the builder's native
platform; no arm64 emulation is required. Runs for the same ref are serialized.
The workflow token has only `contents: read`; Docker Hub publication uses the token
you configured above. See [GitHub's action pinning guidance](https://docs.github.com/en/actions/reference/security/secure-use)
and [Docker's cross-compilation guide](https://docs.docker.com/build/building/multi-platform/).

| Pushed tag | Docker tags | Binary version |
| --- | --- | --- |
| `v1.2.3` | `1.2.3`, `1.2`, `latest` | `1.2.3` |
| `v1.2.3-rc.1` | `1.2.3-rc.1` | `1.2.3-rc.1` |
| `v0.1.0` | `0.1.0`, `0.1`, `latest` | `0.1.0` |
| `main` or `banana` | None: no release push trigger | `dev` for a manual branch dry run |
| `v1.2.bad` | None: validation fails before image build | None |

Versions follow SemVer's numeric components and prerelease identifiers. Leading
zeroes in numeric identifiers, build metadata (`+build`), and versions longer than
Docker's 128-character tag limit are rejected. Build metadata is excluded because
`+` cannot appear in a Docker tag. Prereleases never update `latest` or the minor
line. Stable releases update their minor line and `latest`, including a rerun or
an older version released later; choose release order accordingly.

Each published image carries OCI labels `org.opencontainers.image.source`,
`revision`, `version`, `licenses` (`MIT`) and `created` (UTC build time).
The version is compiled into the binary and included in the listening log.

Check a published image, replacing `<version>` with its release version:

```sh
docker run --rm kobylinski/sundew:<version> --version
docker run --rm -p 8025:8025 kobylinski/sundew:<version>
```

`--version` prints only the version and exits before reading configuration or
opening a listener. Builds without `--build-arg VERSION=…` print `dev`.

## If a run fails

Open the failed job in Actions. For missing settings or login/permission errors,
correct the named setting or token access and rerun the failed tag job. For vet,
test, acceptance or image-build failures, fix the code on the integration branch,
verify it, and create a new version tag; do not move an existing release tag.
Publication cannot begin before all verification and both platform builds pass.
If the registry push itself fails, some tags or platform blobs may already exist;
rerun the same unchanged tag after resolving the registry problem. Check the
version image and all intended aliases when it succeeds. A dry run verifies builds
and tests but does not validate registry credentials or push permissions.

The implementation was verified locally and on the approved test engine without
publishing. The operator performs the first GitHub Actions dry run and actual
release after configuring the three settings.
