<script>
  import Icon from './Icon.svelte';
  import Highlight from './Highlight.svelte';
  import LinkedText from './LinkedText.svelte';
  import { time, dateTime } from './data.js';
  let { message, selected = false, query = '', onopen } = $props();
</script>

<article class="message-row" class:selected data-message={message.id}>
  <button class="message-hit" type="button" onclick={() => onopen(message.id, 'message')} aria-label={'View message to ' + message.to + ': ' + message.body} aria-pressed={selected}></button>
  <div class="message-row-content">
    <span class="row-recipient"><span class="eyebrow">To</span><strong><Highlight text={message.to} {query} /></strong></span>
    <time class="row-time" datetime={message.created_at} title={dateTime(message.created_at)}>{time(message.created_at)}</time>
    <span class="row-body"><LinkedText text={message.body} {query} /></span>
    <span class="row-sender"><span>From</span> <Highlight text={message.from} {query} /></span>
    <span class="row-meta">
      <span class={'status status-' + message.status}><span class="status-dot"></span>{message.status}</span>
      <span class="row-provider">{message.provider}</span>
      <span class="row-direction"><Icon name={message.direction === 'inbound' ? 'incoming' : 'outgoing'} size={13} />{message.direction}</span>
    </span>
  </div>
  <div class="row-actions">
    <button type="button" class="raw-button" onclick={() => onopen(message.id, 'request')} aria-label={'View raw request for message to ' + message.to}><Icon name="code" size={14} /><span>Raw</span></button>
  </div>
</article>
