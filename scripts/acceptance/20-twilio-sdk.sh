#!/usr/bin/env bash
set -euo pipefail

: "${SUNDEW_TEST_IMAGE:?run through scripts/acceptance.sh}"
: "${SUNDEW_TEST_PROJECT:?run through scripts/acceptance.sh}"
: "${SUNDEW_TEST_NETWORK:?run through scripts/acceptance.sh}"
: "${SUNDEW_TEST_URL:?run through scripts/acceptance.sh}"

# Only invented credentials. Each official SDK runs on the isolated run network.
python_image="sundew-sdk-python:${SUNDEW_TEST_PROJECT}"
node_image="sundew-sdk-node:${SUNDEW_TEST_PROJECT}"
cleanup() {
  docker image rm -f "$python_image" "$node_image" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker build --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" -t "$python_image" - <<'DOCKERFILE'
FROM python:3.14-alpine
RUN pip install --no-cache-dir twilio==9.11.2
DOCKERFILE

docker build --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" -t "$node_image" - <<'DOCKERFILE'
FROM node:24-alpine
RUN npm install --prefix /opt/sdk --omit=dev --no-audit --no-fund twilio@6.1.2 && npm cache clean --force
ENV NODE_PATH=/opt/sdk/node_modules
DOCKERFILE

docker run --rm -i --network "$SUNDEW_TEST_NETWORK" \
  --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" \
  -e "SUNDEW_TEST_URL=$SUNDEW_TEST_URL" "$python_image" python - <<'PYTHON'
import os, re
from twilio.rest import Client
from twilio.base.exceptions import TwilioRestException

account = 'AC00000000000000000000000000000000'
def client(base, token):
    result = Client(account, token)
    result.api.base_url = base
    return result

def check_message(c, body):
    m = c.messages.create(to='+15551234567', from_='+15557654321', body=body)
    assert re.fullmatch(r'SM[0-9a-f]{32}', m.sid), m.sid
    assert m.status == 'queued' and m.body == body
    assert m.to == '+15551234567' and m.from_ == '+15557654321'
    assert m.account_sid == account and m.direction == 'outbound-api'
    assert m.num_segments == '1' and m.num_media == '0'
    assert m.date_created is not None and m.date_updated is not None
    assert m.date_sent is None and m.error_code is None
    fetched = c.messages(m.sid).fetch()
    assert fetched.sid == m.sid and fetched.body == body

check_message(client(os.environ['SUNDEW_TEST_URL'], 'arbitrary-invented-token'), 'Python SDK dew')
try:
    client(os.environ['SUNDEW_TEST_URL'], 'arbitrary-invented-token').messages.create(
        to='+15005550009', from_='+15005550006', body='magic failure')
except TwilioRestException as error:
    assert error.status == 400 and error.code == 21614, (error.status, error.code)
else:
    raise AssertionError('magic number did not raise documented error')
print('Python official SDK send/fetch/magic error: passed')
PYTHON

docker run --rm -i --network "$SUNDEW_TEST_NETWORK" \
  --label "com.docker.compose.project=$SUNDEW_TEST_PROJECT" \
  -e "SUNDEW_TEST_URL=$SUNDEW_TEST_URL" "$node_image" node - <<'JAVASCRIPT'
const assert = require('node:assert/strict');
const twilio = require('twilio');
const account = 'AC00000000000000000000000000000000';
function client(base, token) {
  const result = twilio(account, token);
  result.api.baseUrl = base;
  return result;
}
async function checkMessage(c, body) {
  const m = await c.messages.create({to: '+15551234567', from: '+15557654321', body});
  assert.match(m.sid, /^SM[0-9a-f]{32}$/);
  assert.equal(m.status, 'queued');
  assert.equal(m.body, body);
  assert.equal(m.to, '+15551234567');
  assert.equal(m.from, '+15557654321');
  assert.equal(m.accountSid, account);
  assert.equal(m.direction, 'outbound-api');
  assert.equal(m.numSegments, '1');
  assert.equal(m.numMedia, '0');
  assert.ok(m.dateCreated instanceof Date && !Number.isNaN(m.dateCreated.valueOf()));
  assert.ok(m.dateUpdated instanceof Date && !Number.isNaN(m.dateUpdated.valueOf()));
  assert.equal(m.dateSent, null);
  assert.equal(m.errorCode, null);
  const fetched = await c.messages(m.sid).fetch();
  assert.equal(fetched.sid, m.sid);
  assert.equal(fetched.body, body);
}
(async () => {
  await checkMessage(client(process.env.SUNDEW_TEST_URL, 'arbitrary-invented-token'), 'Node SDK dew');
  await assert.rejects(
    client(process.env.SUNDEW_TEST_URL, 'arbitrary-invented-token').messages.create({to: '+15005550009', from: '+15005550006', body: 'magic failure'}),
    error => error.status === 400 && error.code === 21614
  );
  console.log('Node official SDK send/fetch/magic error: passed');
})().catch(error => { console.error(error); process.exit(1); });
JAVASCRIPT
