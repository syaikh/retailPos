import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('inventory-store', () => {
  let store: ReturnType<typeof import('../inventory-store.svelte').useInventoryStore>;

  beforeEach(() => {
    vi.resetModules();
  });

  it('returns expected API shape', async () => {
    const { useInventoryStore } = await import('../inventory-store.svelte');
    store = useInventoryStore();
    expect(store).toHaveProperty('warningThreshold');
    expect(store).toHaveProperty('criticalThreshold');
    expect(store).toHaveProperty('setThresholds');
  });

  it('returns default thresholds', async () => {
    const { useInventoryStore } = await import('../inventory-store.svelte');
    store = useInventoryStore();
    expect(store.warningThreshold).toBe(10);
    expect(store.criticalThreshold).toBe(5);
  });

  it('sets thresholds via setThresholds', async () => {
    const { useInventoryStore } = await import('../inventory-store.svelte');
    store = useInventoryStore();
    store.setThresholds({ warning: 20, critical: 10 });
    expect(store.warningThreshold).toBe(20);
    expect(store.criticalThreshold).toBe(10);
  });
});