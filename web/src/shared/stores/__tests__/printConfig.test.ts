import { describe, it, expect, beforeEach, vi } from 'vitest';

function freshImport() {
  vi.resetModules();
  return import('../printConfig.svelte');
}

describe('PrintConfigStore', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it('defaults to preview mode and the dev agent URL', async () => {
    const { printConfig } = await import('../printConfig.svelte');
    expect(printConfig.mode).toBe('preview');
    expect(printConfig.agentUrl).toBe('http://localhost:9123');
  });

  it('setMode updates and persists to localStorage', async () => {
    const { printConfig } = await freshImport();
    printConfig.setMode('silent');
    expect(printConfig.mode).toBe('silent');
    const raw = localStorage.getItem('pos.printConfig');
    expect(raw).toBeTruthy();
    expect(JSON.parse(raw as string).mode).toBe('silent');
  });

  it('toggleMode flips between preview and silent', async () => {
    const { printConfig } = await freshImport();
    expect(printConfig.mode).toBe('preview');
    printConfig.toggleMode();
    expect(printConfig.mode).toBe('silent');
    printConfig.toggleMode();
    expect(printConfig.mode).toBe('preview');
  });

  it('setAgentUrl trims surrounding whitespace and persists; empty falls back to default', async () => {
    const { printConfig } = await freshImport();
    printConfig.setAgentUrl('  http://example:9000/  ');
    expect(printConfig.agentUrl).toBe('http://example:9000/');
    printConfig.setAgentUrl('   ');
    expect(printConfig.agentUrl).toBe('http://localhost:9123');
  });

  it('hydrates mode and agentUrl from localStorage', async () => {
    localStorage.setItem(
      'pos.printConfig',
      JSON.stringify({ mode: 'silent', agentUrl: 'http://shop:1234' })
    );
    const { printConfig } = await freshImport();
    expect(printConfig.mode).toBe('silent');
    expect(printConfig.agentUrl).toBe('http://shop:1234');
  });

  it('ignores malformed localStorage JSON and keeps defaults', async () => {
    localStorage.setItem('pos.printConfig', '{not valid json');
    const { printConfig } = await freshImport();
    expect(printConfig.mode).toBe('preview');
    expect(printConfig.agentUrl).toBe('http://localhost:9123');
  });

  it('seeds mode from VITE_PRINT_MODE env when storage is empty', async () => {
    vi.stubEnv('VITE_PRINT_MODE', 'silent');
    const { printConfig } = await freshImport();
    expect(printConfig.mode).toBe('silent');
  });
});
