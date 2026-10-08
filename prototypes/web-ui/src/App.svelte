<script>
  // PROTOTYPE: the selected split-inbox design.
  // All data and mutations are local; no backend calls are made.
  import { tick } from 'svelte';
  import Icon from './Icon.svelte';
  import Inbox from './Inbox.svelte';
  import Install from './Install.svelte';
  import PreviewControls from './PreviewControls.svelte';
  import { fixtures, makeMessage } from './data.js';

  const params = new URLSearchParams(location.search);
  const allowedStates = ['standard', 'loading', 'empty', 'offline', 'long', 'many', 'inbound', 'failed'];
  const initialScenario = allowedStates.includes(params.get('state')) ? params.get('state') : 'standard';
  const initialMessages = fixtures(initialScenario);
  let theme = $state(['light', 'dark'].includes(params.get('theme')) ? params.get('theme') : 'auto');
  let systemDark = $state(matchMedia('(prefers-color-scheme: dark)').matches);
  let resolvedTheme = $derived(theme === 'auto' ? (systemDark ? 'dark' : 'light') : theme);
  $effect(() => {
    const preference = matchMedia('(prefers-color-scheme: dark)');
    const update = (event) => { systemDark = event.matches; };
    preference.addEventListener('change', update);
    return () => preference.removeEventListener('change', update);
  });
  let scenario = $state(initialScenario);
  let view = $state(params.get('view') === 'install' ? 'install' : 'messages');
  let query = $state(params.get('q') ?? '');
  let messages = $state(initialMessages);
  let selectedId = $state(params.get('message') || initialMessages[0]?.id || '');
  let detailOpen = $state(Boolean(params.get('message')));
  let tab = $state(['message', 'request', 'response'].includes(params.get('tab')) ? params.get('tab') : 'message');
  let limit = $state(50);
  let announcement = $state('');
  let toast = $state('');
  let pending = $state([]);
  let deleteTarget = $state(null);
  let confirmDialog = $state(null);
  let cancelButton = $state(null);
  let searchInput = $state(null);
  let serial = 300;
  const showControls = import.meta.env.DEV && params.get('controls') === '1';
  let filtered = $derived(messages.filter((message) => [message.to, message.from, message.body, message.account]
    .some((value) => value.toLowerCase().includes(query.trim().toLowerCase()))));
  let items = $derived(filtered.slice(0, limit));
  let hasMore = $derived(filtered.length > limit);
  let selected = $derived(messages.find((message) => message.id === selectedId));
  let isOffline = $derived(scenario === 'offline');
  let isLoading = $derived(scenario === 'loading');

  function url(key, value) {
    const next = new URL(location.href);
    if (value) next.searchParams.set(key, value); else next.searchParams.delete(key);
    return next;
  }
  function remember(key, value) { history.replaceState({}, '', url(key, value)); }
  async function navigate(next) {
    view = next;
    history.pushState({}, '', url('view', next === 'messages' ? '' : next));
    detailOpen = false;
    await tick();
    document.getElementById(next === 'install' ? 'install-heading' : 'messages-heading')?.focus();
    window.scrollTo({ top: 0, behavior: 'instant' });
  }
  function popstate() {
    const next = new URLSearchParams(location.search);
    view = next.get('view') === 'install' ? 'install' : 'messages';
  }
  function setQuery(value) {
    query = value;
    limit = 50;
    remember('q', value);
    announcement = '';
    detailOpen = false;
    const needle = value.trim().toLowerCase();
    const matches = messages.filter((message) => [message.to, message.from, message.body, message.account].some((field) => field.toLowerCase().includes(needle)));
    if (!matches.some((message) => message.id === selectedId)) selectedId = matches[0]?.id ?? '';
    remember('message', '');
  }
  function clearSearch() { setQuery(''); searchInput?.focus(); }
  async function retry() {
    setScenario('standard');
    await tick();
    document.getElementById('messages-heading')?.focus();
  }
  async function openMessage(id, nextTab = 'message') {
    selectedId = id;
    detailOpen = true;
    tab = nextTab;
    remember('message', id);
    remember('tab', nextTab === 'message' ? '' : nextTab);
    await tick();
    document.getElementById('detail-title')?.focus();
  }
  async function closeMessage() {
    const previousId = selectedId;
    detailOpen = false;
    selectedId = '';
    remember('message', '');
    remember('tab', '');
    await tick();
    document.querySelector('[data-message="' + previousId + '"] .message-hit')?.focus();
  }
  function changeTab(next) { tab = next; remember('tab', next === 'message' ? '' : next); }
  function setTheme(next) { theme = next; remember('theme', next === 'auto' ? '' : next); }
  function setScenario(next) {
    view = 'messages';
    remember('view', '');
    scenario = next;
    messages = fixtures(next);
    pending = [];
    query = '';
    limit = 50;
    selectedId = messages[0]?.id ?? '';
    detailOpen = false;
    tab = 'message';
    remember('state', next === 'standard' ? '' : next);
    remember('q', '');
    remember('message', '');
    remember('tab', '');
  }
  function tell(message) {
    toast = message;
    announcement = message;
    setTimeout(() => { if (toast === message) toast = ''; }, 4000);
  }
  function arrive() {
    if (isLoading || isOffline) setScenario('standard');
    serial++;
    const stamp = new Date().toISOString();
    const message = makeMessage(serial, {
      body: 'Your sign-in code is ' + String(482913 + serial) + '. It expires in 10 minutes.',
      created_at: stamp, updated_at: stamp, status: 'queued',
    });
    const scroll = document.getElementById('message-list-scroll');
    if (window.scrollY > 160 || (scroll?.scrollTop ?? 0) > 40) {
      pending = [message, ...pending];
      announcement = pending.length + ' new messages waiting.';
    } else {
      messages = [message, ...messages];
      if (!selectedId) selectedId = message.id;
      tell(query && ![message.to, message.from, message.body, message.account].some((v) => v.toLowerCase().includes(query.trim().toLowerCase()))
        ? 'New message caught outside this search.' : 'New message caught.');
    }
    if (scenario === 'empty') { scenario = 'standard'; remember('state', ''); }
  }
  async function revealNew() {
    messages = [...pending, ...messages];
    pending = [];
    await tick();
    document.getElementById('messages-heading')?.focus({ preventScroll: true });
    document.getElementById('message-list-scroll')?.scrollTo({ top: 0 });
    window.scrollTo({ top: 0, behavior: 'instant' });
    announcement = 'Newest messages shown.';
  }
  async function askDelete(target) {
    deleteTarget = target;
    await tick();
    confirmDialog.showModal();
    cancelButton?.focus();
  }
  async function remove() {
    const all = deleteTarget === 'all';
    const removedId = all ? null : deleteTarget.id;
    messages = all ? [] : messages.filter((message) => message.id !== removedId);
    if (all) pending = [];
    if (all || removedId === selectedId) {
      selectedId = messages[0]?.id ?? '';
      detailOpen = false;
      remember('message', '');
    }
    confirmDialog.close();
    tell(all ? 'All messages deleted.' : 'Message deleted.');
    await tick();
    document.getElementById('messages-heading')?.focus();
  }
  function keyboard(event) {
    if (confirmDialog?.open) return;
    if (event.key === '/' && !event.ctrlKey && !event.metaKey && !event.target.closest('input, textarea, select, [contenteditable]')) {
      event.preventDefault();
      if (view === 'messages') searchInput?.focus();
    }
    if (event.key === 'Escape') {
      if (event.target === searchInput && query) { setQuery(''); event.preventDefault(); }
      else if (detailOpen) { closeMessage(); event.preventDefault(); }
    }
  }
</script>

<svelte:window onkeydown={keyboard} onpopstate={popstate} />
<svelte:head><title>{view === 'install' ? 'Install' : 'Messages'} · Sundew prototype</title></svelte:head>

<div class="app" data-theme={resolvedTheme} class:with-controls={showControls}>
  <a class="skip-link" href="#main">Skip to content</a>
  <header class="app-header">
    <a class="wordmark" href="?view=messages" onclick={(event) => { event.preventDefault(); navigate('messages'); }}>Sundew<span class="wordmark-period">.</span></a>
    <nav aria-label="Main navigation">
      <a href="?view=messages" aria-current={view === 'messages' ? 'page' : undefined} onclick={(event) => { event.preventDefault(); navigate('messages'); }}><Icon name="inbox" size={16} />Messages</a>
      <a href="?view=install" aria-current={view === 'install' ? 'page' : undefined} onclick={(event) => { event.preventDefault(); navigate('install'); }}><Icon name="book" size={16} />Install</a>
    </nav>
    <span class="connection" class:offline={isOffline} class:loading={isLoading}><span></span>{isOffline ? 'Disconnected' : isLoading ? 'Connecting' : 'Live'}</span>
  </header>

  <main id="main" class:install-main={view === 'install'}>
    {#if view === 'install'}
      <Install onback={() => navigate('messages')} />
    {:else}
      <div class="page-heading">
        <div><div class="title-row"><h1 id="messages-heading" tabindex="-1">Messages</h1>{#if !isLoading && !isOffline}<span class="message-count">{items.length}{hasMore ? '+' : ''}</span>{/if}</div><p>Every text your application sends, caught here.</p></div>
        <button class="button secondary delete-all" disabled={!messages.length || isLoading || isOffline} onclick={() => askDelete('all')}><Icon name="trash" size={16} /><span>Delete all</span></button>
      </div>
      <div class="inbox-toolbar">
        <div class="search-field">
          <Icon name="search" size={18} />
          <input bind:this={searchInput} value={query} oninput={(event) => setQuery(event.currentTarget.value)} type="search" autocomplete="off" spellcheck="false" aria-label="Search to, from, body or account" placeholder="Search to, from, body or account…" disabled={isLoading || isOffline} />
          {#if query}<button class="search-clear" aria-label="Clear search" onclick={clearSearch}><Icon name="close" size={14} /></button>{:else}<kbd>/</kbd>{/if}
        </div>
        <span class="sort-label">Newest first <span>↓</span></span>
      </div>
      <div class="sr-only" role="status">{query && !isLoading ? (items.length + (hasMore ? ' or more' : '') + ' matching messages loaded.') : ''}</div>
      {#if pending.length}<button class="new-message-banner" onclick={revealNew}>{pending.length} new {pending.length === 1 ? 'message' : 'messages'} <span>Show newest ↑</span></button>{/if}

      {#if isLoading}
        <section class="state-panel loading-state" aria-label="Loading messages" aria-busy="true">
          <span class="sr-only" role="status">Loading messages…</span>
          {#each Array(5) as _}<div class="skeleton-row" aria-hidden="true"><span></span><span></span><span></span></div>{/each}
        </section>
      {:else if isOffline}
        <section class="state-panel">
          <div class="state-icon error-icon"><Icon name="alert" size={28} /></div>
          <h2>Can't reach Sundew</h2>
          <p>The inbox couldn't be loaded. Check that Sundew is running, then try again.</p>
          <button class="button primary" onclick={retry}>Try again</button>
          <button class="back-link" onclick={() => navigate('install')}>View installation guide<Icon name="arrow" size={15} /></button>
        </section>
      {:else if !messages.length}
        <section class="state-panel">
          <div class="state-icon"><Icon name="inbox" size={30} /></div>
          <h2>Ready for your first message</h2>
          <p>Point your application's SMS provider at Sundew. Messages will appear here as they arrive.</p>
          <button class="button primary" onclick={() => navigate('install')}>Set up your application<Icon name="arrow" size={16} /></button>
          <span class="empty-live"><span class="status-dot"></span>Listening for messages</span>
        </section>
      {:else if !filtered.length}
        <section class="state-panel">
          <div class="state-icon"><Icon name="search" size={28} /></div>
          <h2>No matching messages</h2><p>No recipient, sender, message body or account contains “{query}”.</p>
          <button class="button secondary" onclick={clearSearch}>Clear search</button>
        </section>
      {:else}
        <Inbox {items} {selected} {detailOpen} {tab} {query} {hasMore} onopen={openMessage} onclose={closeMessage} ondelete={askDelete} ontab={changeTab} onload={() => limit += 50} />
      {/if}
      <footer class="app-footer"><span>Messages are kept in memory until Sundew stops.</span><a href="?view=install" onclick={(event) => { event.preventDefault(); navigate('install'); }}>Installation guide<Icon name="arrow" size={13} /></a></footer>
    {/if}
  </main>

  <div class="announcer sr-only" role="status" aria-live="polite" aria-atomic="true">{announcement}</div>
  {#if toast}<div class="toast" aria-hidden="true"><Icon name="check" size={16} />{toast}</div>{/if}

  <dialog bind:this={confirmDialog} class="delete-dialog" aria-labelledby="delete-title" aria-describedby="delete-description">
    <div class="dialog-icon"><Icon name="trash" size={24} /></div>
    <h2 id="delete-title">{deleteTarget === 'all' ? 'Delete all messages?' : 'Delete this message?'}</h2>
    <p id="delete-description">{deleteTarget === 'all' ? 'Every caught message will be deleted, including messages not loaded or hidden by search. This cannot be undone.' : 'The message to ' + (deleteTarget?.to ?? '') + ' will be permanently removed from this inbox.'}</p>
    <div class="dialog-actions"><button class="button secondary" bind:this={cancelButton} onclick={() => confirmDialog.close()}>Cancel</button><button class="button danger" onclick={remove}>{deleteTarget === 'all' ? 'Delete all messages' : 'Delete message'}</button></div>
  </dialog>
  {#if showControls}<PreviewControls {theme} {scenario} ontheme={setTheme} onscenario={setScenario} onarrival={arrive} />{/if}
</div>
