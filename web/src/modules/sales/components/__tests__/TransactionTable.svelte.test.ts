import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'TransactionTable.svelte'), 'utf-8');
}

describe('TransactionTable.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $props()', () => {
    expect(src).toContain('= $props()');
  });

  it('uses $bindable() for sortBy/sortDir', () => {
    expect(src).toContain('$bindable(');
  });

  it('imports Badge, Pagination, Skeleton', () => {
    expect(src).toContain("import { Badge, Pagination, Skeleton, SortableHeader } from '$shared/ui'");
  });

  it('imports Banknote icon', () => {
    expect(src).toContain("import { Banknote } from 'lucide-svelte'");
  });

  it('imports formatDateTimeInJakarta', () => {
    expect(src).toContain("import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime'");
  });

  it('has getPaymentMethodVariant function', () => {
    expect(src).toContain('function getPaymentMethodVariant');
  });

  it('has formatDateTime function', () => {
    expect(src).toContain('const formatDateTime');
  });

  it('renders Skeleton on loading', () => {
    expect(src).toContain('<Skeleton');
  });

  it('renders empty state with Banknote icon', () => {
    expect(src).toContain('<Banknote');
    expect(src).toContain('No transactions found');
  });

  it('renders sortable table headers', () => {
    expect(src).toContain('INVOICE');
    expect(src).toContain('DATE');
    expect(src).toContain('CUSTOMER');
    expect(src).toContain('PAYMENT');
    expect(src).toContain('TOTAL (RP)');
  });

  it('shows Walk-in / General for sales without customer', () => {
    expect(src).toContain("Walk-in / General");
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });

  it('has salesData binding', () => {
    expect(src).toContain('salesData');
  });

  it('has handleSort function', () => {
    expect(src).toContain('function handleSort');
  });

  it('has handlePageChange function', () => {
    expect(src).toContain('function handlePageChange');
  });

  it('renders rows as clickable', () => {
    expect(src).toContain('onclick={() => handleRowClick');
  });
});
