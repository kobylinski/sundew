#!/usr/bin/env bash
# Print Docker tags, one per line; the first is also the embedded version.
set -euo pipefail
tag=${1:-}
number='(0|[1-9][0-9]*)'
identifier='(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
if [[ ! $tag =~ ^v${number}\.${number}\.${number}(-${identifier}(\.${identifier})*)?$ ]]; then
  echo 'Expected a version tag such as v1.2.3 or v1.2.3-rc.1 (no build metadata)' >&2
  exit 1
fi
version=${tag#v}
if (( ${#version} > 128 )); then
  echo 'Version exceeds the Docker tag limit of 128 characters' >&2
  exit 1
fi
printf '%s\n' "$version"
if [[ $version != *-* ]]; then
  printf '%s\nlatest\n' "${version%.*}"
fi
