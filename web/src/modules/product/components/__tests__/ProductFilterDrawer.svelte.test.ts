import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ProductFilterDrawer.svelte'), 'utf-8');
}

describe('ProductFilterDrawer.svelte source-structure guards', () => {
  const src = getSource();

  it('imports SearchBar from shared/ui and Brand type', () => {
    expect(src).toContain("import { SearchBar } from '$shared/ui'");
    expect(src).toContain("import type { Brand } from '$modules/product/types'");
  });

  it('uses $bindable for open, selectedCategories, selectedBrandIDs', () => {
    expect(src).toContain('open = $bindable(false)');
    expect(src).toContain('selectedCategories = $bindable<string[]>([])');
    expect(src).toContain('selectedBrandIDs = $bindable<number[]>([])');
  });

  it('keeps temporary selection state synced on open', () => {
    expect(src).toContain('tempSelectedCategories = $state<string[]>([])');
    expect(src).toContain('tempSelectedBrandIDs = $state<number[]>([])');
    expect(src).toMatch(/\$effect\(\(\) => \{\s*if \(open\) \{/);
  });

  it('resets search query when opening or switching tabs', () => {
    expect(src).toContain("searchQuery = ''");
  });

  it('filters categories excluding All and matching search case-insensitively', () => {
    expect(src).toContain("categories.filter(cat => cat !== 'All' && cat.toLowerCase().includes(searchQuery.toLowerCase()))");
  });

  it('filters brands by name matching search case-insensitively', () => {
    expect(src).toContain('brands.filter(b => b.name.toLowerCase().includes(searchQuery.toLowerCase()))');
  });

  it('toggles category and brand selections in temp state', () => {
    expect(src).toContain('function toggleCategory(cat: string)');
    expect(src).toContain('function toggleBrand(id: number)');
  });

  it('applyFilters writes temp state back with All fallback for empty categories', () => {
    expect(src).toContain("selectedCategories = tempSelectedCategories.length > 0 ? [...tempSelectedCategories] : ['All']");
    expect(src).toContain('selectedBrandIDs = tempSelectedBrandIDs.length > 0 ? [...tempSelectedBrandIDs] : []');
    expect(src).toContain('onApply?.()');
    expect(src).toContain('open = false');
  });

  it('resetFilters clears both temp selections', () => {
    expect(src).toContain('function resetFilters()');
    expect(src).toMatch(/function resetFilters\(\) \{\s*tempSelectedCategories = \[\];\s*tempSelectedBrandIDs = \[\];\s*\}/);
  });

  it('closes drawer on Escape keydown via window listener', () => {
    expect(src).toContain('<svelte:window onkeydown={handleKeydown} />');
    expect(src).toContain("e.key === 'Escape'");
  });

  it('renders dialog with modal semantics and accessible label', () => {
    expect(src).toContain('role="dialog"');
    expect(src).toContain('aria-modal="true"');
    expect(src).toContain('aria-label={labels.filterProduk}');
  });

  it('renders category and brand tabs with selection count badges', () => {
    expect(src).toContain('{labels.category}');
    expect(src).toContain('{labels.brand}');
    expect(src).toContain('{tempSelectedCategories.length}');
    expect(src).toContain('{tempSelectedBrandIDs.length}');
  });

  it('shows empty-state messages for filtered categories and brands', () => {
    expect(src).toContain('{labels.noCategoriesFound}');
    expect(src).toContain('{labels.noBrandsFound}');
  });

  it('shows active filter count badge on apply button', () => {
    expect(src).toContain('{labels.applyFilter}');
    expect(src).toContain('{#if activeCount > 0}');
  });
});
