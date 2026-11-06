/** Build a filled area + stroke SVG path from numeric samples. */
export function sparkPaths(
  values: number[],
  width = 120,
  height = 36,
  pad = 2,
): { line: string; area: string } {
  if (!values.length) return { line: '', area: '' };
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = max - min || 1;
  const n = values.length;
  const pts = values.map((v, i) => {
    const x = pad + (i / Math.max(n - 1, 1)) * (width - pad * 2);
    const y = pad + (1 - (v - min) / span) * (height - pad * 2);
    return [x, y] as const;
  });
  const line = pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(' ');
  const area =
    line +
    ` L${pts[pts.length - 1][0].toFixed(1)},${height - pad}` +
    ` L${pts[0][0].toFixed(1)},${height - pad} Z`;
  return { line, area };
}

export function pushSample(buf: number[], value: number, max = 40): number[] {
  const next = buf.length ? [...buf, value] : Array(8).fill(value);
  if (next.length > max) next.shift();
  return next;
}
