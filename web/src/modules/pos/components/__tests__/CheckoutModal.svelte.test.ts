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
    expect(src).toContain("import { X, Check, User, ChevronRight, Plus, Trash2 } from 'lucide-svelte'");
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

  it('uses allocation-based state instead of single payment method', () => {
    expect(src).toContain('let allocations = $state<AllocationRow[]>([])');
    expect(src).toContain('interface AllocationRow');
    expect(src).toContain('methodCode: string');
    expect(src).toContain('amount: number');
  });

  it('computes totalAllocated, remainingBalance, and canComplete', () => {
    expect(src).toContain('const totalAllocated = $derived');
    expect(src).toContain('const remainingBalance = $derived');
    expect(src).toContain('const canComplete = $derived');
  });

  it('has addAllocation and removeAllocation functions', () => {
    expect(src).toContain('function addAllocation');
    expect(src).toContain('function removeAllocation');
  });

  it('has onfinalize callback that accepts PaymentAllocation array', () => {
    expect(src).toContain('onfinalize = (payments: PaymentAllocation[]) => {}');
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

  it('renders payment allocation rows', () => {
    expect(src).toContain('{#each allocations as alloc');
    expect(src).toContain('removeAllocation(alloc.id)');
  });

  it('renders remaining balance indicator', () => {
    expect(src).toContain('remainingBalance');
  });

  it('renders denomination quick buttons for cash rows', () => {
    expect(src).toContain('denom >= 1000000');
    expect(src).toContain('`${denom / 1000000}jt`');
    expect(src).toContain('`${denom / 1000}rb`');
  });

  it('renders Batal and Selesai action buttons', () => {
    expect(src).toContain('Batal [F3]');
    expect(src).toContain('Selesai [Enter]');
    expect(src).toContain('onfinalize');
  });

  it('handles F7 key for exact cash amount', () => {
    expect(src).toContain("e.key === 'F7'");
    expect(src).toContain("a.methodCode === 'CASH'");
  });

  it('has handleFinalize that maps allocations to PaymentAllocation[]', () => {
    expect(src).toContain('function handleFinalize');
    expect(src).toContain('payment_method_code: a.methodCode');
    expect(src).toContain('amount: a.amount');
  });

  it('renders remove payment button', () => {
    expect(src).toContain('Trash2');
    expect(src).toContain('removeAllocation');
  });

  it('renders customer selector with User icon', () => {
    expect(src).toContain('<User size={14}');
    expect(src).toContain('selectedCustomerLabel');
  });
});
