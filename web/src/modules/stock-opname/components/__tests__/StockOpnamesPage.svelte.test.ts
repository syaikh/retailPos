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

  it('warns when location scope is combined with other scopes', () => {
    expect(src).toContain("createRows.some((r) => r.scope_type === 'location')");
    expect(src).toContain('must be the only scope');
  });
});
