<script>
  import CopyButton from './CopyButton.svelte';
  import Icon from './Icon.svelte';
  import { image } from './install.js';
  const account = 'AC11111111111111111111111111111111';
  let { onback } = $props();
  let format = $state('docker');
  const docker = 'docker run --rm -p 8025:8025 \\\n  "' + image + '"';
  const compose = [
    'services:', '  sundew:', '    image: "' + image + '"',
    '    ports:', '      - "8025:8025"', '',
    '  # Merge into your existing application service:',
    '  app:', '    environment:', '      TWILIO_API_BASE: http://sundew:8025',
  ].join('\n');
  const send = [
    'curl -u "' + account + ':development" \\',
    '  --data-urlencode "From=+15550001001" \\',
    '  --data-urlencode "To=+15551234567" \\',
    '  --data-urlencode "Body=Your sign-in code is 482913." \\',
    '  "http://localhost:8025/2010-04-01/Accounts/' + account + '/Messages.json"',
  ].join('\n');
  const read = [
    'curl --get \\',
    '  --data-urlencode "to=+15551234567" \\',
    '  http://localhost:8025/api/v1/messages/latest',
  ].join('\n');
  const env = [
    ['SUNDEW_ADDR', 'Listening address.', ':8025'],
    ['SUNDEW_BASE_URL', 'Base URL in the links returned by provider façades.', ''],
    ['SUNDEW_CALLBACK_DELAY', 'Delay between simulated status callbacks. For example, 1s.', ''],
    ['SUNDEW_CALLBACK_OUTCOME', 'Final delivery outcome: delivered or failed.', 'delivered'],
    ['SUNDEW_INBOUND_URL', "Your application's inbound SMS webhook URL.", ''],
  ];
</script>

<div class="install-layout">
  <aside class="install-aside">
    <p class="eyebrow">Quick start</p>
    <h1 id="install-heading" tabindex="-1">Your SMS.<br /> Right here.</h1>
    <p>Point your application at Sundew. Every message it sends lands in your inbox, ready to inspect or read from a test.</p>
    <p class="install-note">Messages stay in memory until Sundew stops. Nothing reaches a phone.</p>
    <button class="back-link" onclick={onback}><Icon name="back" size={16} />Back to messages</button>
  </aside>
  <div class="install-steps">
    <section class="install-step">
      <span class="step-number">01</span>
      <div class="step-content">
        <h2>Run Sundew</h2>
        <p>Replace <code>&lt;namespace&gt;</code> with your Docker Hub namespace and <code>&lt;version&gt;</code> with a release tag. Open <code>http://localhost:8025</code> to see your messages.</p>
        <div class="code-toolbar">
          <div class="format-switch" aria-label="Run example">
            <button class:active={format === 'docker'} aria-pressed={format === 'docker'} onclick={() => format = 'docker'}>Docker</button>
            <button class:active={format === 'compose'} aria-pressed={format === 'compose'} onclick={() => format = 'compose'}>Compose</button>
          </div>
          <CopyButton text={format === 'docker' ? docker : compose} label="Copy command" />
        </div>
        <pre class="install-code" aria-label={format === 'docker' ? 'Docker command' : 'Compose configuration'}>{format === 'docker' ? docker : compose}</pre>
        {#if format === 'compose'}<p class="code-hint">Merge this snippet into your Compose file; <code>app</code> is your existing application service.</p>{/if}
      </div>
    </section>
    <section class="install-step">
      <span class="step-number">02</span>
      <div class="step-content">
        <h2>Point your Twilio client here</h2>
        <p>Set the base URL in your application's Twilio adapter. Use any account SID and token for development.</p>
        <dl class="endpoint-list">
          <div><dt>App runs on your machine</dt><dd><code>http://localhost:8025</code><CopyButton text="http://localhost:8025" label="Copy local base URL" compact /></dd></div>
          <div><dt>App runs in the same Compose network</dt><dd><code>http://sundew:8025</code><CopyButton text="http://sundew:8025" label="Copy Compose base URL" compact /></dd></div>
        </dl>
        <p class="code-hint"><code>TWILIO_API_BASE</code> is an example application variable. Your adapter must use it to override the SDK's base URL.</p>
        <details class="try-request">
          <summary>Try a request with curl</summary>
          <div class="code-toolbar"><span>Send a development message</span><CopyButton text={send} label="Copy request" /></div>
          <pre class="install-code" aria-label="Example send request">{send}</pre>
        </details>
      </div>
    </section>
    <section class="install-step">
      <span class="step-number">03</span>
      <div class="step-content">
        <h2>Read a message from your test</h2>
        <p>Ask for the latest message sent to a number. URL-encode the <code>+</code>; curl does that for you here.</p>
        <div class="code-toolbar"><span>Latest message</span><CopyButton text={read} label="Copy query" /></div>
        <pre class="install-code" aria-label="Latest message query">{read}</pre>
        <p class="code-hint">Use <code>GET /api/v1/messages/stream</code> to wait for new messages. <code>POST /api/v1/reset</code> empties the store between tests. The API reference is served at <code>/api/v1/openapi.json</code>.</p>
      </div>
    </section>
    <section class="install-step">
      <span class="step-number">04</span>
      <div class="step-content">
        <h2>Configure when you need to</h2>
        <p>Pass environment variables to the container. There are no accounts to create or database volumes to configure.</p>
        <div class="config-list">
          {#each env as [name, description, defaultValue]}
            <div><code>{name}</code><p>{description}{#if defaultValue}<span class="default-value">{' '}Default: <code>{defaultValue}</code></span>{/if}</p></div>
          {/each}
        </div>
      </div>
    </section>
  </div>
</div>
