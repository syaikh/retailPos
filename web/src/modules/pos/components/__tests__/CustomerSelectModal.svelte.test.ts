import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CustomerSelectModal.svelte'), 'utf-8');
}

describe('CustomerSelectModal.svelte source-structure guards', () => {
  const src = getSource();

  it('imports fly from svelte/transition', () => {
    expect(src).toContain("import { fly } from 'svelte/transition'");
  });

  it('imports Button, Input from shared/ui', () => {
    expect(src).toContain("import { Button, Input } from '$shared/ui'");
  });

  it('imports X from lucide-svelte', () => {
    expect(src).toContain("import { X } from 'lucide-svelte'");
  });

  it('uses $bindable for showCustomerModal and customerSearch', () => {
    expect(src).toContain('showCustomerModal = $bindable(');
    expect(src).toContain('customerSearch = $bindable(');
  });

  it('has onselectcustomer callback', () => {
    expect(src).toContain('onselectcustomer');
  });

  it('renders Pilih Customer heading', () => {
    expect(src).toContain('Pilih Customer');
  });

  it('renders walk-in option', () => {
    expect(src).toContain('Walk-in / Umum');
  });

  it('renders no customers found message', () => {
    expect(src).toContain('Customer tidak ditemukan');
  });

  it('renders customer search input', () => {
    expect(src).toContain('Cari berdasarkan');
  });
});
