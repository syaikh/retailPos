import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'RackStockPanel.svelte'), 'utf-8');
}

describe('RackStockPanel.svelte source-structure guards', () => {
  const src = getSource();

  it('imports rack stock service functions', () => {
    expect(src).toContain("getLocationStock, setLocationStock, transferLocationStock } from '../services/inventory-service'");
  });

  it('imports storage locations service', () => {
    expect(src).toContain("getStorageLocations } from '$modules/storage-location/services/storage-location-service'");
  });

  it('uses $props with productId, canAdjust and onChanged', () => {
    expect(src).toContain('= $props()');
    expect(src).toContain('productId = null as number | null');
    expect(src).toContain('canAdjust = false');
    expect(src).toContain('onChanged = () => {}');
  });

  it('imports labels from $shared/i18n', () => {
    expect(src).toContain("import { labels } from '$shared/i18n'");
  });

  it('has loading and empty states', () => {
    expect(src).toContain('aria-busy="true"');
    expect(src).toContain('{labels.belumAdaStokRak}');
  });

  it('gates add/set/transfer actions behind canAdjust', () => {
    expect(src).toContain('{#if canAdjust}');
    expect(src).toContain('{labels.tambahStok}');
    expect(src).toContain('{labels.setStokRak}');
    expect(src).toContain('{labels.transfer}');
  });

  it('validates set quantity is non-negative', () => {
    expect(src).toContain('setQuantity < 0');
    expect(src).toContain('labels.jumlahTidakBolehNegatif');
  });

  it('validates transfer destination differs from source and quantity is positive', () => {
    expect(src).toContain('toLocationId === fromLocationId');
    expect(src).toContain('labels.lokasiAsalDanTujuanHarusBerbeda');
    expect(src).toContain('transferQuantity <= 0');
    expect(src).toContain('labels.jumlahHarusLebihDari0');
  });

  it('calls onChanged after successful set and transfer', () => {
    expect(src).toContain('await load();\n      onChanged();');
  });

  it('documents rack stock as a sub-account of global stock', () => {
    expect(src).toContain('labels.stokRakSubAccount');
  });

  it('loads rack rows independently of location metadata (read-only safe)', () => {
    expect(src).toContain('rows = await getLocationStock(productId);');
    expect(src).toContain('if (canAdjust) {');
    expect(src).toContain('getStorageLocations({ is_active: true, limit: 500, offset: 0 })');
  });
});
