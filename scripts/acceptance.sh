#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
run_dir=$(mktemp -d "${TMPDIR:-/tmp}/sundew-acceptance.XXXXXX")
lane="${SUNDEW_TEST_LANE:-b099379f}"
if [[ ! $lane =~ ^[a-z0-9]+$ ]]; then
  echo "SUNDEW_TEST_LANE must contain only lowercase letters and digits" >&2
  rm -rf "$run_dir"
  exit 1
fi
export SUNDEW_TEST_PROJECT="sundew-${lane}-$(date +%s)-$$-${RANDOM}"
export SUNDEW_TEST_NETWORK="${SUNDEW_TEST_PROJECT}_default"
export SUNDEW_TEST_IMAGE="sundew-acceptance:${SUNDEW_TEST_PROJECT}"
export SUNDEW_TEST_CLIENT_IMAGE="python:3.14-alpine"
export SUNDEW_TEST_COMPOSE="$run_dir/compose.yml"
network_created=false
client_was_present=false
if docker image inspect "$SUNDEW_TEST_CLIENT_IMAGE" >/dev/null 2>&1; then
  client_was_present=true
fi
cleanup() {
  result=$?
  trap - EXIT
  if (( result != 0 )) && [[ -f $SUNDEW_TEST_COMPOSE ]]; then
    docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" logs >&2 || true
  fi
  containers=$(docker ps -aq --filter "label=com.docker.compose.project=$SUNDEW_TEST_PROJECT") || result=1
  if [[ -n ${containers:-} ]]; then
    while IFS= read -r container; do
      docker rm -f "$container" >/dev/null || result=1
    done <<< "$containers"
  fi
  if [[ -f $SUNDEW_TEST_COMPOSE ]]; then
    docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" down --volumes --remove-orphans >&2 || result=1
  fi
  if [[ $network_created == true ]]; then
    docker network rm "$SUNDEW_TEST_NETWORK" >/dev/null || result=1
  fi
  if docker image inspect "$SUNDEW_TEST_IMAGE" >/dev/null 2>&1; then
    docker image rm "$SUNDEW_TEST_IMAGE" >/dev/null || result=1
  fi
  if [[ $client_was_present == false ]]; then
    if docker image inspect "$SUNDEW_TEST_CLIENT_IMAGE" >/dev/null 2>&1; then
      docker image rm "$SUNDEW_TEST_CLIENT_IMAGE" >/dev/null || result=1
    fi
  fi
  rm -rf "$run_dir"
  exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Reserve the application's address before starting the probe. Looking up a
# not-yet-running service name can block in DNS and distort the startup timing.
# Docker chooses the isolated subnet; nothing assumes a host-side address.
docker network create --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" "$SUNDEW_TEST_NETWORK" >/dev/null
network_created=true
gateway=$(docker network inspect --format '{{(index .IPAM.Config 0).Gateway}}' "$SUNDEW_TEST_NETWORK")
IFS=. read -r ip_a ip_b ip_c ip_d <<< "$gateway"
ip_number=$(( (ip_a << 24) + (ip_b << 16) + (ip_c << 8) + ip_d + 2 ))
export SUNDEW_TEST_SERVER_IP
SUNDEW_TEST_SERVER_IP=$(printf '%d.%d.%d.%d' "$(( (ip_number >> 24) & 255 ))" "$(( (ip_number >> 16) & 255 ))" "$(( (ip_number >> 8) & 255 ))" "$(( ip_number & 255 ))")
cat > "$SUNDEW_TEST_COMPOSE" <<'YAML'
services:
  sundew:
    image: ${SUNDEW_TEST_IMAGE}
    networks:
      default:
        ipv4_address: ${SUNDEW_TEST_SERVER_IP}
  client:
    image: ${SUNDEW_TEST_CLIENT_IMAGE}
    environment:
      SUNDEW_TEST_URL: ${SUNDEW_TEST_URL:-http://sundew:8025}
networks:
  default:
    name: ${SUNDEW_TEST_NETWORK}
    external: true
YAML

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
