<script>
  import Highlight from './Highlight.svelte';
  import { linkParts } from './links.js';
  let { text = '', query = '' } = $props();
  let parts = $derived(linkParts(text));
</script>

{#each parts as part}{#if part.href}<a class="content-link" href={part.href} target="_blank" rel="noopener noreferrer" aria-label={part.text + ' (opens in a new tab)'}><Highlight text={part.text} {query} /></a>{:else}<Highlight text={part.text} {query} />{/if}{/each}
