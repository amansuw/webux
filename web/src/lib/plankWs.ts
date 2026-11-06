/** Hermes JSON-RPC client over WebUX `/ws/plank` bridge. */

export type MascotState =
  | 'idle'
  | 'welcome'
  | 'thinking'
  | 'searching'
  | 'coding'
  | 'working'
  | 'organizing'
  | 'heavy'
  | 'success'
  | 'error'
  | 'sleeping'
  | 'approval';

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  toolName?: string;
  toolId?: string;
  pending?: boolean;
}

export interface PlankSession {
  id: string;
  title: string;
  preview: string;
  started_at?: string;
  message_count?: number;
  source?: string;
}

export interface PlankStatus {
  online: boolean;
  serving: boolean;
  url: string;
  model: string;
  provider: string;
  skin: string;
  personality: string;
  agent_name: string;
  tagline: string;
  error?: string;
  started_by_us?: boolean;
}

type EventHandler = (ev: {
  type: string;
  session_id?: string;
  payload?: Record<string, unknown>;
}) => void;

type RpcResult = { ok: true; result: unknown } | { ok: false; error: string };

function uid(prefix = 'm'): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 10)}`;
}

function classifyTool(name: string): MascotState {
  const n = name.toLowerCase();
  if (/search|web|browser|ddg|crawl|fetch_url/.test(n)) return 'searching';
  if (/code|shell|terminal|file|edit|write|read|git|patch/.test(n)) return 'coding';
  if (/memory|kanban|db|sql|organize|note/.test(n)) return 'organizing';
  return 'working';
}

/** Short label for a tool chip — never duplicate the tool name. */
function toolPreview(payload: Record<string, unknown>, name: string): string {
  const ctx = String(payload.context ?? payload.summary ?? '').trim();
  if (ctx && ctx.toLowerCase() !== name.toLowerCase()) return ctx.slice(0, 120);

  const args = payload.args;
  if (args && typeof args === 'object' && !Array.isArray(args)) {
    const a = args as Record<string, unknown>;
    for (const key of ['command', 'cmd', 'query', 'path', 'file', 'url', 'pattern']) {
      const v = a[key];
      if (typeof v === 'string' && v.trim()) return v.trim().slice(0, 120);
    }
  }
  return '';
}

export class PlankClient {
  private ws: WebSocket | null = null;
  private pending = new Map<string, { resolve: (r: RpcResult) => void }>();
  private reqSeq = 0;
  private destroyed = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private sleepTimer: ReturnType<typeof setTimeout> | null = null;
  private successTimer: ReturnType<typeof setTimeout> | null = null;

  sessionId: string | null = null;
  connected = false;
  running = false;
  mascot: MascotState = 'welcome';
  messages: ChatMessage[] = [];
  sessions: PlankSession[] = [];
  toolsActive = new Set<string>();

  onChange: (() => void) | null = null;
  onEvent: EventHandler | null = null;

  connect() {
    if (this.destroyed) return;
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = localStorage.getItem('webux_token');
    const q = token ? `?access_token=${encodeURIComponent(token)}` : '';
    // Cookie auth is primary; bearer query unused by server but harmless.
    this.ws = new WebSocket(`${proto}//${location.host}/ws/plank${q}`);

    this.ws.onopen = () => {
      this.connected = true;
      this.setMascot(this.messages.length === 0 ? 'welcome' : 'idle');
      this.notify();
      this.armSleep();
    };

    this.ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(String(e.data));
        this.handleFrame(msg);
      } catch {
        /* ignore non-json */
      }
    };

    this.ws.onclose = () => {
      this.connected = false;
      this.running = false;
      this.notify();
      if (!this.destroyed) {
        this.reconnectTimer = setTimeout(() => this.connect(), 2500);
      }
    };

    this.ws.onerror = () => this.ws?.close();
  }

  destroy() {
    this.destroyed = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.sleepTimer) clearTimeout(this.sleepTimer);
    if (this.successTimer) clearTimeout(this.successTimer);
    this.ws?.close();
    this.ws = null;
  }

  wake() {
    if (this.mascot === 'sleeping') this.setMascot('idle');
    this.armSleep();
  }

  private armSleep() {
    if (this.sleepTimer) clearTimeout(this.sleepTimer);
    this.sleepTimer = setTimeout(() => {
      if (!this.running && this.connected) this.setMascot('sleeping');
      this.notify();
    }, 90_000);
  }

  private setMascot(s: MascotState) {
    this.mascot = s;
  }

  private notify() {
    this.onChange?.();
  }

  private nextId(): string {
    this.reqSeq += 1;
    return `p${this.reqSeq}`;
  }

  private rpc(method: string, params: Record<string, unknown> = {}): Promise<RpcResult> {
    return new Promise((resolve) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        resolve({ ok: false, error: 'Cannot. Reason: bridge offline.' });
        return;
      }
      const id = this.nextId();
      this.pending.set(id, { resolve });
      this.ws.send(JSON.stringify({ jsonrpc: '2.0', id, method, params }));
      setTimeout(() => {
        if (this.pending.has(id)) {
          this.pending.delete(id);
          resolve({ ok: false, error: 'Cannot. Reason: RPC timeout.' });
        }
      }, 60_000);
    });
  }

  private handleFrame(msg: Record<string, unknown>) {
    if (msg.method === 'event') {
      const params = (msg.params ?? {}) as {
        type?: string;
        session_id?: string;
        payload?: Record<string, unknown>;
      };
      this.handleEvent(params.type ?? '', params.session_id, params.payload ?? {});
      return;
    }

    if (msg.id != null) {
      const id = String(msg.id);
      const waiter = this.pending.get(id);
      if (!waiter) return;
      this.pending.delete(id);
      if (msg.error) {
        const err = msg.error as { message?: string };
        waiter.resolve({ ok: false, error: err.message || 'RPC error' });
      } else {
        waiter.resolve({ ok: true, result: msg.result });
      }
    }
  }

  private handleEvent(type: string, sessionId: string | undefined, payload: Record<string, unknown>) {
    this.onEvent?.({ type, session_id: sessionId, payload });
    this.wake();

    switch (type) {
      case 'gateway.ready':
      case 'plank.bridge':
        this.setMascot(this.messages.length === 0 ? 'welcome' : 'idle');
        break;

      case 'message.start':
        this.running = true;
        this.setMascot('thinking');
        if (!this.messages.some((m) => m.pending && m.role === 'assistant')) {
          this.messages = [
            ...this.messages,
            { id: uid('a'), role: 'assistant', content: '', pending: true },
          ];
        }
        break;

      case 'message.delta': {
        const text = String(payload.text ?? '');
        this.appendAssistant(text);
        if (this.toolsActive.size === 0) this.setMascot('thinking');
        break;
      }

      case 'thinking.delta':
      case 'reasoning.delta':
        this.running = true;
        this.setMascot('thinking');
        break;

      case 'tool.start': {
        this.running = true;
        const name = String(payload.name ?? payload.tool ?? 'tool');
        const tid = String(payload.tool_id ?? name);
        this.toolsActive.add(tid);
        const preview = toolPreview(payload, name);
        this.messages = [
          ...this.messages,
          {
            id: uid('t'),
            role: 'tool',
            content: preview,
            toolName: name,
            toolId: tid,
            pending: true,
          },
        ];
        this.setMascot(classifyTool(name));
        break;
      }

      case 'tool.progress':
      case 'tool.generating':
        this.setMascot(this.mascot === 'searching' || this.mascot === 'coding' || this.mascot === 'organizing' ? this.mascot : 'working');
        break;

      case 'tool.complete': {
        const tid = String(payload.tool_id ?? '');
        const name = String(payload.name ?? payload.tool ?? '');
        if (tid) this.toolsActive.delete(tid);
        const summary = String(payload.summary ?? '').trim();
        if (tid || name) {
          this.messages = this.messages.map((m) => {
            if (m.role !== 'tool') return m;
            const match = (tid && m.toolId === tid) || (!tid && m.toolName === name && m.pending);
            if (!match) return m;
            return {
              ...m,
              pending: false,
              content: summary && summary.toLowerCase() !== (m.toolName || '').toLowerCase()
                ? summary
                : m.content,
            };
          });
        }
        if (this.toolsActive.size === 0 && this.running) this.setMascot('thinking');
        break;
      }

      case 'status.update': {
        const kind = String(payload.kind ?? '');
        if (kind === 'compacting' || kind === 'loop' || kind === 'process') {
          this.setMascot('heavy');
        }
        break;
      }

      case 'approval.request':
      case 'clarify.request':
      case 'sudo.request':
      case 'secret.request':
        this.setMascot('approval');
        break;

      case 'message.complete': {
        const text = payload.text != null ? String(payload.text) : '';
        if (text) this.finalizeAssistant(text);
        else this.finalizeAssistant();
        this.running = false;
        this.toolsActive.clear();
        this.flashSuccess();
        break;
      }

      case 'error': {
        const message = String(payload.message ?? 'Unknown fault');
        this.running = false;
        this.toolsActive.clear();
        this.messages = [
          ...this.messages.filter((m) => !m.pending),
          { id: uid('e'), role: 'system', content: `Cannot. Reason: ${message}` },
        ];
        this.setMascot('error');
        if (this.successTimer) clearTimeout(this.successTimer);
        this.successTimer = setTimeout(() => {
          this.setMascot('idle');
          this.notify();
        }, 2200);
        break;
      }
    }

    this.notify();
  }

  private appendAssistant(delta: string) {
    const idx = [...this.messages].reverse().findIndex((m) => m.role === 'assistant');
    if (idx === -1) {
      this.messages = [...this.messages, { id: uid('a'), role: 'assistant', content: delta, pending: true }];
      return;
    }
    const real = this.messages.length - 1 - idx;
    const cur = this.messages[real];
    this.messages = this.messages.map((m, i) =>
      i === real ? { ...m, content: m.content + delta, pending: true } : m
    );
    void cur;
  }

  private finalizeAssistant(full?: string) {
    const idx = [...this.messages].reverse().findIndex((m) => m.role === 'assistant');
    if (idx === -1) {
      if (full) {
        this.messages = [...this.messages, { id: uid('a'), role: 'assistant', content: full }];
      }
      return;
    }
    const real = this.messages.length - 1 - idx;
    this.messages = this.messages.map((m, i) =>
      i === real
        ? { ...m, content: full != null && full.length ? full : m.content, pending: false }
        : m
    );
  }

  private flashSuccess() {
    this.setMascot('success');
    if (this.successTimer) clearTimeout(this.successTimer);
    this.successTimer = setTimeout(() => {
      this.setMascot('idle');
      this.notify();
      this.armSleep();
    }, 1600);
  }

  async ensureSession(): Promise<string | null> {
    if (this.sessionId) return this.sessionId;
    const res = await this.rpc('session.create', { source: 'webux', title: 'WebUX' });
    if (!res.ok) return null;
    const result = res.result as { session_id?: string };
    this.sessionId = result.session_id ?? null;
    this.notify();
    return this.sessionId;
  }

  async resume(sessionId: string): Promise<boolean> {
    const res = await this.rpc('session.resume', { session_id: sessionId });
    if (!res.ok) return false;
    this.sessionId = sessionId;
    // Hydrate history if available
    const hist = await this.rpc('session.history', { session_id: sessionId });
    if (hist.ok) {
      const result = hist.result as { messages?: Array<{ role?: string; content?: string }> };
      const msgs = result.messages ?? [];
      this.messages = msgs
        .filter((m) => m.role === 'user' || m.role === 'assistant')
        .map((m) => ({
          id: uid(),
          role: (m.role as 'user' | 'assistant') || 'assistant',
          content: String(m.content ?? ''),
        }));
    }
    this.setMascot(this.messages.length ? 'idle' : 'welcome');
    this.notify();
    return true;
  }

  async refreshSessions(): Promise<void> {
    const res = await this.rpc('session.list', {});
    if (!res.ok) return;
    const result = res.result as { sessions?: PlankSession[] };
    const all = result.sessions ?? [];
    this.sessions = all.filter((s) => {
      const src = (s.source || '').toLowerCase();
      return !src || src === 'webux' || src === 'cli' || src === 'dashboard';
    });
    this.notify();
  }

  async send(text: string): Promise<string | null> {
    const trimmed = text.trim();
    if (!trimmed || this.running) return null;
    this.wake();

    const sid = await this.ensureSession();
    if (!sid) return 'Cannot. Reason: session create failed.';

    this.messages = [
      ...this.messages,
      { id: uid('u'), role: 'user', content: trimmed },
      { id: uid('a'), role: 'assistant', content: '', pending: true },
    ];
    this.running = true;
    this.setMascot('thinking');
    this.notify();

    const res = await this.rpc('prompt.submit', { session_id: sid, text: trimmed });
    if (!res.ok) {
      this.running = false;
      this.messages = this.messages.filter((m) => !m.pending);
      this.messages = [...this.messages, { id: uid('e'), role: 'system', content: res.error }];
      this.setMascot('error');
      this.notify();
      return res.error;
    }
    return null;
  }

  async interrupt(): Promise<void> {
    if (!this.sessionId) return;
    await this.rpc('session.interrupt', { session_id: this.sessionId });
    this.running = false;
    this.toolsActive.clear();
    this.setMascot('idle');
    this.notify();
  }

  newChat() {
    this.sessionId = null;
    this.messages = [];
    this.running = false;
    this.toolsActive.clear();
    this.setMascot('welcome');
    this.notify();
  }
}

export async function fetchPlankStatus(): Promise<PlankStatus> {
  const res = await fetch('/api/plank/status', { credentials: 'include' });
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  return res.json();
}
