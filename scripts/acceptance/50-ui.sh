#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" run --rm -T client python - <<'PY'
import json, os, urllib.request, urllib.error
from html.parser import HTMLParser
base = os.environ['SUNDEW_TEST_URL']
def get(path):
    with urllib.request.urlopen(base + path, timeout=10) as response:
        return response.status, response.headers, response.read()
class Assets(HTMLParser):
    paths = []
    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        path = attrs.get('src') if tag == 'script' else attrs.get('href') if tag == 'link' else None
        if path: self.paths.append(path)
status, headers, body = get('/')
assert status == 200 and b'id="app"' in body and b'UI is not built' not in body
assert headers['Cache-Control'] == 'no-cache'
csp = headers['Content-Security-Policy']
assert "script-src 'self'" in csp and "style-src 'self'" in csp and "connect-src 'self'" in csp
assert 'unsafe-inline' not in csp and 'unsafe-eval' not in csp
assets = Assets(); assets.feed(body.decode())
assert any(path.endswith('.js') for path in assets.paths)
assert any(path.endswith('.css') for path in assets.paths)
for path in assets.paths:
    assert path.startswith('/assets/'), path
    status, headers, content = get(path)
    assert status == 200 and content
    assert 'immutable' in headers['Cache-Control']
    assert ('javascript' in headers['Content-Type']) if path.endswith('.js') else ('text/css' in headers['Content-Type'])
for path in ['/install', '/messages/deep-link']:
    status, headers, fallback = get(path)
    assert status == 200 and fallback == body and headers['Cache-Control'] == 'no-cache'
status, headers, api = get('/api/v1/messages')
assert status == 200 and headers.get_content_type() == 'application/json'
assert isinstance(json.loads(api)['items'], list)
assert get('/healthz')[2] == b'ok\n'
try: get('/assets/missing.js')
except urllib.error.HTTPError as error: assert error.code == 404
else: raise AssertionError('missing asset returned SPA HTML')
print('Embedded production UI/assets/deep links/CSP/API/health: passed')
PY
