<script>
  import Icon from './Icon.svelte';
  let { text, label = 'Copy', compact = false } = $props();
  let copied = $state(false);
  let failed = $state(false);
  $effect(() => {
    // A different message or command must not inherit the previous copy status.
    void text;
    copied = false;
    failed = false;
  });
  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      copied = true;
      failed = false;
      setTimeout(() => copied = false, 1800);
    } catch {
      failed = true;
    }
  }
</script>

<button type="button" class:icon-button={compact} class:copy-button={!compact} onclick={copy} aria-label={copied ? 'Copied' : label} title={label}>
  <Icon name={copied ? 'check' : 'copy'} size={15} />
  {#if !compact}<span>{copied ? 'Copied' : label}</span>{/if}
</button>
<span class="sr-only" role="status">{copied ? label + ' completed.' : failed ? 'Clipboard unavailable. Select the text to copy it.' : ''}</span>
