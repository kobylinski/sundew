import { test } from 'node:test';
import assert from 'node:assert/strict';
import { InboxClient, initialState, normalize, listURL, reconcile, suggestions, matches, sorted } from './model.js';
const message = (id, overrides = {}) => normalize({ id, created_at: '2026-10-08T08:00:00Z', updated_at: '2026-10-08T08:00:00Z', to: '+15550001', from: '+15550002', body: 'Login CODE', account: 'ACtest', status: 'queued', ...overrides });
const response = data => ({ ok: true, status: 200, json: async () => data });
const deferred = () => { let resolve; const promise = new Promise(r => resolve = r); return { promise, resolve }; };
class Stream {
  handlers = {}; close() { this.closed = true; }
  addEventListener(type, handler) { this.handlers[type] = handler; }
  send(type, message) { this.handlers[type]({ data: JSON.stringify({ type, message }) }); }
}
function client(fetcher, away = false) {
  const streams = [];
  const result = new InboxClient(() => {}, { fetcher, stream: () => { const s = new Stream(); streams.push(s); return s; }, away: () => away });
  result.start(); return { client: result, streams };
}
test('filter mapping encodes plus, preserves exact phone + query, resets cursor', () => {
  const url = new URL(listURL({ q: ' CODE & link ', phone: { field: 'to', number: '+1555' } }, 'opaque/+='), 'http://local');
  assert.equal(url.searchParams.get('to'), '+1555'); assert.equal(url.searchParams.get('q'), 'CODE & link');
  assert.equal(url.searchParams.get('cursor'), 'opaque/+='); assert.equal(url.searchParams.get('limit'), '50');
  assert.equal(new URL(listURL(), 'http://local').searchParams.has('cursor'), false);
  assert.ok(matches(message('a'), { q: 'code', phone: { field: 'from', number: '+15550002' } }));
  assert.ok(!matches(message('a'), { q: 'code', phone: { field: 'to', number: '+1555' } }));
});
test('nullable inbound collections and phone suggestion field distinction', () => {
  const m = message('a', { media_urls: null, options: null, exchange: null });
  assert.deepEqual(m.media_urls, []); assert.deepEqual(m.exchange.header, {});
  const found = suggestions([m, m, message('b', { from: m.to })], '+1555');
  assert.equal(found.filter(p => p.number === m.to).length, 2);
  assert.ok(suggestions(Array.from({ length: 20 }, (_, i) => message(''+i, { to: '+1'+i })), '+').length <= 8);
});
test('reconciliation deduplicates, stages arrivals, applies updates/deletions/reset', () => {
  const a = message('a'); let state = { ...initialState(), items: [a], selectedId: 'a', selected: a, filters: { q: 'code' } };
  state = reconcile(state, { type: 'message.created', message: message('b') }, true);
  assert.deepEqual(state.items.map(m => m.id), ['a']); assert.equal(state.pending.length, 1);
  state = reconcile(state, { type: 'message.updated', message: message('b', { status: 'delivered' }) }, true);
  assert.equal(state.pending[0].status, 'delivered');
  state = reconcile(state, { type: 'message.updated', message: message('a', { body: 'No match' }) });
  assert.equal(state.items.length, 0); assert.equal(state.selected.body, 'No match');
  state = reconcile(state, { type: 'message.deleted', message: a }); assert.equal(state.selectedId, '');
  assert.equal(reconcile(state, { type: 'store.reset' }).pending.length, 0);
  assert.equal(sorted([a, a]).length, 1);
  assert.equal(sorted([message('a', { created_at: '2026-10-08T08:00:00.100000001Z' }), message('b', { created_at: '2026-10-08T08:00:00.1Z' })])[0].id, 'a');
});
test('open stream first, buffer snapshot races, reconnect revalidates list + detail', async () => {
  const pending = deferred(); const urls = [];
  const { client: c, streams } = client(async url => { urls.push(url); return urls.length === 1 ? pending.promise : response(url.includes('?') ? { items: [message('b')], next_cursor: '' } : message('b')); });
  try {
    assert.equal(urls.length, 0); streams[0].onopen();
    streams[0].send('message.created', message('b')); streams[0].send('message.deleted', message('a'));
    pending.resolve(response({ items: [message('a')], next_cursor: '' }));
    await new Promise(r => setImmediate(r));
    assert.deepEqual(c.state.items.map(m => m.id), ['b']);
    streams[0].onerror(); assert.equal(c.state.connection, 'reconnecting');
    c.connect(); streams[1].onopen(); await new Promise(r => setImmediate(r));
    assert.ok(urls.some(url => url.endsWith('/b'))); assert.equal(c.state.connection, 'live');
  } finally { c.stop(); }
});
test('stale search response cannot overwrite a newer filter response', async () => {
  const old = deferred(); let count = 0;
  const { client: c } = client(async () => ++count === 1 ? old.promise : response({ items: [message('new')], next_cursor: '' }));
  try {
    const first = c.refresh(); c.setFilters({ q: 'new' }); const second = c.refresh(false); await second;
    old.resolve(response({ items: [message('old')], next_cursor: 'old' })); await first;
    assert.deepEqual(c.state.items.map(m => m.id), ['new']); assert.equal(c.state.cursor, '');
  } finally { c.stop(); }
});
test('pagination does not resurrect a row deleted while page is in flight', async () => {
  const pending = deferred(); const { client: c } = client(() => pending.promise);
  try {
    c.emit({ items: [message('a')], loaded: true, loading: false, cursor: 'page2' });
    const more = c.more(); c.event({ type: 'message.deleted', message: message('b') });
    pending.resolve(response({ items: [message('b')], next_cursor: '' })); await more;
    assert.deepEqual(c.state.items.map(m => m.id), ['a']);
  } finally { c.stop(); }
});
test('failed deletion retains rows; success accepts 204 without JSON; all deletes have no filters', async () => {
  const urls = []; let fail = true;
  const { client: c } = client(async (url, options) => {
    urls.push([url, options]);
    return fail ? { ok: false, status: 503, json: async () => ({ error: { message: 'offline' } }) } : { ok: true, status: 204, json: () => { throw Error('204 parsed'); } };
  });
  try {
    c.emit({ items: [message('a')], loaded: true });
    assert.equal(await c.remove('all'), false); assert.equal(c.state.items.length, 1);
    fail = false; assert.equal(await c.remove('all'), true); assert.equal(c.state.items.length, 0);
    assert.equal(urls[1][0], '/api/v1/messages'); assert.equal(urls[1][1].method, 'DELETE');
  } finally { c.stop(); }
});
