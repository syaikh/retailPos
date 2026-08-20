import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ProductFiltersToolbar.svelte'), 'utf-8');
}

describe('ProductFiltersToolbar.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button, SearchBar, BulkActionDropdown, Dropdown from shared/ui', () => {
    expect(src).toContain("import { Button, SearchBar, BulkActionDropdown, Dropdown, FilterChipBar } from '$shared/ui'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('uses $bindable for searchQuery, selectedCategories, filterStatus, lowStockOnly', () => {
    expect(src).toContain('searchQuery = $bindable');
    expect(src).toContain('selectedCategories = $bindable');
    expect(src).toContain('filterStatus = $bindable');
    expect(src).toContain('lowStockOnly = $bindable');
  });

  it('uses $bindable for selectedBrandIDs and unified onfilter callback', () => {
    expect(src).toContain('selectedBrandIDs = $bindable<number[]>([])');
    expect(src).toContain('onfilter = () => {}');
    expect(src).not.toContain('onfiltercategory');
  });

  it('uses $props', () => {
    expect(src).toContain('= $props()');
  });

  it('has status dropdown via Dropdown component and category filter button', () => {
    expect(src).toContain('<Dropdown');
    expect(src).toContain('placement="bottom-start"');
  });

  it('renders filter chips section', () => {
    expect(src).toContain('activeChips = $derived');
    expect(src).toContain('clearLabel={labels.clearAll}');
  });

  it('includes brand chips and clears them via chip removal', () => {
    expect(src).toContain("chips.push({ type: 'brand', label: t('brandsCount', { count: selectedBrandIDs.length }) })");
    expect(src).toContain("if (type === 'brand') { selectedBrandIDs = []; }");
  });

  it('derives hasActiveFilters from categories and brands', () => {
    expect(src).toContain('hasActiveFilters = $derived');
    expect(src).toContain('selectedBrandIDs.length > 0');
  });

  it('shows Add Product button with canCreate guard', () => {
    expect(src).toContain('{labels.tambahProduk}');
    expect(src).toContain('disabled={!canCreate}');
  });

  it('has Low Stock toggle switch', () => {
    expect(src).toContain('{labels.lowStock}');
  });
});
