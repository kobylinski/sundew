<script>
  import Icon from './Icon.svelte';
  import CopyButton from './CopyButton.svelte';
  import { dateTime, rawRequest, rawResponse } from './data.js';
  let { message, tab = 'message', onchange, onclose, ondelete } = $props();
  const tabs = ['message', 'request', 'response'];
  function tabKeys(event) {
    let index = tabs.indexOf(tab);
    if (event.key === 'ArrowRight') index = (index + 1) % tabs.length;
    else if (event.key === 'ArrowLeft') index = (index + tabs.length - 1) % tabs.length;
    else if (event.key === 'Home') index = 0;
    else if (event.key === 'End') index = tabs.length - 1;
    else return;
    event.preventDefault();
    event.stopPropagation();
    onchange(tabs[index]);
    event.currentTarget.querySelectorAll('[role=tab]')[index].focus();
  }
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
  <div class="detail-tabs" role="tablist" tabindex="-1" aria-label="Message information" onkeydown={tabKeys}>
    {#each tabs as item}
      <button type="button" role="tab" id={'tab-' + item} aria-selected={tab === item} aria-controls={'panel-' + item} tabindex={tab === item ? 0 : -1} onclick={() => onchange(item)}>
        <Icon name={item === 'message' ? 'message' : item === 'request' ? 'code' : 'response'} size={15} />
        {item === 'message' ? 'Message' : item === 'request' ? 'Raw request' : 'Response'}
      </button>
    {/each}
  </div>

  <div role="tabpanel" id={'panel-' + tab} aria-labelledby={'tab-' + tab} tabindex="0" class="detail-content">
    {#if tab === 'message'}
      <div class="message-copy">
        <div class="body-label"><span class="eyebrow">Message content</span><CopyButton text={message.body} label="Copy text" /></div>
        <p class="message-body">{message.body || 'No text content'}</p>
        <div class="content-meta"><Icon name="message" size={13} /><span>{message.segments} {message.segments === 1 ? 'segment' : 'segments'}</span>{#if media.length}<span class="content-divider">·</span><span>{media.length} {media.length === 1 ? 'attachment' : 'attachments'}</span>{/if}</div>
      </div>
      {#if message.error_code}
        <div class="failure-note"><Icon name="alert" /><div><strong>Delivery failed</strong><p>Provider error code <code>{message.error_code}</code>.</p></div></div>
      {/if}
      <h3 class="section-title">Message details</h3>
      <dl class="message-metadata">
        <div class="metadata-half"><dt>From</dt><dd>{message.from}</dd></div>
        <div class="metadata-half"><dt>To</dt><dd>{message.to}</dd></div>
        <div class="metadata-half"><dt>Caught</dt><dd><time datetime={message.created_at}>{dateTime(message.created_at)}</time></dd></div>
        <div class="metadata-half"><dt>Updated</dt><dd><time datetime={message.updated_at}>{dateTime(message.updated_at)}</time></dd></div>
        <div><dt>Account</dt><dd><code>{message.account}</code><CopyButton text={message.account} label="Copy account" compact /></dd></div>
        <div><dt>Message ID</dt><dd><code>{message.id}</code><CopyButton text={message.id} label="Copy message ID" compact /></dd></div>
        <div><dt>Provider ID</dt><dd>{#if message.provider_id}<code>{message.provider_id}</code><CopyButton text={message.provider_id} label="Copy provider ID" compact />{:else}<span class="muted">Not recorded</span>{/if}</dd></div>
      </dl>
      <section class="detail-section" aria-label="Provider options">
        <h3>Provider options</h3>
        {#if Object.keys(options).length}
          <dl class="option-list">
            {#each Object.entries(options) as [key, value]}<div><dt>{key}</dt><dd>{value}</dd></div>{/each}
          </dl>
        {:else}<p class="muted">No provider options.</p>{/if}
      </section>
      <section class="detail-section" aria-label="Media URLs">
        <h3>Media URLs <span class="quiet-count">{media.length}</span></h3>
        {#if media.length}
          {#each media as url}<a class="media-link" href={url} target="_blank" rel="noreferrer">{url}<Icon name="external" size={14} /></a>{/each}
        {:else}<p class="muted">No media attached.</p>{/if}
      </section>
    {:else if tab === 'request'}
      <div class="body-label"><span class="eyebrow">Request as received</span>{#if request}<CopyButton text={request} label="Copy request" />{/if}</div>
      {#if request}
        <pre class="raw-code" aria-label="Raw HTTP request">{request}</pre>
      {:else}<p class="muted">No provider request was recorded for this simulated inbound message.</p>{/if}
    {:else}
      <div class="body-label"><span class="eyebrow">Response Sundew returned</span>{#if response}<CopyButton text={response} label="Copy response" />{/if}</div>
      {#if response}
        <span class="response-code">{message.exchange.response_status}</span>
        <pre class="raw-code" aria-label="Raw HTTP response">{response}</pre>
        <p class="raw-caption">The original response stays the same as delivery status changes.</p>
      {:else}<p class="muted">No provider response was recorded for this simulated inbound message.</p>{/if}
    {/if}
  </div>
  <footer class="detail-footer"><button type="button" class="text-danger" onclick={() => ondelete(message)}><Icon name="trash" size={15} />Delete message</button></footer>
</section>
