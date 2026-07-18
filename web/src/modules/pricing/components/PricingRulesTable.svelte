<script lang="ts">
  import { Button, Skeleton, SortableHeader, Tooltip } from '$shared/ui';
  import { Pencil, Trash2, DollarSign, Copy, CheckSquare, Square, Power, PowerOff } from 'lucide-svelte';
  import type { PricingRule, PricingType, PricingMethod } from '../types';

  let {
    rules = [],
    loading = false,
    searchQuery = '',
    sortBy = 'name',
    sortDir = 'asc',
    canEdit = false,
    canDelete = false,
    canCreate = false,
    pricingTypes = [],
    pricingMethods = [],
    onsort = (_col: string) => {},
    onedit = (_rule: PricingRule) => {},
    ondelete = (_rule: PricingRule) => {},
    onduplicate = (_rule: PricingRule) => {},
    onbulkactivate = (_ids: number[]) => {},
    onbulkdeactivate = (_ids: number[]) => {},
    onbulkdelete = (_ids: number[]) => {},
    oncreate = () => {},
    targetNames = new Map<string, string>(),
  }: {
    rules: PricingRule[];
    loading: boolean;
    searchQuery: string;
    sortBy: string;
    sortDir: 'asc' | 'desc';
    canEdit: boolean;
    canDelete: boolean;
    canCreate: boolean;
    pricingTypes: { value: string; label: string }[];
    pricingMethods: { value: string; label: string }[];
    onsort?: (col: string) => void;
    onedit?: (rule: PricingRule) => void;
    ondelete?: (rule: PricingRule) => void;
    onduplicate?: (rule: PricingRule) => void;
    onbulkactivate?: (ids: number[]) => void;
    onbulkdeactivate?: (ids: number[]) => void;
    onbulkdelete?: (ids: number[]) => void;
    oncreate?: () => void;
    targetNames?: Map<string, string>;
  } = $props();

  let selectedIds = $state<Set<number>>(new Set());

  let allSelected = $derived(rules.length > 0 && rules.every(r => selectedIds.has(r.id)));
  let someSelected = $derived(rules.some(r => selectedIds.has(r.id)) && !allSelected);
  let selectedCount = $derived(selectedIds.size);

  function toggleSelect(id: number) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selectedIds = next;
  }

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(rules.map(r => r.id));
    }
  }

  function clearSelection() {
    selectedIds = new Set();
  }

  function targetLabel(rule: PricingRule): string {
    if (rule.product_id) {
      const name = targetNames.get(`product:${rule.product_id}`);
      return name || `Product #${rule.product_id}`;
    }
    if (rule.category_id) {
      const name = targetNames.get(`category:${rule.category_id}`);
      return name || `Category #${rule.category_id}`;
    }
    if (rule.brand_id) {
      const name = targetNames.get(`brand:${rule.brand_id}`);
      return name || `Brand #${rule.brand_id}`;
    }
    return '-';
  }

  function methodLabel(method: string): string {
    return pricingMethods.find(m => m.value === method)?.label || method;
  }

  function formatPrice(v: number): string {
    return v?.toLocaleString('id-ID') || '0';
  }

  function valueLabel(rule: PricingRule): string {
    switch (rule.pricing_method) {
      case 'fixed_price': return formatPrice(rule.pricing_value);
      case 'discount_percent': return `${rule.pricing_value}%`;
      case 'discount_amount': return `-${formatPrice(rule.pricing_value)}`;
      case 'markup_percent': return `+${rule.pricing_value}%`;
      default: return String(rule.pricing_value);
    }
  }

  function timeAgo(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    const now = Date.now();
    const then = new Date(dateStr).getTime();
    const diff = now - then;
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return 'Baru saja';
    if (mins < 60) return `${mins} menit lalu`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs} jam lalu`;
    const days = Math.floor(hrs / 24);
    if (days < 30) return `${days} hari lalu`;
    const months = Math.floor(days / 30);
    return `${months} bln lalu`;
  }

  function formatDateTime(dateStr: string | undefined): string {
    if (!dateStr) return '';
    try {
      return new Date(dateStr).toLocaleString('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch { return dateStr; }
  }
</script>

{#if loading}
  <div class="overflow-x-auto">
  <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label="Loading pricing rules">
    <colgroup>
      <col style="width: 3%;" />
      <col style="width: 17%;" />
      <col style="width: 10%;" />
      <col style="width: 7%;" />
      <col style="width: 10%;" />
      <col style="width: 10%;" />
      <col style="width: 6%;" />
      <col style="width: 6%;" />
      <col style="width: 7%;" />
      <col style="width: 9%;" />
      <col style="width: 7%;" />
    </colgroup>
    <thead><tr><th></th><th>Nama</th><th>Target</th><th>Tipe</th><th>Metode</th><th class="text-right">Nilai</th><th>Min Qty</th><th>Prioritas</th><th>Status</th><th>Diperbarui</th><th>Aksi</th></tr></thead>
    <tbody>{#each Array(5) as _}<tr>{#each Array(11) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
  </table>
  </div>
{:else if rules.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-text-muted" role="status">
    <DollarSign class="w-12 h-12 mb-3" aria-hidden="true" />
    <p class="text-sm">Belum ada pricing rules</p>
    {#if canCreate}
      <p class="text-xs text-text-muted mt-1">Klik "Tambah Rule" untuk membuat aturan harga pertama.</p>
    {/if}
  </div>
{:else}
  <div class="overflow-x-auto">
  <table class="w-full min-w-[700px] lg:min-w-[1100px]" style="table-layout: fixed;" aria-label="Pricing rules">
    <colgroup>
      <col style="width: 3%;" />
      <col style="width: 16%;" />
      <col class="hidden lg:table-column" style="width: 10%;" />
      <col class="hidden lg:table-column" style="width: 7%;" />
      <col style="width: 10%;" />
      <col style="width: 10%;" />
      <col class="hidden lg:table-column" style="width: 6%;" />
      <col class="hidden lg:table-column" style="width: 6%;" />
      <col style="width: 7%;" />
      <col class="hidden lg:table-column" style="width: 9%;" />
      <col style="width: 7%;" />
    </colgroup>
    <thead class="bg-muted/50 sticky top-0 z-10">
      <tr class="border-b text-left text-sm text-text-muted">
        <th class="px-3 py-3">
          <button type="button" onclick={toggleSelectAll} class="text-text-muted hover:text-text-primary transition-colors" aria-label={allSelected ? 'Batalkan semua pilihan' : 'Pilih semua'}>
            {#if allSelected}
              <CheckSquare size={16} class="text-primary-light" />
            {:else if someSelected}
              <span class="relative flex items-center justify-center w-4 h-4">
                <Square size={16} class="text-text-muted" />
                <span class="absolute inset-0 flex items-center justify-center"><span class="w-2 h-0.5 bg-primary-light rounded"></span></span>
              </span>
            {:else}
              <Square size={16} />
            {/if}
          </button>
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label="NAMA" column="name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold hidden lg:table-cell" scope="col">TARGET</th>
        <th class="px-4 py-3 font-semibold hidden lg:table-cell">
          <SortableHeader label="TIPE" column="pricing_type" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label="METODE" column="pricing_method" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold text-right" scope="col">NILAI</th>
        <th class="px-4 py-3 font-semibold text-right hidden lg:table-cell">
          <SortableHeader label="MIN QTY" column="minimum_quantity" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
        </th>
        <th class="px-4 py-3 font-semibold text-right hidden lg:table-cell">
          <SortableHeader label="PRIORITAS" column="priority" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label="STATUS" column="is_active" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold hidden lg:table-cell">
          <SortableHeader label="DIPERBARUI" column="updated_at" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold text-center" scope="col">AKSI</th>
      </tr>
    </thead>
    <tbody>
      {#each rules as rule (rule.id)}
        <tr class="border-b border-border hover:bg-surface-hover/50 transition-colors {selectedIds.has(rule.id) ? 'bg-primary-subtle/10' : ''}">
          <td class="px-3 py-3">
            <button type="button" onclick={() => toggleSelect(rule.id)} class="text-text-muted hover:text-text-primary transition-colors" aria-label={selectedIds.has(rule.id) ? `Batalkan pilihan ${rule.name}` : `Pilih ${rule.name}`}>
              {#if selectedIds.has(rule.id)}
                <CheckSquare size={16} class="text-primary-light" />
              {:else}
                <Square size={16} />
              {/if}
            </button>
          </td>
          <td class="px-4 py-3 font-medium text-sm truncate max-w-0"><Tooltip content={rule.name} delay={400}><span class="truncate block">{rule.name}</span></Tooltip></td>
          <td class="px-4 py-3 text-xs truncate max-w-0 hidden lg:table-cell"><Tooltip content={targetLabel(rule)} delay={400}><span class="truncate block">{targetLabel(rule)}</span></Tooltip></td>
          <td class="px-4 py-3 hidden lg:table-cell"><span class="inline-flex items-center rounded-md bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700 truncate">{pricingTypes.find(t => t.value === rule.pricing_type)?.label || rule.pricing_type}</span></td>
          <td class="px-4 py-3 text-xs text-text-muted">{methodLabel(rule.pricing_method)}</td>
          <td class="px-4 py-3 text-xs text-right font-medium tabular-nums">{valueLabel(rule)}</td>
          <td class="px-4 py-3 text-right text-xs tabular-nums hidden lg:table-cell">{rule.minimum_quantity}{rule.maximum_quantity ? `–${rule.maximum_quantity}` : ''}</td>
          <td class="px-4 py-3 text-right text-xs tabular-nums hidden lg:table-cell">{rule.priority}</td>
          <td class="px-4 py-3">
            {#if rule.is_active}
              <span class="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Aktif</span>
            {:else}
              <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600">Nonaktif</span>
            {/if}
          </td>
          <td class="px-4 py-3 text-xs text-text-muted hidden lg:table-cell"><Tooltip content={formatDateTime(rule.updated_at || rule.created_at)} delay={400}><span class="truncate block">{timeAgo(rule.updated_at || rule.created_at)}</span></Tooltip></td>
          <td class="px-4 py-3">
            <div class="flex items-center justify-center gap-1" role="group" aria-label="Actions for {rule.name}">
              {#if canEdit}
                <Button variant="ghost" size="sm" class="text-text-muted hover:text-primary-light" onclick={() => onedit(rule)} aria-label="Edit {rule.name}"><Pencil class="w-4 h-4" /></Button>
              {/if}
              {#if canCreate}
                <Button variant="ghost" size="sm" class="text-text-muted hover:text-accent-light" onclick={() => onduplicate(rule)} aria-label="Duplikasi {rule.name}"><Copy class="w-4 h-4" /></Button>
              {/if}
              {#if canDelete}
                <Button variant="ghost" size="sm" class="text-text-muted hover:text-danger hover:bg-danger-subtle" onclick={() => ondelete(rule)} aria-label="Delete {rule.name}"><Trash2 class="w-4 h-4" /></Button>
              {/if}
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
  </div>

  {#if selectedCount > 0}
    <div class="flex items-center justify-between px-4 py-3 bg-primary-subtle/15 border-t border-primary-default/15">
      <span class="text-sm text-text-primary font-medium">{selectedCount} rule dipilih</span>
      <div class="flex items-center gap-2">
        {#if canEdit}
          <Button variant="secondary" size="sm" onclick={() => { onbulkactivate([...selectedIds]); clearSelection(); }}>
            <Power size={14} /> Aktifkan
          </Button>
          <Button variant="secondary" size="sm" onclick={() => { onbulkdeactivate([...selectedIds]); clearSelection(); }}>
            <PowerOff size={14} /> Nonaktifkan
          </Button>
        {/if}
        {#if canDelete}
          <Button variant="danger" size="sm" onclick={() => { onbulkdelete([...selectedIds]); clearSelection(); }}>
            <Trash2 size={14} /> Hapus
          </Button>
        {/if}
        <Button variant="ghost" size="sm" onclick={clearSelection}>Batal</Button>
      </div>
    </div>
  {/if}
{/if}
