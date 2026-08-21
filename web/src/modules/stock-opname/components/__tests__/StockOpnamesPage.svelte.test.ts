import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'StockOpnamesPage.svelte'), 'utf-8');
}

describe('StockOpnamesPage.svelte location-scope source guards', () => {
  const src = getSource();

  it('imports getStorageLocations for the location scope options', () => {
    expect(src).toContain("getStorageLocations } from '$modules/storage-location/services/storage-location-service'");
  });

  it('loads active storage locations as location scope options', () => {
    expect(src).toContain("getStorageLocations({ is_active: true, limit: 500, offset: 0 })");
    expect(src).toContain("type === 'location'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('warns when location scope is combined with other scopes', () => {
    expect(src).toContain("createRows.some((r) => r.scope_type === 'location')");
    expect(src).toContain('labels.storageLocationScopeOnly');
  });

  it('imports onMount from svelte', () => {
    expect(src).toContain("import { onMount } from 'svelte';");
  });

  it('performs the first load in onMount, not the auto-reload effect', () => {
    expect(src).toContain('onMount(() => {');
    expect(src).toContain('store.loadSessions(store.currentFilters);');
  });

  it('auto-reload effect tracks only filter inputs, not pagination state', () => {
    expect(src).toContain('void store.searchFilter');
    expect(src).toContain('void store.statusFilter');
    expect(src).not.toMatch(/^\s*store\.currentFilters;\s*$/m);
  });

  it('skips reload on first effect run via firstRun guard', () => {
    const firstRunDecl = src.indexOf('let firstRun = true;');
    expect(firstRunDecl).toBeGreaterThan(-1);
    const effectIdx = src.indexOf('$effect(() => {', firstRunDecl);
    expect(effectIdx).toBeGreaterThan(firstRunDecl);
    const guardIdx = src.indexOf('if (firstRun) {', effectIdx);
    expect(guardIdx).toBeGreaterThan(effectIdx);
    const resetIdx = src.indexOf('firstRun = false;', guardIdx);
    expect(resetIdx).toBeGreaterThan(guardIdx);
  });
});
