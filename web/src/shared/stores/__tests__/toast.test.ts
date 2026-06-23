import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

describe('toast store', () => {
  let toast: typeof import('../toast.svelte').toast;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.resetModules();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns expected API shape', async () => {
    const { toast: t } = await import('../toast.svelte');
    toast = t;
    expect(toast).toHaveProperty('subscribe');
    expect(toast).toHaveProperty('success');
    expect(toast).toHaveProperty('error');
    expect(toast).toHaveProperty('warning');
    expect(toast).toHaveProperty('info');
    expect(toast).toHaveProperty('remove');
  });

  it('success adds toast with success variant', async () => {
    const { toast: t } = await import('../toast.svelte');
    toast = t;
    toast.success('Test message');
    let toasts: typeof import('../toast.svelte').ToastMessage[] = [];
    toast.subscribe((value) => { toasts = value; })();
    expect(toasts).toHaveLength(1);
    expect(toasts[0].variant).toBe('success');
    expect(toasts[0].message).toBe('Test message');
    expect(toasts[0].id).toBeDefined();
  });

  it('error adds toast with error variant', async () => {
    const { toast: t } = await import('../toast.svelte');
    toast = t;
    toast.error('Error message');
    let toasts: typeof import('../toast.svelte').ToastMessage[] = [];
    toast.subscribe((value) => { toasts = value; })();
    expect(toasts[0].variant).toBe('error');
    expect(toasts[0].message).toBe('Error message');
  });

  it('warning adds toast with warning variant', async () => {
    const { toast: t } = await import('../toast.svelte');
    toast = t;
    toast.warning('Warning message');
    let toasts: typeof import('../toast.svelte').ToastMessage[] = [];
    toast.subscribe((value) => { toasts = value; })();
    expect(toasts[0].variant).toBe('warning');
  });

  it('info adds toast with info variant', async () => {
    const { toast: t } = await import('../toast.svelte');
    toast = t;
    toast.info('Info message');
    let toasts: typeof import('../toast.svelte').ToastMessage[] = [];
    toast.subscribe((value) => { toasts = value; })();
    expect(toasts[0].variant).toBe('info');
  });

  it('remove removes toast by id', async () => {
    const { toast: t } = await import('../toast.svelte');
    toast = t;
    const id = toast.success('Test');
    let toasts: typeof import('../toast.svelte').ToastMessage[] = [];
    toast.subscribe((value) => { toasts = value; })();
    expect(toasts).toHaveLength(1);
    toast.remove(id);
    toast.subscribe((value) => { toasts = value; })();
    expect(toasts).toHaveLength(0);
  });

  it('auto-dismisses based on duration', async () => {
    const { toast: t } = await import('../toast.svelte');
    toast = t;
    toast.success('Test', 4000);
    let toasts: typeof import('../toast.svelte').ToastMessage[] = [];
    toast.subscribe((value) => { toasts = value; })();
    expect(toasts).toHaveLength(1);
    vi.advanceTimersByTime(4000);
    await vi.runAllTimersAsync();
    toast.subscribe((value) => { toasts = value; })();
    expect(toasts).toHaveLength(0);
  });
});