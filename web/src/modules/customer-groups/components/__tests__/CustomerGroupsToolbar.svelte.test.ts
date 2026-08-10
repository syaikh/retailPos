import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CustomerGroupsToolbar.svelte'), 'utf-8');
}

describe('CustomerGroupsToolbar.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button, SearchBar, BulkActionDropdown, FilterChipBar from shared/ui', () => {
    expect(src).toContain("import { Button, SearchBar, BulkActionDropdown, FilterChipBar } from '$shared/ui'");
  });

  it('imports Users and UsersRound icons for has_customers filter', () => {
    expect(src).toContain('Users');
    expect(src).toContain('UsersRound');
  });

  it('has hasCustomersFilter bindable prop', () => {
    expect(src).toContain('hasCustomersFilter = $bindable');
  });

  it('has activeFilters derived with has_customers chip', () => {
    expect(src).toContain("type: 'has_customers'");
    expect(src).toContain('labels.filterChipHasCustomers');
    expect(src).toContain('labels.filterChipNoCustomers');
  });

  it('has clearFilterChip handler for has_customers', () => {
    expect(src).toContain("else if (type === 'has_customers') hasCustomersFilter = 'all'");
  });

  it('has clearAllFilters that resets hasCustomersFilter', () => {
    expect(src).toContain("hasCustomersFilter = 'all'");
  });

  it('has segmented status filter group with aria-label', () => {
    expect(src).toContain('aria-label={labels.filterStatus}');
  });

  it('has segmented customer filter group with aria-label', () => {
    expect(src).toContain('aria-label={labels.filterCustomer}');
  });

  it('has aria-pressed on all filter buttons', () => {
    const ariaPressedCount = (src.match(/aria-pressed/g) || []).length;
    expect(ariaPressedCount).toBeGreaterThanOrEqual(6);
  });

  it('has card wrapper class', () => {
    expect(src).toContain('class="card px-4 py-3"');
  });

  it('has BulkActionDropdown with customer_groups module', () => {
    expect(src).toContain('module="customer_groups"');
  });

  it('has FilterChipBar component', () => {
    expect(src).toContain('<FilterChipBar');
  });
});
