<script>
  import Icon from './Icon.svelte';
  import CopyButton from './CopyButton.svelte';
  import LinkedText from './LinkedText.svelte';
  import { webHref } from './links.js';
  import { dateTime, rawRequest, rawResponse } from './format.js';
  let { message, onclose, ondelete, deleting } = $props();
  let request = $derived(rawRequest(message));
  let response = $derived(rawResponse(message));
  let options = $derived(message.options ?? {});
  let media = $derived(message.media_urls ?? []);
</script>

<section class="message-detail" aria-label="Message details">
  <header class="detail-heading">
    <div><span class="eyebrow">{message.direction === 'inbound' ? 'Incoming message' : 'Message to'}</span><h2 tabindex="-1" id="detail-title">{message.to}</h2><p class="detail-sender">From <span>{message.from}</span></p></div>
    <button type="button" class="icon-button close-detail" onclick={onclose} aria-label="Back to messages" title="Back to messages"><Icon name="close" /></button>
  </header>
  <div class="detail-intro">
    <span class={'status status-' + message.status}><span class="status-dot"></span>{message.status}</span>
    <span class="direction"><Icon name={message.direction === 'inbound' ? 'incoming' : 'outgoing'} size={14} />{message.direction}</span>
    <span class="provider-tag">{message.provider}</span>
  </div>
  <div class="detail-tools" role="group" aria-label="Message tools">
    <CopyButton text={message.body} label="Copy text" />
    <button type="button" class="text-danger" disabled={deleting} onclick={() => ondelete(message)}><Icon name="trash" size={15} />Delete message</button>
  </div>
  <section class="message-copy" aria-labelledby="content-heading">
    <h3 id="content-heading" class="section-title">Message content</h3>
    <p class="message-body"><LinkedText text={message.body || 'No text content'} /></p>
    <div class="content-meta"><Icon name="message" size={13} /><span>{message.segments} {message.segments === 1 ? 'segment' : 'segments'}</span>{#if media.length}<span class="content-divider">·</span><span>{media.length} {media.length === 1 ? 'attachment' : 'attachments'}</span>{/if}</div>
  </section>
  {#if message.error_code}
    <div class="failure-note"><Icon name="alert" /><div><strong>Delivery failed</strong><p>Provider error code <code>{message.error_code}</code>.</p></div></div>
  {/if}
  <section class="detail-section" aria-labelledby="metadata-heading">
    <h3 id="metadata-heading">Message details</h3>
    <dl class="message-metadata">
      <div class="metadata-half"><dt>From</dt><dd>{message.from}</dd></div>
      <div class="metadata-half"><dt>To</dt><dd>{message.to}</dd></div>
      <div class="metadata-half"><dt>Caught</dt><dd><time datetime={message.created_at}>{dateTime(message.created_at)}</time></dd></div>
      <div class="metadata-half"><dt>Updated</dt><dd><time datetime={message.updated_at}>{dateTime(message.updated_at)}</time></dd></div>
      <div><dt>Account</dt><dd><code>{message.account}</code><CopyButton text={message.account} label="Copy account" compact /></dd></div>
      <div><dt>Message ID</dt><dd><code>{message.id}</code><CopyButton text={message.id} label="Copy message ID" compact /></dd></div>
      <div><dt>Provider ID</dt><dd>{#if message.provider_id}<code>{message.provider_id}</code><CopyButton text={message.provider_id} label="Copy provider ID" compact />{:else}<span class="muted">Not recorded</span>{/if}</dd></div>
    </dl>
  </section>
  <section class="detail-section" aria-label="Provider options">
    <h3>Provider options</h3>
    {#if Object.keys(options).length}
      <dl class="option-list">
        {#each Object.entries(options) as [key, value]}
          <div><dt>{key}</dt><dd>{#if key === 'status_callback' && webHref(value)}<a class="content-link" href={webHref(value)} target="_blank" rel="noopener noreferrer" aria-label={value + ' (opens in a new tab)'}>{value}</a>{:else}{value}{/if}</dd></div>
        {/each}
      </dl>
    {:else}<p class="muted">No provider options.</p>{/if}
  </section>
  <section class="detail-section" aria-label="Media URLs">
    <h3>Media URLs <span class="quiet-count">{media.length}</span></h3>
    {#if media.length}
      {#each media as url}
        {#if webHref(url)}<a class="media-link" href={webHref(url)} target="_blank" rel="noopener noreferrer" aria-label={url + ' (opens in a new tab)'}>{url}<Icon name="external" size={14} /></a>{:else}<span class="media-text">{url}</span>{/if}
      {/each}
    {:else}<p class="muted">No media attached.</p>{/if}
  </section>
  <section class="detail-section" aria-labelledby="request-heading">
    <div class="body-label"><h3 id="request-heading" tabindex="-1">Raw request</h3>{#if request}<CopyButton text={request} label="Copy request" />{/if}</div>
    {#if request}
      <pre class="raw-code" aria-label="Raw HTTP request">{request}</pre>
    {:else}<p class="muted">No provider request was recorded for this simulated inbound message.</p>{/if}
  </section>
  <section class="detail-section" aria-labelledby="response-heading">
    <div class="body-label"><h3 id="response-heading" tabindex="-1">Raw response</h3>{#if response}<CopyButton text={response} label="Copy response" />{/if}</div>
    {#if response}
      <span class="response-code">{message.exchange.response_status}</span>
      <pre class="raw-code" aria-label="Raw HTTP response">{response}</pre>
      <p class="raw-caption">The original response stays the same as delivery status changes.</p>
    {:else}<p class="muted">No provider response was recorded for this simulated inbound message.</p>{/if}
  </section>
</section>
