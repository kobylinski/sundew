#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
check() {
  actual=$(scripts/release-tags.sh "$1")
  [[ $actual == "$2" ]] || { echo "Wrong tags for $1" >&2; exit 1; }
}
check v1.2.3 $'1.2.3\n1.2\nlatest'
check v1.2.3-rc.1 '1.2.3-rc.1'
check v0.1.0 $'0.1.0\n0.1\nlatest'
check v0.0.0-alpha.0 '0.0.0-alpha.0'
check v1.2.3-0a '1.2.3-0a'
for invalid in main v1.2 1.2.3 v01.2.3 v1.2.3-01 v1.2.3-rc..1 v1.2.3+build v1.2.3- ''; do
  if scripts/release-tags.sh "$invalid" >/dev/null 2>&1; then
    echo "Accepted invalid tag: $invalid" >&2
    exit 1
  fi
done
echo 'Release tag cases passed'
