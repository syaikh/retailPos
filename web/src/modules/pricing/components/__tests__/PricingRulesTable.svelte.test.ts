import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PricingRulesTable.svelte'), 'utf-8');
}

describe('PricingRulesTable.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Tooltip, Badge, Dropdown from shared/ui', () => {
    expect(src).toContain("import { Button, Skeleton, SortableHeader, Tooltip, Badge, Dropdown } from '$shared/ui'");
  });

  it('imports checkbox and bulk action icons', () => {
    expect(src).toContain('CheckSquare');
    expect(src).toContain('Square');
    expect(src).toContain('Power');
    expect(src).toContain('PowerOff');
    expect(src).toContain('Copy');
    expect(src).toContain('MoreVertical');
  });

  it('imports PricingRule, PricingType, PricingMethod types', () => {
    expect(src).toContain("import type { PricingRule, PricingType, PricingMethod } from '../types'");
  });

  it('has canCreate prop', () => {
    expect(src).toContain('canCreate = false');
    expect(src).toContain('canCreate: boolean');
  });

  it('has onduplicate callback prop', () => {
    expect(src).toContain('onduplicate = (_rule: PricingRule) => {}');
    expect(src).toContain('onduplicate?: (rule: PricingRule) => void');
  });

  it('has bulk action callback props', () => {
    expect(src).toContain('onbulkactivate = (_ids: number[]) => {}');
    expect(src).toContain('onbulkdeactivate = (_ids: number[]) => {}');
    expect(src).toContain('onbulkdelete = (_ids: number[]) => {}');
  });

  it('has oncreate callback prop', () => {
    expect(src).toContain('oncreate = () => {}');
    expect(src).toContain('oncreate?: () => void');
  });

  it('has targetNames prop with Map type', () => {
    expect(src).toContain('targetNames = new Map<string, string>()');
    expect(src).toContain('targetNames?: Map<string, string>');
  });

  it('has selectedIds state for row selection', () => {
    expect(src).toContain('let selectedIds = $state<Set<number>>(new Set())');
  });

  it('has allSelected derived', () => {
    expect(src).toContain('let allSelected = $derived(rules.length > 0 && rules.every(r => selectedIds.has(r.id)))');
  });

  it('has someSelected derived', () => {
    expect(src).toContain('let someSelected = $derived(rules.some(r => selectedIds.has(r.id)) && !allSelected)');
  });

  it('has selectedCount derived', () => {
    expect(src).toContain('let selectedCount = $derived(selectedIds.size)');
  });

  it('has toggleSelect function', () => {
    expect(src).toContain('function toggleSelect(id: number)');
  });

  it('has toggleSelectAll function', () => {
    expect(src).toContain('function toggleSelectAll()');
  });

  it('has clearSelection function', () => {
    expect(src).toContain('function clearSelection()');
  });

  it('targetLabel uses targetNames map for product resolution', () => {
    expect(src).toContain("targetNames.get(`product:${rule.product_id}`)");
    expect(src).toContain("targetNames.get(`category:${rule.category_id}`)");
    expect(src).toContain("targetNames.get(`brand:${rule.brand_id}`)");
  });

  it('has timeAgo function', () => {
    expect(src).toContain('function timeAgo(dateStr: string | undefined): string');
    expect(src).toContain('Baru saja');
    expect(src).toContain('menit lalu');
    expect(src).toContain('jam lalu');
    expect(src).toContain('hari lalu');
    expect(src).toContain('bln lalu');
  });

  it('has formatDateTime function', () => {
    expect(src).toContain('function formatDateTime(dateStr: string | undefined): string');
    expect(src).toContain("toLocaleString('id-ID'");
  });

  it('has checkbox column in table header', () => {
    expect(src).toContain('toggleSelectAll');
    expect(src).toContain('aria-label={allSelected ? \'Batalkan semua pilihan\' : \'Pilih semua\'}');
  });

  it('has three-state select all (all/some/none)', () => {
    expect(src).toContain('{#if allSelected}');
    expect(src).toContain('{:else if someSelected}');
    expect(src).toContain('<CheckSquare');
    expect(src).toContain('<Square');
  });

  it('has checkbox in each row', () => {
    expect(src).toContain('onclick={() => toggleSelect(rule.id)}');
    expect(src).toContain('aria-label={selectedIds.has(rule.id) ? `Batalkan pilihan ${rule.name}` : `Pilih ${rule.name}`}');
  });

  it('applies selected row styling', () => {
    expect(src).toContain("selectedIds.has(rule.id) ? 'bg-primary-subtle/10' : ''");
  });

  it('has duplicate action in kebab menu when canCreate', () => {
    expect(src).toContain('{#if canCreate}');
    expect(src).toContain('onduplicate(rule)');
    expect(src).toContain('Duplikasi');
  });

  it('shows empty state hint when canCreate', () => {
    expect(src).toContain('{#if canCreate}');
    expect(src).toContain('Klik "Tambah Rule" untuk membuat aturan harga pertama.');
  });

  it('has bulk action bar when items are selected', () => {
    expect(src).toContain('{#if selectedCount > 0}');
    expect(src).toContain('{selectedCount} rule dipilih');
  });

  it('bulk activate button calls onbulkactivate', () => {
    expect(src).toContain('onbulkactivate([...selectedIds])');
  });

  it('bulk deactivate button calls onbulkdeactivate', () => {
    expect(src).toContain('onbulkdeactivate([...selectedIds])');
  });

  it('bulk delete button calls onbulkdelete', () => {
    expect(src).toContain('onbulkdelete([...selectedIds])');
  });

  it('bulk activate/deactivate visible when canEdit', () => {
    expect(src).toContain('{#if canEdit}');
    expect(src).toContain('Aktifkan');
    expect(src).toContain('Nonaktifkan');
  });

  it('bulk delete visible when canDelete', () => {
    expect(src).toContain('{#if canDelete}');
    expect(src).toContain('Hapus');
  });

  it('has clear selection cancel button', () => {
    expect(src).toContain('Batal');
    expect(src).toContain('onclick={clearSelection}');
  });

  it('uses Tooltip for rule name', () => {
    expect(src).toContain('<Tooltip content={rule.name} delay={400}>');
  });

  it('uses Tooltip for target label', () => {
    expect(src).toContain('<Tooltip content={targetLabel(rule)} delay={400}>');
  });

  it('uses Tooltip for updated_at column', () => {
    expect(src).toContain('<Tooltip content={formatDateTime(rule.updated_at || rule.created_at)} delay={400}>');
    expect(src).toContain('{timeAgo(rule.updated_at || rule.created_at)}');
  });

  it('has updated_at sortable column (DIPERBARUI)', () => {
    expect(src).toContain('DIPERBARUI');
    expect(src).toContain('column="updated_at"');
  });

  it('has approval sortable column', () => {
    expect(src).toContain('column="status"');
    expect(src).toContain('APPROVAL');
  });

  it('NILAI column is right-aligned', () => {
    expect(src).toContain('NILAI');
    expect(src).toContain('text-right');
  });

  it('has responsive column classes (hidden lg:table-cell)', () => {
    expect(src).toContain('hidden lg:table-cell');
  });

  it('has responsive table min-width', () => {
    expect(src).toContain('min-w-[700px] lg:min-w-[1100px]');
  });

  it('has method column sortable', () => {
    expect(src).toContain('column="pricing_method"');
    expect(src).toContain('METODE');
  });

  it('loading skeleton has 11 columns', () => {
    expect(src).toContain('{#each Array(11) as _}');
  });

  it('does not use role="grid" on data table', () => {
    expect(src).not.toContain('role="grid"');
  });

  it('has aria-label on table', () => {
    expect(src).toContain('aria-label="Pricing rules"');
  });

  it('has aria-busy on loading table', () => {
    expect(src).toContain('aria-busy="true"');
    expect(src).toContain('aria-label="Loading pricing rules"');
  });

  it('has aria-label on actions group', () => {
    expect(src).toContain('aria-label="Actions for {rule.name}"');
  });

  it('action buttons use icon size', () => {
    expect(src).toContain('size="icon"');
  });

  it('imports MoreVertical icon from lucide-svelte for kebab menu', () => {
    expect(src).toContain('MoreVertical');
  });

  it('has onviewaudit callback prop', () => {
    expect(src).toContain('onviewaudit = (_rule: PricingRule) => {}');
    expect(src).toContain('onviewaudit?: (rule: PricingRule) => void');
  });

  it('kebab menu has audit action', () => {
    expect(src).toContain('onviewaudit(rule)');
    expect(src).toContain('Audit');
  });

  it('action column has 8% width', () => {
    expect(src).toContain('width: 8%;" />');
  });

  it('type column has 8% width', () => {
    expect(src).toContain('width: 8%;" />');
  });

  it('has showDetailCols prop for toggling detail columns', () => {
    expect(src).toContain('showDetailCols = false');
    expect(src).toContain('showDetailCols?: boolean');
  });

  it('uses shared Badge component for approval status', () => {
    expect(src).toContain('<Badge variant={approvalVariant');
    expect(src).toContain('approvalLabel(rule.status');
  });

  it('uses shared Badge component for pricing type', () => {
    expect(src).toContain('<Badge variant={typeVariant()}');
  });

  it('has approvalVariant helper function', () => {
    expect(src).toContain('function approvalVariant(status: string)');
    expect(src).toContain("case 'approved': return 'success'");
    expect(src).toContain("case 'pending': return 'warning'");
    expect(src).toContain("case 'rejected': return 'danger'");
    expect(src).toContain("default: return 'muted'");
  });

  it('has approvalLabel helper function', () => {
    expect(src).toContain('function approvalLabel(status: string)');
    expect(src).toContain("case 'approved': return 'Approved'");
    expect(src).toContain("case 'pending': return 'Pending'");
    expect(src).toContain("case 'rejected': return 'Rejected'");
    expect(src).toContain("default: return 'Draft'");
  });

  it('uses Dropdown for kebab action menu', () => {
    expect(src).toContain('<Dropdown placement="bottom-end"');
    expect(src).toContain('{#snippet content({ close })}');
  });

  it('kebab menu items are conditional on permissions', () => {
    expect(src).toContain("rule.status === 'draft' && canEdit");
    expect(src).toContain("rule.status === 'pending' && canEdit");
    expect(src).toContain('{#if canEdit}');
    expect(src).toContain('{#if canCreate}');
    expect(src).toContain('{#if canDelete}');
  });

  it('kebab menu has labeled actions', () => {
    expect(src).toContain('Ajukan');
    expect(src).toContain('Approve');
    expect(src).toContain('Reject');
    expect(src).toContain('Edit');
    expect(src).toContain('Duplikasi');
    expect(src).toContain('Hapus');
  });
});
