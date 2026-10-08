<script>
  import MessageRow from './MessageRow.svelte';
  import MessageDetail from './MessageDetail.svelte';
  import Icon from './Icon.svelte';
  let { items, selected, detailOpen, tab, query, onopen, onclose, ondelete, ontab, hasMore, onload } = $props();
</script>

{#if detailOpen && selected}
  <div class="ledger-detail">
    <button class="back-link" onclick={onclose}><Icon name="back" size={17} />All messages</button>
    <MessageDetail message={selected} {tab} onchange={ontab} {onclose} {ondelete} />
  </div>
{:else}
  <section class="ledger-list" aria-label="Caught messages" id="message-list-scroll">
    <div class="ledger-labels" aria-hidden="true"><span>Recipient / sender</span><span>Message</span><span>Status</span><span>Time</span><span></span></div>
    {#each items as message (message.id)}
      <MessageRow {message} {query} mode="ledger" {onopen} {ondelete} />
    {/each}
    <div class="list-end">{#if hasMore}<button class="button secondary" onclick={onload}>Load 50 more</button>{:else}<span>All caught messages shown</span>{/if}</div>
  </section>
{/if}
