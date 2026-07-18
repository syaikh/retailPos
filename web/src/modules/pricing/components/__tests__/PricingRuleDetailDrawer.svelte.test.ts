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

  it('imports Drawer, Button, Badge, Skeleton from shared/ui', () => {
    expect(src).toContain("import { Drawer, Button, Badge, Skeleton } from '$shared/ui'");
  });

  it('imports Pencil and Trash2 icons from lucide-svelte', () => {
    expect(src).toContain('Pencil');
    expect(src).toContain('Trash2');
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

  it('has timeAgo function', () => {
    expect(src).toContain('function timeAgo(dateStr: string | undefined): string');
    expect(src).toContain('Baru saja');
    expect(src).toContain('menit lalu');
    expect(src).toContain('jam lalu');
    expect(src).toContain('hari lalu');
    expect(src).toContain('bln lalu');
    expect(src).toContain('thn lalu');
  });

  it('timeAgo returns dash for undefined', () => {
    expect(src).toContain("if (!dateStr) return '-'");
  });

  it('has formatDate function', () => {
    expect(src).toContain('function formatDate(dateStr: string | undefined): string');
    expect(src).toContain("toLocaleDateString('id-ID'");
  });

  it('formatDate returns dash for undefined', () => {
    const fnIdx = src.indexOf('function formatDate(');
    expect(fnIdx).toBeGreaterThan(-1);
    const fnBody = src.substring(fnIdx, fnIdx + 100);
    expect(fnBody).toContain("return '-'");
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

  it('has methodLabel function with Indonesian labels', () => {
    expect(src).toContain('function methodLabel(m: string): string');
    expect(src).toContain("fixed_price: 'Harga Tetap'");
    expect(src).toContain("discount_percent: 'Diskon %'");
    expect(src).toContain("discount_amount: 'Diskon Rp'");
    expect(src).toContain("markup_percent: 'Markup %'");
  });

  it('has typeLabel function', () => {
    expect(src).toContain('function typeLabel(t: string): string');
    expect(src).toContain("special_price' ? 'Harga Spesial' : 'Promosi'");
  });

  it('has targetLabel function using targetNames map', () => {
    expect(src).toContain('function targetLabel(r: PricingRule): string');
    expect(src).toContain("targetNames.get(`product:${r.product_id}`)");
    expect(src).toContain("targetNames.get(`category:${r.category_id}`)");
    expect(src).toContain("targetNames.get(`brand:${r.brand_id}`)");
  });

  it('has targetScope function', () => {
    expect(src).toContain('function targetScope(r: PricingRule): string');
    expect(src).toContain("'Produk'");
    expect(src).toContain("'Kategori'");
    expect(src).toContain("'Merek'");
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
    expect(src).toContain("case 'approved': return 'Disetujui'");
    expect(src).toContain("case 'pending': return 'Menunggu'");
    expect(src).toContain("case 'rejected': return 'Ditolak'");
    expect(src).toContain("default: return 'Draft'");
  });

  it('has dayLabel function for Indonesian day abbreviations', () => {
    expect(src).toContain('function dayLabel(d: string): string');
    expect(src).toContain("mon: 'Sen'");
    expect(src).toContain("tue: 'Sel'");
    expect(src).toContain("wed: 'Rab'");
    expect(src).toContain("thu: 'Kam'");
    expect(src).toContain("fri: 'Jum'");
    expect(src).toContain("sat: 'Sab'");
    expect(src).toContain("sun: 'Min'");
  });

  it('has dayLabels derived from recurrence_days', () => {
    expect(src).toContain('const dayLabels = $derived');
    expect(src).toContain('rule?.recurrence_days?.map(dayLabel).join');
  });

  it('has handleClose function that calls onclose', () => {
    expect(src).toContain('function handleClose()');
    expect(src).toContain('onclose()');
  });

  it('renders Drawer with title from rule name', () => {
    expect(src).toContain("<Drawer bind:open title={rule?.name || 'Detail Rule'}");
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

  it('shows approval and active status badges', () => {
    expect(src).toContain('approvalVariant(rule.status)');
    expect(src).toContain('approvalLabel(rule.status)');
    expect(src).toContain("rule.is_active ? 'Aktif' : 'Nonaktif'");
  });

  it('has Harga section showing value, method, type', () => {
    expect(src).toContain('>Harga<');
    expect(src).toContain('valueLabel(rule)');
    expect(src).toContain('methodLabel(rule.pricing_method)');
    expect(src).toContain('typeLabel(rule.pricing_type)');
  });

  it('has Target section showing scope and name', () => {
    expect(src).toContain('>Target<');
    expect(src).toContain('targetScope(rule)');
    expect(src).toContain('targetLabel(rule)');
  });

  it('has Kuantitas section showing min qty, max qty (conditional), priority', () => {
    expect(src).toContain('>Kuantitas<');
    expect(src).toContain('rule.minimum_quantity');
    expect(src).toContain('{#if rule.maximum_quantity}');
    expect(src).toContain('rule.priority');
  });

  it('has Jadwal section with conditional day and time ranges', () => {
    expect(src).toContain('>Jadwal<');
    expect(src).toContain('{#if dayLabels}');
    expect(src).toContain('{#if rule.time_from || rule.time_to}');
    expect(src).toContain('rule.time_from');
    expect(src).toContain('rule.time_to');
    expect(src).toContain('formatDate(rule.effective_from)');
    expect(src).toContain('formatDate(rule.effective_until)');
  });

  it('has Kondisi section with combine, customer group, and store', () => {
    expect(src).toContain('>Kondisi<');
    expect(src).toContain("rule.allow_combine ? 'Ya' : 'Tidak'");
    expect(src).toContain('{#if rule.customer_group_id}');
    expect(src).toContain('{#if rule.store_id}');
  });

  it('resolves customer group name from customerGroups array', () => {
    expect(src).toContain('customerGroups.find(cg => cg.id === rule.customer_group_id)');
  });

  it('resolves store name from stores array', () => {
    expect(src).toContain('stores.find(s => s.id === rule.store_id)');
  });

  it('has Waktu section showing created and updated timestamps', () => {
    expect(src).toContain('>Waktu<');
    expect(src).toContain('formatDateTime(rule.created_at)');
    expect(src).toContain('timeAgo(rule.updated_at)');
  });

  it('renders footer snippet with Edit button when canEdit', () => {
    expect(src).toContain('{#snippet footer()}');
    expect(src).toContain('{#if canEdit}');
    expect(src).toContain('onedit(rule!)');
    expect(src).toContain('Edit');
  });

  it('renders Delete button when canDelete', () => {
    expect(src).toContain('{#if canDelete}');
    expect(src).toContain('ondelete(rule!)');
    expect(src).toContain('Hapus');
  });

  it('does not show max qty when rule.maximum_quantity is falsy', () => {
    const maxIdx = src.indexOf('{#if rule.maximum_quantity}');
    expect(maxIdx).toBeGreaterThan(-1);
    const block = src.substring(maxIdx, maxIdx + 300);
    expect(block).toContain('Max. Qty');
  });
});
