/** Semi-circle (CasaOS-style) arc paths for a 0–100 value. */
export function gaugeArcs(
  pct: number,
  r = 54,
  cx = 60,
  cy = 58,
  stroke = 8,
): { track: string; value: string; length: number; offset: number } {
  const start = Math.PI; // left
  const end = 0; // right
  const x0 = cx + r * Math.cos(start);
  const y0 = cy + r * Math.sin(start);
  const x1 = cx + r * Math.cos(end);
  const y1 = cy + r * Math.sin(end);
  const track = `M ${x0} ${y0} A ${r} ${r} 0 0 1 ${x1} ${y1}`;
  const length = Math.PI * r;
  const clamped = Math.max(0, Math.min(100, pct));
  const offset = length * (1 - clamped / 100);
  return { track, value: track, length, offset };
}
