<script lang="ts">
  type Point = { t: string; v: number; label?: string };

  let {
    points = [],
    color = 'var(--blue)',
    formatValue = (n: number) => String(n),
    height = 140,
  }: {
    points?: Point[];
    color?: string;
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

  let maxV = $derived(Math.max(1, ...points.map((p) => p.v)));
  let coords = $derived(
    points.map((p, i) => {
      const x = padL + (i / Math.max(points.length - 1, 1)) * (W - padL - padR);
      const y = padT + (1 - p.v / maxV) * (H - padT - padB);
      return { x, y, ...p };
    }),
  );
  let line = $derived(
    coords.map((c, i) => `${i === 0 ? 'M' : 'L'}${c.x.toFixed(1)},${c.y.toFixed(1)}`).join(' '),
  );
  let area = $derived(
    coords.length
      ? line +
          ` L${coords[coords.length - 1].x.toFixed(1)},${H - padB}` +
          ` L${coords[0].x.toFixed(1)},${H - padB} Z`
      : '',
  );
  let yTicks = $derived([0, 0.5, 1].map((f) => ({
    y: padT + (1 - f) * (H - padT - padB),
    label: formatValue(maxV * f),
  })));
  let xLabels = $derived(
    coords.length
      ? [0, Math.floor(coords.length / 2), coords.length - 1]
          .filter((i, idx, a) => a.indexOf(i) === idx)
          .map((i) => ({ x: coords[i].x, label: shortTime(coords[i].t) }))
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
    if (!wrapEl || !coords[idx]) {
      tipStyle = 'display:none';
      return;
    }
    const c = coords[idx];
    const rect = wrapEl.getBoundingClientRect();
    const px = (c.x / W) * rect.width;
    const py = (c.y / H) * rect.height;
    const tipW = tipEl?.offsetWidth || 120;
    const tipH = tipEl?.offsetHeight || 44;
    const gap = 10;

    // Prefer above; flip below when near the top
    let top = py - tipH - gap;
    if (top < 4) top = py + gap;
    if (top + tipH > rect.height - 4) top = Math.max(4, rect.height - tipH - 4);

    // Clamp horizontally inside the chart
    let left = px - tipW / 2;
    left = Math.max(4, Math.min(left, rect.width - tipW - 4));

    tipStyle = `left:${left}px;top:${top}px;display:block`;
  }

  function onMove(e: MouseEvent) {
    if (!wrapEl || !coords.length) return;
    const rect = wrapEl.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width) * W;
    let best = 0;
    let bestDist = Infinity;
    for (let i = 0; i < coords.length; i++) {
      const d = Math.abs(coords[i].x - x);
      if (d < bestDist) {
        bestDist = d;
        best = i;
      }
    }
    hoverIdx = best;
    // Wait a frame so tip content/size is current
    requestAnimationFrame(() => placeTip(best));
  }
  function onLeave() {
    hoverIdx = null;
    tipStyle = 'display:none';
  }

  let tip = $derived(hoverIdx != null ? coords[hoverIdx] : null);
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
      <path d={area} fill={color} opacity="0.15" />
      <path d={line} fill="none" stroke={color} stroke-width="2" />
      {#if tip}
        <line x1={tip.x} x2={tip.x} y1={padT} y2={H - padB} class="cross" />
        <circle cx={tip.x} cy={tip.y} r="3.5" fill={color} stroke="#fff" stroke-width="1.2" />
      {/if}
    </svg>
  </div>
  {#if tip}
    <div class="tip" bind:this={tipEl} style={tipStyle}>
      <div class="tip-t">{shortTime(tip.t)}</div>
      <div class="tip-v" style="color:{color}">{formatValue(tip.v)}</div>
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
    border-radius: var(--r-md); padding: 0.35rem 0.5rem; pointer-events: none;
    white-space: nowrap; box-shadow: 0 4px 16px rgba(0,0,0,.45); z-index: 6;
  }
  .tip-t { font-size: 0.65rem; color: var(--text-tertiary); }
  .tip-v { font-size: 0.85rem; font-weight: 600; font-family: var(--font-mono); }
</style>
