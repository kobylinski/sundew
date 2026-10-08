#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" run --rm -T client python - <<'PY'
import json, os, urllib.request, urllib.error
from html.parser import HTMLParser
base = os.environ['SUNDEW_TEST_URL']
def get(path, method='GET'):
    request = urllib.request.Request(base + path, method=method)
    with urllib.request.urlopen(request, timeout=10) as response:
        return response.status, response.headers, response.read()
def error(path, expected, method='GET'):
    try: get(path, method)
    except urllib.error.HTTPError as response:
        with response:
            assert response.code == expected, (path, response.code)
            payload = response.read()
            assert b'id="app"' not in payload and b'UI is not built' not in payload
            return response.headers, payload
    raise AssertionError('expected HTTP error for ' + path)
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
directives = {parts[0]: parts[1:] for part in csp.split(';') if (parts := part.split())}
for directive in ['script-src', 'style-src', 'connect-src']:
    assert directives[directive] == ["'self'"], (directive, directives[directive])
assert headers['X-Content-Type-Options'] == 'nosniff'
assets = Assets(); assets.feed(body.decode())
assert any(path.endswith('.js') for path in assets.paths)
assert any(path.endswith('.css') for path in assets.paths)
for path in assets.paths:
    assert path.startswith('/assets/'), path
    status, headers, content = get(path)
    assert status == 200 and content
    assert 'immutable' in headers['Cache-Control']
    assert ('javascript' in headers['Content-Type']) if path.endswith('.js') else ('text/css' in headers['Content-Type'])
    status, head, empty = get(path, 'HEAD')
    assert status == 200 and empty == b''
    assert head['Content-Type'] == headers['Content-Type']
    assert head['Cache-Control'] == headers['Cache-Control']
    assert head['X-Content-Type-Options'] == 'nosniff'
for path in ['/install', '/messages/deep-link']:
    status, headers, fallback = get(path)
    assert status == 200 and fallback == body and headers['Cache-Control'] == 'no-cache'
    assert headers['Content-Security-Policy'] == csp
    status, head, empty = get(path, 'HEAD')
    assert status == 200 and empty == b'' and head['Cache-Control'] == 'no-cache'
status, headers, api = get('/api/v1/messages')
assert status == 200 and headers.get_content_type() == 'application/json'
assert isinstance(json.loads(api)['items'], list)
assert get('/healthz')[2] == b'ok\n'
assert get('/healthz', 'HEAD')[2] == b''
for path in ['/assets/missing.js', '/api', '/api/not-a-route', '/healthz/missing', '/2010-04-01/not-a-route']:
    error(path, 404)
headers, payload = error('/healthz', 405, 'POST')
assert {'GET', 'HEAD'} <= {part.strip() for part in headers['Allow'].split(',')}
headers, payload = error('/api/v1/messages/unknown', 404)
assert headers.get_content_type() == 'application/json'
assert json.loads(payload)['error']['code'] == 'not_found'
for path, status, code in [
    ('/2010-04-01/Accounts/ACreview/Messages.json', 405, 20004),
    ('/2010-04-01/Accounts/ACreview/Messages/SMunknown.json', 404, 20404),
]:
    headers, payload = error(path, status)
    assert headers.get_content_type() == 'application/json'
    assert json.loads(payload)['code'] == code
print('Embedded UI/assets/HEAD/cache/CSP/deep links and reserved API/provider/health routes: passed')
PY
