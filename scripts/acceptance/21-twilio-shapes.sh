#!/usr/bin/env bash
set -euo pipefail
: "${SUNDEW_TEST_PROJECT:?Run scripts/acceptance.sh}"
: "${SUNDEW_TEST_COMPOSE:?Run scripts/acceptance.sh}"
docker compose -p "$SUNDEW_TEST_PROJECT" -f "$SUNDEW_TEST_COMPOSE" run --rm -T client python - <<'PY'
import base64, json, os, urllib.error, urllib.parse, urllib.request

base = os.environ['SUNDEW_TEST_URL']
account = 'ACreview'
path = '/2010-04-01/Accounts/' + account + '/Messages.json'
def request(method, url, form=None, authorization=None):
    headers = {'Content-Type': 'application/x-www-form-urlencoded'}
    if authorization is not None:
        headers['Authorization'] = authorization
    data = urllib.parse.urlencode(form, doseq=True).encode() if form is not None else None
    req = urllib.request.Request(base + url, data=data, headers=headers, method=method)
    try:
        response = urllib.request.urlopen(req, timeout=10)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        assert response.headers.get_content_type() == 'application/json'
        return response.status, json.load(response), response.headers

def send(body, auth=None, **options):
    form = {'To': '+15551000002', 'From': '+15551000001', 'Body': body, **options}
    status, message, _ = request('POST', path, form, auth)
    assert status == 201
    return message

# Exercise actual form decoding and response serialization at encoding boundaries.
for body, expected in [('a'*160, 1), ('a'*161, 2), ('a'*306, 2), ('a'*307, 3),
                       ('界'*70, 1), ('界'*71, 2), ('^'*80, 1), ('^'*81, 2), ('😀'*36, 2)]:
    message = send(body)
    assert message['body'] == body and message['num_segments'] == str(expected)
    assert message['account_sid'] == account
    status, fetched, _ = request('GET', '/2010-04-01/Accounts/' + account + '/Messages/' + message['sid'] + '.json')
    assert status == 200 and fetched['sid'] == message['sid']
    assert fetched['num_segments'] == str(expected)

for auth, expected_account in [(None, account), ('Bearer invented-review', account),
                               ('Basic malformed', account),
                               ('Basic ' + base64.b64encode(b'ACcredential:invented-review-token').decode(), 'ACcredential')]:
    assert send('permissive auth', auth)['account_sid'] == expected_account

message = send('', MediaUrl=['https://media.invalid/one', 'https://media.invalid/two'])
assert message['num_media'] == '2' and message['num_segments'] == '0'
status, missing, _ = request('GET', '/2010-04-01/Accounts/' + account + '/Messages/SMmissing.json')
assert status == 404 and missing['code'] == 20404 and missing['status'] == 404
for method, url, allowed in [('PUT', path, 'POST'),
                             ('POST', '/2010-04-01/Accounts/' + account + '/Messages/SMmissing.json', 'GET')]:
    status, error, headers = request(method, url)
    assert status == 405 and error['status'] == 405
    assert isinstance(error['code'], int) and error['message'] and error['more_info']
    assert allowed in headers.get('Allow', '')
print('Twilio encoding/media/permissive auth/provider errors passed')
PY
