#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
run_dir=$(mktemp -d "${TMPDIR:-/tmp}/sundew-acceptance.XXXXXX")
export SUNDEW_TEST_PROJECT="sundew-$(date +%s)-$$-${RANDOM}"
export SUNDEW_TEST_NETWORK="${SUNDEW_TEST_PROJECT}_default"
export SUNDEW_TEST_IMAGE="sundew-acceptance:${SUNDEW_TEST_PROJECT}"
export SUNDEW_TEST_CLIENT_IMAGE="python:3.14-alpine"
export SUNDEW_TEST_COMPOSE="$run_dir/compose.yml"
client_was_present=false
if docker image inspect "$SUNDEW_TEST_CLIENT_IMAGE" >/dev/null 2>&1; then
  client_was_present=true
fi
cleanup() {
  result=$?
  trap - EXIT
  if (( result != 0 )); then
    docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" logs >&2 || true
  fi
  containers=$(docker ps -aq --filter "label=com.docker.compose.project=$SUNDEW_TEST_PROJECT") || result=1
  if [[ -n ${containers:-} ]]; then
    while IFS= read -r container; do
      docker rm -f "$container" >/dev/null || result=1
    done <<< "$containers"
  fi
  docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" down --volumes --remove-orphans >&2 || result=1
  docker image rm "$SUNDEW_TEST_IMAGE" >/dev/null 2>&1 || true
  if [[ $client_was_present == false ]]; then
    docker image rm "$SUNDEW_TEST_CLIENT_IMAGE" >/dev/null 2>&1 || true
  fi
  rm -rf "$run_dir"
  exit "$result"
}
cat > "$SUNDEW_TEST_COMPOSE" <<'YAML'
services:
  sundew:
    image: ${SUNDEW_TEST_IMAGE}
  client:
    image: ${SUNDEW_TEST_CLIENT_IMAGE}
    environment:
      SUNDEW_TEST_URL: ${SUNDEW_TEST_URL:-http://sundew:8025}
YAML
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

docker build -t "$SUNDEW_TEST_IMAGE" .
size=$(docker image inspect --format '{{.Size}}' "$SUNDEW_TEST_IMAGE")
if (( size >= 30000000 )); then
  echo "Image is $size bytes; expected under 30 MB" >&2
  exit 1
fi
echo "Image size: $size bytes"
docker pull "$SUNDEW_TEST_CLIENT_IMAGE"
docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" create sundew

# A step inherits the project, Compose file, image names and URL. Each step's
# client must run in this project, so a remote DOCKER_HOST works unchanged.
export SUNDEW_TEST_URL="http://sundew:8025"
steps=0
for step in scripts/acceptance/*.sh; do
  [[ -f $step && -x $step ]] || continue
  echo "Acceptance: $step"
  "$step"
  steps=$((steps + 1))
done
if (( steps == 0 )); then
  echo "No executable acceptance steps found" >&2
  exit 1
fi
