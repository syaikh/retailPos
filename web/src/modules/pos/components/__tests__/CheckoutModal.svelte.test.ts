import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CheckoutModal.svelte'), 'utf-8');
}

describe('CheckoutModal.svelte source-structure guards', () => {
  const src = getSource();

  it('imports fly from svelte/transition', () => {
    expect(src).toContain("import { fly } from 'svelte/transition'");
  });

  it('imports Button from shared/ui', () => {
    expect(src).toContain("import { Button } from '$shared/ui'");
  });

  it('imports X, Check, Search from lucide-svelte', () => {
    expect(src).toContain("import { X, Check, Search } from 'lucide-svelte'");
  });

  it('uses $bindable for showCheckoutModal, paymentMethod, cashReceived', () => {
    expect(src).toContain('showCheckoutModal = $bindable(');
    expect(src).toContain('paymentMethod = $bindable(');
    expect(src).toContain('cashReceived = $bindable(');
  });

  it('has onfinalize and onselectcustomer callbacks', () => {
    expect(src).toContain('onfinalize');
    expect(src).toContain('onselectcustomer');
  });

  it('defines quickCashPresets', () => {
    expect(src).toContain('const quickCashPresets = [50000, 100000, 150000, 200000]');
  });

  it('renders modal with Pembayaran Selesai heading', () => {
    expect(src).toContain('Pembayaran Selesai');
  });

  it('renders Total Tagihan display', () => {
    expect(src).toContain('Total Tagihan');
  });

  it('renders payment method selector in modal', () => {
    expect(src).toContain('Metode Pembayaran');
  });

  it('renders cash received input for Cash method', () => {
    expect(src).toContain('cash-received-input');
    expect(src).toContain('Cash Received');
  });

  it('renders quick cash preset buttons', () => {
    expect(src).toContain('preset.toLocaleString');
  });

  it('renders change due section', () => {
    expect(src).toContain('Uang Kembali');
    expect(src).toContain('Kembali');
  });

  it('renders Batal and Selesai buttons', () => {
    expect(src).toContain('Batal [F3]');
    expect(src).toContain('Selesai');
  });
});
