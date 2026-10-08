<script>
  import Icon from './Icon.svelte';
  let { variant, theme, scenario, onvariant, ontheme, onscenario, onarrival } = $props();
  const variants = ['A', 'B', 'C'];
  const names = { A: 'Split inbox', B: 'Compact list', C: 'Message stream' };
  function cycle(delta) { onvariant(variants[(variants.indexOf(variant) + delta + 3) % 3]); }
  function keys(event) {
    if (event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return;
    if (event.target.closest('input, textarea, select, [contenteditable], [role=tablist], dialog')) return;
    if (event.key === 'ArrowLeft') { event.preventDefault(); cycle(-1); }
    if (event.key === 'ArrowRight') { event.preventDefault(); cycle(1); }
  }
</script>

<svelte:window onkeydown={keys} />
<aside class="prototype-switcher" aria-label="Prototype controls">
  <span class="prototype-label">Prototype <span>· mock data</span></span>
  <div class="variant-switch">
    <button aria-label="Previous layout" onclick={() => cycle(-1)}><Icon name="back" size={16} /></button>
    <span><b>{variant}</b> {names[variant]}</span>
    <button aria-label="Next layout" onclick={() => cycle(1)}><Icon name="arrow" size={16} /></button>
  </div>
  <select aria-label="Preview state" value={scenario} onchange={(event) => onscenario(event.currentTarget.value)}>
    <option value="standard">Normal inbox</option>
    <option value="loading">Loading</option>
    <option value="empty">Empty</option>
    <option value="offline">API unreachable</option>
    <option value="long">Long message</option>
    <option value="many">240 messages</option>
    <option value="inbound">Inbound message</option>
    <option value="failed">Failed message</option>
  </select>
  <button class="arrival-control" onclick={onarrival}>+ Arrival</button>
  <button class="theme-control" aria-label={theme === 'light' ? 'Preview dark theme' : 'Preview light theme'} onclick={() => ontheme(theme === 'light' ? 'dark' : 'light')}><Icon name={theme === 'light' ? 'moon' : 'sun'} size={17} /></button>
</aside>
