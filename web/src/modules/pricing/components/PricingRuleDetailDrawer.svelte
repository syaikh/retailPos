<script lang="ts">
  import { Drawer, Button, Badge } from '$shared/ui';
  import { Pencil, Trash2, DollarSign, Target, Hash, Clock, Settings, CalendarDays } from 'lucide-svelte';
  import { labels, t, formatLocaleDate } from '$shared/i18n';
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
    if (seconds < 60) return labels.justNow;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return t('minutesAgo', { n: minutes });
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return t('hoursAgo', { n: hours });
    const days = Math.floor(hours / 24);
    if (days < 30) return t('daysAgo', { n: days });
    const months = Math.floor(days / 30);
    if (months < 12) return t('monthsAgo', { n: months });
    const years = Math.floor(months / 12);
    return t('yearsAgo', { n: years });
  }

  function formatDateTime(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    try {
      return formatLocaleDate(new Date(dateStr), { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
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
      case 'fixed_price': return labels.hargaTetap;
      case 'discount_percent': return labels.diskonsPersen;
      case 'discount_amount': return labels.diskonsNominal;
      case 'markup_percent': return labels.markupPersen;
      default: return r.pricing_method;
    }
  }

  function methodLabel(m: string): string {
    const map: Record<string, string> = {
      fixed_price: labels.hargaTetap,
      discount_percent: labels.methodDiscountPercent,
      discount_amount: labels.methodDiscountAmount,
      markup_percent: labels.methodMarkupPercent,
    };
    return map[m] || m;
  }

  function typeLabel(t: string): string {
    return t === 'special_price' ? labels.hargaSpesial : labels.promosi;
  }

  function targetLabel(r: PricingRule): string {
    if (r.product_id) return targetNames.get(`product:${r.product_id}`) || t('productNumber', { value: r.product_id });
    if (r.category_id) return targetNames.get(`category:${r.category_id}`) || t('categoryNumber', { value: r.category_id });
    if (r.brand_id) return targetNames.get(`brand:${r.brand_id}`) || t('brandNumber', { value: r.brand_id });
    return '-';
  }

  function targetScope(r: PricingRule): string {
    if (r.product_id) return labels.produk;
    if (r.category_id) return labels.kategori;
    if (r.brand_id) return labels.merek;
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
      case 'approved': return labels.statusApproved;
      case 'pending': return labels.statusPending;
      case 'rejected': return labels.statusRejected;
      default: return labels.statusDraft;
    }
  }

  function dayShort(d: string): string {
    const map: Record<string, string> = {
      mon: labels.dayMonShort, tue: labels.dayTueShort, wed: labels.dayWedShort, thu: labels.dayThuShort, fri: labels.dayFriShort, sat: labels.daySatShort, sun: labels.daySunShort,
    };
    return map[d] || d;
  }

  function dayFull(d: string): string {
    const map: Record<string, string> = {
      mon: labels.dayMon, tue: labels.dayTue, wed: labels.dayWed, thu: labels.dayThu, fri: labels.dayFri, sat: labels.daySat, sun: labels.daySun,
    };
    return map[d] || d;
  }

  const activeDays = $derived(rule?.recurrence_days || []);
  const inactiveDays = $derived(ALL_DAYS.filter(d => !activeDays.includes(d)));

  function handleClose() {
    onclose();
  }
</script>

<Drawer bind:open title={rule?.name || labels.detailRule} onclose={handleClose}>
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
            <Badge variant={rule.is_active ? 'success' : 'muted'} size="sm">{rule.is_active ? labels.aktif : labels.nonaktif}</Badge>
            <Badge variant="default" size="sm">{typeLabel(rule.pricing_type)}</Badge>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-2.5">
        <div class="bg-surface-default rounded-xl p-3 border border-border/40">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1.5 flex items-center gap-1">
            <DollarSign size={10} /> {labels.nilai}
          </p>
          <p class="text-base font-bold text-primary-light tabular-nums leading-tight">{valueLabel(rule)}</p>
          <p class="text-[11px] text-text-muted mt-0.5">{valueSubLabel(rule)}</p>
        </div>
        <div class="bg-surface-default rounded-xl p-3 border border-border/40">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1.5 flex items-center gap-1">
            <Target size={10} /> {labels.target}
          </p>
          <p class="text-sm font-semibold text-text-primary leading-tight truncate">{targetScope(rule)}</p>
          <p class="text-[11px] text-text-muted mt-0.5 truncate">{targetLabel(rule)}</p>
        </div>
        <div class="bg-surface-default rounded-xl p-3 border border-border/40">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1.5 flex items-center gap-1">
            <Hash size={10} /> {labels.kuantitas}
          </p>
          <p class="text-sm font-semibold text-text-primary tabular-nums leading-tight">
            {rule.minimum_quantity}{#if rule.maximum_quantity} – {rule.maximum_quantity}{:else}+{/if}
          </p>
          <p class="text-[11px] text-text-muted mt-0.5">{labels.prioritas} {rule.priority}</p>
        </div>
        <div class="bg-surface-default rounded-xl p-3 border border-border/40">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1.5 flex items-center gap-1">
            <Clock size={10} /> {labels.diperbarui}
          </p>
          <p class="text-sm font-semibold text-text-primary leading-tight">{timeAgo(rule.updated_at)}</p>
          <p class="text-[11px] text-text-muted mt-0.5 font-mono">{formatDateTime(rule.updated_at)}</p>
        </div>
      </div>

      <div class="bg-surface-default rounded-xl border border-border/40 divide-y divide-border/30">
        <div class="px-3.5 py-3">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2.5 flex items-center gap-1.5">
            <DollarSign size={10} /> {labels.harga}
          </p>
          <div class="space-y-2 text-sm">
            <div class="flex items-center justify-between">
              <span class="text-text-muted">{labels.metode}</span>
              <span class="text-text-secondary font-medium">{methodLabel(rule.pricing_method)}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-text-muted">{labels.gabungkanRule}</span>
              <span class="text-text-secondary font-medium">{rule.allow_combine ? labels.yes : labels.no}</span>
            </div>
          </div>
        </div>

        <div class="px-3.5 py-3">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2.5 flex items-center gap-1.5">
            <CalendarDays size={10} /> {labels.jadwal}
          </p>
          <div class="space-y-3 text-sm">
            {#if activeDays.length > 0}
              <div>
                <span class="text-text-muted text-xs block mb-1.5">{labels.hariAktif}</span>
                <div class="flex flex-wrap gap-1">
                  {#each ALL_DAYS as day}
                    {@const isActive = activeDays.includes(day)}
                    <span class="inline-flex items-center justify-center min-w-7 h-7 px-1 rounded-lg text-[11px] font-semibold transition-colors
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
                <span class="text-text-muted">{labels.jamAktif}</span>
                <span class="text-text-secondary font-medium font-mono text-xs">{rule.time_from || '00:00'} – {rule.time_to || '23:59'}</span>
              </div>
            {/if}
            <div class="flex items-center justify-between">
              <span class="text-text-muted">{labels.berlakuDari}</span>
              <span class="text-text-secondary font-medium text-xs">{formatDateTime(rule.effective_from)}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-text-muted">{labels.berlakuSampai}</span>
              <span class="text-text-secondary font-medium text-xs">{formatDateTime(rule.effective_until)}</span>
            </div>
          </div>
        </div>

        <div class="px-3.5 py-3">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2.5 flex items-center gap-1.5">
            <Settings size={10} /> {labels.kondisi}
          </p>
          <div class="space-y-2 text-sm">
            {#if rule.customer_group_id}
              <div class="flex items-center justify-between">
                <span class="text-text-muted">{labels.groupPelanggan}</span>
                <span class="text-text-secondary font-medium">{customerGroups.find(cg => cg.id === rule.customer_group_id)?.name || `#${rule.customer_group_id}`}</span>
              </div>
            {/if}
            {#if rule.store_id}
              <div class="flex items-center justify-between">
                <span class="text-text-muted">{labels.outlet}</span>
                <span class="text-text-secondary font-medium">{stores.find(s => s.id === rule.store_id)?.name || `#${rule.store_id}`}</span>
              </div>
            {/if}
            {#if !rule.customer_group_id && !rule.store_id}
              <p class="text-xs text-text-muted italic">{labels.appliesToAllGroupsAndStores}</p>
            {/if}
          </div>
        </div>

        <div class="px-3.5 py-3">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-2.5 flex items-center gap-1.5">
            <Clock size={10} /> {labels.riwayat}
          </p>
          <div class="space-y-2 text-sm">
            <div class="flex items-center justify-between">
              <span class="text-text-muted">{labels.dibuat}</span>
              <span class="text-text-secondary font-medium text-xs">{formatDateTime(rule.created_at)}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-text-muted">{labels.idRule}</span>
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
        {labels.edit}
      </Button>
    {/if}
    {#if canDelete}
      <Button variant="danger" onclick={() => ondelete(rule!)}>
        <Trash2 class="w-4 h-4 mr-2" />
        {labels.hapus}
      </Button>
    {/if}
  {/snippet}
</Drawer>
