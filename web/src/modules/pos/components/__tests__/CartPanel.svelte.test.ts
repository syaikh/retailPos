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

  it('imports cart-related lucide-svelte icons', () => {
    expect(src).toContain("import { ShoppingCart, X, Minus, Plus, Wallet, Printer, Hand, RotateCcw } from 'lucide-svelte'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('uses svelte transitions', () => {
    expect(src).toContain("import { slide } from 'svelte/transition'");
    expect(src).toContain("import { flip } from 'svelte/animate'");
  });

  it('has cart and totalAmount as props', () => {
    expect(src).toContain('cart = []');
    expect(src).toContain('totalAmount = 0');
    expect(src).toContain('totalItems = 0');
    expect(src).toContain('onupdateqty');
    expect(src).toContain('onremovefromcart');
    expect(src).toContain('onclearcart');
    expect(src).toContain('oncheckout');
    expect(src).toContain('onprintreceipt');
  });

  it('has hold and recall callbacks', () => {
    expect(src).toContain('onholdsale');
    expect(src).toContain('onopenparkedmodal');
    expect(src).toContain('parkedSaleCount');
  });

  it('renders cart header with ShoppingCart icon', () => {
    expect(src).toContain('<ShoppingCart size={18}');
  });

  it('shows empty state when cart is empty', () => {
    expect(src).toContain('{labels.yourCartIsEmpty}');
    expect(src).toContain('{labels.addProductsToStartSelling}');
  });

  it('renders quantity controls for each item', () => {
    expect(src).toContain('<Minus size={14} />');
    expect(src).toContain('<Plus size={14} />');
    expect(src).toContain('aria-label={labels.decreaseQuantity}');
    expect(src).toContain('aria-label={labels.increaseQuantity}');
  });

  it('renders checkout button with Wallet icon', () => {
    expect(src).toContain('<Wallet size={16} />');
    expect(src).toContain("t('payWithAmount'");
  });

  it('renders print and recall buttons', () => {
    expect(src).toContain('<Printer size={12} />');
    expect(src).toContain('{labels.recall}');
    expect(src).toContain('{labels.print}');
  });

  it('has Badge for item count', () => {
    expect(src).toContain('<Badge variant="primary" size="sm">');
  });
});
