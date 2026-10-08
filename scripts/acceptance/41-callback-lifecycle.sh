#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_IMAGE:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_NETWORK:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_CLIENT_IMAGE:?Run scripts/acceptance.sh}"
receiver="${SUNDEW_TEST_PROJECT}-lifecycle-receiver"
application="${SUNDEW_TEST_PROJECT}-lifecycle-app"
callback_delay_seconds=1
cleanup() {
  result=$?
  trap - EXIT
  if (( result != 0 )); then
    docker logs "$receiver" >&2 || true
    docker logs "$application" >&2 || true
  fi
  docker rm -f "$receiver" "$application" >/dev/null 2>&1 || true
  exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

receiver_source=$(cat <<'PY'
import json, threading, urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
records = {}
condition = threading.Condition()
class Receiver(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass
    def do_POST(self):
        payload = self.rfile.read(int(self.headers.get('Content-Length', '0')))
        form = urllib.parse.parse_qs(payload.decode())
        with condition:
            records.setdefault(self.path, []).append(form)
            condition.notify_all()
        self.send_response(503 if self.path == '/reject' else 204)
        self.end_headers()
    def do_GET(self):
        query = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query)
        target = query.get('target', [''])[0]
        count = int(query.get('count', ['0'])[0])
        timeout = float(query.get('timeout', ['8'])[0])
        with condition:
            condition.wait_for(lambda: len(records.get(target, [])) >= count, timeout=timeout)
            payload = json.dumps(records.get(target, [])).encode()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)
ThreadingHTTPServer(('0.0.0.0', 8080), Receiver).serve_forever()
PY
)
docker run -d --name "$receiver" --network "$SUNDEW_TEST_NETWORK" \
  --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" \
  "$SUNDEW_TEST_CLIENT_IMAGE" python -u -c "$receiver_source" >/dev/null

docker run -d --name "$application" --network "$SUNDEW_TEST_NETWORK" \
  --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" \
  -e "SUNDEW_CALLBACK_DELAY=${callback_delay_seconds}s" "$SUNDEW_TEST_IMAGE" >/dev/null

docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" run --rm -T client python - "$receiver" "$application" "$callback_delay_seconds" <<'PY'
import base64, json, sys, time, urllib.error, urllib.parse, urllib.request
receiver = 'http://' + sys.argv[1] + ':8080'
base = 'http://' + sys.argv[2] + ':8025'
callback_delay = float(sys.argv[3])
# Two transitions would arrive after one and two delays; half a delay adds margin.
absence_window = 2.5 * callback_delay
auth = 'Basic ' + base64.b64encode(b'ACreview:invented-review-token').decode()
def call(host, method, path, form=None):
    data = urllib.parse.urlencode(form).encode() if form is not None else None
    req = urllib.request.Request(host + path, data=data, method=method,
          headers={'Content-Type': 'application/x-www-form-urlencoded', 'Authorization': auth})
    with urllib.request.urlopen(req, timeout=15) as response:
        payload = response.read()
        return json.loads(payload) if payload and response.headers.get_content_type() == 'application/json' else payload

def ready(host, path):
    deadline = time.monotonic() + 15
    while True:
        try:
            call(host, 'GET', path)
            return
        except (OSError, urllib.error.URLError):
            if time.monotonic() >= deadline:
                raise
            time.sleep(0.02)

def records(target, count, timeout=8):
    query = urllib.parse.urlencode({'target': target, 'count': count, 'timeout': timeout})
    return call(receiver, 'GET', '/records?' + query)

def send(target):
    return call(base, 'POST', '/2010-04-01/Accounts/ACreview/Messages.json',
                {'From': '+15551000001', 'To': '+15551000002', 'Body': 'callback lifecycle',
                 'StatusCallback': receiver + target})

ready(receiver, '/records')
ready(base, '/healthz')
# The receiver waits on its recorded state, including the bounded absence checks.
for operation in ['delete', 'reset']:
    target = '/' + operation
    message = send(target)
    assert [row['MessageStatus'][0] for row in records(target, 1)] == ['queued']
    caught = call(base, 'GET', '/api/v1/messages/latest')
    assert caught['provider_id'] == message['sid']
    if operation == 'delete':
        call(base, 'DELETE', '/api/v1/messages/' + caught['id'])
    else:
        call(base, 'POST', '/api/v1/reset')
    assert [row['MessageStatus'][0] for row in records(target, 2, absence_window)] == ['queued']
    assert call(base, 'GET', '/api/v1/messages')['items'] == []

message = send('/reject')
rejected = records('/reject', 3)
assert [row['MessageStatus'][0] for row in rejected] == ['queued', 'sent', 'delivered']
assert all(row['MessageSid'] == [message['sid']] for row in rejected)
assert call(base, 'GET', '/api/v1/messages/latest')['status'] == 'delivered'
print('Callback delete/reset cancellation and HTTP-rejection continuation passed')
PY
