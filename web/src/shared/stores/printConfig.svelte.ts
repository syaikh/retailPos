// Print configuration store.
//
// Controls how receipts are emitted after a sale:
//   - mode "preview": render the on-screen 58mm overlay and open the browser
//     print dialog (current behaviour, requires a manual confirm).
//   - mode "silent":  POST the receipt payload to the local print agent, which
//     routes it to a real thermal printer / virtual printer / file without any
//     browser dialog. This is how real high-volume retail POS prints.
//
// The configuration persists to localStorage and can be seeded from Vite env
// vars (VITE_PRINT_MODE, VITE_PRINT_AGENT_URL). In production the agent URL
// would point at the register's local print bridge instead of a dev default.

export type PrintMode = 'preview' | 'silent';

const STORAGE_KEY = 'pos.printConfig';
const DEFAULT_AGENT_URL = 'http://localhost:9123';

interface PrintConfigShape {
  mode: PrintMode;
  agentUrl: string;
}

function safeParse(raw: string | null): Partial<PrintConfigShape> {
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw);
    return typeof parsed === 'object' && parsed !== null ? parsed : {};
  } catch {
    return {};
  }
}

class PrintConfigStore {
  mode = $state<PrintMode>('preview');
  agentUrl = $state<string>(DEFAULT_AGENT_URL);
  /** When true, the build-time mode is enforced and the mode toggle (preview/silent) is disabled. The agent-URL gear remains available so a register can still be pointed at its local agent. */
  locked = $state(false);

  constructor() {
    const envMode = (import.meta.env.VITE_PRINT_MODE as string | undefined) || '';
    const envUrl = (import.meta.env.VITE_PRINT_AGENT_URL as string | undefined) || '';
    const stored = safeParse(this.readStorage());

    const validMode: PrintMode = envMode === 'silent' ? 'silent' : 'preview';
    const forcedSilent = envMode === 'silent';
    this.locked = forcedSilent;
    // A `silent` build locks the mode so a stored preference or the UI toggle
    // can never revert a register back to preview.
    this.mode = forcedSilent ? 'silent' : (stored.mode as PrintMode) || validMode;
    this.agentUrl = stored.agentUrl || envUrl || DEFAULT_AGENT_URL;
  }

  private readStorage(): string | null {
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch {
      return null;
    }
  }

  private persist() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ mode: this.mode, agentUrl: this.agentUrl }));
    } catch {
      /* storage unavailable (private mode / SSR) — ignore */
    }
  }

  setMode(mode: PrintMode) {
    if (this.locked) return;
    this.mode = mode;
    this.persist();
  }

  toggleMode() {
    this.setMode(this.mode === 'silent' ? 'preview' : 'silent');
  }

  setAgentUrl(url: string) {
    this.agentUrl = url.trim() || DEFAULT_AGENT_URL;
    this.persist();
  }
}

export const printConfig = new PrintConfigStore();
