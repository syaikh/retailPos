import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PricingRulesToolbar.svelte'), 'utf-8');
}

describe('PricingRulesToolbar.svelte source-structure guards', () => {
  const src = getSource();

  it('imports FilterChipBar from shared/ui', () => {
    expect(src).toContain("import { Button, SearchBar, Dropdown, BulkActionDropdown, FilterChipBar } from '$shared/ui'");
  });

  it('imports Calculator, ChevronDown, Plus icons (no Columns3)', () => {
    expect(src).toContain("import { Plus, ChevronDown, Calculator } from 'lucide-svelte'");
    expect(src).not.toContain('Columns3');
  });

  it('imports debounce utility', () => {
    expect(src).toContain("import { debounce } from '$shared/utils/debounce'");
  });

  it('has onimport callback prop', () => {
    expect(src).toContain('onimport = () => {}');
    expect(src).toContain('onimport?: () => void');
  });

  it('has onsimulate callback prop', () => {
    expect(src).toContain('onsimulate = () => {}');
    expect(src).toContain('onsimulate?: () => void');
  });

  it('has activeFilters derived', () => {
    expect(src).toContain('let activeFilters = $derived.by');
  });

  it('builds filter chips for approval', () => {
    expect(src).toContain("type: 'approval'");
    expect(src).toContain("label: labels.approvalChip.replace('{value}', approvalLabels[approvalFilter] || approvalFilter)");
  });

  it('builds filter chips for status', () => {
    expect(src).toContain("type: 'status'");
    expect(src).toContain("label: labels.statusChip.replace('{value}', statusLabels[statusFilter] || statusFilter)");
  });

  it('builds filter chips for type', () => {
    expect(src).toContain("type: 'type'");
    expect(src).toContain("label: labels.typeChip.replace('{value}', label)");
  });

  it('builds filter chips for method', () => {
    expect(src).toContain("type: 'method'");
    expect(src).toContain("label: labels.methodChip.replace('{value}', label)");
  });

  it('has clearAllFilters function', () => {
    expect(src).toContain('function clearAllFilters()');
    expect(src).toContain("approvalFilter = 'all'");
    expect(src).toContain("statusFilter = 'all'");
    expect(src).toContain("typeFilter = 'all'");
    expect(src).toContain("methodFilter = 'all'");
  });

  it('has clearFilterChip function for individual chip clearing', () => {
    expect(src).toContain('function clearFilterChip(type: string)');
    expect(src).toContain("case 'approval': approvalFilter = 'all'");
    expect(src).toContain("case 'status': statusFilter = 'all'");
    expect(src).toContain("case 'type': typeFilter = 'all'");
    expect(src).toContain("case 'method': methodFilter = 'all'");
  });

  it('renders FilterChipBar with active filters', () => {
    expect(src).toContain('<FilterChipBar');
    expect(src).toContain('chips={activeFilters}');
    expect(src).toContain('onclear={clearFilterChip}');
    expect(src).toContain('onclearall={clearAllFilters}');
  });

  it('renders BulkActionDropdown for import/export', () => {
    expect(src).toContain('<BulkActionDropdown module="pricing_rules"');
    expect(src).toContain('canExport={canCreate}');
    expect(src).toContain('canImport={canCreate}');
    expect(src).toContain('onImport={onimport}');
  });

  it('renders simulate button with Calculator icon', () => {
    expect(src).toContain('onclick={onsimulate}');
    expect(src).toContain('labels.simulation');
    expect(src).toContain('Calculator');
  });

  it('typeLabel default is localized (all types)', () => {
    expect(src).toContain('typeLabel = labels.allTypes');
  });

  it('has aria-label on type filter dropdown trigger', () => {
    expect(src).toContain("aria-label={labels.filterTipe.replace('{typeLabel}', typeLabel)}");
  });

  it('has aria-label on method filter dropdown trigger', () => {
    expect(src).toContain("aria-label={labels.filterMetode.replace('{methodLabel}', methodLabel)}");
  });

  it('SearchBar has id="pricing-search"', () => {
    expect(src).toContain('id="pricing-search"');
  });

  it('filter chip clear buttons have aria-labels handled by FilterChipBar', () => {
    expect(src).toContain('FilterChipBar');
  });

  it('has status filter group with aria-label', () => {
    expect(src).toContain('aria-label={labels.filterStatusActive}');
  });

  it('filter chips use FilterChipBar component', () => {
    expect(src).toContain('chips={activeFilters}');
    expect(src).toContain('FilterChipBar');
  });

  it('type filter button changes style when active', () => {
    expect(src).toContain("typeFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary'");
  });

  it('method filter button changes style when active', () => {
    expect(src).toContain("methodFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary'");
  });

  it('does not have showDetailCols prop (removed)', () => {
    expect(src).not.toContain('showDetailCols');
    expect(src).not.toContain('Columns3');
  });

  it('filter dropdowns use h-8 consistent height', () => {
    expect(src).toContain('h-8 px-3 rounded-lg');
  });
});
