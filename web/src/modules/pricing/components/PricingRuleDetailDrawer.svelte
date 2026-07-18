<script lang="ts">
  import { Drawer, Button, Badge, Skeleton } from '$shared/ui';
  import { Pencil, Trash2 } from 'lucide-svelte';
  import type { PricingRule } from '../types';

  let {
    open = $bindable(false),
    rule = null,
    canEdit = false,
    canDelete = false,
    targetNames = new Map<string, string>(),
    customerGroups = [],
    stores = [],
    onclose = () => {},
    onedit = () => {},
    ondelete = () => {},
  }: {
    open?: boolean;
    rule?: PricingRule | null;
    canEdit?: boolean;
    canDelete?: boolean;
    targetNames?: Map<string, string>;
    customerGroups?: { id: number; name: string }[];
    stores?: { id: number; name: string }[];
    onclose?: () => void;
    onedit?: (rule: PricingRule) => void;
    ondelete?: (rule: PricingRule) => void;
  } = $props();

  function timeAgo(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
    if (seconds < 60) return 'Baru saja';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes} menit lalu`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours} jam lalu`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days} hari lalu`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months} bln lalu`;
    const years = Math.floor(months / 12);
    return `${years} thn lalu`;
  }

  function formatDate(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    try {
      return new Date(dateStr).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' });
    } catch { return dateStr; }
  }

  function formatDateTime(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    try {
      return new Date(dateStr).toLocaleString('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch { return dateStr; }
  }

  function formatPrice(v: number): string {
    return v?.toLocaleString('id-ID') || '0';
  }

  function valueLabel(r: PricingRule): string {
    switch (r.pricing_method) {
      case 'fixed_price': return `Rp ${formatPrice(r.pricing_value)}`;
      case 'discount_percent': return `${r.pricing_value}%`;
      case 'discount_amount': return `-Rp ${formatPrice(r.pricing_value)}`;
      case 'markup_percent': return `+${r.pricing_value}%`;
      default: return String(r.pricing_value);
    }
  }

  function methodLabel(m: string): string {
    const map: Record<string, string> = {
      fixed_price: 'Harga Tetap',
      discount_percent: 'Diskon %',
      discount_amount: 'Diskon Rp',
      markup_percent: 'Markup %',
    };
    return map[m] || m;
  }

  function typeLabel(t: string): string {
    return t === 'special_price' ? 'Harga Spesial' : 'Promosi';
  }

  function targetLabel(r: PricingRule): string {
    if (r.product_id) return targetNames.get(`product:${r.product_id}`) || `Product #${r.product_id}`;
    if (r.category_id) return targetNames.get(`category:${r.category_id}`) || `Category #${r.category_id}`;
    if (r.brand_id) return targetNames.get(`brand:${r.brand_id}`) || `Brand #${r.brand_id}`;
    return '-';
  }

  function targetScope(r: PricingRule): string {
    if (r.product_id) return 'Produk';
    if (r.category_id) return 'Kategori';
    if (r.brand_id) return 'Merek';
    return '-';
  }

  function approvalVariant(status: string): 'success' | 'warning' | 'danger' | 'muted' {
    switch (status) {
      case 'approved': return 'success';
      case 'pending': return 'warning';
      case 'rejected': return 'danger';
      default: return 'muted';
    }
  }

  function approvalLabel(status: string): string {
    switch (status) {
      case 'approved': return 'Disetujui';
      case 'pending': return 'Menunggu';
      case 'rejected': return 'Ditolak';
      default: return 'Draft';
    }
  }

  function dayLabel(d: string): string {
    const map: Record<string, string> = {
      mon: 'Sen', tue: 'Sel', wed: 'Rab', thu: 'Kam', fri: 'Jum', sat: 'Sab', sun: 'Min',
    };
    return map[d] || d;
  }

  const dayLabels = $derived(
    rule?.recurrence_days?.map(dayLabel).join(', ') || null
  );

  function handleClose() {
    onclose();
  }
</script>

<Drawer bind:open title={rule?.name || 'Detail Rule'} onclose={handleClose}>
  {#if rule}
    <div class="space-y-6">
      <div class="flex items-start gap-4">
        <div class="w-12 h-12 rounded-xl bg-primary-subtle flex items-center justify-center text-primary-light text-lg font-bold shrink-0">
          {rule.name.charAt(0).toUpperCase()}
        </div>
        <div class="flex-1 min-w-0">
          <h3 class="text-base font-semibold text-text-primary truncate">{rule.name}</h3>
          <div class="mt-1.5 flex items-center gap-2">
            <Badge variant={approvalVariant(rule.status)} size="sm">{approvalLabel(rule.status)}</Badge>
            <Badge variant={rule.is_active ? 'success' : 'muted'} size="sm">{rule.is_active ? 'Aktif' : 'Nonaktif'}</Badge>
          </div>
        </div>
      </div>

      <div class="space-y-5">
        <div>
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Harga</h4>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-text-muted">Nilai</span>
              <span class="text-text-secondary font-semibold">{valueLabel(rule)}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-text-muted">Metode</span>
              <span class="text-text-secondary">{methodLabel(rule.pricing_method)}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-text-muted">Tipe</span>
              <Badge variant="default" size="sm">{typeLabel(rule.pricing_type)}</Badge>
            </div>
          </div>
        </div>

        <div>
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Target</h4>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-text-muted">Cakupan</span>
              <span class="text-text-secondary">{targetScope(rule)}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-text-muted">Nama</span>
              <span class="text-text-secondary text-right max-w-[220px] truncate">{targetLabel(rule)}</span>
            </div>
          </div>
        </div>

        <div>
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Kuantitas</h4>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-text-muted">Min. Qty</span>
              <span class="text-text-secondary tabular-nums">{rule.minimum_quantity}</span>
            </div>
            {#if rule.maximum_quantity}
              <div class="flex justify-between">
                <span class="text-text-muted">Max. Qty</span>
                <span class="text-text-secondary tabular-nums">{rule.maximum_quantity}</span>
              </div>
            {/if}
            <div class="flex justify-between">
              <span class="text-text-muted">Prioritas</span>
              <span class="text-text-secondary tabular-nums">{rule.priority}</span>
            </div>
          </div>
        </div>

        <div>
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Jadwal</h4>
          <div class="space-y-2 text-sm">
            {#if dayLabels}
              <div class="flex justify-between">
                <span class="text-text-muted">Hari Aktif</span>
                <span class="text-text-secondary">{dayLabels}</span>
              </div>
            {/if}
            {#if rule.time_from || rule.time_to}
              <div class="flex justify-between">
                <span class="text-text-muted">Jam Aktif</span>
                <span class="text-text-secondary">{rule.time_from || '00:00'} – {rule.time_to || '23:59'}</span>
              </div>
            {/if}
            <div class="flex justify-between">
              <span class="text-text-muted">Berlaku Dari</span>
              <span class="text-text-secondary">{formatDate(rule.effective_from)}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-text-muted">Berlaku Sampai</span>
              <span class="text-text-secondary">{formatDate(rule.effective_until)}</span>
            </div>
          </div>
        </div>

        <div>
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Kondisi</h4>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-text-muted">Gabungkan Rule</span>
              <span class="text-text-secondary">{rule.allow_combine ? 'Ya' : 'Tidak'}</span>
            </div>
            {#if rule.customer_group_id}
              <div class="flex justify-between">
                <span class="text-text-muted">Group Pelanggan</span>
                <span class="text-text-secondary">{customerGroups.find(cg => cg.id === rule.customer_group_id)?.name || `#${rule.customer_group_id}`}</span>
              </div>
            {/if}
            {#if rule.store_id}
              <div class="flex justify-between">
                <span class="text-text-muted">Outlet</span>
                <span class="text-text-secondary">{stores.find(s => s.id === rule.store_id)?.name || `#${rule.store_id}`}</span>
              </div>
            {/if}
          </div>
        </div>

        <div>
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Waktu</h4>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-text-muted">Dibuat</span>
              <span class="text-text-secondary">{formatDateTime(rule.created_at)}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-text-muted">Diperbarui</span>
              <span class="text-text-secondary">{timeAgo(rule.updated_at)}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  {/if}

  {#snippet footer()}
    {#if canEdit}
      <Button variant="secondary" onclick={() => onedit(rule!)}>
        <Pencil class="w-4 h-4 mr-2" />
        Edit
      </Button>
    {/if}
    {#if canDelete}
      <Button variant="danger" onclick={() => ondelete(rule!)}>
        <Trash2 class="w-4 h-4 mr-2" />
        Hapus
      </Button>
    {/if}
  {/snippet}
</Drawer>
