<script>
  import { tick, onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import Inbox from './Inbox.svelte';
  import Install from './Install.svelte';
  import { initialState, InboxClient, suggestions } from './model.js';

  function route() {
    const match = location.pathname.match(/^\/messages\/([^/]+)$/);
    let id = '';
    try { id = match ? decodeURIComponent(match[1]) : ''; } catch { /* Invalid route returns to inbox. */ }
    return { view: location.pathname === '/install' ? 'install' : 'messages', id };
  }
  const initial = route();
  let view = $state(initial.view);
  let detailOpen = $state(Boolean(initial.id));
  let systemDark = $state(matchMedia('(prefers-color-scheme: dark)').matches);
  let resolvedTheme = $derived(systemDark ? 'dark' : 'light');
  $effect(() => {
    const preference = matchMedia('(prefers-color-scheme: dark)');
    const update = event => { systemDark = event.matches; };
    preference.addEventListener('change', update);
    return () => preference.removeEventListener('change', update);
  });
  let inbox = $state(initialState());
  let query = $state('');
  let phoneFilter = $state(null);
  let showPhones = $state(false);
  let activePhone = $state(-1);
  let announcement = $state('');
  let toast = $state('');
  let deleteTarget = $state(null);
  let confirmDialog = $state(null);
  let cancelButton = $state(null);
  let searchInput = $state(null);
  let toastTimer;
  const client = new InboxClient(next => inbox = next, {
    notice: message => { announcement = message; },
    missing: async () => {
      detailOpen = false;
      if (view === 'messages') history.replaceState({}, '', '/');
      tell('The message is no longer available.');
      await focusInbox();
    },
  });
  onMount(() => {
    if (view === 'messages') client.start(initial.id);
    return () => { client.stop(); clearTimeout(toastTimer); };
  });
  let messages = $derived(inbox.items);
  let items = $derived(inbox.items);
  let pending = $derived(inbox.pending);
  let hasMore = $derived(Boolean(inbox.cursor));
  let selected = $derived(inbox.selected ?? (detailOpen ? null : items[0]));
  let phones = $derived(inbox.phones.length ? inbox.phones : suggestions([...items, ...pending], query));
  let phonesOpen = $derived(showPhones && query.trim().startsWith('+') && phones.length > 0);
  let isOffline = $derived(Boolean(inbox.error));
  let isLoading = $derived(inbox.loading && !inbox.loaded);

  async function focusInbox() { await tick(); document.getElementById('messages-heading')?.focus(); }
  async function navigate(next) {
    view = next; detailOpen = false;
    history.pushState({}, '', next === 'install' ? '/install' : '/');
    if (next === 'install') client.stop(); else client.start();
    await tick();
    document.getElementById(next === 'install' ? 'install-heading' : 'messages-heading')?.focus();
    window.scrollTo({ top: 0, behavior: 'instant' });
  }
  async function popstate() {
    const next = route(); view = next.view; detailOpen = Boolean(next.id);
    if (view === 'install') client.stop();
    else { client.start(next.id); if (next.id) await client.select(next.id); }
    await tick();
    document.getElementById(view === 'install' ? 'install-heading' : detailOpen ? 'detail-title' : 'messages-heading')?.focus();
  }
  function filter() {
    detailOpen = false; history.replaceState({}, '', '/');
    client.setFilters({ q: query, phone: phoneFilter });
  }
  function setQuery(value) {
    query = value; showPhones = true; activePhone = -1; filter();
  }
  function clearSearch() { setQuery(''); searchInput?.focus(); }
  function choosePhone(phone) {
    phoneFilter = phone; query = ''; showPhones = false; activePhone = -1;
    filter(); searchInput?.focus();
    announcement = 'Showing messages ' + phone.field + ' ' + phone.number + '.';
  }
  function clearPhone() { phoneFilter = null; filter(); searchInput?.focus(); }
  function clearFilters() { phoneFilter = null; clearSearch(); }
  function phoneKeys(event) {
    if (event.key === 'Escape' && phonesOpen) {
      event.preventDefault(); event.stopPropagation(); showPhones = false; activePhone = -1;
    } else if ((event.key === 'ArrowDown' || event.key === 'ArrowUp') && phones.length) {
      event.preventDefault(); showPhones = true;
      activePhone = activePhone < 0 ? (event.key === 'ArrowDown' ? 0 : phones.length - 1)
        : (activePhone + (event.key === 'ArrowDown' ? 1 : -1) + phones.length) % phones.length;
    } else if (event.key === 'Enter' && phonesOpen && activePhone >= 0) {
      event.preventDefault(); choosePhone(phones[activePhone]);
    }
  }
  async function retry() { client.connect(); await client.refresh(); await focusInbox(); }
  async function openMessage(id, section = 'message') {
    detailOpen = true; history.pushState({}, '', '/messages/' + encodeURIComponent(id));
    await client.select(id); await tick();
    if (inbox.selectedId !== id) return;
    const heading = document.getElementById(section === 'request' ? 'request-heading' : 'detail-title');
    heading?.focus({ preventScroll: true }); heading?.scrollIntoView({ block: 'start', behavior: 'instant' });
  }
  async function closeMessage() {
    const previousId = inbox.selectedId;
    detailOpen = false; client.leaveDetail();
    history.pushState({}, '', '/');
    await tick();
    document.querySelector('[data-message="' + CSS.escape(previousId) + '"] .message-hit')?.focus();
  }
  function tell(message) {
    toast = message; announcement = message; clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toast = ''; }, 4000);
  }
  async function revealNew() {
    client.reveal(); await focusInbox(); window.scrollTo({ top: 0, behavior: 'instant' }); announcement = 'Newest messages shown.';
  }
  async function askDelete(target) {
    deleteTarget = target; await tick(); confirmDialog.showModal(); cancelButton?.focus();
  }
  async function remove() {
    const all = deleteTarget === 'all';
    if (await client.remove(deleteTarget)) {
      confirmDialog.close(); detailOpen = false; history.replaceState({}, '', '/');
      tell(all ? 'All messages deleted.' : 'Message deleted.'); await focusInbox();
    } else { announcement = 'Deletion failed. Your messages have been kept. Try again.'; }
  }
  function keyboard(event) {
    if (confirmDialog?.open) return;
    if (event.key === '/' && !event.ctrlKey && !event.metaKey && !event.target.closest('input, textarea, select, [contenteditable]')) {
      event.preventDefault(); if (view === 'messages') searchInput?.focus();
    }
    if (event.key === 'Escape') {
      if (event.target === searchInput && query) { setQuery(''); event.preventDefault(); }
      else if (detailOpen) { closeMessage(); event.preventDefault(); }
    }
  }
</script>

<svelte:window onkeydown={keyboard} onpopstate={popstate} />
<svelte:head><title>{view === 'install' ? 'Install' : 'Messages'} · Sundew</title></svelte:head>

<div class="app" data-theme={resolvedTheme}>
  <a class="skip-link" href="#main">Skip to content</a>
  <header class="app-header">
    <a class="wordmark" href="/" onclick={(event) => { event.preventDefault(); navigate('messages'); }}>Sundew<span class="wordmark-period">.</span></a>
    <nav aria-label="Main navigation">
      <a href="/" aria-current={view === 'messages' ? 'page' : undefined} onclick={(event) => { event.preventDefault(); navigate('messages'); }}><Icon name="inbox" size={16} />Messages</a>
      <a href="/install" aria-current={view === 'install' ? 'page' : undefined} onclick={(event) => { event.preventDefault(); navigate('install'); }}><Icon name="book" size={16} />Install</a>
    </nav>
    <span class="connection" class:offline={isOffline || inbox.connection !== 'live'} class:loading={isLoading}><span></span>{isOffline ? 'Disconnected' : inbox.connection === 'live' ? 'Live' : inbox.connection === 'connecting' ? 'Connecting' : inbox.connection === 'disconnected' ? 'Disconnected' : 'Reconnecting'}</span>
  </header>

  <main id="main" class:install-main={view === 'install'}>
    {#if view === 'install'}
      <Install onback={() => navigate('messages')} />
    {:else}
      <div class="page-heading">
        <div><div class="title-row"><h1 id="messages-heading" tabindex="-1">Messages</h1>{#if !isLoading && !isOffline}<span class="message-count">{items.length}{hasMore ? '+' : ''}</span>{/if}</div><p>Every text your application sends, caught here.</p></div>
        <button class="button secondary delete-all" disabled={(!messages.length && !hasMore && !phoneFilter && !query) || inbox.loading || inbox.deleting || !inbox.loaded} onclick={() => askDelete('all')}><Icon name="trash" size={16} /><span>Delete all</span></button>
      </div>
      <div class="inbox-toolbar">
        <div class="search-area">
          <div class="search-field">
            <Icon name="search" size={18} />
            <input bind:this={searchInput} value={query} oninput={(event) => setQuery(event.currentTarget.value)} onfocus={() => { showPhones = true; }} onblur={() => { showPhones = false; }} onkeydown={phoneKeys} type="search" role="combobox" aria-autocomplete="list" aria-expanded={phonesOpen} aria-controls="phone-options" aria-activedescendant={phonesOpen && activePhone >= 0 ? 'phone-option-' + activePhone : undefined} autocomplete="off" spellcheck="false" aria-label="Search to, from, body or account" placeholder="Search messages or + phone…" disabled={isLoading} />
            {#if query}<button class="search-clear" aria-label="Clear search" onclick={clearSearch}><Icon name="close" size={14} /></button>{:else}<kbd>/</kbd>{/if}
          </div>
          {#if phonesOpen}
            <div class="phone-picker">
              <p>Select a phone <span>From matching messages</span></p>
              <div id="phone-options" role="listbox" aria-label="Phone numbers">
                {#each phones as phone, index}
                  <button type="button" role="option" id={'phone-option-' + index} aria-selected={index === activePhone} tabindex="-1" onpointerdown={(event) => event.preventDefault()} onclick={() => choosePhone(phone)}><span class="phone-field">{phone.field === 'to' ? 'To' : 'From'}</span><span>{phone.number}</span><Icon name="arrow" size={14} /></button>
                {/each}
              </div>
            </div>
          {/if}
          {#if phoneFilter}<div class="active-filters"><button class="phone-filter" onclick={clearPhone} aria-label={'Remove phone filter ' + phoneFilter.field + ' ' + phoneFilter.number}><span>{phoneFilter.field === 'to' ? 'To' : 'From'} <strong>{phoneFilter.number}</strong></span><Icon name="close" size={12} /></button></div>{/if}
        </div>
      </div>
      <div class="sr-only" role="status">{(query || phoneFilter) && !isLoading ? (items.length + (hasMore ? ' or more' : '') + ' matching messages loaded.') : ''}</div>
      {#if pending.length}<button class="new-message-banner" onclick={revealNew}>{pending.length} new {pending.length === 1 ? 'message' : 'messages'} <span>Show newest ↑</span></button>{/if}

      {#if inbox.loaded && (isOffline || inbox.connection !== 'live')}
        <div class="connection-notice" role="status"><p>{inbox.error || 'Live connection lost. Reconnecting…'} Existing messages are kept.</p><button class="button secondary" onclick={retry}>Try again</button><a href="/install" onclick={(event) => { event.preventDefault(); navigate('install'); }}>Installation guide</a></div>
      {/if}
      {#if isLoading}
        <section class="state-panel loading-state" aria-label="Loading messages" aria-busy="true">
          <span class="sr-only" role="status">Loading messages…</span>
          {#each Array(5) as _}<div class="skeleton-row" aria-hidden="true"><span></span><span></span><span></span></div>{/each}
        </section>
      {:else if isOffline && !inbox.loaded}
        <section class="state-panel">
          <div class="state-icon error-icon"><Icon name="alert" size={28} /></div>
          <h2>Can't reach Sundew</h2>
          <p>The inbox couldn't be loaded. Check that Sundew is running, then try again.</p>
          <button class="button primary" onclick={retry}>Try again</button>
          <button class="back-link" onclick={() => navigate('install')}>View installation guide<Icon name="arrow" size={15} /></button>
        </section>
      {:else if !items.length && !query && !phoneFilter && !selected}
        <section class="state-panel">
          <div class="state-icon"><Icon name="inbox" size={30} /></div>
          <h2>Ready for your first message</h2>
          <p>Point your application's SMS provider at Sundew. Messages will appear here as they arrive.</p>
          <button class="button primary" onclick={() => navigate('install')}>Set up your application<Icon name="arrow" size={16} /></button>
          <span class="empty-live"><span class="status-dot"></span>Listening for messages</span>
        </section>
      {:else if !items.length && !selected}
        <section class="state-panel">
          <div class="state-icon"><Icon name="search" size={28} /></div>
          <h2>No matching messages</h2><p>{phoneFilter ? 'No messages match this phone and search.' : 'No recipient, sender, message body or account contains “' + query + '”.'}</p>
          <button class="button secondary" onclick={clearFilters}>Clear filters</button>
        </section>
      {:else}
        <Inbox {items} {selected} {detailOpen} {query} {hasMore} onopen={openMessage} onclose={closeMessage} ondelete={askDelete} loading={inbox.loading} deleting={inbox.deleting} onload={() => client.more()} />
      {/if}
    {/if}
  </main>

  <div class="announcer sr-only" role="status" aria-live="polite" aria-atomic="true">{announcement}</div>
  {#if toast}<div class="toast" aria-hidden="true"><Icon name="check" size={16} />{toast}</div>{/if}

  <dialog bind:this={confirmDialog} class="delete-dialog" aria-labelledby="delete-title" aria-describedby="delete-description">
    <div class="dialog-icon"><Icon name="trash" size={24} /></div>
    <h2 id="delete-title">{deleteTarget === 'all' ? 'Delete all messages?' : 'Delete this message?'}</h2>
    <p id="delete-description">{deleteTarget === 'all' ? 'Every caught message will be deleted, including messages not loaded or hidden by search. This cannot be undone.' : 'The message to ' + (deleteTarget?.to ?? '') + ' will be permanently removed from this inbox.'}</p>
    {#if inbox.error}<p role="alert">{inbox.error}</p>{/if}
    <div class="dialog-actions"><button class="button secondary" bind:this={cancelButton} disabled={inbox.deleting} onclick={() => confirmDialog.close()}>Cancel</button><button class="button danger" disabled={inbox.deleting} onclick={remove}>{deleteTarget === 'all' ? 'Delete all messages' : 'Delete message'}</button></div>
  </dialog>
</div>
