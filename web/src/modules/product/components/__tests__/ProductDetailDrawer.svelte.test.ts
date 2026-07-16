import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ProductDetailDrawer.svelte'), 'utf-8');
}

describe('ProductDetailDrawer.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Badge, Button, and Drawer from shared UI', () => {
    expect(src).toContain("import { Badge, Button, Drawer } from '$shared/ui'");
  });

  it('imports lucide icons (Pencil, Trash2, Copy, Percent)', () => {
    expect(src).toContain("import { Pencil, Trash2, Copy, Percent } from 'lucide-svelte'");
  });

  it('imports formatDateTimeInJakarta', () => {
    expect(src).toContain("import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime'");
  });

  it('imports toast store', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('uses $props() for component props', () => {
    expect(src).toContain('= $props()');
  });

  it('has selectedProduct prop', () => {
    expect(src).toContain('selectedProduct');
  });

  it('has bindable showDetailDrawer', () => {
    expect(src).toContain('showDetailDrawer = $bindable()');
  });

  it('has bindable showCopySuccess', () => {
    expect(src).toContain('showCopySuccess = $bindable()');
  });

  it('has formatCurrency function', () => {
    expect(src).toContain('function formatCurrency');
  });

  it('has formatDate function', () => {
    expect(src).toContain('function formatDate');
  });

  it('has statusInfo function', () => {
    expect(src).toContain('function statusInfo');
  });

  it('has copyToClipboard function', () => {
    expect(src).toContain('function copyToClipboard');
  });

  it('has margin and marginPct derived', () => {
    expect(src).toContain('let margin = $derived');
    expect(src).toContain('let marginPct = $derived');
  });

  it('uses stock_stk, margVal, margPctVal, margIsLoss, uomLabel derived', () => {
    expect(src).toContain('let stock_stk');
    expect(src).toContain('let margVal');
    expect(src).toContain('let margPctVal');
    expect(src).toContain('let margIsLoss');
    expect(src).toContain('let uomLabel');
  });

  it('renders detail drawer with showDetailDrawer condition', () => {
    expect(src).toContain('{#if selectedProduct}');
  });

  it('has edit and delete buttons', () => {
    expect(src).toContain('<Pencil size={15}');
    expect(src).toContain('<Trash2 size={15}');
  });

  it('has onedit and ondelete callbacks on buttons', () => {
    expect(src).toContain('onclick={onedit}');
    expect(src).toContain('onclick={ondelete}');
  });
});
