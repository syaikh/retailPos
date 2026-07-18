import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PricingRulesPage.svelte'), 'utf-8');
}

describe('PricingRulesPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports pricing service functions', () => {
    expect(src).toContain("import { getPricingRules, createPricingRule, updatePricingRule, deletePricingRule, submitPricingRule, approvePricingRule, rejectPricingRule, searchProducts, getCustomerGroups, getStores, checkConflicts } from '../services/pricing-service'");
  });

  it('imports product service functions including getProductById', () => {
    expect(src).toContain("import { getCategories, getBrands, getProductById } from '$modules/product/services/product-service'");
  });

  it('imports toast store', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('imports auth store', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
  });

  it('imports PricingRule type', () => {
    expect(src).toContain("import type { PricingRule } from '../types'");
  });

  it('imports extracted child components', () => {
    expect(src).toContain("import PricingRulesToolbar from './PricingRulesToolbar.svelte'");
    expect(src).toContain("import PricingRulesTable from './PricingRulesTable.svelte'");
  });

  it('imports ImportWizard from shared/ui', () => {
    expect(src).toContain("import { Button, Input, Modal, Pagination, ConfirmDeleteModal, Badge, ImportWizard } from '$shared/ui'");
  });

  it('imports PriceSimulationModal', () => {
    expect(src).toContain("import PriceSimulationModal from './PriceSimulationModal.svelte'");
  });

  it('uses $state for rules and pagination', () => {
    expect(src).toContain('let rules = $state');
    expect(src).toContain('let total = $state');
    expect(src).toContain('let limit = $state');
    expect(src).toContain('let offset = $state');
  });

  it('uses $state for modal and form state', () => {
    expect(src).toContain('let showModal = $state');
    expect(src).toContain('let showDeleteModal = $state');
    expect(src).toContain('let form = $state');
  });

  it('has showImportWizard and showSimulation state', () => {
    expect(src).toContain('let showImportWizard = $state');
    expect(src).toContain('let showSimulation = $state');
  });

  it('has showDetailCols state for column visibility toggle', () => {
    expect(src).toContain('let showDetailCols = $state(false)');
  });

  it('uses $derived for sorting', () => {
    expect(src).toContain('let sortedRules = $derived');
  });

  it('has productNames state for name resolution', () => {
    expect(src).toContain('let productNames = $state<Map<number, string>>(new Map())');
  });

  it('has targetNames derived that builds map from categories, brands, productNames', () => {
    expect(src).toContain('let targetNames = $derived');
    expect(src).toContain("map.set(`category:${c.id}`, c.name)");
    expect(src).toContain("map.set(`brand:${b.id}`, b.name)");
    expect(src).toContain("map.set(`product:${id}`, name)");
  });

  it('has maximum_quantity defaulting to empty string in form', () => {
    expect(src).toContain("maximum_quantity: '' as number | string");
  });

  it('resets maximum_quantity to empty string in resetForm', () => {
    const resetIdx = src.indexOf('function resetForm()');
    expect(resetIdx).toBeGreaterThan(-1);
    const resetBlock = src.substring(resetIdx, resetIdx + 500);
    expect(resetBlock).toContain("maximum_quantity: ''");
  });

  it('uses Number() conversion in max qty validation', () => {
    expect(src).toContain('form.minimum_quantity > Number(form.maximum_quantity)');
  });

  it('removes empty maximum_quantity from payload before saving', () => {
    expect(src).toContain("payload.maximum_quantity === ''");
  });

  it('has fetchRules function', () => {
    expect(src).toContain('async function fetchRules()');
  });

  it('has resolveProductNames function', () => {
    expect(src).toContain('async function resolveProductNames()');
    expect(src).toContain('getProductById(id)');
  });

  it('calls resolveProductNames after fetching rules', () => {
    const fetchIdx = src.indexOf('async function fetchRules()');
    expect(fetchIdx).toBeGreaterThan(-1);
    const fetchBlock = src.substring(fetchIdx, fetchIdx + 800);
    expect(fetchBlock).toContain('resolveProductNames()');
  });

  it('has saveRule function', () => {
    expect(src).toContain('async function saveRule(e: Event)');
  });

  it('has confirmDelete function', () => {
    expect(src).toContain('async function confirmDelete()');
  });

  it('has openAdd and openEdit functions', () => {
    expect(src).toContain('function openAdd()');
    expect(src).toContain('function openEdit(rule: PricingRule)');
  });

  it('has handleDuplicate function that creates copy of rule', () => {
    expect(src).toContain('function handleDuplicate(rule: PricingRule)');
    expect(src).toContain('`${rule.name} (Salinan)`');
  });

  it('handleDuplicate sets modalMode to add and clears effective dates', () => {
    const dupIdx = src.indexOf('function handleDuplicate(rule: PricingRule)');
    expect(dupIdx).toBeGreaterThan(-1);
    const dupBlock = src.substring(dupIdx, dupIdx + 900);
    expect(dupBlock).toContain("modalMode = 'add'");
    expect(dupBlock).toContain("selectedRule = null");
    expect(dupBlock).toContain("effective_from: ''");
    expect(dupBlock).toContain("effective_until: ''");
  });

  it('has handleImportComplete function', () => {
    expect(src).toContain('function handleImportComplete()');
    expect(src).toContain("toast.success('Import pricing rules selesai.')");
  });

  it('has handleBulkActivate function using Promise.allSettled', () => {
    expect(src).toContain('async function handleBulkActivate(ids: number[])');
    expect(src).toContain('Promise.allSettled');
    expect(src).toContain("updatePricingRule(id, { is_active: true })");
  });

  it('has handleBulkDeactivate function using Promise.allSettled', () => {
    expect(src).toContain('async function handleBulkDeactivate(ids: number[])');
    expect(src).toContain("updatePricingRule(id, { is_active: false })");
  });

  it('has handleBulkDelete function using Promise.allSettled', () => {
    expect(src).toContain('async function handleBulkDelete(ids: number[])');
    expect(src).toContain('deletePricingRule(id)');
  });

  it('has handleKeydown function with Ctrl+N and Ctrl+K shortcuts', () => {
    expect(src).toContain('function handleKeydown(e: KeyboardEvent)');
    expect(src).toContain("e.key === 'n'");
    expect(src).toContain("e.key === 'k'");
    expect(src).toContain('#pricing-search');
  });

  it('handleKeydown ignores input/textarea/select elements', () => {
    expect(src).toContain('e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement || e.target instanceof HTMLSelectElement');
  });

  it('has validateForm function with all validation rules', () => {
    expect(src).toContain('function validateForm(): Record<string, string>');
    expect(src).toContain("errors.name = 'Nama rule wajib diisi.'");
    expect(src).toContain("errors.target = 'Pilih minimal satu target.'");
    expect(src).toContain("errors.qty = 'Max Qty harus lebih besar dari Min Qty.'");
    expect(src).toContain("errors.dates = 'Tanggal selesai tidak boleh sebelum tanggal mulai.'");
  });

  it('has handleSort function', () => {
    expect(src).toContain('function handleSort(col: string)');
  });

  it('has handlePageChange function', () => {
    expect(src).toContain('function handlePageChange(newOffset: number, newLimit: number)');
  });

  it('has product, category, and brand search functions', () => {
    expect(src).toContain('function handleProductSearch()');
    expect(src).toContain('function handleCategorySearch()');
    expect(src).toContain('function handleBrandSearch()');
  });

  it('has recurrence day helpers', () => {
    expect(src).toContain('function toggleDay(day: string)');
    expect(src).toContain('function selectAllDays()');
    expect(src).toContain('function selectWorkDays()');
    expect(src).toContain('function selectWeekend()');
    expect(src).toContain('function clearDays()');
  });

  it('has permission checks via auth store', () => {
    expect(src).toContain("includes('pricing:create')");
    expect(src).toContain("includes('pricing:update')");
    expect(src).toContain("includes('pricing:delete')");
  });

  it('fetches customer groups, stores, categories, and brands on mount', () => {
    expect(src).toContain('getCustomerGroups()');
    expect(src).toContain('getStores()');
    expect(src).toContain('getCategories()');
    expect(src).toContain('getBrands()');
  });

  it('onMount returns cleanup function for keydown listener', () => {
    expect(src).toContain("window.addEventListener('keydown', handleKeydown)");
    expect(src).toContain("return () => window.removeEventListener('keydown', handleKeydown)");
  });

  it('renders PricingRulesToolbar and PricingRulesTable', () => {
    expect(src).toContain('<PricingRulesToolbar');
    expect(src).toContain('<PricingRulesTable');
  });

  it('passes new bulk and duplicate callbacks to PricingRulesTable', () => {
    expect(src).toContain('onduplicate={handleDuplicate}');
    expect(src).toContain('onbulkactivate={handleBulkActivate}');
    expect(src).toContain('onbulkdeactivate={handleBulkDeactivate}');
    expect(src).toContain('onbulkdelete={handleBulkDelete}');
  });

  it('passes targetNames and canCreate to PricingRulesTable', () => {
    expect(src).toContain('{targetNames}');
    expect(src).toContain('{canCreate}');
  });

  it('passes onimport and onsimulate to PricingRulesToolbar', () => {
    expect(src).toContain('onimport={() => showImportWizard = true}');
    expect(src).toContain('onsimulate={() => showSimulation = true}');
  });

  it('passes showDetailCols to PricingRulesToolbar and PricingRulesTable', () => {
    expect(src).toContain('bind:showDetailCols');
    expect(src).toContain('{showDetailCols}');
  });

  it('renders Modal for add/edit', () => {
    expect(src).toContain('<Modal');
    expect(src).toContain("'Tambah Pricing Rule'");
    expect(src).toContain("'Edit Pricing Rule'");
  });

  it('renders ConfirmDeleteModal', () => {
    expect(src).toContain('<ConfirmDeleteModal');
  });

  it('renders ImportWizard with module="pricing_rules"', () => {
    expect(src).toContain('<ImportWizard');
    expect(src).toContain('module="pricing_rules"');
    expect(src).toContain('displayName="Pricing Rules"');
    expect(src).toContain('onComplete={handleImportComplete}');
  });

  it('renders PriceSimulationModal', () => {
    expect(src).toContain('<PriceSimulationModal');
    expect(src).toContain('bind:open={showSimulation}');
  });

  it('has summary preview derived', () => {
    expect(src).toContain('let summaryPreview = $derived');
  });

  it('uses payload.effective_from with T00:00:00Z suffix', () => {
    expect(src).toContain("payload.effective_from = payload.effective_from + 'T00:00:00Z'");
  });

  it('uses payload.effective_until with T23:59:59Z suffix', () => {
    expect(src).toContain("payload.effective_until = payload.effective_until + 'T23:59:59Z'");
  });

  it('has workDaysSelected and weekendSelected as $derived (not $derived with function call)', () => {
    expect(src).toContain("const workDaysSelected = $derived(['mon', 'tue', 'wed', 'thu', 'fri'].every");
    expect(src).toContain("const weekendSelected = $derived(['sat', 'sun'].every");
  });

  it('has skip-to-content link for accessibility', () => {
    expect(src).toContain('Lewati ke tabel rules');
    expect(src).toContain('sr-only');
  });

  it('has no inline page heading (consistent with other pages)', () => {
    expect(src).not.toContain('<h1');
  });

  it('has aria-live region on table container', () => {
    expect(src).toContain('aria-live="polite"');
    expect(src).toContain('aria-atomic="true"');
  });

  it('typeLabel default is Indonesian', () => {
    expect(src).toContain("'Semua Tipe'");
  });

  it('imports checkConflicts from pricing service', () => {
    expect(src).toContain('checkConflicts');
  });

  it('imports ConflictRule type from pricing service', () => {
    expect(src).toContain("import type { ConflictRule } from '../services/pricing-service'");
  });

  it('imports AlertTriangle icon for conflict warnings', () => {
    expect(src).toContain('AlertTriangle');
  });

  it('has conflict detection state variables', () => {
    expect(src).toContain('conflictRules = $state<ConflictRule[]>([])');
    expect(src).toContain('checkingConflicts = $state(false)');
    expect(src).toContain('showConflictWarning = $state(false)');
  });

  it('has runConflictCheck function', () => {
    expect(src).toContain('async function runConflictCheck()');
    expect(src).toContain('await checkConflicts(');
  });

  it('has doSave function', () => {
    expect(src).toContain('async function doSave()');
  });

  it('has handleViewAudit function that uses sessionStorage', () => {
    expect(src).toContain('function handleViewAudit(rule: PricingRule)');
    expect(src).toContain("sessionStorage.setItem('auditLogFilter'");
    expect(src).toContain("window.location.href = '/admin/audit-logs'");
  });

  it('saveRule checks conflicts before saving', () => {
    expect(src).toContain('if (!showConflictWarning)');
    expect(src).toContain('showConflictWarning = true');
  });

  it('conflict warning UI shows AlertTriangle and conflict list', () => {
    expect(src).toContain('showConflictWarning && conflictRules.length > 0');
    expect(src).toContain('Konflik Ditemukan');
  });

  it('modal footer changes text when conflict warning shown', () => {
    expect(src).toContain('showConflictWarning ? \'Tetap Simpan\'');
  });

  it('passes onviewaudit handler to table', () => {
    expect(src).toContain('onviewaudit={handleViewAudit}');
  });
});
