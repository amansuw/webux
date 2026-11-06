<script lang="ts">
  import type { MascotState } from '$lib/plankWs';

  let {
    state = 'idle',
    size = 96,
    label = '',
  }: {
    state?: MascotState;
    size?: number;
    label?: string;
  } = $props();

  const show = $derived({
    thinking: state === 'thinking' || state === 'welcome',
    searching: state === 'searching',
    coding: state === 'coding',
    working: state === 'working' || state === 'heavy' || state === 'approval',
    organizing: state === 'organizing',
    heavy: state === 'heavy',
    success: state === 'success',
    error: state === 'error',
    sleeping: state === 'sleeping',
    approval: state === 'approval',
  });

  const eyes = $derived.by(() => {
    if (state === 'sleeping' || state === 'success') return 'closed';
    if (state === 'error') return 'angry';
    if (state === 'thinking' || state === 'searching') return 'up';
    return 'normal';
  });

  const mouth = $derived.by(() => {
    if (state === 'thinking' || state === 'searching') return 'flat';
    if (state === 'error') return 'flat';
    if (state === 'sleeping') return 'soft';
    return 'smile';
  });
</script>

<figure class="mascot" class:reduced={false} data-state={state} style={`--sz:${size}px`} aria-label={label || `P.L.A.N.K. ${state}`}>
  <svg
    class="plank-svg"
    class:bob={state !== 'sleeping' && state !== 'error'}
    class:sleep={state === 'sleeping'}
    width={size}
    height={size}
    viewBox="0 0 120 140"
    role="img"
  >
    <title>P.L.A.N.K. — {state}</title>

    <!-- Sleeping bag -->
    {#if show.sleeping}
      <g class="prop sleep-bag">
        <ellipse cx="60" cy="118" rx="42" ry="16" fill="#3d6b45" opacity="0.95" />
        <ellipse cx="60" cy="114" rx="36" ry="10" fill="#5a9a62" />
      </g>
    {/if}

    <!-- Body: 2x4 with side/bottom depth -->
    <g class="body">
      <path d="M28 18 H86 L98 30 V118 H40 L28 106 Z" fill="#c9a66b" stroke="#1a140d" stroke-width="2.5" stroke-linejoin="round" />
      <path d="M86 18 L98 30 V118 L86 106 Z" fill="#6b4a28" stroke="#1a140d" stroke-width="2" />
      <path d="M28 106 L40 118 H98 L86 106 Z" fill="#5a3d22" stroke="#1a140d" stroke-width="2" />
      <!-- grain -->
      <path d="M40 34 V96" stroke="#1a140d" stroke-width="1.2" opacity="0.35" />
      <path d="M52 28 V100" stroke="#1a140d" stroke-width="1" opacity="0.25" />
      <path d="M68 32 V98" stroke="#1a140d" stroke-width="1.1" opacity="0.3" />
      <!-- notch -->
      <path d="M78 18 L86 18 L86 26 Z" fill="#a8844e" stroke="#1a140d" stroke-width="1.5" />
    </g>

    <!-- Hard hat -->
    {#if show.approval}
      <g class="prop hat">
        <ellipse cx="60" cy="20" rx="28" ry="8" fill="#d4a537" stroke="#1a140d" stroke-width="2" />
        <path d="M40 20 Q60 4 80 20" fill="#e0c44e" stroke="#1a140d" stroke-width="2" />
      </g>
    {/if}

    <!-- Face -->
    <g class="face">
      {#if eyes === 'closed'}
        <path d="M42 52 Q48 46 54 52" fill="none" stroke="#c0392b" stroke-width="2.5" stroke-linecap="round" />
        <path d="M62 54 Q66 48 72 54" fill="none" stroke="#c0392b" stroke-width="2.2" stroke-linecap="round" />
      {:else if eyes === 'angry'}
        <circle cx="48" cy="52" r="9" fill="none" stroke="#c0392b" stroke-width="2.8" />
        <circle cx="48" cy="52" r="3.2" fill="#2a5fd4" />
        <circle cx="68" cy="54" r="6.5" fill="none" stroke="#c0392b" stroke-width="2.5" />
        <circle cx="68" cy="54" r="2.4" fill="#2a5fd4" />
        <path d="M40 44 L56 48" stroke="#1a140d" stroke-width="2" />
        <path d="M62 46 L76 42" stroke="#1a140d" stroke-width="2" />
      {:else if eyes === 'up'}
        <circle cx="48" cy="50" r="9" fill="none" stroke="#c0392b" stroke-width="2.8" />
        <circle cx="48" cy="46" r="3.2" fill="#2a5fd4" />
        <circle cx="68" cy="52" r="6.5" fill="none" stroke="#c0392b" stroke-width="2.5" />
        <circle cx="68" cy="48" r="2.4" fill="#2a5fd4" />
      {:else}
        <g class="blink">
          <circle cx="48" cy="52" r="9" fill="none" stroke="#c0392b" stroke-width="2.8" />
          <circle cx="48" cy="52" r="3.2" fill="#2a5fd4" />
          <circle cx="68" cy="54" r="6.5" fill="none" stroke="#c0392b" stroke-width="2.5" />
          <circle cx="68" cy="54" r="2.4" fill="#2a5fd4" />
        </g>
      {/if}

      {#if mouth === 'flat'}
        <path d="M48 78 H72" fill="none" stroke="#2a5fd4" stroke-width="2.6" stroke-linecap="round" />
      {:else if mouth === 'soft'}
        <path d="M50 78 Q60 82 70 78" fill="none" stroke="#2a5fd4" stroke-width="2.4" stroke-linecap="round" />
      {:else}
        <path d="M48 76 Q60 88 72 76" fill="none" stroke="#2a5fd4" stroke-width="2.8" stroke-linecap="round" />
      {/if}
    </g>

    <!-- Thinking bars / list -->
    {#if state === 'thinking'}
      <g class="prop think">
        <rect x="88" y="28" width="18" height="3" rx="1" fill="#c9a227" class="bar1" />
        <rect x="90" y="36" width="14" height="3" rx="1" fill="#6bbf6b" class="bar2" />
        <rect x="92" y="44" width="10" height="3" rx="1" fill="#4e8de0" class="bar3" />
        <circle cx="102" cy="20" r="5" fill="#e0c44e" opacity="0.9" />
      </g>
    {/if}

    <!-- Magnifier -->
    {#if show.searching}
      <g class="prop mag">
        <circle cx="98" cy="70" r="12" fill="#4e8de0" stroke="#1a140d" stroke-width="2" opacity="0.85" />
        <circle cx="98" cy="70" r="7" fill="#9ad0ff" opacity="0.5" />
        <path d="M106 78 L116 90" stroke="#1a140d" stroke-width="3.5" stroke-linecap="round" />
      </g>
    {/if}

    <!-- Code pane -->
    {#if show.coding}
      <g class="prop code">
        <rect x="86" y="40" width="28" height="22" rx="3" fill="#1e2a44" stroke="#4e8de0" stroke-width="1.5" />
        <text x="92" y="52" font-size="7" fill="#6bbf6b" font-family="monospace">&lt;&gt;</text>
        <rect x="90" y="55" width="16" height="2" fill="#6bbf6b" opacity="0.7" />
        <rect x="90" y="59" width="12" height="2" fill="#6bbf6b" opacity="0.5" />
      </g>
    {/if}

    <!-- Barbell / tools -->
    {#if show.working}
      <g class="prop lift">
        {#if show.heavy}
          <rect x="30" y="8" width="60" height="4" rx="1" fill="#4e8de0" />
          <rect x="24" y="2" width="10" height="16" rx="2" fill="#2a5fd4" stroke="#1a140d" stroke-width="1" />
          <rect x="86" y="2" width="10" height="16" rx="2" fill="#2a5fd4" stroke="#1a140d" stroke-width="1" />
        {:else}
          <path d="M18 70 L34 78" stroke="#4e8de0" stroke-width="3" stroke-linecap="round" />
          <circle cx="16" cy="68" r="4" fill="#8b7d6b" stroke="#1a140d" stroke-width="1" />
          <path d="M100 72 L112 60" stroke="#8b72e0" stroke-width="2.5" stroke-linecap="round" />
          <rect x="108" y="54" width="8" height="3" fill="#8b72e0" />
        {/if}
      </g>
    {/if}

    <!-- Drawers -->
    {#if show.organizing}
      <g class="prop drawers">
        <rect x="92" y="78" width="20" height="8" rx="1" fill="#8b72e0" stroke="#1a140d" stroke-width="1" />
        <rect x="92" y="88" width="20" height="8" rx="1" fill="#6b52c0" stroke="#1a140d" stroke-width="1" />
        <rect x="92" y="98" width="20" height="8" rx="1" fill="#8b72e0" stroke="#1a140d" stroke-width="1" />
      </g>
    {/if}

    <!-- Success check -->
    {#if show.success}
      <g class="prop ok">
        <circle cx="96" cy="28" r="12" fill="#6bbf6b" stroke="#1a140d" stroke-width="2" />
        <path d="M90 28 L94 33 L104 22" fill="none" stroke="#fff" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
        <circle cx="78" cy="18" r="2" fill="#9ad0ff" class="spark" />
        <circle cx="110" cy="22" r="1.5" fill="#9ad0ff" class="spark" />
        <circle cx="104" cy="40" r="1.8" fill="#9ad0ff" class="spark" />
      </g>
    {/if}

    <!-- Error fire laptop -->
    {#if show.error}
      <g class="prop fire">
        <rect x="78" y="88" width="30" height="16" rx="2" fill="#8b72e0" stroke="#1a140d" stroke-width="1.5" />
        <path d="M88 88 Q92 70 96 88" fill="#e05c6a" />
        <path d="M96 88 Q100 72 104 88" fill="#e0c44e" />
        <path d="M92 88 Q96 76 100 88" fill="#e05c6a" opacity="0.8" />
      </g>
    {/if}

    <!-- Zz -->
    {#if show.sleeping}
      <g class="prop zz">
        <text x="90" y="36" class="z1" fill="#8b7d6b" font-size="12" font-family="IBM Plex Sans, sans-serif">z</text>
        <text x="100" y="24" class="z2" fill="#8b7d6b" font-size="16" font-family="IBM Plex Sans, sans-serif">Z</text>
      </g>
    {/if}
  </svg>
  {#if label}
    <figcaption>{label}</figcaption>
  {/if}
</figure>

<style>
  .mascot {
    --plank-amber: #c9a227;
    display: inline-flex;
    flex-direction: column;
    align-items: center;
    gap: 0.35rem;
    margin: 0;
    user-select: none;
  }
  figcaption {
    font-family: var(--font-mono);
    font-size: 0.7rem;
    color: var(--text-tertiary);
    letter-spacing: 0.04em;
  }
  .plank-svg {
    display: block;
    overflow: visible;
  }
  .bob {
    animation: bob 2.8s ease-in-out infinite;
  }
  .sleep {
    animation: none;
    transform: translateY(4px);
  }
  .blink {
    animation: blink 4.5s ease-in-out infinite;
  }
  .bar1, .bar2, .bar3 {
    animation: pulse 1.2s ease-in-out infinite;
  }
  .bar2 { animation-delay: 0.15s; }
  .bar3 { animation-delay: 0.3s; }
  .spark {
    animation: sparkle 1s ease-in-out infinite;
  }
  .z1, .z2 {
    animation: rise 2s ease-in-out infinite;
  }
  .z2 { animation-delay: 0.4s; }

  @keyframes bob {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-4px); }
  }
  @keyframes blink {
    0%, 92%, 100% { opacity: 1; }
    94%, 96% { opacity: 0.15; }
  }
  @keyframes pulse {
    0%, 100% { opacity: 0.45; }
    50% { opacity: 1; }
  }
  @keyframes sparkle {
    0%, 100% { opacity: 0.3; transform: scale(0.8); }
    50% { opacity: 1; transform: scale(1.2); }
  }
  @keyframes rise {
    0%, 100% { opacity: 0.35; transform: translateY(0); }
    50% { opacity: 1; transform: translateY(-6px); }
  }

  @media (prefers-reduced-motion: reduce) {
    .bob, .blink, .bar1, .bar2, .bar3, .spark, .z1, .z2 {
      animation: none !important;
    }
  }
</style>
