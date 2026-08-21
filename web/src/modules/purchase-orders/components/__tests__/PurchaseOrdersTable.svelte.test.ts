import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PurchaseOrdersTable.svelte'), 'utf-8');
}

describe('PurchaseOrdersTable.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Copy icon for PO number copy button', () => {
    expect(src).toContain("Copy } from 'lucide-svelte'");
  });

  it('has handleCopyPO function using clipboard API', () => {
    expect(src).toContain('navigator.clipboard.writeText(poNumber)');
  });

  it('shows checkmark briefly after copy', () => {
    expect(src).toContain("copiedPOs = new Set([...copiedPOs, poId])");
    expect(src).toContain('setTimeout');
    expect(src).toContain('next.delete(poId)');
  });

  it('stops click propagation on copy button', () => {
    expect(src).toContain('e.stopPropagation(); handleCopyPO(po.id, po.po_number)');
  });

  it('has aria-label on copy button', () => {
    expect(src).toContain('labels.copyPoNumber');
  });

  it('has whitespace-nowrap on table headers', () => {
    const count = (src.match(/whitespace-nowrap/g) || []).length;
    expect(count).toBeGreaterThanOrEqual(6);
  });

  it('has min-w-[1000px] on data table', () => {
    expect(src).toContain('min-w-[1000px]');
  });

  it('defaults to updated_at descending sort', () => {
    expect(src).toContain("sortBy = 'updated_at'");
    expect(src).toContain("sortDir = 'desc'");
  });

  it('renders Updated column sortable by updated_at', () => {
    expect(src).toContain('<SortableHeader label={labels.updatedAtLabel} column="updated_at"');
  });

  it('renders updated_at cell value', () => {
    expect(src).toContain('{formatDate(po.updated_at)}');
  });

  it('no longer renders a created_at column', () => {
    expect(src).not.toContain('labels.createdAtLabel');
    expect(src).not.toContain('column="created_at"');
    expect(src).not.toContain('{formatDate(po.created_at)}');
  });
});
