<script lang="ts">
  import { Button, Skeleton, SortableHeader, Tooltip, Badge, Dropdown } from '$shared/ui';
  import { Pencil, Trash2, DollarSign, Copy, CheckSquare, Square, Send, Check, X, Power, PowerOff, MoreVertical } from 'lucide-svelte';
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
    showDetailCols = false,
    onsort = (_col: string) => {},
    onedit = (_rule: PricingRule) => {},
    ondelete = (_rule: PricingRule) => {},
    onduplicate = (_rule: PricingRule) => {},
    onbulkactivate = (_ids: number[]) => {},
    onbulkdeactivate = (_ids: number[]) => {},
    onbulkdelete = (_ids: number[]) => {},
    oncreate = () => {},
    onsubmitapproval = (_rule: PricingRule) => {},
    onapprove = (_rule: PricingRule) => {},
    onreject = (_rule: PricingRule) => {},
    onviewaudit = (_rule: PricingRule) => {},
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
    showDetailCols?: boolean;
    onsort?: (col: string) => void;
    onedit?: (rule: PricingRule) => void;
    ondelete?: (rule: PricingRule) => void;
    onduplicate?: (rule: PricingRule) => void;
    onbulkactivate?: (ids: number[]) => void;
    onbulkdeactivate?: (ids: number[]) => void;
    onbulkdelete?: (ids: number[]) => void;
    oncreate?: () => void;
    onsubmitapproval?: (rule: PricingRule) => void;
    onapprove?: (rule: PricingRule) => void;
    onreject?: (rule: PricingRule) => void;
    onviewaudit?: (rule: PricingRule) => void;
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
      case 'approved': return 'Approved';
      case 'pending': return 'Pending';
      case 'rejected': return 'Rejected';
      default: return 'Draft';
    }
  }

  function typeVariant(): 'default' {
    return 'default';
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
      <col style="width: 16%;" />
      <col style="width: 10%;" />
      <col style="width: 10%;" />
      <col style="width: 10%;" />
      <col style="width: 8%;" />
      <col style="width: 7%;" />
      <col style="width: 6%;" />
      <col style="width: 6%;" />
      <col style="width: 8%;" />
      <col style="width: 8%;" />
    </colgroup>
    <thead><tr><th></th><th>Nama</th><th>Nilai</th><th>Metode</th><th>Target</th><th>Tipe</th><th>Approval</th><th>Min Qty</th><th>Prioritas</th><th>Diperbarui</th><th>Aksi</th></tr></thead>
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
      <col style="width: 10%;" />
      <col style="width: 10%;" />
      <col style="width: 10%;" />
      <col style="width: 8%;" />
      <col style="width: 7%;" />
      <col style="width: 6%;" />
      <col style="width: 6%;" />
      <col class="hidden lg:table-column" style="width: 8%;" />
      <col style="width: 8%;" />
    </colgroup>
    <thead class="bg-muted/50 sticky top-0 z-10">
      <tr class="border-b text-left text-xs text-text-muted">
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
        <th class="px-4 py-3 font-semibold text-right" scope="col">NILAI</th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label="METODE" column="pricing_method" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold {showDetailCols ? '' : 'hidden lg:table-cell'}" scope="col">TARGET</th>
        <th class="px-4 py-3 font-semibold {showDetailCols ? '' : 'hidden lg:table-cell'}">
          <SortableHeader label="TIPE" column="pricing_type" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label="APPROVAL" column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold text-right {showDetailCols ? '' : 'hidden lg:table-cell'}">
          <SortableHeader label="MIN QTY" column="minimum_quantity" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
        </th>
        <th class="px-4 py-3 font-semibold text-right {showDetailCols ? '' : 'hidden lg:table-cell'}">
          <SortableHeader label="PRIORITAS" column="priority" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
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
          <td class="px-4 py-4 font-medium text-sm truncate"><Tooltip content={rule.name} delay={400}><span class="truncate block">{rule.name}</span></Tooltip></td>
          <td class="px-4 py-4 text-sm text-right font-semibold tabular-nums text-primary-light">{valueLabel(rule)}</td>
          <td class="px-4 py-4 text-sm text-text-secondary">{methodLabel(rule.pricing_method)}</td>
          <td class="px-4 py-4 text-sm truncate {showDetailCols ? '' : 'hidden lg:table-cell'}"><Tooltip content={targetLabel(rule)} delay={400}><span class="truncate block">{targetLabel(rule)}</span></Tooltip></td>
          <td class="px-4 py-4 {showDetailCols ? '' : 'hidden lg:table-cell'}"><Badge variant={typeVariant()} size="sm">{pricingTypes.find(t => t.value === rule.pricing_type)?.label || rule.pricing_type}</Badge></td>
          <td class="px-4 py-4"><Badge variant={approvalVariant(rule.status || 'draft')} size="sm">{approvalLabel(rule.status || 'draft')}</Badge></td>
          <td class="px-4 py-4 text-right text-sm tabular-nums {showDetailCols ? '' : 'hidden lg:table-cell'}">{rule.minimum_quantity}{rule.maximum_quantity ? `–${rule.maximum_quantity}` : ''}</td>
          <td class="px-4 py-4 text-right text-sm tabular-nums {showDetailCols ? '' : 'hidden lg:table-cell'}">{rule.priority}</td>
          <td class="px-4 py-4 text-sm text-text-muted hidden lg:table-cell"><Tooltip content={formatDateTime(rule.updated_at || rule.created_at)} delay={400}><span class="truncate block">{timeAgo(rule.updated_at || rule.created_at)}</span></Tooltip></td>
          <td class="px-4 py-4">
            <div class="flex items-center justify-center" role="group" aria-label="Actions for {rule.name}">
              <Dropdown placement="bottom-end" items={[]}>
                {#snippet content({ close })}
                  {#if rule.status === 'draft' && canEdit}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-primary-light hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onsubmitapproval(rule); close(); }}>
                      <Send size={14} /> Ajukan
                    </button>
                  {/if}
                  {#if rule.status === 'pending' && canEdit}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-success-light hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onapprove(rule); close(); }}>
                      <Check size={14} /> Approve
                    </button>
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onreject(rule); close(); }}>
                      <X size={14} /> Reject
                    </button>
                  {/if}
                  {#if canEdit}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onedit(rule); close(); }}>
                      <Pencil size={14} /> Edit
                    </button>
                  {/if}
                  {#if canCreate}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onduplicate(rule); close(); }}>
                      <Copy size={14} /> Duplikasi
                    </button>
                  {/if}
                  {#if canDelete}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger-subtle transition-colors" role="menuitem" onclick={() => { ondelete(rule); close(); }}>
                      <Trash2 size={14} /> Hapus
                    </button>
                  {/if}
                  <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onviewaudit(rule); close(); }}>
                    <span class="w-3.5 h-3.5 flex items-center justify-center"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg></span> Audit
                  </button>
                {/snippet}
                {#snippet trigger({ toggle })}
                  <Button variant="ghost" size="icon" class="text-text-muted hover:text-text-primary" onclick={toggle} aria-label="Aksi untuk {rule.name}">
                    <MoreVertical size={16} />
                  </Button>
                {/snippet}
              </Dropdown>
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
