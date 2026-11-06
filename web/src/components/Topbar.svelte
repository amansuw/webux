<script lang="ts">
  import { wsStore } from '$lib/ws';

  let {
    onMenu,
    menuOpen = false,
  }: {
    onMenu?: () => void;
    menuOpen?: boolean;
  } = $props();

  let wsConnected = $state(false);
  let hostname = $state('');
  let username = $state('');

  $effect(() => {
    const unsub = wsStore.connected.subscribe((v) => {
      wsConnected = v;
    });
    return unsub;
  });

  fetch('/api/system/info')
    .then((r) => (r.ok ? r.json() : null))
    .then((d) => {
      if (d?.hostname) hostname = d.hostname;
    })
    .catch(() => {});

  fetch('/auth/whoami')
    .then((r) => (r.ok ? r.json() : null))
    .then((d) => {
      if (d?.username) username = d.username;
    })
    .catch(() => {});

  async function logout() {
    await fetch('/auth/logout', { method: 'POST' });
    window.location.reload();
  }
</script>

<header class="topbar">
  <div class="topbar-left">
    <button
      type="button"
      class="menu-btn"
      aria-label={menuOpen ? 'Close menu' : 'Open menu'}
      aria-expanded={menuOpen}
      onclick={() => onMenu?.()}
    >
      <span class="menu-bars" aria-hidden="true">{menuOpen ? '✕' : '☰'}</span>
    </button>
    {#if hostname}
      <span class="hostname-pill">
        <span class="host-dot"></span>
        <span class="mono hostname-text">{hostname}</span>
      </span>
    {/if}
  </div>

  <div class="topbar-right">
    <div class="ws-indicator" title={wsConnected ? 'Live — WebSocket connected' : 'Not connected'}>
      <span class="dot {wsConnected ? 'dot-green' : 'dot-gray'}"></span>
      <span class="ws-label mono">{wsConnected ? 'live' : 'offline'}</span>
    </div>
    {#if username}
      <div class="user-pill">
        <span class="user-icon">◉</span>
        <span class="mono user-name">{username}</span>
        <button class="logout-btn" onclick={logout} title="Sign out">⏻</button>
      </div>
    {/if}
  </div>
</header>

<style>
  .topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 1.25rem;
    height: 48px;
    flex-shrink: 0;
    background: var(--bg-panel);
    border-bottom: 1px solid var(--border-subtle);
    gap: 0.75rem;
  }

  .topbar-left {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    min-width: 0;
  }

  .menu-btn {
    display: none;
    align-items: center;
    justify-content: center;
    width: 2.1rem;
    height: 2.1rem;
    background: var(--bg-raised);
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    color: var(--text-primary);
    cursor: pointer;
    flex-shrink: 0;
    font-size: 1.05rem;
    line-height: 1;
  }
  .menu-btn:hover {
    border-color: var(--accent);
    color: var(--accent);
  }

  .hostname-pill {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    background: var(--bg-raised);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-md);
    padding: 0.2rem 0.55rem;
    min-width: 0;
  }
  .host-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 4px var(--accent);
    flex-shrink: 0;
  }
  .hostname-text {
    font-size: 0.72rem;
    color: var(--nav-item);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .topbar-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-shrink: 0;
  }

  .ws-indicator {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    cursor: default;
  }
  .ws-label {
    font-size: 0.68rem;
    color: var(--text-tertiary);
  }

  .user-pill {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    background: var(--bg-raised);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-md);
    padding: 0.2rem 0.2rem 0.2rem 0.55rem;
  }
  .user-icon {
    font-size: var(--icon-sm);
    color: var(--accent);
    line-height: 1;
  }
  .user-name {
    font-size: 0.78rem;
    color: var(--nav-item);
  }

  .logout-btn {
    background: none;
    border: none;
    padding: 0.25rem 0.45rem;
    cursor: pointer;
    color: var(--text-tertiary);
    font-size: var(--icon-md);
    border-radius: var(--r-sm);
    line-height: 1;
    transition: color 0.15s, background 0.15s;
  }
  .logout-btn:hover {
    color: var(--red);
    background: var(--bg-hover);
  }

  @media (max-width: 900px) {
    .topbar {
      padding: 0 0.75rem;
    }
    .menu-btn {
      display: inline-flex;
    }
    .user-name {
      display: none;
    }
    .user-pill {
      padding: 0.15rem 0.15rem 0.15rem 0.4rem;
    }
  }

  @media (max-width: 480px) {
    .ws-label {
      display: none;
    }
  }
</style>
