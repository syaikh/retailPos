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

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('has getPaymentMethodVariant function', () => {
    expect(src).toContain('function getPaymentMethodVariant');
  });

  it('has splitPaymentMethods function for multiple payment display', () => {
    expect(src).toContain('function splitPaymentMethods');
    expect(src).toContain('sale.payment_method');
  });

  it('has formatDateTime function', () => {
    expect(src).toContain('const formatDateTime');
  });

  it('renders Skeleton on loading', () => {
    expect(src).toContain('<Skeleton');
  });

  it('renders empty state with Banknote icon', () => {
    expect(src).toContain('<Banknote');
    expect(src).toContain('{labels.noTransactionsFound}');
    expect(src).toContain('{labels.tryAdjustingSearchOrDateRange}');
  });

  it('renders sortable table headers with localized labels', () => {
    expect(src).toContain('label={labels.invoiceLabel}');
    expect(src).toContain('label={labels.dateLabel}');
    expect(src).toContain('{labels.customerLabel}');
    expect(src).toContain('{labels.itemsLabel}');
    expect(src).toContain('label={labels.paymentLabel}');
    expect(src).toContain('label={labels.totalRp}');
  });

  it('shows localized Walk-in / General for sales without customer', () => {
    expect(src).toContain('labels.walkInGeneral');
  });

  it('shows items as a plain number and localizes more count', () => {
    expect(src).toContain('sale.items?.length');
    expect(src).toContain("t('moreWithCount'");
  });

  it('shows localized loading aria-label', () => {
    expect(src).toContain('labels.loadingTransactions');
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
