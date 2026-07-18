<script lang="ts">
  import { Drawer, Button, Badge } from '$shared/ui';
  import { Pencil, Trash2, DollarSign, Target, Hash, Clock, Settings, CalendarDays } from 'lucide-svelte';
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

  const ALL_DAYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const;

  function timeAgo(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
    if (seconds < 60) return 'Baru saja';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m lalu`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}j lalu`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}h lalu`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months}bln lalu`;
    const years = Math.floor(months / 12);
    return `${years}thn lalu`;
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

  function valueSubLabel(r: PricingRule): string {
    switch (r.pricing_method) {
      case 'fixed_price': return 'Harga Tetap';
      case 'discount_percent': return 'Diskon Persen';
      case 'discount_amount': return 'Diskon nominal';
      case 'markup_percent': return 'Markup Persen';
      default: return r.pricing_method;
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

  function dayShort(d: string): string {
    const map: Record<string, string> = {
      mon: 'Sn', tue: 'Sl', wed: 'Rb', thu: 'Km', fri: 'Jm', sat: 'Sb', sun: 'Mg',
    };
    return map[d] || d;
  }

  function dayFull(d: string): string {
    const map: Record<string, string> = {
      mon: 'Senin', tue: 'Selasa', wed: 'Rabu', thu: 'Kamis', fri: 'Jumat', sat: 'Sabtu', sun: 'Minggu',
    };
    return map[d] || d;
  }

  const activeDays = $derived(rule?.recurrence_days || []);
  const inactiveDays = $derived(ALL_DAYS.filter(d => !activeDays.includes(d)));

  function handleClose() {
    onclose();
  }
</script>

<Drawer bind:open title={rule?.name || 'Detail Rule'} onclose={handleClose}>
  {#if rule}
    <div class="space-y-5">
      <div class="flex items-start gap-3.5">
        <div class="w-11 h-11 rounded-xl bg-primary-subtle flex items-center justify-center text-primary-light text-base font-bold shrink-0">
          {rule.name.charAt(0).toUpperCase()}
        </div>
        <div class="flex-1 min-w-0">
          <h3 class="text-sm font-semibold text-text-primary truncate leading-snug">{rule.name}</h3>
          <div class="mt-1.5 flex items-center gap-1.5 flex-wrap">
            <Badge variant={approvalVariant(rule.status)} size="sm">{approvalLabel(rule.status)}</Badge>
            <Badge variant={rule.is_active ? 'success' : 'muted'} size="sm">{rule.is_active ? 'Aktif' : 'Nonaktif'}</Badge>
            <Badge variant="default" size="sm">{typeLabel(rule.pricing_type)}</Badge>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-2.5">
        <div class="bg-surface-default rounded-xl p-3 border border-border/40">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1.5 flex items-center gap-1">
            <DollarSign size={10} /> Nilai
          </p>
          <p class="text-base font-bold text-primary-light tabular-nums leading-tight">{valueLabel(rule)}</p>
          <p class="text-[11px] text-text-muted mt-0.5">{valueSubLabel(rule)}</p>
        </div>
        <div class="bg-surface-default rounded-xl p-3 border border-border/40">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1.5 flex items-center gap-1">
            <Target size={10} /> Target
          </p>
          <p class="text-sm font-semibold text-text-primary leading-tight truncate">{targetScope(rule)}</p>
          <p class="text-[11px] text-text-muted mt-0.5 truncate">{targetLabel(rule)}</p>
        </div>
        <div class="bg-surface-default rounded-xl p-3 border border-border/40">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1.5 flex items-center gap-1">
            <Hash size={10} /> Kuantitas
          </p>
          <p class="text-sm font-semibold text-text-primary tabular-nums leading-tight">
            {rule.minimum_quantity}{#if rule.maximum_quantity} – {rule.maximum_quantity}{:else}+{/if}
          </p>
          <p class="text-[11px] text-text-muted mt-0.5">Prioritas {rule.priority}</p>
        </div>
        <div class="bg-surface-default rounded-xl p-3 border border-border/40">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1.5 flex items-center gap-1">
            <Clock size={10} /> Diperbarui
          </p>
          <p class="text-sm font-semibold text-text-primary leading-tight">{timeAgo(rule.updated_at)}</p>
          <p class="text-[11px] text-text-muted mt-0.5 font-mono">{formatDateTime(rule.updated_at)}</p>
        </div>
      </div>

      <div class="bg-surface-default rounded-xl border border-border/40 divide-y divide-border/30">
        <div class="px-3.5 py-3">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2.5 flex items-center gap-1.5">
            <DollarSign size={10} /> Harga
          </p>
          <div class="space-y-2 text-sm">
            <div class="flex items-center justify-between">
              <span class="text-text-muted">Metode</span>
              <span class="text-text-secondary font-medium">{methodLabel(rule.pricing_method)}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-text-muted">Gabungkan Rule</span>
              <span class="text-text-secondary font-medium">{rule.allow_combine ? 'Ya' : 'Tidak'}</span>
            </div>
          </div>
        </div>

        <div class="px-3.5 py-3">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2.5 flex items-center gap-1.5">
            <CalendarDays size={10} /> Jadwal
          </p>
          <div class="space-y-3 text-sm">
            {#if activeDays.length > 0}
              <div>
                <span class="text-text-muted text-xs block mb-1.5">Hari Aktif</span>
                <div class="flex flex-wrap gap-1">
                  {#each ALL_DAYS as day}
                    {@const isActive = activeDays.includes(day)}
                    <span class="inline-flex items-center justify-center w-7 h-7 rounded-lg text-[11px] font-semibold transition-colors
                      {isActive ? 'bg-primary-subtle text-primary-light' : 'bg-bg-secondary text-text-muted/40 line-through'}"
                      title={dayFull(day)}
                    >
                      {dayShort(day)}
                    </span>
                  {/each}
                </div>
              </div>
            {/if}
            {#if rule.time_from || rule.time_to}
              <div class="flex items-center justify-between">
                <span class="text-text-muted">Jam Aktif</span>
                <span class="text-text-secondary font-medium font-mono text-xs">{rule.time_from || '00:00'} – {rule.time_to || '23:59'}</span>
              </div>
            {/if}
            <div class="flex items-center justify-between">
              <span class="text-text-muted">Berlaku Dari</span>
              <span class="text-text-secondary font-medium text-xs">{formatDateTime(rule.effective_from)}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-text-muted">Berlaku Sampai</span>
              <span class="text-text-secondary font-medium text-xs">{formatDateTime(rule.effective_until)}</span>
            </div>
          </div>
        </div>

        <div class="px-3.5 py-3">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2.5 flex items-center gap-1.5">
            <Settings size={10} /> Kondisi
          </p>
          <div class="space-y-2 text-sm">
            {#if rule.customer_group_id}
              <div class="flex items-center justify-between">
                <span class="text-text-muted">Group Pelanggan</span>
                <span class="text-text-secondary font-medium">{customerGroups.find(cg => cg.id === rule.customer_group_id)?.name || `#${rule.customer_group_id}`}</span>
              </div>
            {/if}
            {#if rule.store_id}
              <div class="flex items-center justify-between">
                <span class="text-text-muted">Outlet</span>
                <span class="text-text-secondary font-medium">{stores.find(s => s.id === rule.store_id)?.name || `#${rule.store_id}`}</span>
              </div>
            {/if}
            {#if !rule.customer_group_id && !rule.store_id}
              <p class="text-xs text-text-muted italic">Berlaku untuk semua group & outlet</p>
            {/if}
          </div>
        </div>

        <div class="px-3.5 py-3">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2.5 flex items-center gap-1.5">
            <Clock size={10} /> Riwayat
          </p>
          <div class="space-y-2 text-sm">
            <div class="flex items-center justify-between">
              <span class="text-text-muted">Dibuat</span>
              <span class="text-text-secondary font-medium text-xs">{formatDateTime(rule.created_at)}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-text-muted">ID Rule</span>
              <span class="text-text-muted font-mono text-xs">#{rule.id}</span>
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
