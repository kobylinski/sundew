#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
compose() { docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" "$@"; }
compose start sundew

# Prove HTTP readiness separately from startup timing: Docker DNS and the
# client container's startup must not be charged to Sundew's process startup.
compose run --rm -T client python -c '
import os, time, urllib.request
deadline = time.monotonic() + 20
while time.monotonic() < deadline:
    try:
        with urllib.request.urlopen(os.environ["SUNDEW_TEST_URL"] + "/healthz", timeout=0.5) as response:
            assert response.status == 200
            assert response.read() == b"ok\n"
            print("GET /healthz 200")
            break
    except (OSError, AssertionError):
        time.sleep(0.05)
else:
    raise SystemExit("health endpoint never became ready")
'
container=$(compose ps -q sundew)
started=$(docker inspect --format '{{.State.StartedAt}}' "$container")
listening=$(docker logs --timestamps "$container" 2>&1 | awk '/Sundew listening on/ {print $1; exit}')
[[ -n $listening ]] || { echo "Missing listening log timestamp" >&2; exit 1; }
compose run --rm -T client python -c '
import calendar, datetime, re, sys
def nanoseconds(stamp):
    match = re.fullmatch(r"(.*?)(?:\.(\d+))?Z", stamp)
    if not match:
        raise SystemExit("unexpected Docker timestamp: " + stamp)
    seconds = calendar.timegm(datetime.datetime.fromisoformat(match[1]).timetuple())
    return seconds * 10**9 + int((match[2] or "").ljust(9, "0")[:9])
elapsed = (nanoseconds(sys.argv[2]) - nanoseconds(sys.argv[1])) / 10**9
print(f"Process startup {elapsed:.3f}s")
if not 0 <= elapsed < 1:
    raise SystemExit("startup must be under one second")
' "$started" "$listening"
