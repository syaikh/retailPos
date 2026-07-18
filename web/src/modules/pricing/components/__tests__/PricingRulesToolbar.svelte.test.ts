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

  it('imports BulkActionDropdown from shared/ui', () => {
    expect(src).toContain("import { Button, SearchBar, Dropdown, BulkActionDropdown } from '$shared/ui'");
  });

  it('imports Calculator, X, ChevronDown, Plus icons', () => {
    expect(src).toContain("import { Plus, ChevronDown, X, Calculator } from 'lucide-svelte'");
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

  it('builds filter chips for status', () => {
    expect(src).toContain("key: 'status'");
    expect(src).toContain("label: `Status: ${statusLabels[statusFilter] || statusFilter}`");
  });

  it('builds filter chips for type', () => {
    expect(src).toContain("key: 'type'");
    expect(src).toContain("label: `Tipe: ${label}`");
  });

  it('builds filter chips for method', () => {
    expect(src).toContain("key: 'method'");
    expect(src).toContain("label: `Metode: ${label}`");
  });

  it('has hasActiveFilters derived', () => {
    expect(src).toContain('let hasActiveFilters = $derived(activeFilters.length > 0)');
  });

  it('has clearAllFilters function', () => {
    expect(src).toContain('function clearAllFilters()');
    expect(src).toContain("statusFilter = 'all'");
    expect(src).toContain("typeFilter = 'all'");
    expect(src).toContain("methodFilter = 'all'");
  });

  it('each filter chip has a clear callback', () => {
    expect(src).toContain('clear: () => { statusFilter = \'all\'; handleFilterChange(); }');
    expect(src).toContain('clear: () => { typeFilter = \'all\'; handleFilterChange(); }');
    expect(src).toContain('clear: () => { methodFilter = \'all\'; handleFilterChange(); }');
  });

  it('renders filter chips when hasActiveFilters', () => {
    expect(src).toContain('{#if hasActiveFilters}');
    expect(src).toContain('Filter aktif:');
  });

  it('renders clear all button when more than 1 active filter', () => {
    expect(src).toContain('{#if activeFilters.length > 1}');
    expect(src).toContain('Hapus semua');
  });

  it('renders BulkActionDropdown for import/export', () => {
    expect(src).toContain('<BulkActionDropdown module="pricing_rules"');
    expect(src).toContain('canExport={canCreate}');
    expect(src).toContain('canImport={canCreate}');
    expect(src).toContain('onImport={onimport}');
  });

  it('renders simulate button with Calculator icon', () => {
    expect(src).toContain('onclick={onsimulate}');
    expect(src).toContain('Simulasi harga');
    expect(src).toContain('Calculator');
  });

  it('typeLabel default is Indonesian (Semua Tipe)', () => {
    expect(src).toContain("typeLabel = 'Semua Tipe'");
  });

  it('has aria-label on type filter dropdown trigger', () => {
    expect(src).toContain("aria-label=\"Filter tipe: {typeLabel}\"");
  });

  it('has aria-label on method filter dropdown trigger', () => {
    expect(src).toContain("aria-label=\"Filter metode: {methodLabel}\"");
  });

  it('SearchBar has id="pricing-search"', () => {
    expect(src).toContain('id="pricing-search"');
  });

  it('filter chips have aria-label for individual chip clear buttons', () => {
    expect(src).toContain('aria-label="Hapus filter {chip.label}"');
  });

  it('has status filter group with aria-label', () => {
    expect(src).toContain('aria-label="Filter status"');
  });

  it('has filter chip area with proper structure', () => {
    expect(src).toContain('activeFilters as chip (chip.key)');
    expect(src).toContain('px-2 py-0.5 rounded-full');
  });

  it('type filter button changes style when active', () => {
    expect(src).toContain("typeFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary'");
  });

  it('method filter button changes style when active', () => {
    expect(src).toContain("methodFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary'");
  });
});
