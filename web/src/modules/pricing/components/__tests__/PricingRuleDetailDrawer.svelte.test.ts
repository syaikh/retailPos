import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PricingRuleDetailDrawer.svelte'), 'utf-8');
}

describe('PricingRuleDetailDrawer.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Drawer, Button, Badge from shared/ui', () => {
    expect(src).toContain("import { Drawer, Button, Badge } from '$shared/ui'");
  });

  it('imports lucide-svelte icons', () => {
    expect(src).toContain('Pencil');
    expect(src).toContain('Trash2');
    expect(src).toContain('DollarSign');
    expect(src).toContain('Target');
    expect(src).toContain('Hash');
    expect(src).toContain('Clock');
    expect(src).toContain('Settings');
    expect(src).toContain('CalendarDays');
  });

  it('imports PricingRule type', () => {
    expect(src).toContain("import type { PricingRule } from '../types'");
  });

  it('has open bindable prop', () => {
    expect(src).toContain('open = $bindable(false)');
    expect(src).toContain('open?: boolean');
  });

  it('has rule prop with null default', () => {
    expect(src).toContain('rule = null');
    expect(src).toContain('rule?: PricingRule | null');
  });

  it('has canEdit and canDelete props', () => {
    expect(src).toContain('canEdit = false');
    expect(src).toContain('canDelete = false');
    expect(src).toContain('canEdit?: boolean');
    expect(src).toContain('canDelete?: boolean');
  });

  it('has targetNames prop with Map type', () => {
    expect(src).toContain('targetNames = new Map<string, string>()');
    expect(src).toContain('targetNames?: Map<string, string>');
  });

  it('has customerGroups and stores array props', () => {
    expect(src).toContain('customerGroups = []');
    expect(src).toContain('stores = []');
    expect(src).toContain('customerGroups?: { id: number; name: string }[]');
    expect(src).toContain('stores?: { id: number; name: string }[]');
  });

  it('has onclose, onedit, ondelete callback props', () => {
    expect(src).toContain('onclose = () => {}');
    expect(src).toContain('onedit = () => {}');
    expect(src).toContain('ondelete = () => {}');
    expect(src).toContain('onclose?: () => void');
    expect(src).toContain('onedit?: (rule: PricingRule) => void');
    expect(src).toContain('ondelete?: (rule: PricingRule) => void');
  });

  it('has timeAgo function with Indonesian labels', () => {
    expect(src).toContain('function timeAgo(dateStr: string | undefined): string');
    expect(src).toContain('return labels.justNow');
    expect(src).toContain("t('minutesAgo', { n: minutes })");
  });

  it('timeAgo returns dash for undefined', () => {
    expect(src).toContain("if (!dateStr) return '-'");
  });

  it('has formatDateTime function', () => {
    expect(src).toContain('function formatDateTime(dateStr: string | undefined): string');
    expect(src).toContain("toLocaleString('id-ID'");
  });

  it('has formatPrice function', () => {
    expect(src).toContain('function formatPrice(v: number): string');
    expect(src).toContain("toLocaleString('id-ID')");
  });

  it('has valueLabel function with all pricing methods', () => {
    expect(src).toContain('function valueLabel(r: PricingRule): string');
    expect(src).toContain("case 'fixed_price': return `Rp ${formatPrice(r.pricing_value)}`");
    expect(src).toContain("case 'discount_percent': return `${r.pricing_value}%`");
    expect(src).toContain("case 'discount_amount': return `-Rp ${formatPrice(r.pricing_value)}`");
    expect(src).toContain("case 'markup_percent': return `+${r.pricing_value}%`");
  });

  it('has valueSubLabel function for stat card subtitle', () => {
    expect(src).toContain('function valueSubLabel(r: PricingRule): string');
    expect(src).toContain('return labels.diskonsPersen');
    expect(src).toContain('return labels.diskonsNominal');
    expect(src).toContain('return labels.markupPersen');
  });

  it('has methodLabel function with Indonesian labels', () => {
    expect(src).toContain('function methodLabel(m: string): string');
    expect(src).toContain('fixed_price: labels.hargaTetap');
    expect(src).toContain('discount_percent: labels.methodDiscountPercent');
    expect(src).toContain('discount_amount: labels.methodDiscountAmount');
    expect(src).toContain('markup_percent: labels.methodMarkupPercent');
  });

  it('has typeLabel function', () => {
    expect(src).toContain('function typeLabel(t: string): string');
    expect(src).toContain("t === 'special_price' ? labels.hargaSpesial : labels.promosi");
  });

  it('has targetLabel function using targetNames map', () => {
    expect(src).toContain('function targetLabel(r: PricingRule): string');
    expect(src).toContain("targetNames.get(`product:${r.product_id}`)");
    expect(src).toContain("targetNames.get(`category:${r.category_id}`)");
    expect(src).toContain("targetNames.get(`brand:${r.brand_id}`)");
  });

  it('has targetScope function', () => {
    expect(src).toContain('function targetScope(r: PricingRule): string');
    expect(src).toContain('return labels.produk');
    expect(src).toContain('return labels.kategori');
    expect(src).toContain('return labels.merek');
  });

  it('has approvalVariant function with all status mappings', () => {
    expect(src).toContain('function approvalVariant(status: string)');
    expect(src).toContain("case 'approved': return 'success'");
    expect(src).toContain("case 'pending': return 'warning'");
    expect(src).toContain("case 'rejected': return 'danger'");
    expect(src).toContain("default: return 'muted'");
  });

  it('has approvalLabel function with Indonesian labels', () => {
    expect(src).toContain('function approvalLabel(status: string): string');
    expect(src).toContain('return labels.statusApproved');
    expect(src).toContain('return labels.statusPending');
    expect(src).toContain('return labels.statusRejected');
    expect(src).toContain('return labels.statusDraft');
  });

  it('has dayShort function for day chip abbreviations', () => {
    expect(src).toContain('function dayShort(d: string): string');
    expect(src).toContain('mon: labels.dayMonShort');
    expect(src).toContain('tue: labels.dayTueShort');
    expect(src).toContain('wed: labels.dayWedShort');
    expect(src).toContain('thu: labels.dayThuShort');
    expect(src).toContain('fri: labels.dayFriShort');
    expect(src).toContain('sat: labels.daySatShort');
    expect(src).toContain('sun: labels.daySunShort');
  });

  it('has dayFull function for full day names', () => {
    expect(src).toContain('function dayFull(d: string): string');
    expect(src).toContain('mon: labels.dayMon');
    expect(src).toContain('sun: labels.daySun');
  });

  it('has activeDays and inactiveDays derived', () => {
    expect(src).toContain('const activeDays = $derived');
    expect(src).toContain('const inactiveDays = $derived');
  });

  it('has ALL_DAYS constant', () => {
    expect(src).toContain("const ALL_DAYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const");
  });

  it('has handleClose function that calls onclose', () => {
    expect(src).toContain('function handleClose()');
    expect(src).toContain('onclose()');
  });

  it('renders Drawer with title from rule name', () => {
    expect(src).toContain("<Drawer bind:open title={rule?.name || labels.detailRule}");
    expect(src).toContain('onclose={handleClose}');
  });

  it('conditionally renders content when rule exists', () => {
    expect(src).toContain('{#if rule}');
  });

  it('shows rule initial in avatar', () => {
    expect(src).toContain('rule.name.charAt(0).toUpperCase()');
  });

  it('shows rule name as heading', () => {
    expect(src).toContain('{rule.name}');
  });

  it('shows approval, active, and type badges in header', () => {
    expect(src).toContain('approvalVariant(rule.status)');
    expect(src).toContain('approvalLabel(rule.status)');
    expect(src).toContain("rule.is_active ? labels.aktif : labels.nonaktif");
    expect(src).toContain('typeLabel(rule.pricing_type)');
  });

  it('has 2x2 stat cards grid at top', () => {
    expect(src).toContain('grid grid-cols-2');
    expect(src).toContain('bg-surface-default rounded-xl');
  });

  it('stat card shows value and valueSubLabel', () => {
    expect(src).toContain('valueLabel(rule)');
    expect(src).toContain('valueSubLabel(rule)');
  });

  it('stat card shows target scope and name', () => {
    expect(src).toContain('targetScope(rule)');
    expect(src).toContain('targetLabel(rule)');
  });

  it('stat card shows quantity range inline', () => {
    expect(src).toContain('rule.minimum_quantity');
    expect(src).toContain('rule.priority');
  });

  it('stat card shows updated time', () => {
    expect(src).toContain('timeAgo(rule.updated_at)');
    expect(src).toContain('formatDateTime(rule.updated_at)');
  });

  it('has card-based sections with dividers', () => {
    expect(src).toContain('divide-y divide-border/30');
  });

  it('has Harga section with method and allow_combine', () => {
    expect(src).toContain('{labels.harga}');
    expect(src).toContain('methodLabel(rule.pricing_method)');
    expect(src).toContain('rule.allow_combine ? labels.yes : labels.no');
  });

  it('has Jadwal section with day chips', () => {
    expect(src).toContain('{labels.jadwal}');
    expect(src).toContain('activeDays.includes(day)');
    expect(src).toContain('dayShort(day)');
    expect(src).toContain('dayFull(day)');
  });

  it('Jadwal has time range and effective dates', () => {
    expect(src).toContain('{#if rule.time_from || rule.time_to}');
    expect(src).toContain('rule.time_from');
    expect(src).toContain('rule.time_to');
    expect(src).toContain('formatDateTime(rule.effective_from)');
    expect(src).toContain('formatDateTime(rule.effective_until)');
  });

  it('has Kondisi section with customer group and store', () => {
    expect(src).toContain('{labels.kondisi}');
    expect(src).toContain('{#if rule.customer_group_id}');
    expect(src).toContain('{#if rule.store_id}');
  });

  it('resolves customer group name from customerGroups array', () => {
    expect(src).toContain('customerGroups.find(cg => cg.id === rule.customer_group_id)');
  });

  it('resolves store name from stores array', () => {
    expect(src).toContain('stores.find(s => s.id === rule.store_id)');
  });

  it('has fallback text when no customer group or store', () => {
    expect(src).toContain('{labels.appliesToAllGroupsAndStores}');
  });

  it('has Riwayat section showing created timestamp and rule ID', () => {
    expect(src).toContain('{labels.riwayat}');
    expect(src).toContain('formatDateTime(rule.created_at)');
    expect(src).toContain('rule.id');
  });

  it('renders footer snippet with Edit button when canEdit', () => {
    expect(src).toContain('{#snippet footer()}');
    expect(src).toContain('{#if canEdit}');
    expect(src).toContain('onedit(rule!)');
    expect(src).toContain('{labels.edit}');
  });

  it('renders Delete button when canDelete', () => {
    expect(src).toContain('{#if canDelete}');
    expect(src).toContain('ondelete(rule!)');
    expect(src).toContain('{labels.hapus}');
  });
});
