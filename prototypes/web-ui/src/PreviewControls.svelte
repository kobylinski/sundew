<script>
  let { theme, scenario, ontheme, onscenario, onarrival } = $props();
</script>

<aside class="prototype-switcher" aria-label="Prototype controls">
  <span class="prototype-label">Prototype <span>· mock data</span></span>
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
  <select aria-label="Preview theme" value={theme} onchange={(event) => ontheme(event.currentTarget.value)}>
    <option value="auto">System theme</option>
    <option value="light">Light</option>
    <option value="dark">Dark</option>
  </select>
  <button class="arrival-control" onclick={onarrival}>+ Arrival</button>
</aside>
