<script>
  import MessageRow from './MessageRow.svelte';
  import MessageDetail from './MessageDetail.svelte';
  import Icon from './Icon.svelte';
  let { items, selected, detailOpen, tab, query, onopen, onclose, ondelete, ontab, hasMore, onload } = $props();
</script>

<div class="split-inbox" class:has-selection={detailOpen}>
  <section class="inbox-list" aria-label="Caught messages" id="message-list-scroll">
    {#each items as message (message.id)}
      <MessageRow {message} {query} selected={selected?.id === message.id} {onopen} {ondelete} />
    {/each}
    <div class="list-end">{#if hasMore}<button class="button secondary" onclick={onload}>Load 50 more</button>{:else}<span>All caught messages shown</span>{/if}</div>
  </section>
  <aside class="detail-pane">
    {#if selected}
      <MessageDetail message={selected} {tab} onchange={ontab} {onclose} {ondelete} />
    {:else}
      <div class="choose-message"><Icon name="inbox" size={28} /><p>Select a message to inspect it.</p></div>
    {/if}
  </aside>
</div>
