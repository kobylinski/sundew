#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_IMAGE:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_NETWORK:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_CLIENT_IMAGE:?Run scripts/acceptance.sh}"
receiver="${SUNDEW_TEST_PROJECT}-receiver"
application="${SUNDEW_TEST_PROJECT}-callback-app"
failed_application="${SUNDEW_TEST_PROJECT}-failed-app"
cleanup() {
  result=$?
  trap - EXIT
  if (( result != 0 )); then
    docker logs "$receiver" >&2 || true
    docker logs "$application" >&2 || true
    docker logs "$failed_application" >&2 || true
  fi
  docker rm -f "$receiver" "$application" "$failed_application" >/dev/null 2>&1 || true
  exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Source travels as a command argument: no bind mounts or host ports, including remote engines.
receiver_source=$(cat <<'PY'
import json, threading, time, urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

records = []
condition = threading.Condition()
class Receiver(BaseHTTPRequestHandler):
    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        with condition:
            records.append({"path": self.path, "form": urllib.parse.parse_qs(body.decode()), "at": time.monotonic()})
            condition.notify_all()
        self.send_response(204)
        self.end_headers()
    def do_GET(self):
        query = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query)
        count = int(query.get("count", ["0"])[0])
        with condition:
            condition.wait_for(lambda: len(records) >= count, timeout=12)
            data = json.dumps(records).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
ThreadingHTTPServer(("0.0.0.0", 8080), Receiver).serve_forever()
PY
)
docker run -d --name "$receiver" --network "$SUNDEW_TEST_NETWORK" \
  --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" \
  "$SUNDEW_TEST_CLIENT_IMAGE" python -u -c "$receiver_source" >/dev/null
for outcome in delivered failed; do
  container="$application"
  [[ $outcome == failed ]] && container="$failed_application"
  docker run -d --name "$container" --network "$SUNDEW_TEST_NETWORK" \
    --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" \
    -e SUNDEW_CALLBACK_DELAY=100ms -e "SUNDEW_CALLBACK_OUTCOME=$outcome" \
    -e "SUNDEW_INBOUND_URL=http://$receiver:8080/inbound" \
    "$SUNDEW_TEST_IMAGE" >/dev/null
 done

docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" run --rm -T client python - "$receiver" "$application" "$failed_application" <<'PY'
import base64, json, sys, time, urllib.error, urllib.parse, urllib.request

receiver, delivered, failed = sys.argv[1:]
def request(base, method, path, data=None, content_type="application/json"):
    headers = {"Content-Type": content_type, "Authorization": "Basic " + base64.b64encode(b"ACacceptance:invented-token").decode()}
    req = urllib.request.Request(base + path, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=15) as res:
        payload = res.read()
        return res.status, json.loads(payload) if payload and res.headers.get("Content-Type", "").startswith("application/json") else payload

def ready(base, path):
    deadline = time.monotonic() + 15
    while True:
        try:
            return request(base, "GET", path)
        except (OSError, urllib.error.URLError):
            if time.monotonic() >= deadline:
                raise
            time.sleep(0.02)

receiver_url = "http://" + receiver + ":8080"
ready(receiver_url, "/records")
expected_count = 0
for host, final in [(delivered, "delivered"), (failed, "failed")]:
    base = "http://" + host + ":8025"
    ready(base, "/healthz")
    target = "/" + final
    form = urllib.parse.urlencode({"From": "+15550000001", "To": "+15550000002", "Body": "callback acceptance", "StatusCallback": receiver_url + target}).encode()
    status, sent = request(base, "POST", "/2010-04-01/Accounts/ACacceptance/Messages.json", form, "application/x-www-form-urlencoded")
    assert status == 201
    expected_count += 3
    records = request(receiver_url, "GET", "/records?count=" + str(expected_count))[1]
    events = [r for r in records if r["path"] == target]
    assert [r["form"]["MessageStatus"][0] for r in events] == ["queued", "sent", final], events
    assert all(r["form"]["MessageSid"] == [sent["sid"]] for r in events)
    assert all(r["form"]["AccountSid"] == ["ACacceptance"] for r in events)
    assert all(events[i + 1]["at"] - events[i]["at"] >= 0.09 for i in range(2)), events
    if final == "failed":
        assert events[-1]["form"]["ErrorCode"][0], events[-1]
    caught = request(base, "GET", "/api/v1/messages/latest")[1]
    assert caught["status"] == final, caught

base = "http://" + delivered + ":8025"
for target in [None, receiver_url + "/override"]:
    payload = {"from": "+15550000002", "to": "+15550000001", "body": "STOP", "account": "ACacceptance"}
    if target:
        payload["url"] = target
    status, inbound = request(base, "POST", "/api/v1/inbound", json.dumps(payload).encode())
    assert status == 201 and inbound["response_status"] == 204, inbound
    assert inbound["message"]["direction"] == "inbound" and inbound["message"]["status"] == "received"
    expected_count += 1
    records = request(receiver_url, "GET", "/records?count=" + str(expected_count))[1]
    record = records[-1]
    assert record["path"] == ("/override" if target else "/inbound")
    assert record["form"]["Body"] == ["STOP"] and record["form"]["From"] == ["+15550000002"]
print("Callbacks queued/sent/delivered, failed error, delay, and inbound default/override passed")
PY
