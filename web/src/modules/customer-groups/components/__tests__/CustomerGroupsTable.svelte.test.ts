import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CustomerGroupsTable.svelte'), 'utf-8');
}

describe('CustomerGroupsTable.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Badge, Button, Skeleton, SortableHeader, Tooltip, Dropdown from shared/ui', () => {
    expect(src).toContain("import { Badge, Button, Skeleton, SortableHeader, Tooltip, Dropdown } from '$shared/ui'");
  });

  it('imports MoreVertical, Users, Pencil, Trash2, Copy icons', () => {
    expect(src).toContain('MoreVertical');
    expect(src).toContain('Users');
    expect(src).toContain('Pencil');
    expect(src).toContain('Trash2');
    expect(src).toContain('Copy');
  });

  it('imports CustomerGroup type', () => {
    expect(src).toContain("import type { CustomerGroup } from '../types'");
  });

  it('has bulk action callback props', () => {
    expect(src).toContain('onbulkactivate = (_ids: number[]) => {}');
    expect(src).toContain('onbulkdeactivate = (_ids: number[]) => {}');
    expect(src).toContain('onbulkdelete = (_ids: number[]) => {}');
  });

  it('has selectedIds state', () => {
    expect(src).toContain('let selectedIds = $state<Set<number>>(new Set())');
  });

  it('has allSelected derived', () => {
    expect(src).toContain('let allSelected = $derived(groups.length > 0 && groups.every(g => selectedIds.has(g.id)))');
  });

  it('has someSelected derived', () => {
    expect(src).toContain('let someSelected = $derived(groups.some(g => selectedIds.has(g.id)) && !allSelected)');
  });

  it('has selectedCount derived', () => {
    expect(src).toContain('let selectedCount = $derived(selectedIds.size)');
  });

  it('has toggleSelect and toggleSelectAll functions', () => {
    expect(src).toContain('function toggleSelect(id: number)');
    expect(src).toContain('function toggleSelectAll()');
  });

  it('has clearSelection function', () => {
    expect(src).toContain('function clearSelection()');
  });

  it('has timeAgo helper function', () => {
    expect(src).toContain('function timeAgo(dateStr: string | undefined)');
  });

  it('has formatDateTime helper function', () => {
    expect(src).toContain('function formatDateTime(dateStr: string | undefined)');
  });

  it('has colgroup with fixed table layout', () => {
    expect(src).toContain('table-layout: fixed');
    expect(src).toContain('<colgroup>');
  });

  it('has py-4 row padding', () => {
    expect(src).toContain('px-4 py-4');
  });

  it('has customer_count column', () => {
    expect(src).toContain('customer_count');
    expect(src).toContain("column=\"customer_count\"");
  });

  it('uses kebab Dropdown for actions', () => {
    expect(src).toContain('<Dropdown placement="bottom-end"');
  });

  it('has color display with avatar circle', () => {
    expect(src).toContain('background-color: ${g.color}');
  });

  it('uses relative time in Tooltip', () => {
    expect(src).toContain('<Tooltip content={formatDateTime');
  });

  it('has truncated name under secondary text', () => {
    expect(src).toContain('truncate block text-xs text-text-muted');
  });

  it('has bulk action bar when selectedCount > 0', () => {
    expect(src).toContain('{#if selectedCount > 0}');
  });

  it('has bulk activate/deactivate/delete buttons', () => {
    expect(src).toContain('Aktifkan');
    expect(src).toContain('Nonaktifkan');
    expect(src).toContain('Hapus');
  });

  it('has aria-label on table', () => {
    expect(src).toContain('aria-label="Customer groups"');
  });

  it('has aria-live on empty state', () => {
    expect(src).toContain('aria-live="polite"');
  });
});
