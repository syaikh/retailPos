import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PurchaseOrdersPage.svelte'), 'utf-8');
}

describe('PurchaseOrdersPage.svelte source-structure guards', () => {
  const src = getSource();

  it('auto-reload effect tracks only filter inputs, not pagination state', () => {
    expect(src).toContain('void store.searchQuery');
    expect(src).toContain('void store.statusFilter');
    expect(src).toContain('void store.supplierFilter');
    expect(src).toContain('void store.startDate');
    expect(src).toContain('void store.endDate');
    expect(src).not.toMatch(/^\s*store\.currentFilters;\s*$/m);
  });

  it('handlePageChange updates page/pageSize and loads without resetting page', () => {
    const fnStart = src.indexOf('function handlePageChange');
    const fnEnd = src.indexOf('}', src.indexOf('store.load(store.currentFilters)', fnStart));
    const fnBody = src.slice(fnStart, fnEnd);
    expect(fnBody).toContain('store.pageSize = newLimit');
    expect(fnBody).toContain('store.page = Math.floor(newOffset / newLimit)');
    expect(fnBody).not.toContain('store.page = 0');
  });

  it('handleSort resets to first page before loading', () => {
    const fnStart = src.indexOf('function handleSort');
    const fnBody = src.slice(fnStart, fnStart + 300);
    const resetIdx = fnBody.indexOf('store.page = 0;');
    const loadIdx = fnBody.indexOf('store.load(store.currentFilters)');
    expect(resetIdx).toBeGreaterThan(-1);
    expect(loadIdx).toBeGreaterThan(resetIdx);
  });

  it('renders Pagination wired to store paging state', () => {
    expect(src).toContain('<Pagination total={store.total} limit={store.pageSize} offset={store.offset} onPageChange={handlePageChange}');
  });
});
