<script>
  let { text = '', query = '' } = $props();
  let at = $derived(query.trim() ? text.toLowerCase().indexOf(query.trim().toLowerCase()) : -1);
</script>

{#if at < 0}{text}{:else}{text.slice(0, at)}<mark>{text.slice(at, at + query.trim().length)}</mark>{text.slice(at + query.trim().length)}{/if}
