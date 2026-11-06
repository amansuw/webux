<script lang="ts">
  import type { Snippet } from 'svelte';
  import { tweened } from 'svelte/motion';
  import { cubicOut } from 'svelte/easing';
  import { gaugeArcs } from '$lib/gauge';

  let {
    pct = 0,
    color = 'var(--accent)',
    textColor = 'var(--text-primary)',
    label = '',
    format = (n: number) => `${Math.round(n)}%`,
    duration = 750,
    children,
  }: {
    pct?: number;
    color?: string;
    textColor?: string;
    label?: string;
    format?: (n: number) => string;
    duration?: number;
    children?: Snippet;
  } = $props();

  const animated = tweened(0, { duration: 750, easing: cubicOut });

  $effect(() => {
    const target = Math.max(0, Math.min(100, Number.isFinite(pct) ? Number(pct) : 0));
    animated.set(target, { duration });
  });

  let display = $derived($animated);
  let arcs = $derived(gaugeArcs(display));
</script>

<div class="gauge">
  <svg viewBox="0 0 120 78" class="gauge-svg" aria-label={label}>
    <path
      d={arcs.track}
      fill="none"
      stroke="var(--bg-raised)"
      stroke-width="8"
      stroke-linecap="round"
    />
    <path
      class="gauge-arc"
      d={arcs.value}
      fill="none"
      stroke={color}
      stroke-width="8"
      stroke-linecap="round"
      stroke-dasharray={arcs.length}
      stroke-dashoffset={arcs.offset}
    />
    <text x="60" y="52" text-anchor="middle" class="gauge-pct" fill={textColor}>
      {format(display)}
    </text>
  </svg>
  {#if label}
    <div class="gauge-label">{label}</div>
  {/if}
  {#if children}
    {@render children()}
  {/if}
</div>

<style>
  .gauge { text-align: center; }
  .gauge-svg { width: 100%; max-width: 140px; height: auto; margin: 0 auto; display: block; }
  .gauge-arc { transition: stroke 0.45s ease; }
  .gauge-pct {
    font-size: 1.15rem;
    font-weight: 700;
    font-family: var(--font-mono);
  }
  .gauge-label {
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--text-primary);
    margin-top: 0.15rem;
  }
</style>
