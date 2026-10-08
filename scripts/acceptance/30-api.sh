#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" run --rm -T client python - <<'PY'
import base64, json, os, urllib.parse, urllib.request

base = os.environ["SUNDEW_TEST_URL"]
def call(method, path, data=None, content_type="application/json"):
    headers = {"Content-Type": content_type}
    if path.startswith("/2010-"):
        headers["Authorization"] = "Basic " + base64.b64encode(b"ACacceptance:invented-token").decode()
    request = urllib.request.Request(base + path, data=data, headers=headers, method=method)
    with urllib.request.urlopen(request, timeout=10) as response:
        payload = response.read()
        return response.status, json.loads(payload) if payload else None

assert call("POST", "/api/v1/reset")[0] == 204
stream = urllib.request.urlopen(base + "/api/v1/messages/stream?to=%2B15550000002", timeout=10)
assert stream.headers["Content-Type"] == "text/event-stream"
# Headers are flushed only after the subscription is established.
assert stream.readline().startswith(b": connected")
form = urllib.parse.urlencode({"From": "+15550000001", "To": "+15550000002", "Body": "acceptance code 123456"}).encode()
assert call("POST", "/2010-04-01/Accounts/ACacceptance/Messages.json", form, "application/x-www-form-urlencoded")[0] == 201
while True:
    line = stream.readline()
    if not line:
        raise AssertionError("stream closed before a created event")
    if line.startswith(b"event:"):
        assert line.strip() == b"event: message.created", line
    if line.startswith(b"data:"):
        event = json.loads(line[5:])
        assert event["type"] == "message.created"
        assert event["message"]["body"] == "acceptance code 123456"
        assert event["message"]["to"] == "+15550000002"
        break
stream.close()
status, latest = call("GET", "/api/v1/messages/latest?to=%2B15550000002")
assert status == 200 and latest["body"] == "acceptance code 123456"
status, page = call("GET", "/api/v1/messages?provider=twilio&body=CODE&q=123456&limit=1")
assert status == 200 and [m["id"] for m in page["items"]] == [latest["id"]]
assert "next_cursor" in page
assert call("GET", "/api/v1/messages/" + latest["id"])[1]["id"] == latest["id"]
assert call("GET", "/api/v1/openapi.json")[1]["openapi"] == "3.1.0"
assert call("POST", "/api/v1/reset")[0] == 204
assert call("GET", "/api/v1/messages")[1] == {"items": [], "next_cursor": ""}
print("API list/latest/get/SSE/reset/OpenAPI passed")
PY
