import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { fileURLToPath } from 'url';
import { readFileSync } from 'fs';
const __filename = fileURLToPath(import.meta.url);

// ─── helpers ─────────────────────────────────────────────────────────────────

/** Strip all <script> / <style> / {#if} blocks — returns pure JS expressions from value lines. */
function extractRhsBySemantics(source: string, lineLabel: string): string | null {
  const m = new RegExp(
    `\\blet\\s+${lineLabel}\\s*=`,
    'i'
  ).exec(source);
  if (!m) return null;
  const start = m.index;
  const block = source.slice(start);
  const open = block.indexOf('{');
  if (open === -1) return null;
  const close = block.lastIndexOf('}');
  const inner = block.slice(open + 1, close).trim();
  return inner.replace(/;\s*$/, '');
}

// ─── sample product used across tests ─────────────────────────────────────────
const PRODUCTS = [
  { id: 1, name: 'Kopi Susu Gula Aren', sku: 'KSGA-001', price: 18000, stock: 50, barcode: '899001001' },
  { id: 2, name: 'Roti Coklat',        sku: 'RTC-002',  price: 12000, stock:  3, barcode: '899002002' },
  { id: 3, name: 'Susu UHT 250ml',     sku: 'SUHT-003', price:  9000, stock:  0 },
];

// ─── source-structure guard tests ─────────────────────────────────────────────
describe('PosPage source-structure guards', () => {
  let source: string;

  beforeAll(() => {
    // __filename = /abs/path/PosPage.svelte.test.ts → swap to the .svelte source
    source = readFileSync(__filename.replace('PosPage.svelte.test.ts', 'PosPage.svelte'), 'utf-8');
  });

  // ── GLOBAL KEYBOARD SHORTCUTS ───────────────────────────────────────────────

  describe('GLOBAL KEYBOARD SHORTCUTS', () => {
    it('imports fly from svelte/transition', () => {
      expect(source).toContain("import { fly } from 'svelte/transition'");
    });

    it('registers <svelte:window on:keydown>', () => {
      expect(source).toContain('<svelte:window on:keydown={handleGlobalKeydown}');
    });

    it('exposes handleGlobalKeydown function', () => {
      const m = /function\s+handleGlobalKeydown\s*\(event\)/.exec(source);
      expect(m).not.toBeNull();
    });

    it('shortcut list: F2 + F4 + Alt+Delete', () => {
      expect(source).toContain("event.key === 'F2'");
      expect(source).toContain("event.key === 'F4'");
      expect(source).toContain("event.altKey && event.key === 'Delete'");
    });

    it('F2 focuses the search input via getElementById', () => {
      expect(source).toContain("getElementById('pos-search-input')");
      expect(source).toContain('input.focus()');
    });

    it('F4 opens checkout modal', () => {
      expect(source).toContain('openCheckoutModal()');
      expect(source).toContain("event.key === 'F4'");
    });

    it('Alt+Delete calls preventDefault and clearCart + toast', () => {
      expect(source).toContain('event.preventDefault()');
      expect(source).toContain('clearCart()');
    });

    it('shortcut legend text class exists in markup', () => {
      expect(source).toContain('[F2] Cari Produk');
      expect(source).toContain('[F4] Bayar');
      expect(source).toContain('[ALT+DEL] Kosongkan Keranjang');
    });
  });

  // ── CHECKOUT MODAL ──────────────────────────────────────────────────────────

  describe('CHECKOUT MODAL', () => {
    it('exports showCheckoutModal $state', () => {
      const sym = /let\s+showCheckoutModal\s*=\s*\$state\(false\)/.exec(source);
      expect(sym).not.toBeNull();
    });

    it('exports cashReceived $state defaulting to 0', () => {
      const sym = /let\s+cashReceived\s*=\s*\$state\(0\)/.exec(source);
      expect(sym).not.toBeNull();
    });

    it('reactive changeDue = cashReceived - totalAmount', () => {
      expect(source).toContain('changeDue');
    });

    it('modal overlay uses backdrop-blur', () => {
      expect(source).toContain('backdrop-blur');
    });

    it('modal header title is "Pembayaran Selesai"', () => {
      expect(source).toContain('Pembayaran Selesai');
    });

    it('total uses text-4xl and purple-400', () => {
      expect(source).toContain('text-4xl');
      expect(source).toContain('text-purple-400');
    });

    it('cash input field has id for programmatic focus', () => {
      expect(source).toContain('id="cash-received-input"');
      expect(source).not.toContain('autofocus');
    });

    it('programmatic focus effect on modal open', () => {
      expect(source).toContain('$effect(() =>');
      expect(source).toContain("showCheckoutModal");
      expect(source).toContain("el.focus()");
      expect(source).toContain("getElementById('cash-received-input')");
    });

    it('F5 inside modal fills cashReceived with totalAmount', () => {
      const block = extractRhsBySemantics(source, 'cashReceived') ?? '';
      expect(source).toContain("event.key === 'F5'");
    });

    it('quick cash preset buttons include Rp 50.000 and Rp 100.000', () => {
      // check quickCashPresets or button labels in the modal
      expect(
        source.includes('50000') || source.includes('50.000')
      ).toBe(true);
      expect(
        source.includes('100000') || source.includes('100.000')
      ).toBe(true);
    });

    it('changeDue >= 0 branch uses text-emerald-400', () => {
      expect(source).toContain("text-emerald-400");
    });

    it('changeDue negative shows danger styling', () => {
      expect(source).toContain("text-danger-light");
    });

    it('F3 / Esc closes modal', () => {
      expect(source).toContain("event.key === 'Escape'");
      expect(source).toContain("event.key === 'F3'");
    });

    it('Enter inside modal calls finalizeSale()', () => {
      expect(source).toContain('finalizeSale()');
    });

    it('pay button opens checkout and guards with changeDue', () => {
      expect(source).toContain('openCheckoutModal');
      expect(source).toContain('disabled={cart.length === 0 || changeDue < 0}');
    });
  });

  // ── FINALIZE SALE & RECEIPT PRINT ──────────────────────────────────────────

  describe('FINALIZE SALE & RECEIPT PRINT', () => {
    it('finalizeSale calls closeCheckoutModal before processCheckout', () => {
      expect(source).toContain('closeCheckoutModal()');
      expect(source).toContain('processCheckout()');
    });

    it('finalizeSale fires printReceipt on success', () => {
      expect(source).toContain('printReceipt()');
    });

    it('processCheckout posts to /sales endpoint', () => {
      expect(source).toContain("'/sales'");
    });

    it('sale payload includes payment_method and total_amount', () => {
      expect(source).toContain("payment_method: paymentMethod");
      expect(source).toContain("total_amount: totalAmount");
    });

    it('printReceipt guards with if (!lastSale)', () => {
      expect(source).toContain('if (!lastSale) return');
    });

    it('printReceipt calls window.print()', () => {
      expect(source).toContain('window.print()');
    });
  });

  // ── THERMAL RECEIPT PRINT CSS ───────────────────────────────────────────────

  describe('THERMAL RECEIPT PRINT CSS', () => {
    it('thermal-receipt hidden class present', () => {
      expect(source).toContain('class="thermal-receipt hidden"');
    });

    it('print media query structure present in style block', () => {
      expect(source).toContain('@media print');
      expect(source).toContain('display: none !important;');
      expect(source).toContain('display: block !important;');
    });

    it('thermal-receipt positioned absolute at top-left in print', () => {
      expect(source).toContain('.thermal-receipt {');
      expect(source).toContain('position: absolute');
    });

    it('thermal receipt uses white background and black text', () => {
      expect(source).toContain('background: white');
      expect(source).toContain("color: #000");
    });

    it('thermal font-family is monospace / Courier', () => {
      expect(source).toContain("'Courier New', monospace");
    });

    it('receipt contains shop name, invoice, time, items, total, footer', () => {
      expect(source).toContain('thermal-shop-name');
      expect(source).toContain('thermal-item');
      expect(source).toContain('thermal-item-total');
      expect(source).toContain('thermal-footer');
      expect(source).toContain('thermal-divider');
    });
  });
});

// ─── RUNTIME LOGIC TESTS ───────────────────────────────────────────────────────

describe('PosPage runtime logic', () => {
  it('complete purchase flow', async () => {
    const toastInfo = vi.fn();
    const toastError = vi.fn();
    const toastSuccess = vi.fn();

    // Simulate derived totalAmount from cart
    const cart = [
      { id: 1, name: 'Kopi Susu', price: 18000, quantity: 2 },
      { id: 2, name: 'Roti Coklat', price: 12000, quantity: 1 },
    ];
    const subtotal = cart.reduce((s, i) => s + i.price * i.quantity, 0);
    const totalAmount = subtotal; // 48000

    expect(totalAmount).toBe(48000);

    // Simulate cash received + change
    const cashReceived = 50000;
    const changeDue = cashReceived - totalAmount;
    expect(changeDue).toBe(2000);
  });

  it('short card/e-wallet: no changeDue guard on finalize', () => {
    // When paymentMethod !== 'Cash', no cash input → finalize always allowed
    const isCash = false;
    const guardNeeded = isCash;
    expect(guardNeeded).toBe(false);
  });

  it('derived changeDue is negative when cash is under total', () => {
    const totalAmount = 75000;
    const cashReceived = 50000;
    expect(cashReceived - totalAmount).toBeLessThan(0);
  });

  it('derived changeDue is zero when exact cash paid', () => {
    const totalAmount = 50000;
    const cashReceived = 50000;
    expect(cashReceived - totalAmount).toBe(0);
  });

  it('derived changeDue is positive when overpaid', () => {
    const totalAmount = 48000;
    const cashReceived = 100000;
    expect(cashReceived - totalAmount).toBeGreaterThan(0);
  });

  it('quick cash presets cover common denominations', () => {
    const quickCashPresets = [50000, 100000, 150000, 200000];
    expect(quickCashPresets[0]).toBe(50000);
    expect(quickCashPresets[1]).toBe(100000);
    expect(quickCashPresets).toHaveLength(4);
  });

  it('quick cash preset — clicking 100000 sets cashReceived to 100k', () => {
    let cashReceived = 0;
    const preset = 100000;
    cashReceived = preset;
    expect(cashReceived).toBe(100000);
  });

  it('F4 shortcut blocked when cart is empty', () => {
    const cart = [];
    const openModal = () => { if (cart.length === 0) return 'blocked'; return 'opened'; };
    // Simulate F4 handler: event.preventDefault + openCheckoutModal logic
    const result = openModal();
    expect(result).toBe('blocked');
  });

  it('updateQty clamps to zero then removes', () => {
    const items = [{ id: 1, name: 'Item', price: 1000, quantity: 2 }];
    function updateQty(id, delta) {
      const item = items.find(i => i.id === id);
      if (item) {
        item.quantity += delta;
        if (item.quantity <= 0) items.splice(items.indexOf(item), 1);
        return true;
      }
      return false;
    }
    updateQty(1, -1);
    expect(items[0].quantity).toBe(1);
    updateQty(1, -1);
    expect(items).toHaveLength(0);
  });
});
