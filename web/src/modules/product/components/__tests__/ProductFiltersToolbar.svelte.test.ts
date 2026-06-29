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

  it('imports Button, SearchBar, Dropdown from shared/ui', () => {
    expect(src).toContain("import { Button, SearchBar, ExportImportButtons, Dropdown } from '$shared/ui'");
  });

  it('uses $bindable for searchQuery, selectedCategories, filterStatus, lowStockOnly', () => {
    expect(src).toContain('searchQuery = $bindable');
    expect(src).toContain('selectedCategories = $bindable');
    expect(src).toContain('filterStatus = $bindable');
    expect(src).toContain('lowStockOnly = $bindable');
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
    expect(src).toContain('Clear all');
  });

  it('shows Add Product button with canManageInventory guard', () => {
    expect(src).toContain('Add Product');
    expect(src).toContain('disabled={!canManageInventory}');
  });

  it('has Low Stock toggle switch', () => {
    expect(src).toContain('Low Stock');
  });
});
