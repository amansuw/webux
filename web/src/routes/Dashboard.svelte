<script lang="ts">
  import { onMount } from 'svelte';
  import HealthChecks from '$components/HealthChecks.svelte';
  import HoverLineChart from '$components/HoverLineChart.svelte';
  import StackedBarChart from '$components/StackedBarChart.svelte';
  import Gauge from '$components/Gauge.svelte';
  import { pushSample } from '$lib/sparkline';

  const PROC_ROWS = 10;

  let hostname = $state('');
  let distro = $state('');
  let kernel = $state('');
  let uptime = $state('');
  let cpuPct = $state(0);
  let memUsed = $state('');
  let memTotal = $state('');
  let memPct = $state(0);
  let memUsedMB = $state(0);
  let diskUsed = $state('');
  let diskTotal = $state('');
  let diskPct = $state(0);
  let load = $state<number[]>([]);
  let loading = $state(true);
  let info: any = $state(null);
  let cpuTempC = $state<number | null>(null);
  let nvmeTempC = $state<number | null>(null);
  let swapPct = $state(0);
  let swapUsed = $state('');
  let swapTotal = $state('');

  let cpuHist = $state<number[]>([]);
  let memHist = $state<number[]>([]);

  let topProcs: any[] = $state(Array.from({ length: PROC_ROWS }, () => null));
  let mounts: any[] = $state([]);
  let expandedMount = $state('');
  let mountUsage: Record<string, { loading: boolean; error: string; entries: any[] }> = $state({});
  let containers: any[] = $state([]);
  let cronJobs: any[] = $state([]);
  let cronCount = $state(0);
  let failedServices: any[] = $state([]);
  let procCount = $state(0);
  let ctrRunning = $state(0);
  let ctrTotal = $state(0);
  let ctrActionPending = $state<string | null>(null);

  let cf: any = $state(null);
  let cfError = $state('');

  function fmtMB(mb: number): string {
    if (mb >= 1024) return (mb / 1024).toFixed(1) + ' GB';
    return mb.toFixed(0) + ' MB';
  }
  function fmtGB(gb: number): string {
    if (gb >= 1024) return (gb / 1024).toFixed(1) + ' TB';
    return gb.toFixed(1) + ' GB';
  }
  function fmtUptime(seconds: number): string {
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    if (d > 0) return `${d}d ${h}h ${m}m`;
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m`;
  }
  function fmtBytes(n: number): string {
    if (n >= 1e12) return (n / 1e12).toFixed(2) + ' TB';
    if (n >= 1e9) return (n / 1e9).toFixed(2) + ' GB';
    if (n >= 1e6) return (n / 1e6).toFixed(1) + ' MB';
    if (n >= 1e3) return (n / 1e3).toFixed(0) + ' KB';
    return n + ' B';
  }
  function fmtNum(n: number): string {
    if (n >= 1e6) return (n / 1e6).toFixed(2) + 'M';
    if (n >= 1e3) return (n / 1e3).toFixed(2) + 'k';
    return String(Math.round(n));
  }
  function barColor(pct: number) {
    if (pct >= 90) return 'var(--red)';
    if (pct >= 75) return 'var(--yellow)';
    return 'var(--accent)';
  }
  function tempColor(t: number | null) {
    if (t == null) return 'var(--text-tertiary)';
    if (t >= 85) return 'var(--red)';
    if (t >= 70) return 'var(--yellow)';
    return 'var(--blue)';
  }
  function fmtRSS(p: any): string {
    const kb = Number(p?.mem_rss_kb ?? p?.rss_kb ?? 0);
    const bytes = kb > 0 ? kb * 1024 : Number(p?.rss_bytes ?? p?.rss ?? 0);
    if (!bytes) return '—';
    if (bytes >= 1e9) return (bytes / 1e9).toFixed(1) + ' GB';
    if (bytes >= 1e6) return (bytes / 1e6).toFixed(1) + ' MB';
    return (bytes / 1e3).toFixed(0) + ' KB';
  }

  /** Dedupe docker ipv4/ipv6 mappings → "host→container/proto" chips */
  function formatPorts(raw: any[] | null | undefined): string[] {
    if (!raw?.length) return [];
    const seen = new Set<string>();
    const out: string[] = [];
    for (const p of raw) {
      const host = Number(p.host_port || p.PublicPort || 0);
      const cont = Number(p.container_port || p.PrivatePort || 0);
      const proto = String(p.protocol || p.Type || 'tcp').toLowerCase();
      if (!host) continue;
      const key = `${host}:${cont}/${proto}`;
      if (seen.has(key)) continue;
      seen.add(key);
      out.push(host === cont ? `${host}/${proto}` : `${host}→${cont}/${proto}`);
    }
    return out.slice(0, 8);
  }

  async function loadStats() {
    try {
      const res = await fetch('/api/system/stats');
      if (!res.ok) return;
      const d = await res.json();
      hostname = d.hostname ?? hostname;
      distro = d.distro ?? distro;
      kernel = d.kernel ?? kernel;
      cpuPct = d.cpu_percent ?? 0;
      load = [d.load_avg_1 ?? 0, d.load_avg_5 ?? 0, d.load_avg_15 ?? 0].filter((x) => x != null);
      if (!load.length && d.load_avg) load = d.load_avg;

      if (d.mem_used_mb != null) {
        memUsedMB = d.mem_used_mb;
        memUsed = fmtMB(d.mem_used_mb);
        memTotal = fmtMB(d.mem_total_mb);
        memPct = d.mem_total_mb > 0 ? (d.mem_used_mb / d.mem_total_mb) * 100 : 0;
      } else if (d.mem_used_human) {
        memUsed = d.mem_used_human;
        memTotal = d.mem_total_human;
        memPct = d.mem_percent ?? 0;
      }

      if (d.swap_total_mb != null) {
        swapTotal = fmtMB(d.swap_total_mb);
        swapUsed = fmtMB(d.swap_used_mb ?? 0);
        swapPct = d.swap_total_mb > 0 ? ((d.swap_used_mb ?? 0) / d.swap_total_mb) * 100 : 0;
      }

      if (d.disk_used_human) {
        diskUsed = d.disk_used_human;
        diskTotal = d.disk_total_human;
        diskPct = d.disk_percent ?? 0;
      } else if (d.disk_used_gb != null) {
        diskUsed = fmtGB(d.disk_used_gb);
        diskTotal = fmtGB(d.disk_total_gb);
        diskPct = d.disk_total_gb > 0 ? (d.disk_used_gb / d.disk_total_gb) * 100 : 0;
      }

      if (d.uptime_human) uptime = d.uptime_human;
      else if (d.uptime_seconds != null) uptime = fmtUptime(d.uptime_seconds);

      cpuTempC = d.cpu_temp_c ?? null;
      nvmeTempC = d.nvme_temp_c ?? null;

      cpuHist = pushSample(cpuHist, cpuPct);
      memHist = pushSample(memHist, memPct);
    } catch {
    } finally {
      loading = false;
    }
  }

  async function loadInfo() {
    try {
      const res = await fetch('/api/system/info');
      if (res.ok) info = await res.json();
    } catch {}
  }

  async function loadProcesses() {
    try {
      const pRes = await fetch('/api/processes?sort=cpu&order=desc').then((r) => (r.ok ? r.json() : null));
      const procs = pRes?.processes ?? pRes?.items ?? (Array.isArray(pRes) ? pRes : []);
      procCount = procs.length;
      const top = [...procs]
        .sort((a, b) => (b.cpu_percent ?? b.cpu ?? 0) - (a.cpu_percent ?? a.cpu ?? 0))
        .slice(0, PROC_ROWS);
      while (top.length < PROC_ROWS) top.push(null);
      topProcs = top;
    } catch {}
  }

  async function loadSideData() {
    try {
      const [dRes, cList, cronRes, svcRes] = await Promise.all([
        fetch('/api/disks').then((r) => (r.ok ? r.json() : null)).catch(() => null),
        fetch('/api/containers?all=true').then((r) => (r.ok ? r.json() : null)).catch(() => null),
        fetch('/api/cron').then((r) => (r.ok ? r.json() : null)).catch(() => null),
        fetch('/api/services?type=service').then((r) => (r.ok ? r.json() : null)).catch(() => null),
      ]);

      const m = dRes?.mounts ?? dRes?.filesystems ?? [];
      mounts = (Array.isArray(m) ? m : [])
        .filter((x: any) => x.mountpoint || x.mount)
        .map((x: any) => ({
          mount: x.mountpoint || x.mount,
          device: x.device || x.source || '',
          fs: x.fstype || x.fs || '',
          size: x.size_human || x.size || '',
          used: x.used_human || x.used || '',
          pct: x.use_percent ?? x.percent ?? x.usage_percent ?? 0,
        }))
        .filter((x: any) => !String(x.mount).startsWith('/snap'))
        .sort((a: any, b: any) => (b.pct || 0) - (a.pct || 0))
        .slice(0, 8);

      const ctrs = cList?.containers ?? (Array.isArray(cList) ? cList : []);
      containers = ctrs.slice(0, 10).map((c: any) => ({ ...c, portLabels: formatPorts(c.ports) }));
      ctrTotal = ctrs.length;
      ctrRunning = ctrs.filter((c: any) => String(c.state || c.Status || '').toLowerCase().includes('running')).length;

      const jobs = cronRes?.jobs ?? (Array.isArray(cronRes) ? cronRes : []);
      cronCount = jobs.length;
      const now = new Date();
      cronJobs = jobs
        .filter((j: any) => j.enabled !== false)
        .map((j: any) => {
          const nextAt = nextCronRun(j.schedule, now);
          return {
            ...j,
            nextAt,
            nextLabel: nextAt ? fmtCronNext(nextAt, now) : '—',
            shortCmd: shortCronCmd(j.command || '', j.comment || ''),
          };
        })
        .sort((a: any, b: any) => {
          const ta = a.nextAt?.getTime() ?? Number.MAX_SAFE_INTEGER;
          const tb = b.nextAt?.getTime() ?? Number.MAX_SAFE_INTEGER;
          return ta - tb;
        })
        .slice(0, 8);

      const svcs = svcRes?.services ?? svcRes?.units ?? (Array.isArray(svcRes) ? svcRes : []);
      failedServices = svcs
        .filter((s: any) => String(s.active_state || s.state || '').toLowerCase() === 'failed')
        .slice(0, 8);
    } catch {}
  }

  async function toggleMount(mount: string) {
    if (expandedMount === mount) {
      expandedMount = '';
      return;
    }
    expandedMount = mount;
    if (mountUsage[mount]?.entries?.length) return;
    mountUsage = { ...mountUsage, [mount]: { loading: true, error: '', entries: [] } };
    try {
      const res = await fetch(`/api/disks/usage?mount=${encodeURIComponent(mount)}&top=8`);
      const d = await res.json().catch(() => null);
      if (!res.ok) throw new Error(d?.error || `usage ${res.status}`);
      mountUsage = {
        ...mountUsage,
        [mount]: { loading: false, error: '', entries: d?.entries ?? [] },
      };
    } catch (e: any) {
      mountUsage = {
        ...mountUsage,
        [mount]: { loading: false, error: e?.message || 'Failed to load', entries: [] },
      };
    }
  }

  async function containerAction(id: string, action: string) {
    if (!id) return;
    ctrActionPending = `${id}:${action}`;
    try {
      await fetch(`/api/containers/${id}/action?runtime=docker`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action }),
      });
      await loadSideData();
    } catch {
    } finally {
      ctrActionPending = null;
    }
  }

  function procStateClass(state: string): string {
    if (state === 'R') return 'badge-green';
    if (state === 'Z') return 'badge-red';
    if (state === 'D' || state === 'T') return 'badge-yellow';
    return 'badge-gray';
  }

  function shortCronCmd(cmd: string, comment: string): string {
    if (comment && !comment.includes('$Id:') && !comment.includes('user-name command')) return comment;
    const base = cmd.split(/\s+/).find((p) => p.startsWith('/') || p.endsWith('.sh')) || cmd;
    const name = base.split('/').pop() || cmd;
    return name.length > 42 ? name.slice(0, 40) + '…' : name;
  }

  function parseCronField(field: string, min: number, max: number): number[] {
    const out = new Set<number>();
    for (const part of field.split(',')) {
      const stepMatch = part.match(/^(\*|\d+(?:-\d+)?)(?:\/(\d+))?$/);
      if (!stepMatch) continue;
      const [, range, stepStr] = stepMatch;
      const step = stepStr ? Number(stepStr) : 1;
      let start = min;
      let end = max;
      if (range !== '*') {
        if (range.includes('-')) {
          const [a, b] = range.split('-').map(Number);
          start = a;
          end = b;
        } else {
          start = end = Number(range);
        }
      }
      for (let v = start; v <= end; v += step) out.add(v);
    }
    return [...out].sort((a, b) => a - b);
  }

  /** Next run for standard 5-field cron (min hour dom mon dow). */
  function nextCronRun(expr: string, from: Date): Date | null {
    if (!expr) return null;
    const parts = expr.trim().split(/\s+/);
    if (parts.length < 5) return null;
    const [miF, hF, domF, monF, dowF] = parts;
    try {
      const mins = parseCronField(miF, 0, 59);
      const hours = parseCronField(hF, 0, 23);
      const doms = parseCronField(domF, 1, 31);
      const mons = parseCronField(monF, 1, 12);
      const dows = parseCronField(dowF, 0, 7).map((d) => (d === 7 ? 0 : d));
      if (!mins.length || !hours.length || !doms.length || !mons.length || !dows.length) return null;

      const cur = new Date(from.getTime());
      cur.setSeconds(0, 0);
      cur.setMinutes(cur.getMinutes() + 1);

      for (let i = 0; i < 366 * 24 * 60; i++) {
        const mon = cur.getMonth() + 1;
        const dom = cur.getDate();
        const dow = cur.getDay();
        const hour = cur.getHours();
        const min = cur.getMinutes();
        const monOk = mons.includes(mon);
        const timeOk = hours.includes(hour) && mins.includes(min);
        // cron: if both DOM and DOW are restricted, either may match
        const domStar = domF === '*';
        const dowStar = dowF === '*';
        let dayOk = true;
        if (!domStar && !dowStar) dayOk = doms.includes(dom) || dows.includes(dow);
        else {
          if (!domStar) dayOk = dayOk && doms.includes(dom);
          if (!dowStar) dayOk = dayOk && dows.includes(dow);
        }
        if (monOk && dayOk && timeOk) return new Date(cur.getTime());
        cur.setMinutes(cur.getMinutes() + 1);
      }
    } catch {
      return null;
    }
    return null;
  }

  function fmtCronNext(at: Date, now: Date): string {
    const ms = at.getTime() - now.getTime();
    const mins = Math.round(ms / 60000);
    if (mins < 60) return `in ${Math.max(1, mins)}m`;
    const hrs = Math.round(mins / 60);
    if (hrs < 36) return `in ${hrs}h`;
    const days = Math.round(hrs / 24);
    if (days < 14) return `in ${days}d`;
    return at.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
  }

  async function loadCloudflare() {
    try {
      const res = await fetch('/api/cloudflare/overview');
      const d = await res.json().catch(() => null);
      if (!res.ok || d?.configured === false) {
        cfError = d?.error || `Cloudflare unavailable (${res.status})`;
        cf = null;
        return;
      }
      cf = d;
      cfError = '';
    } catch (e: any) {
      cfError = e?.message || 'Cloudflare fetch failed';
      cf = null;
    }
  }

  function capabilities(): { label: string; active: boolean }[] {
    if (!info) return [];
    return [
      { label: 'Docker', active: info.has_docker },
      { label: 'Podman', active: info.has_podman },
      { label: 'Ansible', active: info.has_ansible },
      { label: 'Puppet', active: info.has_puppet },
      { label: 'UFW', active: info.has_ufw },
      { label: 'nftables', active: info.has_nftables },
      { label: 'iptables', active: info.has_iptables },
    ].filter((c) => c.active !== undefined);
  }

  let tempPct = $derived(cpuTempC != null ? Math.min(100, cpuTempC) : 0);

  let cfReqPoints = $derived((cf?.series ?? []).map((s: any) => ({ t: s.t, v: Number(s.requests || 0) })));
  let cfBytePoints = $derived((cf?.series ?? []).map((s: any) => ({ t: s.t, v: Number(s.bytes || 0) })));
  let cfStatusRows = $derived(
    (cf?.series ?? []).map((s: any) => ({
      t: s.t,
      values: {
        s2xx: Number(s.s2xx || 0),
        s3xx: Number(s.s3xx || 0),
        s4xx: Number(s.s4xx || 0),
        s5xx: Number(s.s5xx || 0),
      },
    })),
  );
  let cfCacheRows = $derived(
    (cf?.series ?? []).map((s: any) => ({
      t: s.t,
      values: {
        hit: Number(s.cached_requests || 0),
        miss: Number(s.uncached || Math.max(0, (s.requests || 0) - (s.cached_requests || 0))),
      },
    })),
  );
  const statusSeries = [
    { key: 's2xx', label: '2xx', color: '#4e8de0' },
    { key: 's3xx', label: '3xx', color: '#e0c44e' },
    { key: 's4xx', label: '4xx', color: '#e879a9' },
    { key: 's5xx', label: '5xx', color: '#a78bfa' },
  ];
  const cacheSeries = [
    { key: 'hit', label: 'Cached', color: '#e879a9' },
    { key: 'miss', label: 'Uncached', color: '#e0c44e' },
  ];
  let cfStatusTotals = $derived.by(() => {
    const map: Record<string, number> = { '2xx': 0, '3xx': 0, '4xx': 0, '5xx': 0 };
    for (const s of cf?.status ?? []) map[s.code] = s.requests || 0;
    return map;
  });
  let cfVisits = $derived(Number(cf?.totals?.page_views ?? cf?.totals?.uniques_sum ?? 0));

  onMount(() => {
    loadStats();
    loadInfo();
    loadProcesses();
    loadSideData();
    loadCloudflare();
    const statsIv = setInterval(loadStats, 4000);
    const procIv = setInterval(loadProcesses, 3000);
    const sideIv = setInterval(loadSideData, 20000);
    const cfIv = setInterval(loadCloudflare, 60000);
    return () => {
      clearInterval(statsIv);
      clearInterval(procIv);
      clearInterval(sideIv);
      clearInterval(cfIv);
    };
  });
</script>

<div class="dash">
  <div class="dash-header">
    <div>
      <h1>{hostname || 'Dashboard'}</h1>
      <p class="subtitle">
        {#if distro}{distro}{/if}
        {#if kernel} · {kernel}{/if}
        {#if uptime} · up {uptime}{/if}
      </p>
    </div>
    {#if info}
      {@const caps = capabilities()}
      {#if caps.length}
        <div class="caps-pills">
          {#each caps as cap}
            <span class="cap-pill" class:cap-active={cap.active} class:cap-inactive={!cap.active}>
              <span class="cap-dot"></span>{cap.label}
            </span>
          {/each}
        </div>
      {/if}
    {/if}
  </div>

  <!-- CasaOS-style system status gauges -->
  <div class="status-panel">
    <div class="status-title">System status</div>
    <div class="gauges">
      <Gauge pct={loading ? 0 : cpuPct} color={barColor(cpuPct)} label="CPU">
        <div class="gauge-meta mono" style="color:{tempColor(cpuTempC)}">
          {#if cpuTempC != null}{cpuTempC.toFixed(0)}°C{:else}—{/if}
          {#if load[0] != null}<span class="dim"> · load {load[0].toFixed(2)}</span>{/if}
        </div>
      </Gauge>

      <Gauge pct={loading ? 0 : memPct} color="var(--blue)" label="Memory">
        <div class="gauge-meta mono">{memUsed || '—'}{#if memTotal}<span class="dim"> / {memTotal}</span>{/if}</div>
      </Gauge>

      <Gauge pct={loading ? 0 : diskPct} color={barColor(diskPct)} label="Disk (/)">
        <div class="gauge-meta mono">{diskUsed || '—'}{#if diskTotal}<span class="dim"> / {diskTotal}</span>{/if}</div>
      </Gauge>

      <Gauge
        pct={loading ? 0 : tempPct}
        color={tempColor(cpuTempC)}
        textColor={tempColor(cpuTempC)}
        label="Thermals"
        format={(n) => (cpuTempC == null && n === 0 ? '—' : `${Math.round(n)}°`)}
      >
        <div class="gauge-meta mono">
          CPU {cpuTempC != null ? cpuTempC.toFixed(0) + '°C' : '—'}
          <span class="dim"> · NVMe {nvmeTempC != null ? nvmeTempC.toFixed(0) + '°C' : '—'}</span>
        </div>
      </Gauge>
    </div>

    <div class="status-strip">
      <div class="strip-item">
        <span class="strip-label">Swap</span>
        <span class="strip-val mono">{swapTotal ? `${swapPct.toFixed(0)}%` : 'off'}</span>
        <div class="mini-bar"><div class="mini-fill" style="width:{Math.min(swapPct,100)}%;background:{barColor(swapPct)}"></div></div>
      </div>
      <div class="strip-item">
        <span class="strip-label">Uptime</span>
        <span class="strip-val mono">{uptime || '—'}</span>
      </div>
      <div class="strip-item">
        <span class="strip-label">Load 5 / 15</span>
        <span class="strip-val mono">{load[1]?.toFixed(2) ?? '—'} / {load[2]?.toFixed(2) ?? '—'}</span>
      </div>
      <a class="strip-item linkish" href="#/services">
        <span class="strip-label">Failed units</span>
        <span class="strip-val mono" style="color:{failedServices.length ? 'var(--red)' : 'var(--accent)'}">{failedServices.length}</span>
      </a>
    </div>
  </div>

  <!-- Cloudflare -->
  <div class="panel cf-panel">
    <div class="panel-head">
      <div>
        <h2>Cloudflare edge</h2>
        <p class="panel-sub">{cf?.zone_name || 'anmaritamang.com'} · last 24h</p>
      </div>
      {#if cf?.fetched_at}
        <span class="panel-meta mono">updated {new Date(cf.fetched_at).toLocaleTimeString()}</span>
      {/if}
    </div>

    {#if cfError && !cf}
      <div class="empty">{cfError}</div>
    {:else if !cf}
      <div class="empty">Loading Cloudflare analytics…</div>
    {:else}
      <div class="cf-top">
        <div class="cf-stat"><span class="kpi-label">Requests</span><span class="big mono">{fmtNum(cf.totals.requests)}</span></div>
        <div class="cf-stat"><span class="kpi-label">Transfer</span><span class="big mono">{fmtBytes(cf.totals.bytes)}</span></div>
        <div class="cf-stat"><span class="kpi-label">Cache hit</span><span class="big mono">{cf.totals.cache_hit_rate.toFixed(1)}%</span></div>
        <div class="cf-stat">
          <span class="kpi-label">Total visits</span>
          <span class="big mono">{fmtNum(cfVisits)}</span>
        </div>
      </div>

      <div class="cf-grid">
        <div class="cf-chart">
          <div class="cf-chart-head">
            <div class="mini-title">Requests over time</div>
            <span class="cf-series-label"><i style="background:#4e8de0"></i> Requests {fmtNum(cf.totals.requests)}</span>
          </div>
          <HoverLineChart points={cfReqPoints} color="#4e8de0" formatValue={fmtNum} height={150} />
        </div>
        <div class="cf-chart">
          <div class="cf-chart-head">
            <div class="mini-title">Data transfer</div>
            <span class="cf-series-label"><i style="background:#4e8de0"></i> Data transfer {fmtBytes(cf.totals.bytes)}</span>
          </div>
          <HoverLineChart points={cfBytePoints} color="#4e8de0" formatValue={fmtBytes} height={150} />
        </div>
        <div class="cf-chart">
          <div class="cf-chart-head">
            <div class="mini-title">Requests by status code</div>
            <div class="cf-series-row">
              {#each statusSeries as s}
                <span class="cf-series-label"><i style="background:{s.color}"></i> {s.label} {fmtNum(cfStatusTotals[s.label] || 0)}</span>
              {/each}
            </div>
          </div>
          <StackedBarChart rows={cfStatusRows} series={statusSeries} formatValue={fmtNum} height={150} />
        </div>
        <div class="cf-chart">
          <div class="cf-chart-head">
            <div class="mini-title">Requests by cache status</div>
            <div class="cf-series-row">
              {#each cacheSeries as s}
                <span class="cf-series-label"><i style="background:{s.color}"></i> {s.label} {fmtNum(s.key === 'hit' ? cf.totals.cached_requests : Math.max(0, cf.totals.requests - cf.totals.cached_requests))}</span>
              {/each}
            </div>
          </div>
          <StackedBarChart rows={cfCacheRows} series={cacheSeries} formatValue={fmtNum} height={150} />
        </div>
      </div>
    {/if}
  </div>

  <div class="ops-grid">
    <div class="panel">
      <div class="panel-head">
        <h2>Top processes <span class="live-dot" title="refreshes every 3s"></span> <span class="count mono">{procCount}</span></h2>
        <a class="panel-link" href="#/processes">View all →</a>
      </div>
      <table class="data-table compact proc-table">
        <thead>
          <tr><th>PID</th><th>Name</th><th>User</th><th>CPU</th><th>RSS</th><th>State</th></tr>
        </thead>
        <tbody>
          {#each topProcs as p}
            {#if p}
              <tr>
                <td class="mono">{p.pid}</td>
                <td class="proc-cell" title={p.cmdline || p.name || p.comm}>
                  <div class="proc-name">{p.name || p.comm}</div>
                  <div class="proc-cmd">{p.cmdline || '\u00a0'}</div>
                </td>
                <td class="mono">{p.user || p.username || '—'}</td>
                <td class="mono" style="color:{(p.cpu_percent??p.cpu??0) > 20 ? 'var(--red)' : 'var(--accent)'}">
                  {Number(p.cpu_percent ?? p.cpu ?? 0).toFixed(1)}%
                </td>
                <td class="mono">{fmtRSS(p)}</td>
                <td>
                  <span class="badge {procStateClass(p.state || '')}" title={p.state_name || p.state || ''}>
                    {p.state || '—'}
                  </span>
                </td>
              </tr>
            {:else}
              <tr class="proc-placeholder">
                <td class="mono dim">—</td>
                <td class="proc-cell">
                  <div class="proc-name dim">—</div>
                  <div class="proc-cmd">{"\u00a0"}</div>
                </td>
                <td class="mono dim">—</td>
                <td class="mono dim">—</td>
                <td class="mono dim">—</td>
                <td><span class="badge badge-gray">—</span></td>
              </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    </div>

    <div class="panel">
      <div class="panel-head">
        <h2>Storage</h2>
        <a class="panel-link" href="#/disks">Disks →</a>
      </div>
      {#each mounts as m}
        {@const open = expandedMount === m.mount}
        {@const usage = mountUsage[m.mount]}
        <div class="mount-row" class:mount-open={open}>
          <button type="button" class="mount-toggle" onclick={() => toggleMount(m.mount)}>
            <span class="chev" class:open>{open ? '▾' : '▸'}</span>
            <div class="mount-body">
              <div class="mount-top">
                <span class="mono mount-path">{m.mount}</span>
                <span class="mono mount-pct" style="color:{barColor(m.pct)}">{Number(m.pct).toFixed(0)}%</span>
              </div>
              <div class="mini-bar"><div class="mini-fill" style="width:{Math.min(Number(m.pct)||0,100)}%;background:{barColor(Number(m.pct)||0)}"></div></div>
              <div class="kpi-sub mono">{m.used || '—'} / {m.size || '—'} · {m.fs || '?'}</div>
            </div>
          </button>
          {#if open}
            <div class="mount-files">
              {#if usage?.loading}
                <div class="kpi-sub">Scanning largest paths…</div>
              {:else if usage?.error}
                <div class="kpi-sub" style="color:var(--red)">{usage.error}</div>
              {:else if usage?.entries?.length}
                {#each usage.entries as e}
                  <div class="file-row">
                    <span class="mono file-path" title={e.path}>{e.path}</span>
                    <span class="mono file-size">{e.human || fmtBytes(e.bytes || 0)}</span>
                  </div>
                {/each}
              {:else}
                <div class="kpi-sub">No large entries found</div>
              {/if}
            </div>
          {/if}
        </div>
      {:else}
        <div class="empty">No mount data</div>
      {/each}
    </div>
  </div>

  <div class="ops-grid">
    <div class="panel">
      <div class="panel-head">
        <h2>Containers <span class="count mono">{ctrRunning}/{ctrTotal}</span></h2>
        <a class="panel-link" href="#/containers">Manage →</a>
      </div>
      <table class="data-table compact ctr-table">
        <thead>
          <tr><th>Name</th><th>Image</th><th>State</th><th>Ports</th><th>Actions</th></tr>
        </thead>
        <tbody>
          {#each containers as c}
            {@const running = String(c.state || '').toLowerCase().includes('running')}
            {@const cid = c.id || c.Id || ''}
            {@const ports = c.portLabels ?? []}
            <tr>
              <td><div class="proc-name trunc">{c.name || c.Names?.[0] || cid?.slice?.(0,12)}</div></td>
              <td class="mono trunc">{c.image || c.Image || '—'}</td>
              <td>
                <span class="badge" class:badge-green={running} class:badge-gray={!running}>
                  {c.state || c.Status || '—'}
                </span>
              </td>
              <td class="ports-cell" title={ports.join(', ') || ''}>
                {#if ports.length}
                  {#each ports.slice(0, 3) as label}
                    <span class="badge badge-blue">{label}</span>
                  {/each}
                  {#if ports.length > 3}
                    <span class="badge badge-gray">+{ports.length - 3}</span>
                  {/if}
                {:else}
                  <span class="dim">—</span>
                {/if}
              </td>
              <td class="actions-cell">
                <div class="ctr-actions">
                  {#if running}
                    <button
                      class="btn-act"
                      disabled={ctrActionPending === `${cid}:stop`}
                      onclick={() => containerAction(cid, 'stop')}
                      title="Stop">Stop</button>
                    <button
                      class="btn-act"
                      disabled={ctrActionPending === `${cid}:restart`}
                      onclick={() => containerAction(cid, 'restart')}
                      title="Restart">Restart</button>
                  {:else}
                    <button
                      class="btn-act"
                      disabled={ctrActionPending === `${cid}:start`}
                      onclick={() => containerAction(cid, 'start')}
                      title="Start">Start</button>
                  {/if}
                </div>
              </td>
            </tr>
          {:else}
            <tr><td colspan="5" class="empty-cell">No containers</td></tr>
          {/each}
        </tbody>
      </table>
    </div>

    <div class="panel">
      <div class="panel-head">
        <h2>Cron <span class="count mono">{cronCount}</span></h2>
        <a class="panel-link" href="#/cron">Manage →</a>
      </div>
      <table class="data-table compact">
        <thead>
          <tr><th>Next</th><th>Schedule</th><th>Job</th><th>Owner</th></tr>
        </thead>
        <tbody>
          {#each cronJobs as j}
            <tr>
              <td class="mono" style="color:var(--accent)">{j.nextLabel}</td>
              <td class="mono trunc" title={j.schedule}>{j.schedule}</td>
              <td>
                <div class="proc-name trunc" title={j.command}>{j.shortCmd}</div>
                {#if j.comment && j.shortCmd !== j.comment}
                  <div class="proc-cmd">{j.command?.split('/').pop()}</div>
                {/if}
              </td>
              <td class="mono">{j.owner || '—'}</td>
            </tr>
          {:else}
            <tr><td colspan="4" class="empty-cell">No cron jobs</td></tr>
          {/each}
        </tbody>
      </table>
      {#if failedServices.length}
        <div class="failed-block">
          <div class="mini-title" style="color:var(--red)">Failed services</div>
          {#each failedServices as s}
            <div class="failed-row mono">{s.name || s.unit}</div>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  <div class="health-row">
    <HealthChecks />
  </div>
</div>

<style>
  .dash {
    max-width: none; padding-bottom: 220px;
    display: flex; flex-direction: column; gap: 0.85rem;
  }
  .dash-header { display:flex; align-items:flex-start; justify-content:space-between; gap:1rem; flex-wrap:wrap; margin:0; }
  .dash-header h1 { font-size:1.3rem; font-weight:700; margin:0 0 0.2rem; }
  .subtitle { font-size:0.8rem; color:var(--text-secondary); margin:0; }

  .caps-pills { display:flex; flex-wrap:wrap; gap:0.55rem; }
  .cap-pill { display:flex; align-items:center; gap:0.3rem; font-size:0.72rem; font-family:var(--font-mono); }
  .cap-active { color:var(--text-secondary); }
  .cap-inactive { color:var(--text-tertiary); opacity:0.45; }
  .cap-dot { width:6px; height:6px; border-radius:50%; background:var(--accent); box-shadow:0 0 4px var(--accent); }
  .cap-inactive .cap-dot { background:var(--text-tertiary); box-shadow:none; }

  .status-panel, .panel {
    background:var(--bg-panel); border:1px solid var(--border-subtle); border-radius:var(--r-lg);
    padding:1rem 1.1rem; margin:0;
  }
  .status-title, .mini-title, .kpi-label {
    font-size:0.65rem; font-weight:600; text-transform:uppercase; letter-spacing:.1em; color:var(--text-tertiary);
  }
  .status-title { margin-bottom:0.75rem; }

  .gauges { display:grid; grid-template-columns:repeat(4,1fr); gap:0.5rem; }
  .gauge-meta { font-size:0.72rem; margin-top:0.15rem; }
  .dim { color:var(--text-tertiary); }

  .status-strip { display:grid; grid-template-columns:repeat(4,1fr); gap:0.65rem; margin-top:1rem; padding-top:0.85rem; border-top:1px solid var(--border-subtle); }
  .strip-item { display:flex; flex-direction:column; gap:0.2rem; }
  .strip-item.linkish { text-decoration:none; }
  .strip-label { font-size:0.62rem; text-transform:uppercase; letter-spacing:.08em; color:var(--text-tertiary); }
  .strip-val { font-size:0.95rem; font-weight:600; color:var(--text-primary); }
  .mini-bar { height:4px; background:var(--bg-raised); border-radius:2px; overflow:hidden; margin-top:0.2rem; }
  .mini-fill { height:100%; border-radius:2px; transition: width 0.75s cubic-bezier(0.22, 1, 0.36, 1), background 0.45s ease; }

  .panel-head { display:flex; align-items:baseline; justify-content:space-between; margin-bottom:0.75rem; gap:0.75rem; }
  .panel-head h2 { font-size:0.95rem; font-weight:600; margin:0; text-transform:none; letter-spacing:0; color:var(--text-primary); }
  .panel-link { font-size:0.75rem; color:var(--accent); text-decoration:none; }
  .panel-sub, .panel-meta, .kpi-sub { font-size:0.7rem; color:var(--text-tertiary); }
  .count { font-size:0.75rem; color:var(--text-tertiary); font-weight:500; margin-left:0.35rem; }
  .live-dot { display:inline-block; width:7px; height:7px; border-radius:50%; background:var(--accent); box-shadow:0 0 6px var(--accent); margin:0 0.2rem; vertical-align:middle; animation:pulse 1.6s infinite; }
  @keyframes pulse { 50% { opacity:0.35; } }

  .cf-top { display:grid; grid-template-columns:repeat(4,1fr); gap:0.65rem; margin-bottom:0.85rem; }
  .cf-stat { background:var(--bg-raised); border-radius:var(--r-md); padding:0.65rem 0.75rem; }
  .big { display:block; font-size:1.25rem; font-weight:700; color:var(--text-primary); margin-top:0.2rem; }
  .cf-panel { overflow: visible; }
  .cf-grid { display:grid; grid-template-columns:1fr 1fr; gap:0.85rem; overflow: visible; }
  .cf-chart { overflow: visible; position: relative; z-index: 1; }
  .cf-chart:hover { z-index: 4; }
  .cf-chart-head { display:flex; flex-wrap:wrap; align-items:baseline; justify-content:space-between; gap:0.4rem; margin-bottom:0.45rem; }
  .cf-series-row { display:flex; flex-wrap:wrap; gap:0.55rem; }
  .cf-series-label { font-size:0.7rem; color:var(--text-tertiary); white-space:nowrap; }
  .cf-series-label i { display:inline-block; width:8px; height:8px; border-radius:50%; margin-right:0.25rem; vertical-align:middle; }

  .ops-grid {
    display:grid; grid-template-columns:1.3fr 1fr; gap:0.85rem; margin:0;
    align-items: stretch;
  }
  .ops-grid > .panel {
    display: flex; flex-direction: column; min-height: 0;
  }
  .ops-grid > .panel > .data-table { flex: 1; }
  .compact { font-size:0.8rem; }
  .compact th, .compact td { padding:0.45rem 0.55rem; vertical-align: middle; }
  .proc-table { table-layout:fixed; width:100%; }
  .proc-table tbody { height: calc(10 * 2.85rem); }
  .proc-table tbody tr { height:2.85rem; max-height:2.85rem; }
  .proc-table tbody td {
    height:2.85rem; max-height:2.85rem; padding-top:0.3rem; padding-bottom:0.3rem;
    vertical-align:middle; overflow:hidden; box-sizing:border-box;
  }
  .proc-cell { line-height:1.15; }
  .proc-placeholder td { opacity:0.35; }
  .proc-name {
    font-weight:500; color:var(--text-primary);
    overflow:hidden; text-overflow:ellipsis; white-space:nowrap;
    height:1.05rem; line-height:1.05rem;
  }
  .proc-cmd {
    font-size:0.68rem; color:var(--text-tertiary);
    overflow:hidden; text-overflow:ellipsis; white-space:nowrap;
    height:0.9rem; line-height:0.9rem;
  }
  .trunc { max-width:180px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }

  .ctr-table { table-layout: fixed; width: 100%; }
  .ctr-table tbody tr { height: 2.6rem; }
  .ctr-table tbody td {
    height: 2.6rem; max-height: 2.6rem; vertical-align: middle;
    overflow: hidden; box-sizing: border-box;
  }
  .ports-cell {
    display: flex; flex-wrap: nowrap; align-items: center; gap: 0.25rem;
    min-width: 0; overflow: hidden;
  }
  .ports-cell .badge { flex-shrink: 0; }
  .actions-cell { width: 8.5rem; }
  .ctr-actions {
    display: flex; gap: 0.3rem; align-items: center; justify-content: flex-start;
    height: 100%;
  }
  .btn-act {
    background: transparent; border: 1px solid var(--border-default); color: var(--text-secondary);
    border-radius: var(--r-sm); height: 1.65rem; padding: 0 0.45rem; font-size: 0.72rem;
    cursor: pointer; line-height: 1; white-space: nowrap;
  }
  .btn-act:hover:not(:disabled) { color: var(--text-primary); border-color: var(--accent); background: var(--accent-dim); }
  .btn-act:disabled { opacity: 0.4; cursor: wait; }
  .mount-row { margin-bottom: 0.65rem; }
  .mount-row:last-child { margin-bottom: 0; }
  .mount-toggle {
    display:flex; gap:0.45rem; align-items:flex-start; width:100%;
    background:transparent; border:0; padding:0; text-align:left; cursor:pointer; color:inherit;
  }
  .chev { color:var(--text-tertiary); font-size:var(--icon-sm); line-height:1.4; width:1.1rem; flex-shrink:0; }
  .chev.open { color:var(--accent); }
  .mount-body { flex:1; min-width:0; }
  .mount-top { display:flex; justify-content:space-between; margin-bottom:0.25rem; }
  .mount-path { font-size:0.8rem; color:var(--text-primary); }
  .mount-pct { font-size:0.8rem; font-weight:600; }
  .mount-files {
    margin:0.4rem 0 0.2rem 1.3rem; padding:0.45rem 0.55rem;
    background:var(--bg-raised); border-radius:var(--r-md); border:1px solid var(--border-subtle);
  }
  .file-row { display:flex; justify-content:space-between; gap:0.75rem; padding:0.18rem 0; font-size:0.72rem; }
  .file-path { color:var(--text-secondary); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; min-width:0; }
  .file-size { color:var(--text-primary); flex-shrink:0; }
  .failed-block { margin-top:0.85rem; padding-top:0.75rem; border-top:1px solid var(--border-subtle); }
  .failed-row { font-size:0.75rem; color:var(--red); padding:0.15rem 0; }
  .empty, .empty-cell { color:var(--text-tertiary); font-size:0.85rem; padding:0.75rem 0; text-align:center; }

  .health-row { width:100%; }

  @media (max-width: 1100px) {
    .gauges, .status-strip, .cf-top, .cf-grid { grid-template-columns:repeat(2,1fr); }
    .ops-grid { grid-template-columns:1fr; }
  }
  @media (max-width: 900px) {
    .dash-header { flex-direction: column; align-items: stretch; gap: 0.65rem; }
    .caps-pills { justify-content: flex-start; max-width: 100%; }
    .status-panel, .panel { padding: 0.85rem; }
    .proc-table, .ctr-table { min-width: 520px; }
    .ops-grid > .panel { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  }
  @media (max-width: 700px) {
    .cf-grid, .cf-top, .status-strip { grid-template-columns: 1fr; }
    .gauges { grid-template-columns: repeat(2, 1fr); gap: 0.75rem; }
    .big { font-size: 1.1rem; }
    .dash { gap: 0.75rem; }
  }
  @media (max-width: 420px) {
    .gauges { grid-template-columns: 1fr 1fr; gap: 0.5rem; }
    .cap-pill { font-size: 0.65rem; }
  }
</style>
