<script lang="ts">
  type Series = { key: string; label: string; color: string };
  type Row = { t: string; values: Record<string, number> };

  let {
    rows = [],
    series = [],
    formatValue = (n: number) => String(n),
    height = 140,
  }: {
    rows?: Row[];
    series?: Series[];
    formatValue?: (n: number) => string;
    height?: number;
  } = $props();

  let wrapEl: HTMLDivElement | undefined = $state();
  let tipEl: HTMLDivElement | undefined = $state();
  let hoverIdx = $state<number | null>(null);
  let tipStyle = $state('display:none');

  const W = 520;
  const H = 140;
  const padL = 36;
  const padR = 8;
  const padT = 10;
  const padB = 22;

  let totals = $derived(rows.map((r) => series.reduce((s, ser) => s + (r.values[ser.key] || 0), 0)));
  let maxV = $derived(Math.max(1, ...totals));
  let barW = $derived(Math.max(2, ((W - padL - padR) / Math.max(rows.length, 1)) * 0.72));
  let bars = $derived(
    rows.map((r, i) => {
      const x = padL + (i + 0.5) * ((W - padL - padR) / Math.max(rows.length, 1)) - barW / 2;
      let y = H - padB;
      const segs: { key: string; color: string; x: number; y: number; h: number; v: number }[] = [];
      for (const ser of series) {
        const v = r.values[ser.key] || 0;
        const h = (v / maxV) * (H - padT - padB);
        y -= h;
        segs.push({ key: ser.key, color: ser.color, x, y, h, v });
      }
      return { t: r.t, x: x + barW / 2, segs };
    }),
  );
  let yTicks = $derived([0, 0.5, 1].map((f) => ({
    y: padT + (1 - f) * (H - padT - padB),
    label: formatValue(maxV * f),
  })));
  let xLabels = $derived(
    bars.length
      ? [0, Math.floor(bars.length / 2), bars.length - 1]
          .filter((i, idx, a) => a.indexOf(i) === idx)
          .map((i) => ({ x: bars[i].x, label: shortTime(bars[i].t) }))
      : [],
  );

  function shortTime(iso: string): string {
    try {
      const d = new Date(iso);
      return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch {
      return iso;
    }
  }

  function placeTip(idx: number) {
    if (!wrapEl || !bars[idx]) {
      tipStyle = 'display:none';
      return;
    }
    const b = bars[idx];
    const rect = wrapEl.getBoundingClientRect();
    const px = (b.x / W) * rect.width;
    const tipW = tipEl?.offsetWidth || 150;
    const tipH = tipEl?.offsetHeight || 90;
    let left = px - tipW / 2;
    left = Math.max(4, Math.min(left, rect.width - tipW - 4));
    let top = 8;
    if (top + tipH > rect.height - 4) top = Math.max(4, rect.height - tipH - 4);
    tipStyle = `left:${left}px;top:${top}px;display:block`;
  }

  function onMove(e: MouseEvent) {
    if (!wrapEl || !bars.length) return;
    const rect = wrapEl.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width) * W;
    let best = 0;
    let bestDist = Infinity;
    for (let i = 0; i < bars.length; i++) {
      const d = Math.abs(bars[i].x - x);
      if (d < bestDist) {
        bestDist = d;
        best = i;
      }
    }
    hoverIdx = best;
    requestAnimationFrame(() => placeTip(best));
  }
  function onLeave() {
    hoverIdx = null;
    tipStyle = 'display:none';
  }

  let tip = $derived(hoverIdx != null ? { ...bars[hoverIdx], idx: hoverIdx } : null);
</script>

<div
  class="chart"
  style="height:{height}px"
  bind:this={wrapEl}
  onmousemove={onMove}
  onmouseleave={onLeave}
  role="img"
>
  <div class="plot">
    <svg viewBox="0 0 {W} {H}" preserveAspectRatio="none" class="svg">
      {#each yTicks as t}
        <line x1={padL} x2={W - padR} y1={t.y} y2={t.y} class="grid" />
        <text x={padL - 4} y={t.y + 3} text-anchor="end" class="tick">{t.label}</text>
      {/each}
      {#each xLabels as t}
        <text x={t.x} y={H - 6} text-anchor="middle" class="tick">{t.label}</text>
      {/each}
      {#each bars as b}
        {#each b.segs as s}
          {#if s.h > 0}
            <rect x={s.x} y={s.y} width={barW} height={s.h} fill={s.color} rx="1" />
          {/if}
        {/each}
      {/each}
      {#if tip}
        <line x1={tip.x} x2={tip.x} y1={padT} y2={H - padB} class="cross" />
      {/if}
    </svg>
  </div>
  {#if tip}
    <div class="tip" bind:this={tipEl} style={tipStyle}>
      <div class="tip-t">{shortTime(tip.t)}</div>
      {#each series as ser}
        {@const v = tip.segs.find((s) => s.key === ser.key)?.v || 0}
        <div class="tip-row">
          <span class="dot" style="background:{ser.color}"></span>
          <span>{ser.label}</span>
          <strong>{formatValue(v)}</strong>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .chart { position: relative; width: 100%; z-index: 1; overflow: visible; }
  .chart:hover { z-index: 5; }
  .plot {
    width: 100%; height: 100%;
    background: var(--bg-raised); border-radius: var(--r-md); overflow: hidden;
  }
  .svg { width: 100%; height: 100%; display: block; }
  .grid { stroke: var(--border-subtle); stroke-width: 1; }
  .cross { stroke: var(--text-tertiary); stroke-width: 1; stroke-dasharray: 3 3; }
  .tick { font-size: 8px; fill: var(--text-tertiary); font-family: var(--font-mono); }
  .tip {
    position: absolute;
    background: var(--bg-panel); border: 1px solid var(--border-default);
    border-radius: var(--r-md); padding: 0.4rem 0.55rem; pointer-events: none;
    white-space: nowrap; box-shadow: 0 4px 16px rgba(0,0,0,.45); z-index: 6;
    min-width: 9rem;
  }
  .tip-t { font-size: 0.65rem; color: var(--text-tertiary); margin-bottom: 0.25rem; }
  .tip-row { display: flex; align-items: center; gap: 0.35rem; font-size: 0.72rem; color: var(--text-secondary); }
  .tip-row strong { margin-left: auto; font-family: var(--font-mono); color: var(--text-primary); font-weight: 600; }
  .dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
</style>
