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

  it('imports icons from lucide-svelte', () => {
    expect(src).toContain("import { X, Check, User, ChevronRight } from 'lucide-svelte'");
  });

  it('imports CurrencyInput from shared/ui', () => {
    expect(src).toContain("import { Button, CurrencyInput } from '$shared/ui'");
  });

  it('has denominations array', () => {
    expect(src).toContain('const denominations = [5000, 10000, 20000, 50000, 100000]');
  });

  it('has showCheckoutModal bindable prop', () => {
    expect(src).toContain('showCheckoutModal = $bindable(false)');
  });

  it('has paymentMethod bindable prop', () => {
    expect(src).toContain('paymentMethod = $bindable(\'Cash\')');
  });

  it('has cashReceived and changeDue props', () => {
    expect(src).toContain('cashReceived = $bindable(0)');
    expect(src).toContain('changeDue = 0');
  });

  it('has onfinalize and onselectcustomer callbacks', () => {
    expect(src).toContain('onfinalize');
    expect(src).toContain('onselectcustomer');
  });

  it('renders dialog with aria-modal', () => {
    expect(src).toContain('role="dialog"');
    expect(src).toContain('aria-modal="true"');
    expect(src).toContain('aria-label="Pembayaran"');
  });

  it('renders total amount display', () => {
    expect(src).toContain('totalAmount.toLocaleString');
  });

  it('renders payment method selector buttons', () => {
    expect(src).toContain('paymentOptions');
    expect(src).toContain('opt.label');
  });

  it('renders customer selector with User icon', () => {
    expect(src).toContain('<User size={14}');
    expect(src).toContain('selectedCustomerLabel');
  });

  it('renders cash received input for Cash method', () => {
    expect(src).toContain('cash-received-input');
    expect(src).toContain('<CurrencyInput id="cash-received-input"');
  });

  it('renders denomination quick buttons', () => {
    expect(src).toContain('denom >= 1000000');
    expect(src).toContain('`${denom / 1000000}jt`');
    expect(src).toContain('`${denom / 1000}rb`');
  });

  it('renders change due section', () => {
    expect(src).toContain('Kembali');
    expect(src).toContain('Math.abs(changeDue)');
  });

  it('has Batal and Selesai action buttons', () => {
    expect(src).toContain('Batal [F3]');
    expect(src).toContain('Selesai [Enter]');
    expect(src).toContain('onfinalize');
  });
});
