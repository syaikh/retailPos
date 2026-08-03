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

  it('has loading and empty states', () => {
    expect(src).toContain('Memuat...');
    expect(src).toContain('Belum ada stok rak untuk produk ini.');
  });

  it('gates add/set/transfer actions behind canAdjust', () => {
    expect(src).toContain('{#if canAdjust}');
    expect(src).toContain('Tambah Stok');
    expect(src).toContain('>Set</Button>');
    expect(src).toContain('>Transfer</Button>');
  });

  it('validates set quantity is non-negative', () => {
    expect(src).toContain('setQuantity < 0');
    expect(src).toContain('Jumlah tidak boleh negatif');
  });

  it('validates transfer destination differs from source and quantity is positive', () => {
    expect(src).toContain('toLocationId === fromLocationId');
    expect(src).toContain('Lokasi asal dan tujuan harus berbeda');
    expect(src).toContain('transferQuantity <= 0');
    expect(src).toContain('Jumlah harus lebih dari 0');
  });

  it('calls onChanged after successful set and transfer', () => {
    expect(src).toContain('await load();\n      onChanged();');
  });

  it('documents rack stock as a sub-account of global stock', () => {
    expect(src).toContain('sub-akun dari stok global');
  });
});
