import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ProductTable.svelte'), 'utf-8');
}

describe('ProductTable.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Badge, Button, Skeleton from shared/ui', () => {
    expect(src).toContain("import { Badge, Button, Skeleton, SortableHeader } from '$shared/ui'");
  });

  it('imports ProductActionsDropdown', () => {
    expect(src).toContain("import ProductActionsDropdown from '$modules/product/components/ProductActionsDropdown.svelte'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('uses $bindable for selectedIds, sortBy, sortDir, showCopySuccess', () => {
    expect(src).toContain('selectedIds = $bindable');
    expect(src).toContain('sortBy = $bindable');
    expect(src).toContain('sortDir = $bindable');
    expect(src).toContain('showCopySuccess = $bindable');
  });

  it('uses $derived for allSelected and someSelected', () => {
    expect(src).toContain('allSelected = $derived');
    expect(src).toContain('someSelected = $derived');
  });

  it('has event callbacks (onsort, onproductclick, onedit, ondelete, onadjuststock, oncopy)', () => {
    expect(src).toContain('onsort');
    expect(src).toContain('onproductclick');
    expect(src).toContain('onedit');
    expect(src).toContain('ondelete');
    expect(src).toContain('onadjuststock');
    expect(src).toContain('oncopy');
  });

  it('has statusInfo helper function', () => {
    expect(src).toContain('function statusInfo');
  });

  it('handles loading state with Skeleton', () => {
    expect(src).toContain('{#if loading}');
  });

  it('handles empty state', () => {
    expect(src).toContain('{labels.noProductsFound}');
    expect(src).toContain('{labels.tryAdjustingOrAddFirstProduct}');
  });

  it('renders table with sortable headers', () => {
    expect(src).toContain('label={labels.productName}');
    expect(src).toContain('label={labels.category}');
  });
});
