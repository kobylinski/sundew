#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
compose() { docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" "$@"; }
probe="${SUNDEW_TEST_PROJECT}-health"

# Arm the probe BEFORE the application starts. Measure on the Docker host's
# clock against State.StartedAt, excluding client startup and CLI/SSH latency.
compose run -d -T --name "$probe" client python -u -c '
import os, time, urllib.request
print("armed", flush=True)
deadline = time.monotonic() + 30
while time.monotonic() < deadline:
    try:
        with urllib.request.urlopen(os.environ.get("SUNDEW_TEST_URL", "http://sundew:8025") + "/healthz", timeout=0.2) as response:
            assert response.status == 200
            assert response.read() == b"ok\n"
            print(time.time_ns(), flush=True)
            break
    except (OSError, AssertionError):
        time.sleep(0.01)
else:
    raise SystemExit("health endpoint never became ready")
'
armed=false
for (( attempt=0; attempt<100; attempt++ )); do
  case "$(docker logs "$probe" 2>&1)" in
    *armed*) armed=true; break ;;
  esac
  sleep 0.1
done
[[ $armed == true ]] || { echo "Health probe did not arm" >&2; exit 1; }
compose start sundew
[[ $(docker wait "$probe") == 0 ]] || { docker logs "$probe" >&2; exit 1; }
ready_ns=$(docker logs "$probe" | tail -n 1)
container=$(compose ps -q sundew)
started=$(docker inspect --format '{{.State.StartedAt}}' "$container")
compose run --rm -T client python -c '
import calendar, datetime, re, sys
stamp, ready = sys.argv[1], int(sys.argv[2])
match = re.fullmatch(r"(.*?)(?:\.(\d+))?Z", stamp)
if not match:
    raise SystemExit("unexpected container start timestamp: " + stamp)
seconds = calendar.timegm(datetime.datetime.fromisoformat(match[1]).timetuple())
started = seconds * 10**9 + int((match[2] or "").ljust(9, "0")[:9])
elapsed = (ready - started) / 10**9
print(f"GET /healthz 200; startup {elapsed:.3f}s")
if not 0 <= elapsed < 1:
    raise SystemExit("startup must be under one second")
' "$started" "$ready_ns"
docker rm "$probe" >/dev/null
