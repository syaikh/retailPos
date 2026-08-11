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

  it('imports kebab and bulk action icons', () => {
    expect(src).toContain('Pencil');
    expect(src).toContain('Trash2');
    expect(src).toContain('Copy');
    expect(src).toContain('Power');
    expect(src).toContain('PowerOff');
    expect(src).toContain('MoreVertical');
  });

  it('imports PricingRule type only', () => {
    expect(src).toContain("import type { PricingRule } from '../types'");
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

  it('has checkbox in table header with indeterminate state', () => {
    expect(src).toContain('toggleSelectAll');
    expect(src).toContain('bind:indeterminate={someSelected}');
    expect(src).toContain('aria-label={labels.pilihSemua}');
  });

  it('uses native checkbox for three-state select all', () => {
    expect(src).toContain('<input type="checkbox"');
    expect(src).toContain('bind:indeterminate={someSelected}');
    expect(src).toContain('onchange={toggleSelectAll}');
  });

  it('has checkbox in each row', () => {
    expect(src).toContain('onchange={() => toggleSelect(rule.id)}');
    expect(src).toContain("t('pilihItem', { name: rule.name })");
  });

  it('applies selected row styling', () => {
    expect(src).toContain("selectedIds.has(rule.id) ? 'bg-muted/30' : ''");
  });

  it('has duplicate action in kebab menu when canCreate', () => {
    expect(src).toContain('{#if canCreate}');
    expect(src).toContain('onduplicate(rule)');
    expect(src).toContain('{labels.duplicate}');
  });

  it('shows empty state hint when canCreate', () => {
    expect(src).toContain('{#if canCreate}');
    expect(src).toContain('{labels.emptyStateCreateRule}');
  });

  it('has bulk action bar when items are selected', () => {
    expect(src).toContain('{#if selectedCount > 0}');
    expect(src).toContain("t('rulesSelected', { count: selectedCount })");
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
    expect(src).toContain('{labels.activate}');
    expect(src).toContain('{labels.deactivate}');
  });

  it('bulk delete visible when canDelete', () => {
    expect(src).toContain('{#if canDelete}');
    expect(src).toContain('{labels.hapus}');
  });

  it('has clear selection cancel button', () => {
    expect(src).toContain('{labels.cancel}');
    expect(src).toContain('onclick={clearSelection}');
  });

  it('uses Tooltip for rule name', () => {
    expect(src).toContain('<Tooltip content={rule.name} delay={400}>');
  });

  it('uses Tooltip for target label', () => {
    expect(src).toContain('<Tooltip content={targetLabel(rule)} delay={400}>');
  });

  it('NILAI column is right-aligned', () => {
    expect(src).toContain('{labels.nilai}');
    expect(src).toContain('text-right');
  });

  it('has method column sortable', () => {
    expect(src).toContain('column="pricing_method"');
    expect(src).toContain('{labels.metode}');
  });

  it('has status sortable column', () => {
    expect(src).toContain('column="status"');
    expect(src).toContain('{labels.status}');
  });

  it('loading skeleton has 5 rows', () => {
    expect(src).toContain('{#each Array(5) as _}');
  });

  it('does not use role="grid" on data table', () => {
    expect(src).not.toContain('role="grid"');
  });

  it('has aria-label on table', () => {
    expect(src).toContain('aria-label={labels.pricingRules}');
  });

  it('has aria-busy on loading table', () => {
    expect(src).toContain('aria-busy="true"');
    expect(src).toContain('aria-label={labels.loadingRules}');
  });

  it('has aria-label on actions group', () => {
    expect(src).toContain("t('actionsFor', { name: rule.name })");
  });

  it('action buttons use icon size', () => {
    expect(src).toContain('size="icon"');
  });

  it('imports MoreVertical icon from lucide-svelte for kebab menu', () => {
    expect(src).toContain('MoreVertical');
  });

  it('has onrowclick callback prop', () => {
    expect(src).toContain('onrowclick = (_rule: PricingRule) => {}');
    expect(src).toContain('onrowclick?: (rule: PricingRule) => void');
  });

  it('row click triggers onrowclick', () => {
    expect(src).toContain('onclick={() => onrowclick(rule)}');
  });

  it('uses shared Badge component for approval status', () => {
    expect(src).toContain('<Badge variant={approvalVariant');
    expect(src).toContain('approvalLabel(rule.status');
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
    expect(src).toContain("case 'approved': return labels.statusApproved");
    expect(src).toContain("case 'pending': return labels.statusPending");
    expect(src).toContain("case 'rejected': return labels.statusRejected");
    expect(src).toContain('default: return labels.statusDraft');
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

  it('kebab menu has labeled actions without Audit', () => {
    expect(src).toContain('{labels.submit}');
    expect(src).toContain('{labels.approve}');
    expect(src).toContain('{labels.reject}');
    expect(src).toContain('{labels.edit}');
    expect(src).toContain('{labels.duplicate}');
    expect(src).toContain('{labels.hapus}');
    expect(src).not.toContain('onviewaudit');
  });

  it('has 7-column layout via colgroup', () => {
    expect(src).toContain('table-layout: fixed');
    expect(src).toContain('width: 30%;" />');
  });
});
