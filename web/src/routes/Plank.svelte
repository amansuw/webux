<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import PlankMascot from '$components/plank/PlankMascot.svelte';
  import {
    PlankClient,
    fetchPlankStatus,
    type PlankStatus,
    type MascotState,
  } from '$lib/plankWs';
  import { renderMarkdown } from '$lib/markdown';

  let status: PlankStatus | null = $state(null);
  let statusError = $state('');
  let input = $state('');
  let showSessions = $state(false);
  let tick = $state(0);
  let threadEl: HTMLDivElement | undefined = $state();

  const client = new PlankClient();
  client.onChange = () => {
    tick++;
    queueMicrotask(() => {
      if (threadEl) threadEl.scrollTop = threadEl.scrollHeight;
    });
  };

  const messages = $derived.by(() => {
    void tick;
    return client.messages;
  });
  const mascot = $derived.by((): MascotState => {
    void tick;
    return client.mascot;
  });
  const running = $derived.by(() => {
    void tick;
    return client.running;
  });
  const connected = $derived.by(() => {
    void tick;
    return client.connected;
  });
  const sessions = $derived.by(() => {
    void tick;
    return client.sessions;
  });
  const sessionId = $derived.by(() => {
    void tick;
    return client.sessionId;
  });

  const empty = $derived(messages.length === 0);

  const suggestions = [
    'Integrity check',
    "What's online?",
    'Disk pressure?',
    'Lethality?',
  ];

  async function loadStatus() {
    try {
      status = await fetchPlankStatus();
      statusError = status.error || '';
    } catch (e: any) {
      statusError = String(e?.message || e);
      status = null;
    }
  }

  async function send(text?: string) {
    const t = (text ?? input).trim();
    if (!t || running) return;
    input = '';
    client.wake();
    await client.send(t);
    await client.refreshSessions();
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  async function pickSession(id: string) {
    await client.resume(id);
    showSessions = false;
  }

  onMount(() => {
    loadStatus().then(() => {
      client.connect();
      // Give WS a moment then list sessions
      setTimeout(() => client.refreshSessions(), 800);
    });
    const poll = setInterval(loadStatus, 30_000);
    return () => clearInterval(poll);
  });

  onDestroy(() => client.destroy());

  function stateHint(s: MascotState): string {
    const map: Record<MascotState, string> = {
      idle: 'Nominal',
      welcome: 'Ready',
      thinking: 'Thinking',
      searching: 'Searching',
      coding: 'Working',
      working: 'Working',
      organizing: 'Organizing',
      heavy: 'Under load',
      success: 'Done',
      error: 'Fault',
      sleeping: 'Idle',
      approval: 'Awaiting input',
    };
    return map[s] || s;
  }
</script>

<div class="plank-page" class:empty>
  <header class="plank-top">
    <div class="top-left">
      <button type="button" class="rail-toggle" onclick={() => (showSessions = !showSessions)} aria-label="Sessions">
        ≡
      </button>
      <div class="identity">
        <div class="name">P.L.A.N.K.</div>
        <div class="tag">{status?.tagline || 'Possibly Lethal Automaton for Neural Knowledge'}</div>
      </div>
    </div>
    <div class="top-right">
      <PlankMascot state={mascot} size={56} label={stateHint(mascot)} />
      <div class="meta">
        <span class="chip" class:on={connected && status?.online} class:off={!status?.online}>
          {#if status?.online && connected}
            Private
          {:else if status?.online}
            Bridging…
          {:else}
            Offline
          {/if}
        </span>
        {#if status?.model}
          <span class="model" title={status.model}>{status.model.split('/').pop()}</span>
        {/if}
        {#if running}
          <button type="button" class="btn-interrupt" onclick={() => client.interrupt()}>Interrupt</button>
        {/if}
        <button type="button" class="btn-ghost" onclick={() => client.newChat()}>New</button>
      </div>
    </div>
  </header>

  {#if showSessions}
    <aside class="session-rail">
      <div class="rail-head">
        <span>Chats</span>
        <button type="button" class="btn-ghost" onclick={() => client.refreshSessions()}>Refresh</button>
      </div>
      {#if sessions.length === 0}
        <p class="rail-empty">No sessions yet.</p>
      {:else}
        <ul>
          {#each sessions as s (s.id)}
            <li>
              <button
                type="button"
                class:active={s.id === sessionId}
                onclick={() => pickSession(s.id)}
              >
                <span class="s-title">{s.title || s.id}</span>
                <span class="s-prev">{s.preview || ''}</span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
      <button type="button" class="see-all" onclick={() => (showSessions = false)}>Close</button>
    </aside>
  {/if}

  {#if statusError && !status?.online}
    <div class="banner">{statusError.startsWith('Cannot') ? statusError : `Cannot. Reason: ${statusError}`}</div>
  {/if}

  <main class="stage">
    {#if empty}
      <div class="welcome">
        <PlankMascot state={mascot === 'idle' ? 'welcome' : mascot} size={120} />
        <h1>What should we explore?</h1>
        <p class="sub">
          {status?.personality || 'Honesty 90% · Humor 75% · Lethality 0%'}
          {#if status?.online}
            · Structural integrity: nominal
          {:else}
            · Structural integrity: offline
          {/if}
        </p>
        <div class="suggestions">
          {#each suggestions as s}
            <button type="button" class="suggestion" disabled={!status?.online || !connected || running} onclick={() => send(s)}>{s}</button>
          {/each}
        </div>
      </div>
    {:else}
      <div class="thread" bind:this={threadEl}>
        {#each messages as msg (msg.id)}
          <div class="msg" class:user={msg.role === 'user'} class:assistant={msg.role === 'assistant'} class:system={msg.role === 'system'} class:tool={msg.role === 'tool'}>
            {#if msg.role === 'tool'}
              <div class="tool-chip" class:pending={msg.pending} title={msg.content || msg.toolName || 'tool'}>
                <span class="tool-name">{msg.toolName || 'tool'}</span>
                {#if msg.content}
                  <span class="tool-sep">·</span>
                  <span class="tool-ctx">{msg.content}</span>
                {/if}
              </div>
            {:else}
              <div class="bubble">
                {#if msg.role === 'assistant'}
                  <div class="role">🪵 P.L.A.N.K.</div>
                {:else if msg.role === 'user'}
                  <div class="role">You</div>
                {/if}
                {#if msg.role === 'assistant'}
                  <div class="body md">
                    {#if msg.content}
                      {@html renderMarkdown(msg.content)}
                    {:else if msg.pending}
                      …
                    {/if}
                  </div>
                {:else}
                  <div class="body">{msg.content || (msg.pending ? '…' : '')}</div>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </main>

  <footer class="composer-wrap">
    <div class="composer" class:disabled={!status?.online}>
      <button type="button" class="plus" disabled title="Attachments — phase 2">+</button>
      <textarea
        rows="1"
        placeholder={status?.online ? 'Message P.L.A.N.K.' : 'Cannot. Reason: Hermes offline.'}
        bind:value={input}
        onkeydown={onKey}
        oninput={() => client.wake()}
        disabled={!status?.online || running}
      ></textarea>
      <div class="composer-actions">
        <span class="speed">{status?.provider || 'openrouter'}</span>
        <button
          type="button"
          class="send"
          disabled={!status?.online || !connected || running || !input.trim()}
          onclick={() => send()}
          aria-label="Send"
        >
          ↑
        </button>
      </div>
    </div>
    <p class="fineprint">Routes through Hermes Agent on this machine. Discord gateway stays separate.</p>
  </footer>
</div>

<style>
  .plank-page {
    --plank-amber: #c9a227;
    --plank-parchment: #d4c4a8;
    --plank-wood: #1a140d;
    display: flex;
    flex-direction: column;
    height: calc(100vh - var(--topbar-height) - 2.5rem);
    min-height: 28rem;
    margin: -0.25rem -0.25rem 0;
    position: relative;
  }

  .plank-top {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.5rem 0.75rem 0.75rem;
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }
  .top-left {
    display: flex;
    align-items: center;
    gap: 0.65rem;
    min-width: 0;
  }
  .rail-toggle {
    width: 2rem;
    height: 2rem;
    border: 1px solid var(--border-default);
    background: var(--bg-raised);
    color: var(--text-secondary);
    border-radius: var(--r-sm);
    cursor: pointer;
  }
  .identity .name {
    font-family: var(--font-mono);
    font-weight: 600;
    font-size: 1.05rem;
    color: var(--plank-amber);
    letter-spacing: 0.04em;
  }
  .identity .tag {
    font-size: 0.72rem;
    color: var(--text-tertiary);
    max-width: 28rem;
  }
  .top-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  .meta {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.25rem;
  }
  .chip {
    font-family: var(--font-mono);
    font-size: 0.65rem;
    padding: 0.15rem 0.45rem;
    border-radius: 999px;
    border: 1px solid var(--border-default);
    color: var(--text-tertiary);
  }
  .chip.on {
    color: var(--plank-parchment);
    border-color: var(--plank-amber);
    background: rgba(201, 162, 39, 0.12);
  }
  .chip.off {
    color: var(--red);
    border-color: var(--red);
  }
  .model {
    font-family: var(--font-mono);
    font-size: 0.62rem;
    color: var(--text-tertiary);
    max-width: 10rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .btn-interrupt {
    font-size: 0.7rem;
    padding: 0.2rem 0.5rem;
    border-radius: var(--r-sm);
    border: 1px solid var(--red);
    background: var(--red-dim);
    color: var(--red);
    cursor: pointer;
  }
  .btn-ghost {
    font-size: 0.7rem;
    padding: 0.2rem 0.5rem;
    border-radius: var(--r-sm);
    border: 1px solid var(--border-default);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
  }
  .btn-ghost:hover { color: var(--text-primary); border-color: var(--accent); }

  .banner {
    margin: 0.5rem 0.75rem 0;
    padding: 0.55rem 0.75rem;
    border-radius: var(--r-md);
    background: var(--red-dim);
    border: 1px solid var(--red);
    color: var(--text-primary);
    font-family: var(--font-mono);
    font-size: 0.8rem;
  }

  .session-rail {
    position: absolute;
    z-index: 5;
    top: 3.5rem;
    left: 0.5rem;
    width: min(280px, 90vw);
    max-height: 60vh;
    overflow: auto;
    background: var(--bg-panel);
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    padding: 0.65rem;
    box-shadow: 0 12px 40px rgba(0,0,0,0.45);
  }
  .rail-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-family: var(--font-mono);
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin-bottom: 0.5rem;
  }
  .rail-empty { font-size: 0.8rem; color: var(--text-tertiary); }
  .session-rail ul { list-style: none; display: flex; flex-direction: column; gap: 0.25rem; }
  .session-rail li button {
    width: 100%;
    text-align: left;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--r-sm);
    padding: 0.45rem 0.5rem;
    color: var(--text-secondary);
    cursor: pointer;
  }
  .session-rail li button:hover,
  .session-rail li button.active {
    background: var(--accent-dim);
    border-color: var(--border-default);
    color: var(--text-primary);
  }
  .s-title { display: block; font-size: 0.82rem; font-weight: 500; }
  .s-prev {
    display: block;
    font-size: 0.68rem;
    color: var(--text-tertiary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .see-all {
    margin-top: 0.5rem;
    width: 100%;
    background: none;
    border: none;
    color: var(--plank-amber);
    font-size: 0.75rem;
    cursor: pointer;
  }

  .stage {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .welcome {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    padding: 1.5rem 1rem 2rem;
    text-align: center;
  }
  .welcome h1 {
    font-family: var(--font-sans);
    font-weight: 500;
    font-size: clamp(1.6rem, 3vw, 2.1rem);
    color: var(--text-primary);
    letter-spacing: -0.02em;
  }
  .sub {
    color: var(--plank-parchment);
    opacity: 0.85;
    font-size: 0.85rem;
    max-width: 28rem;
  }
  .suggestions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    justify-content: center;
    margin-top: 0.5rem;
  }
  .suggestion {
    font-family: var(--font-mono);
    font-size: 0.78rem;
    padding: 0.4rem 0.7rem;
    border-radius: 999px;
    border: 1px solid var(--border-default);
    background: var(--bg-raised);
    color: var(--text-secondary);
    cursor: pointer;
  }
  .suggestion:hover:not(:disabled) {
    border-color: var(--plank-amber);
    color: var(--plank-parchment);
  }
  .suggestion:disabled { opacity: 0.45; cursor: not-allowed; }

  .thread {
    flex: 1;
    overflow-y: auto;
    padding: 1rem 1rem 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
  }
  .msg { display: flex; }
  .msg.user { justify-content: flex-end; }
  .msg.assistant, .msg.system { justify-content: flex-start; }
  .msg.tool { justify-content: flex-start; }
  .bubble {
    max-width: min(640px, 92%);
    padding: 0.65rem 0.85rem;
    border-radius: var(--r-lg);
    background: var(--bg-raised);
    border: 1px solid var(--border-subtle);
  }
  .msg.user .bubble {
    background: var(--accent-dim);
    border-color: rgba(46, 216, 152, 0.35);
  }
  .msg.system .bubble {
    background: var(--red-dim);
    border-color: var(--red);
    font-family: var(--font-mono);
    font-size: 0.82rem;
  }
  .role {
    font-size: 0.68rem;
    color: var(--text-tertiary);
    margin-bottom: 0.25rem;
    font-family: var(--font-mono);
  }
  .body {
    white-space: pre-wrap;
    word-break: break-word;
    font-size: 0.92rem;
    line-height: 1.55;
  }
  .body.md {
    white-space: normal;
  }
  .body.md :global(p) {
    margin: 0 0 0.65em;
  }
  .body.md :global(p:last-child) {
    margin-bottom: 0;
  }
  .body.md :global(strong) {
    font-weight: 600;
    color: var(--text-primary);
  }
  .body.md :global(em) {
    font-style: italic;
  }
  .body.md :global(a) {
    color: var(--plank-amber);
  }
  .body.md :global(code) {
    font-family: var(--font-mono);
    font-size: 0.84em;
    background: var(--bg-base);
    border: 1px solid var(--border-subtle);
    border-radius: 3px;
    padding: 0.05em 0.35em;
  }
  .body.md :global(pre) {
    margin: 0.55em 0;
    padding: 0.65rem 0.75rem;
    overflow-x: auto;
    background: var(--bg-base);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-sm);
  }
  .body.md :global(pre code) {
    background: none;
    border: none;
    padding: 0;
    font-size: 0.8rem;
    line-height: 1.45;
  }
  .body.md :global(ul),
  .body.md :global(ol) {
    margin: 0.4em 0 0.65em;
    padding-left: 1.25rem;
  }
  .body.md :global(li) {
    margin: 0.2em 0;
  }
  .body.md :global(li > p) {
    margin: 0;
  }
  .body.md :global(blockquote) {
    margin: 0.5em 0;
    padding-left: 0.75rem;
    border-left: 2px solid var(--plank-amber);
    color: var(--text-secondary);
  }
  .body.md :global(h1),
  .body.md :global(h2),
  .body.md :global(h3),
  .body.md :global(h4) {
    margin: 0.75em 0 0.35em;
    font-weight: 600;
    line-height: 1.3;
    color: var(--plank-parchment);
  }
  .body.md :global(h1) { font-size: 1.15em; }
  .body.md :global(h2) { font-size: 1.05em; }
  .body.md :global(h3),
  .body.md :global(h4) { font-size: 1em; }
  .body.md :global(table) {
    width: 100%;
    border-collapse: collapse;
    margin: 0.55em 0 0.75em;
    font-size: 0.84rem;
    display: block;
    overflow-x: auto;
  }
  .body.md :global(th),
  .body.md :global(td) {
    border: 1px solid var(--border-default);
    padding: 0.35rem 0.55rem;
    text-align: left;
    vertical-align: top;
  }
  .body.md :global(th) {
    background: var(--bg-base);
    color: var(--plank-parchment);
    font-weight: 600;
    white-space: nowrap;
  }
  .body.md :global(td) {
    color: var(--text-secondary);
  }
  .body.md :global(hr) {
    border: none;
    border-top: 1px solid var(--border-subtle);
    margin: 0.75em 0;
  }
  .tool-chip {
    display: inline-flex;
    align-items: baseline;
    gap: 0.35rem;
    max-width: min(640px, 92%);
    font-family: var(--font-mono);
    font-size: 0.72rem;
    color: var(--plank-parchment);
    background: rgba(201, 162, 39, 0.1);
    border: 1px solid rgba(201, 162, 39, 0.35);
    border-radius: var(--r-sm);
    padding: 0.3rem 0.55rem;
  }
  .tool-chip.pending {
    border-style: dashed;
    opacity: 0.9;
  }
  .tool-name {
    font-weight: 600;
    flex-shrink: 0;
  }
  .tool-sep {
    color: var(--text-tertiary);
    flex-shrink: 0;
  }
  .tool-ctx {
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .composer-wrap {
    flex-shrink: 0;
    padding: 0.75rem 1rem 0.5rem;
  }
  .composer {
    display: flex;
    align-items: flex-end;
    gap: 0.5rem;
    background: var(--bg-panel);
    border: 1px solid var(--border-default);
    border-radius: 1.25rem;
    padding: 0.55rem 0.65rem;
    max-width: 820px;
    margin: 0 auto;
  }
  .composer:focus-within {
    border-color: var(--plank-amber);
    box-shadow: 0 0 0 1px rgba(201, 162, 39, 0.25);
  }
  .composer.disabled { opacity: 0.7; }
  .plus {
    width: 2rem;
    height: 2rem;
    border-radius: 999px;
    border: 1px solid var(--border-subtle);
    background: transparent;
    color: var(--text-tertiary);
    cursor: default;
    flex-shrink: 0;
  }
  textarea {
    flex: 1;
    resize: none;
    border: none;
    background: transparent;
    color: var(--text-primary);
    font-family: var(--font-sans);
    font-size: 0.95rem;
    line-height: 1.4;
    min-height: 1.4rem;
    max-height: 8rem;
    outline: none;
    padding: 0.35rem 0;
  }
  .composer-actions {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    flex-shrink: 0;
  }
  .speed {
    font-family: var(--font-mono);
    font-size: 0.65rem;
    color: var(--text-tertiary);
    text-transform: lowercase;
  }
  .send {
    width: 2rem;
    height: 2rem;
    border-radius: 999px;
    border: none;
    background: var(--plank-amber);
    color: var(--plank-wood);
    font-weight: 700;
    cursor: pointer;
  }
  .send:disabled {
    background: var(--bg-hover);
    color: var(--text-tertiary);
    cursor: not-allowed;
  }
  .fineprint {
    text-align: center;
    font-size: 0.65rem;
    color: var(--text-tertiary);
    margin-top: 0.45rem;
  }

  @media (max-width: 700px) {
    .identity .tag { display: none; }
    .plank-page { height: calc(100vh - var(--topbar-height) - 1rem); }
    .top-right :global(figure figcaption) { display: none; }
  }
</style>
