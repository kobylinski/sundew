// PROTOTYPE ONLY. Fixtures have exactly the JSON fields of core.Message.
// Values, HTTP bodies, credentials and timestamps are fictional examples.
export const account = 'AC11111111111111111111111111111111';
const epoch = Date.parse('2026-10-08T03:58:42Z');

export function makeMessage(index, overrides = {}) {
  const id = 'msg_' + String(index).padStart(4, '0');
  const providerID = 'SM' + String(index).padStart(32, '0');
  const to = overrides.to ?? '+15551234567';
  const from = overrides.from ?? '+15550001001';
  const messageAccount = overrides.account ?? account;
  const body = overrides.body ?? 'Your sign-in code is 482913. It expires in 10 minutes.';
  const created = new Date(epoch - (index - 1) * 83000).toISOString();
  const options = overrides.options ?? { status_callback: 'http://app:3000/webhooks/sms/status' };
  const form = new URLSearchParams({ To: to, From: from, Body: body });
  if (options.status_callback) form.set('StatusCallback', options.status_callback);
  const message = {
    id,
    provider: 'twilio',
    provider_id: providerID,
    account: messageAccount,
    direction: 'outbound',
    from,
    to,
    body,
    segments: Math.ceil(body.length / 160),
    media_urls: [],
    status: 'delivered',
    options,
    exchange: {
      method: 'POST',
      path: '/2010-04-01/Accounts/' + messageAccount + '/Messages.json',
      query: '',
      header: {
        'Content-Type': ['application/x-www-form-urlencoded'],
        Authorization: ['Basic ' + btoa(messageAccount + ':development')],
        'User-Agent': ['twilio-node/fixture'],
      },
      body: form.toString(),
      response_status: 201,
      response_header: { 'Content-Type': ['application/json'] },
      response_body: JSON.stringify({
        sid: providerID, account_sid: messageAccount, from, to, body,
        status: 'queued', num_segments: String(Math.ceil(body.length / 160)),
        error_code: null, error_message: null,
        uri: '/2010-04-01/Accounts/' + messageAccount + '/Messages/' + providerID + '.json',
      }, null, 2),
    },
    created_at: created,
    updated_at: new Date(Date.parse(created) + 2000).toISOString(),
    ...overrides,
  };
  return message;
}

const longBody = 'Your order #1048 is ready for collection.\n\n'
  + 'Collection point: 24 Garden Street, entrance B. We are open Monday–Friday, 08:00–18:00. '
  + 'Please bring your collection code 729 481 and a photo ID. If someone is collecting on your behalf, '
  + 'share this message with them.\n\n'
  + 'Your order will be held for 7 days. Questions? Reply to this message and our team will help.\n\n'
  + 'Reference: ORDER-1048-TEST-VERY-LONG-REFERENCE-TO-CHECK-WRAPPING-0123456789';

export function fixtures(scenario = 'standard') {
  if (scenario === 'empty' || scenario === 'loading') return [];
  const items = [
    makeMessage(1, { status: 'queued', updated_at: new Date(epoch).toISOString() }),
    makeMessage(2, { to: '+15559876543', body: 'Thanks for your order. Your receipt is ready at https://example.test/r/1048.' }),
    makeMessage(3, {
      from: '+15559876543', to: '+15550001001', body: 'STOP', direction: 'inbound', status: 'received',
      provider_id: '', segments: 0, options: null,
      // A simulated inbound message may have no captured provider exchange.
      exchange: { method: '', path: '', query: '', header: {}, body: '', response_status: 0, response_header: {}, response_body: '' },
    }),
    makeMessage(4, { to: '+15557654321', body: 'Your appointment is tomorrow at 10:30. Reply YES to confirm.', status: 'failed', error_code: '30003' }),
    makeMessage(5, { to: '+15551234567', body: 'Your parcel is on its way. Track it at https://example.test/track/6021.', status: 'sent' }),
    makeMessage(6, {
      to: '+15552345678', body: 'Here is the collection point for your order.',
      media_urls: ['https://example.test/media/collection-point.jpg'],
      options: { status_callback: 'http://app:3000/webhooks/sms/status', validity: '3600', sender_id: 'SundewDemo' },
    }),
    makeMessage(7, { to: '+15554567890', body: 'You changed your password. If this was you, no action is needed.', account: 'AC22222222222222222222222222222222' }),
    makeMessage(8, { to: '+15553456789', body: 'Your booking is confirmed for Friday, 16 October at 14:00. See you then.' }),
  ];
  if (scenario === 'long') {
    items[0] = makeMessage(1, { body: longBody, segments: 5 });
  }
  if (scenario === 'links') {
    items[0] = makeMessage(1, {
      body: 'Track your parcel at https://example.test/track/6021.\nPlain text: javascript:alert(1)\nHTML fragment: <img src=x onerror=alert(1)>',
      media_urls: ['https://example.test/media/collection-point.jpg', 'javascript:alert(1)', 'data:text/html,<b>example</b>'],
      options: { status_callback: 'javascript:alert(1)' },
    });
  }
  if (scenario === 'inbound') return [items[2], ...items.filter((m) => m.id !== items[2].id)]
    .map((m, i) => ({ ...m, created_at: new Date(epoch - i * 83000).toISOString(), updated_at: new Date(epoch - i * 83000 + 2000).toISOString() }));
  if (scenario === 'failed') return [items[3], ...items.filter((m) => m.id !== items[3].id)]
    .map((m, i) => ({ ...m, created_at: new Date(epoch - i * 83000).toISOString(), updated_at: new Date(epoch - i * 83000 + 2000).toISOString() }));
  if (scenario === 'many') {
    return [...items, ...Array.from({ length: 232 }, (_, i) => makeMessage(i + 9, {
      to: '+1555' + String(1000000 + i).padStart(7, '0'),
      body: 'Your verification code is ' + String(482914 + i) + '. This is a development test message.',
      status: i % 11 === 0 ? 'failed' : 'delivered',
      ...(i % 11 === 0 ? { error_code: '30003' } : {}),
    }))];
  }
  return items;
}

export const time = (value) => new Intl.DateTimeFormat('en-GB', {
  hour: '2-digit', minute: '2-digit', timeZone: 'Europe/Warsaw',
}).format(new Date(value));

export const dateTime = (value) => new Intl.DateTimeFormat('en-GB', {
  day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit',
  minute: '2-digit', second: '2-digit', timeZone: 'Europe/Warsaw', timeZoneName: 'short',
}).format(new Date(value));

export function rawRequest(message) {
  const x = message.exchange;
  if (!x.method) return '';
  return [x.method + ' ' + x.path + (x.query ? '?' + x.query : '') + ' HTTP/1.1',
    ...Object.entries(x.header ?? {}).flatMap(([key, values]) => values.map((value) => key + ': ' + value)),
    '', x.body].join('\n');
}

export function rawResponse(message) {
  const x = message.exchange;
  if (!x.response_status) return '';
  return ['HTTP/1.1 ' + x.response_status,
    ...Object.entries(x.response_header ?? {}).flatMap(([key, values]) => values.map((value) => key + ': ' + value)),
    '', x.response_body].join('\n');
}
