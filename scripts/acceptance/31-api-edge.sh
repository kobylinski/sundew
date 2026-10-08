#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" run --rm -T client python - <<'PY'
import base64, json, os, urllib.error, urllib.parse, urllib.request

base = os.environ['SUNDEW_TEST_URL']
auth = 'Basic ' + base64.b64encode(b'ACreview:invented-review-token').decode()
def call(method, path, data=None, headers=None):
    req = urllib.request.Request(base + path, data=data, headers=headers or {}, method=method)
    try:
        response = urllib.request.urlopen(req, timeout=10)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        body = response.read()
        assert not body or response.headers.get_content_type() == 'application/json'
        return response.status, json.loads(body) if body else None

def send(to, body):
    form = urllib.parse.urlencode({'From': '+15551000001', 'To': to, 'Body': body}).encode()
    status, message = call('POST', '/2010-04-01/Accounts/ACreview/Messages.json', form,
                           {'Content-Type': 'application/x-www-form-urlencoded', 'Authorization': auth,
                            'Cookie': 'invented-review-cookie'})
    assert status == 201
    return message

def query(path, **params):
    return call('GET', path + '?' + urllib.parse.urlencode(params))

def error(method, path, status, data=None):
    actual, body = call(method, path, data, {'Content-Type': 'application/json'})
    assert actual == status and body['error']['code'] and body['error']['message']

assert call('POST', '/api/v1/reset')[0] == 204
sent = [send('+15551000002', 'older Code'), send('+15551000003', 'excluded'),
        send('+15551000002', 'Newest CODE')]
status, latest = query('/api/v1/messages/latest', to='+15551000002', q='newest')
assert status == 200 and latest['provider_id'] == sent[-1]['sid']
assert latest['exchange']['header']['Authorization'] == [auth]
assert latest['exchange']['header']['Cookie'] == ['invented-review-cookie']
filters = dict(to=latest['to'], **{'from': latest['from']}, account='ACreview', provider='twilio',
               body='code', q='NEWEST', since=latest['created_at'])
assert query('/api/v1/messages', **filters)[1]['items'][0]['id'] == latest['id']
assert query('/api/v1/messages', q='newest', to='+15551000003')[1]['items'] == []
ids, cursor = [], ''
while True:
    status, page = query('/api/v1/messages', limit=1, cursor=cursor)
    assert status == 200 and len(page['items']) == 1
    ids.append(page['items'][0]['provider_id'])
    cursor = page['next_cursor']
    if not cursor:
        break
    assert len(ids) < 4
assert ids == [message['sid'] for message in reversed(sent)]
error('GET', '/api/v1/messages?limit=0', 400)
error('GET', '/api/v1/messages?cursor=malformed', 400)
error('GET', '/api/v1/messages/latest?since=bad', 400)
error('GET', '/api/v1/messages/missing', 404)
error('DELETE', '/api/v1/messages/missing', 404)
error('PUT', '/api/v1/messages', 405)
error('POST', '/api/v1/inbound', 400, b'{"provider":"unknown"}')

stream = urllib.request.urlopen(base + '/api/v1/messages/stream?to=%2B15551000002&q=stream', timeout=10)
assert stream.readline().startswith(b': connected')
def event(kind):
    while True:
        line = stream.readline()
        assert line, 'stream ended before expected event'
        if line.startswith(b'data:'):
            value = json.loads(line[5:])
            assert value['type'] == kind
            return value
send('+15551000003', 'stream excluded')
send('+15551000002', 'stream included')
created = event('message.created')['message']
assert created['body'] == 'stream included'
assert call('DELETE', '/api/v1/messages/' + created['id'])[0] == 204
assert event('message.deleted')['message']['id'] == created['id']
error('GET', '/api/v1/messages/' + created['id'], 404)
assert call('DELETE', '/api/v1/messages')[0] == 204
assert event('store.reset')['type'] == 'store.reset'
stream.close()
assert call('GET', '/api/v1/messages')[1] == {'items': [], 'next_cursor': ''}
error('GET', '/api/v1/messages/latest', 404)
print('API combined filters/paging/raw capture/errors/SSE delete/reset passed')
PY
