import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CartPanel.svelte'), 'utf-8');
}

describe('CartPanel.svelte source-structure guards', () => {
  const src = getSource();

  it('imports slide from svelte/transition and flip from svelte/animate', () => {
    expect(src).toContain("import { slide } from 'svelte/transition'");
    expect(src).toContain("import { flip } from 'svelte/animate'");
  });

  it('imports Badge, Button from shared/ui', () => {
    expect(src).toContain("import { Badge, Button } from '$shared/ui'");
  });

  it('imports cart-related lucide-svelte icons', () => {
    expect(src).toContain("import { ShoppingCart, X, Minus, Plus, Wallet, Printer, Search } from 'lucide-svelte'");
  });

  it('uses $bindable for paymentMethod', () => {
    expect(src).toContain('paymentMethod = $bindable(');
  });

  it('has event callbacks (onupdateqty, onremovefromcart, onclearcart, oncheckout, onprintreceipt, onselectcustomer)', () => {
    expect(src).toContain('onupdateqty');
    expect(src).toContain('onremovefromcart');
    expect(src).toContain('onclearcart');
    expect(src).toContain('oncheckout');
    expect(src).toContain('onprintreceipt');
    expect(src).toContain('onselectcustomer');
  });

  it('renders cart header with ShoppingCart icon', () => {
    expect(src).toContain('<ShoppingCart size={18}');
    expect(src).toContain('Cart');
  });

  it('renders empty cart state', () => {
    expect(src).toContain('Your cart is empty');
  });

  it('renders DPP and PPN lines when taxAmount > 0', () => {
    expect(src).toContain('DPP');
    expect(src).toContain('PPN 11%');
  });

  it('renders payment method selector', () => {
    expect(src).toContain('Payment method');
  });

  it('renders customer selector button', () => {
    expect(src).toContain('selectedCustomerLabel');
    expect(src).toContain('onselectcustomer');
  });

  it('renders Bayar checkout button with F4 hint', () => {
    expect(src).toContain('Bayar [F4]');
  });

  it('renders print receipt button', () => {
    expect(src).toContain('Print Last Receipt');
  });
});
