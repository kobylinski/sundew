<script>
  import MessageRow from './MessageRow.svelte';
  import MessageDetail from './MessageDetail.svelte';
  let { items, selected, detailOpen, tab, query, onopen, onclose, ondelete, ontab, hasMore, onload } = $props();
</script>

<section class="message-stream" aria-label="Caught messages" id="message-list-scroll">
  <div class="stream-date"><span>8 October 2026</span><span>Newest first</span></div>
  {#each items as message (message.id)}
    <div class="stream-entry" class:expanded={detailOpen && selected?.id === message.id}>
      <MessageRow {message} {query} mode="stream" selected={detailOpen && selected?.id === message.id} {onopen} {ondelete} />
      {#if detailOpen && selected?.id === message.id}
        <div class="stream-inspector"><MessageDetail {message} {tab} onchange={ontab} {onclose} {ondelete} /></div>
      {/if}
    </div>
  {/each}
  <div class="list-end">{#if hasMore}<button class="button secondary" onclick={onload}>Load 50 more</button>{:else}<span>All caught messages shown</span>{/if}</div>
</section>
