import { describe, it, expect } from 'vitest';

describe('product-utils', () => {
  describe('statusInfo', () => {
    it('returns success for active status', async () => {
      const { statusInfo } = await import('../product-utils');
      expect(statusInfo('active')).toEqual({ variant: 'success', label: 'Active' });
    });

    it('returns muted for draft', async () => {
      const { statusInfo } = await import('../product-utils');
      expect(statusInfo('draft')).toEqual({ variant: 'muted', label: 'Draft' });
    });

    it('returns muted for inactive', async () => {
      const { statusInfo } = await import('../product-utils');
      expect(statusInfo('inactive')).toEqual({ variant: 'muted', label: 'Inactive' });
    });

    it('returns danger for discontinued', async () => {
      const { statusInfo } = await import('../product-utils');
      expect(statusInfo('discontinued')).toEqual({ variant: 'danger', label: 'Discontinued' });
    });

    it('returns danger for archived', async () => {
      const { statusInfo } = await import('../product-utils');
      expect(statusInfo('archived')).toEqual({ variant: 'danger', label: 'Archived' });
    });

    it('returns muted for unknown status', async () => {
      const { statusInfo } = await import('../product-utils');
      expect(statusInfo('unknown')).toEqual({ variant: 'muted', label: '- ' });
    });

    it('handles undefined status', async () => {
      const { statusInfo } = await import('../product-utils');
      expect(statusInfo()).toEqual({ variant: 'muted', label: '- ' });
    });
  });

  describe('formatCurrency', () => {
    it('formats number as IDR', async () => {
      const { formatCurrency } = await import('../product-utils');
      expect(formatCurrency(15000)).toBe('Rp 15.000');
    });

    it('returns dash for null', async () => {
      const { formatCurrency } = await import('../product-utils');
      expect(formatCurrency(null as unknown as number)).toBe('-');
    });

    it('returns dash for NaN', async () => {
      const { formatCurrency } = await import('../product-utils');
      expect(formatCurrency(NaN)).toBe('-');
    });
  });

  describe('validateProductForm', () => {
    it('returns null for valid form', async () => {
      const { validateProductForm } = await import('../product-utils');
      expect(validateProductForm({ name: 'Test', sku: 'SKU-001', category: 'Food', price: 10000, stock: 10 })).toBeNull();
    });

    it('returns error for empty name', async () => {
      const { validateProductForm } = await import('../product-utils');
      expect(validateProductForm({ name: '', sku: 'SKU-001', category: 'Food', price: 10000, stock: 10 })).toBe('Please complete all required fields');
    });

    it('returns error for zero price', async () => {
      const { validateProductForm } = await import('../product-utils');
      expect(validateProductForm({ name: 'Test', sku: 'SKU-001', category: 'Food', price: 0, stock: 10 })).toBe('Price must be greater than zero');
    });

    it('returns error for negative stock', async () => {
      const { validateProductForm } = await import('../product-utils');
      expect(validateProductForm({ name: 'Test', sku: 'SKU-001', category: 'Food', price: 10000, stock: -1 })).toBe('Stock must not be negative');
    });
  });

  describe('buildProductPayload', () => {
    it('includes category_name derived from category', async () => {
      const { buildProductPayload } = await import('../product-utils');
      const form = {
        name: 'Test', sku: 'TST', barcode: '', category: 'Food',
        brand_id: null, price: 10000, cost: 5000, stock: 10,
        unit_of_measure_id: null, tax_class_id: null, weight_grams: null,
        description: '', status: 'draft',
      };
      const payload = buildProductPayload(form);
      expect(payload.category_name).toBe('Food');
      expect(payload.barcode).toBeUndefined();
      expect(payload.description).toBeUndefined();
    });
  });
});
