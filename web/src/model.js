// The API boundary owns snapshots, cancellation and stream reconciliation. Views
// consume one immutable snapshot; no Svelte or DOM is needed to exercise races.
export const apiRoot = '/api/v1/messages';
export function normalize(message) {
  return { ...message, media_urls: message.media_urls ?? [], options: message.options ?? {},
    exchange: { ...message.exchange, header: message.exchange?.header ?? {}, response_header: message.exchange?.response_header ?? {} } };
}
export function listURL({ q = '', phone = null } = {}, cursor = '') {
  const params = new URLSearchParams({ limit: '50' });
  if (q.trim()) params.set('q', q.trim());
  if (phone) params.set(phone.field, phone.number);
  if (cursor) params.set('cursor', cursor);
  return apiRoot + '?' + params;
}
export function matches(message, { q = '', phone = null } = {}) {
  return (!phone || message[phone.field] === phone.number) &&
    [message.to, message.from, message.body, message.account].some(value => (value ?? '').toLowerCase().includes(q.trim().toLowerCase()));
}
export function sorted(messages) {
  return [...new Map(messages.map(m => [m.id, m])).values()].sort((a, b) =>
    Date.parse(b.created_at) - Date.parse(a.created_at) ||
    (b.created_at.match(/\.(\d+)/)?.[1] ?? '').padEnd(9, '0').localeCompare((a.created_at.match(/\.(\d+)/)?.[1] ?? '').padEnd(9, '0')) || b.id.localeCompare(a.id));
}
export function suggestions(messages, prefix) {
  if (!prefix.trim().startsWith('+')) return [];
  return ['to', 'from'].flatMap(field => [...new Set(messages.map(m => m[field]))]
    .filter(number => number?.startsWith(prefix.trim())).sort().map(number => ({ field, number }))).slice(0, 8);
}
export function initialState() {
  return { items: [], pending: [], cursor: '', filters: {}, loading: true, loaded: false,
    error: '', connection: 'connecting', selected: null, selectedId: '', deleting: false, phones: [] };
}
export function reconcile(state, event, stage = false) {
  if (event.type === 'store.reset') return { ...state, items: [], pending: [], cursor: '', selected: null, selectedId: '' };
  if (!event.message?.id) return state;
  const message = normalize(event.message);
  const remove = event.type === 'message.deleted' || !matches(message, state.filters);
  const inItems = state.items.some(m => m.id === message.id);
  const inPending = state.pending.some(m => m.id === message.id);
  let items = state.items.filter(m => m.id !== message.id);
  let pending = state.pending.filter(m => m.id !== message.id);
  if (!remove) {
    if (inPending || (!inItems && stage)) pending = sorted([...pending, message]);
    else items = sorted([...items, message]);
  }
  const deletedSelection = event.type === 'message.deleted' && state.selectedId === message.id;
  return { ...state, items, pending,
    selectedId: deletedSelection ? '' : state.selectedId,
    selected: deletedSelection ? null : state.selectedId === message.id ? message : state.selected };
}
export class InboxClient {
  constructor(change, { fetcher = (url, options) => fetch(url, options), stream = url => new EventSource(url), away = () => window.scrollY > 160, missing = () => {}, notice = () => {} } = {}) {
    this.change = change; this.fetcher = fetcher; this.stream = stream; this.away = away;
    this.missing = missing; this.notice = notice; this.state = initialState();
    this.generation = 0; this.detailGeneration = 0; this.delay = 500; this.running = false;
  }
  emit(patch = {}) { this.state = { ...this.state, ...patch }; this.change(this.state); }
  async request(url, options = {}) {
    const response = await this.fetcher(url, options);
    if (!response.ok) {
      const error = new Error('Cannot reach Sundew. Try again.'); error.status = response.status;
      try { error.message = (await response.json()).error?.message || error.message; } catch { /* transport or non-JSON error */ }
      throw error;
    }
    return response.status === 204 ? null : response.json();
  }
  start(id = '') {
    if (this.running) return;
    this.running = true; this.emit({ selectedId: id }); this.connect();
  }
  stop() {
    this.running = false; this.generation++; this.detailGeneration++;
    clearTimeout(this.retryTimer); clearTimeout(this.searchTimer);
    this.source?.close(); this.listAbort?.abort(); this.pageAbort?.abort(); this.phoneAbort?.abort(); this.detailAbort?.abort();
    this.buffer = null; this.pageEvents = null;
    this.emit({ connection: 'disconnected' });
  }
  connect() {
    if (!this.running) return;
    clearTimeout(this.retryTimer); this.source?.close();
    const source = this.stream(apiRoot + '/stream'); this.source = source;
    source.onopen = () => {
      if (source !== this.source || !this.running) return;
      this.delay = 500; this.emit({ connection: 'live' });
      this.refresh();
      if (this.state.selectedId) this.select(this.state.selectedId);
    };
    for (const type of ['message.created', 'message.updated', 'message.deleted', 'store.reset']) {
      source.addEventListener(type, event => {
        if (source !== this.source || !this.running) return;
        try { this.event({ ...JSON.parse(event.data), type }); }
        catch { this.emit({ error: 'Live update could not be read. Reconnecting…' }); source.onerror(); }
      });
    }
    source.onerror = () => {
      if (source !== this.source || !this.running) return;
      source.close(); this.source = null;
      this.emit({ connection: 'reconnecting' });
      // A failed first connection must still reach the API-unreachable state.
      if (!this.state.loaded) { this.refresh(); if (this.state.selectedId) this.select(this.state.selectedId); }
      this.retryTimer = setTimeout(() => this.connect(), this.delay);
      this.delay = Math.min(this.delay * 2, 30000);
    };
  }
  event(event) {
    if (this.buffer) { this.buffer.push(event); return; }
    this.pageEvents?.push(event);
    const oldId = this.state.selectedId;
    if (event.message?.id === oldId) { this.detailGeneration++; this.detailAbort?.abort(); }
    this.state = reconcile(this.state, event, this.away()); this.emit();
    if (oldId && !this.state.selectedId) { this.detailGeneration++; this.detailAbort?.abort(); this.missing(); }
    if (!this.state.selectedId && this.state.items.length) this.emit({ selectedId: this.state.items[0].id, selected: this.state.items[0] });
    if (event.type === 'message.created') this.notice(this.state.pending.length ? this.state.pending.length + ' new messages waiting.' : 'New message caught.');
  }
  setFilters(filters) {
    clearTimeout(this.searchTimer); this.listAbort?.abort(); this.pageAbort?.abort(); this.phoneAbort?.abort(); this.pageEvents = null;
    this.generation++; this.buffer = null; this.detailGeneration++; this.detailAbort?.abort();
    this.emit({ filters, selectedId: '', selected: null, pending: [], cursor: '', phones: [] });
    this.searchTimer = setTimeout(() => { this.refresh(false); this.loadPhones(); }, 200);
  }
  async loadPhones() {
    const q = this.state.filters.q?.trim() ?? '';
    if (!q.startsWith('+')) return;
    this.phoneAbort?.abort(); const abort = new AbortController(); this.phoneAbort = abort;
    try {
      const page = await this.request(listURL({ q }), { signal: abort.signal });
      if (!abort.signal.aborted && q === this.state.filters.q?.trim()) this.emit({ phones: suggestions(page.items, q) });
    } catch { /* List error is surfaced by refresh; autocomplete stays optional. */ }
  }
  async refresh(preserve = true) {
    clearTimeout(this.searchTimer); this.listAbort?.abort(); this.pageAbort?.abort(); this.pageEvents = null;
    const abort = new AbortController(); this.listAbort = abort;
    const version = ++this.generation; const before = this.state;
    const target = preserve ? Math.max(50, before.items.length + before.pending.length) : 50;
    const buffered = []; this.buffer = buffered;
    this.emit({ loading: true });
    try {
      let items = [], cursor = '';
      do {
        const page = await this.request(listURL(before.filters, cursor), { signal: abort.signal });
        items = sorted([...items, ...(page.items ?? []).map(normalize)]); cursor = page.next_cursor || '';
      } while (cursor && items.length < target);
      if (version !== this.generation || abort.signal.aborted || !this.running) return;
      const known = new Set(before.items.map(m => m.id));
      const stage = preserve && before.loaded && this.away();
      this.emit({ items: stage ? items.filter(m => known.has(m.id)) : items,
        pending: stage ? items.filter(m => !known.has(m.id)) : [], cursor, loaded: true, loading: false, error: '' });
      this.buffer = null;
      for (const event of buffered) this.event(event);
      if (!this.state.selectedId && this.state.items.length) this.emit({ selectedId: this.state.items[0].id, selected: this.state.items[0] });
    } catch (error) {
      if (version !== this.generation || abort.signal.aborted || !this.running) return;
      this.buffer = null;
      this.emit({ loading: false, error: error.message, connection: 'disconnected' });
      for (const event of buffered) this.event(event);
    }
  }
  async more() {
    if (!this.state.cursor || this.state.loading) return;
    const version = this.generation; const cursor = this.state.cursor;
    const pageEvents = []; this.pageEvents = pageEvents;
    const abort = new AbortController(); this.pageAbort = abort;
    this.emit({ loading: true });
    try {
      const page = await this.request(listURL(this.state.filters, cursor), { signal: abort.signal });
      // Do not resurrect deletions/updates while a page is in flight.
      if (version !== this.generation || abort.signal.aborted || !this.running) return;
      this.pageEvents = null;
      this.state = { ...this.state, items: sorted([...this.state.items, ...(page.items ?? []).map(normalize)]), cursor: page.next_cursor || '', loading: false, error: '' };
      for (const event of pageEvents) this.state = reconcile(this.state, event, this.away());
      this.emit();
    } catch (error) { if (version === this.generation && !abort.signal.aborted && this.running) this.emit({ loading: false, error: error.message }); }
    finally {
      if (this.pageEvents === pageEvents) this.pageEvents = null;
      if (this.pageAbort === abort) this.pageAbort = null;
    }
  }
  async select(id) {
    this.detailAbort?.abort(); const abort = new AbortController(); this.detailAbort = abort;
    const version = ++this.detailGeneration;
    this.emit({ selectedId: id, selected: this.state.items.find(m => m.id === id) ?? null });
    try {
      const message = await this.request(apiRoot + '/' + encodeURIComponent(id), { signal: abort.signal });
      if (version === this.detailGeneration && !abort.signal.aborted && this.running) this.emit({ selected: normalize(message) });
    } catch (error) {
      if (version !== this.detailGeneration || abort.signal.aborted || !this.running) return;
      if (error.status === 404) {
        this.emit({ items: this.state.items.filter(m => m.id !== id), selected: null, selectedId: '' }); this.missing();
      } else this.emit({ error: error.message });
    }
  }
  leaveDetail() { this.detailGeneration++; this.detailAbort?.abort(); }
  reveal() { this.emit({ items: sorted([...this.state.items, ...this.state.pending]), pending: [] }); }
  async remove(target) {
    if (this.state.deleting) return false;
    this.emit({ deleting: true });
    try {
      await this.request(apiRoot + (target === 'all' ? '' : '/' + encodeURIComponent(target.id)), { method: 'DELETE' });
      // Abort snapshots taken before deletion; the stream may already have arrived.
      this.generation++; this.listAbort?.abort(); this.pageAbort?.abort(); this.pageEvents = null; const buffered = this.buffer ?? []; this.buffer = null;
      for (const event of buffered) this.event(event);
      this.event(target === 'all' ? { type: 'store.reset' } : { type: 'message.deleted', message: target });
      this.emit({ deleting: false, loading: false }); return true;
    } catch (error) { this.emit({ deleting: false, error: error.message }); return false; }
  }
}
