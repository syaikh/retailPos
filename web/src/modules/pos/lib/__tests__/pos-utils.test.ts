import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { setLocale } from '$shared/i18n';
import type { CartItem } from '../../types';

const mockCart: CartItem[] = [
  { id: 1, cart_session_id: 1, product_id: 1, product_name: 'Item A', unit_price: 10000, original_price: 10000, quantity: 2, tax_rate: 11, discount: 0, cost: 0, subtotal: 20000, dpp_amount: 0, tax_amount: 0 },
  { id: 2, cart_session_id: 1, product_id: 2, product_name: 'Item B', unit_price: 5000, original_price: 5000, quantity: 3, tax_rate: 0, discount: 0, cost: 0, subtotal: 15000, dpp_amount: 0, tax_amount: 0 },
];

describe('pos-utils', () => {
  beforeAll(() => setLocale('id'));
  afterAll(() => setLocale('en'));

  describe('formatCurrency', () => {
    it('formats number as IDR', async () => {
      const { formatCurrency } = await import('../pos-utils');
      expect(formatCurrency(15000)).toBe('Rp 15.000');
    });

    it('returns default for null', async () => {
      const { formatCurrency } = await import('../pos-utils');
      expect(formatCurrency(null as unknown as number)).toBe('Rp 0');
    });
  });

  describe('calculateSubtotal', () => {
    it('calculates correct subtotal', async () => {
      const { calculateSubtotal } = await import('../pos-utils');
      expect(calculateSubtotal(mockCart)).toBe(35000);
    });

    it('returns 0 for empty cart', async () => {
      const { calculateSubtotal } = await import('../pos-utils');
      expect(calculateSubtotal([])).toBe(0);
    });
  });

  describe('calculateTax', () => {
    it('calculates PPN for taxable items', async () => {
      const { calculateTax } = await import('../pos-utils');
      const tax = calculateTax(mockCart);
      // Item A: price=10000, qty=2, rate=11%
      // lineTotal = 20000, dpp = round(20000*100/111) = round(18018) = 18018
      // tax = 20000 - 18018 = 1982
      expect(tax).toBe(1982);
    });

    it('returns 0 when no items have tax', async () => {
      const { calculateTax } = await import('../pos-utils');
      const noTaxCart = [{ ...mockCart[0], tax_rate: 0 }];
      expect(calculateTax(noTaxCart)).toBe(0);
    });

    it('returns 0 for empty cart', async () => {
      const { calculateTax } = await import('../pos-utils');
      expect(calculateTax([])).toBe(0);
    });
  });

  describe('calculateTotalItems', () => {
    it('counts total quantity', async () => {
      const { calculateTotalItems } = await import('../pos-utils');
      expect(calculateTotalItems(mockCart)).toBe(5);
    });

    it('returns 0 for empty cart', async () => {
      const { calculateTotalItems } = await import('../pos-utils');
      expect(calculateTotalItems([])).toBe(0);
    });
  });
});
