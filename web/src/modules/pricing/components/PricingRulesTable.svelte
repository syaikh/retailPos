<script lang="ts">
  import { Button, Skeleton, SortableHeader, Tooltip, Badge, Dropdown } from '$shared/ui';
  import { Pencil, Trash2, Copy, Send, Check, X, Power, PowerOff, MoreVertical, DollarSign } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';
  import type { PricingRule } from '../types';

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
    onrowclick = (_rule: PricingRule) => {},
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
    onrowclick?: (rule: PricingRule) => void;
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
      return name || t('productNumber', { value: rule.product_id });
    }
    if (rule.category_id) {
      const name = targetNames.get(`category:${rule.category_id}`);
      return name || t('categoryNumber', { value: rule.category_id });
    }
    if (rule.brand_id) {
      const name = targetNames.get(`brand:${rule.brand_id}`);
      return name || t('brandNumber', { value: rule.brand_id });
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
      case 'approved': return labels.statusApproved;
      case 'pending': return labels.statusPending;
      case 'rejected': return labels.statusRejected;
      default: return labels.statusDraft;
    }
  }

  function handleRowKeydown(e: KeyboardEvent, rule: PricingRule) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onrowclick(rule);
    }
  }
</script>

{#if loading}
  <div class="overflow-x-auto">
  <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label={labels.loadingRules}>
    <colgroup>
      <col style="width: 3%;" />
      <col style="width: 30%;" />
      <col style="width: 12%;" />
      <col style="width: 14%;" />
      <col style="width: 22%;" />
      <col style="width: 10%;" />
      <col style="width: 9%;" />
    </colgroup>
    <thead><tr><th></th><th>{labels.nama}</th><th>{labels.nilai}</th><th>{labels.metode}</th><th>{labels.target}</th><th>{labels.status}</th><th>{labels.actions}</th></tr></thead>
    <tbody>{#each Array(5) as _}<tr>{#each Array(7) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
  </table>
  </div>
  {:else if rules.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-text-muted" role="status">
    <DollarSign class="w-12 h-12 mb-3" aria-hidden="true" />
    <p class="text-sm">{labels.belumAdaRules}</p>
    {#if canCreate}
      <p class="text-xs text-text-muted mt-1">{labels.emptyStateCreateRule}</p>
    {/if}
  </div>
  {:else}
  <div class="overflow-x-auto">
  <table class="w-full" style="table-layout: fixed;" aria-label={labels.pricingRules}>
    <colgroup>
      <col style="width: 3%;" />
      <col style="width: 30%;" />
      <col style="width: 12%;" />
      <col style="width: 14%;" />
      <col style="width: 22%;" />
      <col style="width: 10%;" />
      <col style="width: 9%;" />
    </colgroup>
    <thead class="bg-muted/50 sticky top-0 z-10">
      <tr class="border-b text-left text-xs text-text-muted">
        <th class="px-3 py-3">
          <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={allSelected} bind:indeterminate={someSelected} onchange={toggleSelectAll} aria-label={labels.pilihSemua} />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label={labels.nama} column="name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold text-right" scope="col">{labels.nilai}</th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label={labels.metode} column="pricing_method" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold" scope="col">{labels.target}</th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label={labels.status} column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold text-center" scope="col">{labels.actions}</th>
      </tr>
    </thead>
    <tbody>
      {#each rules as rule (rule.id)}
        <tr
          class="border-b border-border transition-colors hover:bg-muted/50 cursor-pointer {selectedIds.has(rule.id) ? 'bg-muted/30' : ''}"
          onclick={() => onrowclick(rule)}
          onkeydown={(e) => handleRowKeydown(e, rule)}
          role="button"
          tabindex="0"
        >
          <td class="px-3 py-4" onclick={(e) => e.stopPropagation()}>
            <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={selectedIds.has(rule.id)} onchange={() => toggleSelect(rule.id)} aria-label={t('pilihItem', { name: rule.name })} />
          </td>
          <td class="px-4 py-4 font-medium text-sm truncate"><Tooltip content={rule.name} delay={400}><span class="truncate block">{rule.name}</span></Tooltip></td>
          <td class="px-4 py-4 text-sm text-right font-semibold tabular-nums text-primary-light">{valueLabel(rule)}</td>
          <td class="px-4 py-4 text-sm text-text-secondary">{methodLabel(rule.pricing_method)}</td>
          <td class="px-4 py-4 text-sm truncate"><Tooltip content={targetLabel(rule)} delay={400}><span class="truncate block">{targetLabel(rule)}</span></Tooltip></td>
          <td class="px-4 py-4"><Badge variant={approvalVariant(rule.status || 'draft')} size="sm">{approvalLabel(rule.status || 'draft')}</Badge></td>
          <td class="px-4 py-4" onclick={(e) => e.stopPropagation()}>
            <div class="flex items-center justify-center" role="group" aria-label={t('actionsFor', { name: rule.name })}>
              <Dropdown placement="bottom-end" items={[]}>
                {#snippet content({ close })}
                  {#if rule.status === 'draft' && canEdit}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-primary-light hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onsubmitapproval(rule); close(); }}>
                      <Send size={14} /> {labels.submit}
                    </button>
                  {/if}
                  {#if rule.status === 'pending' && canEdit}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-success-light hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onapprove(rule); close(); }}>
                      <Check size={14} /> {labels.approve}
                    </button>
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onreject(rule); close(); }}>
                      <X size={14} /> {labels.reject}
                    </button>
                  {/if}
                  {#if canEdit}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onedit(rule); close(); }}>
                      <Pencil size={14} /> {labels.edit}
                    </button>
                  {/if}
                  {#if canCreate}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem" onclick={() => { onduplicate(rule); close(); }}>
                      <Copy size={14} /> {labels.duplicate}
                    </button>
                  {/if}
                  {#if canDelete}
                    <button type="button" class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger-subtle transition-colors" role="menuitem" onclick={() => { ondelete(rule); close(); }}>
                      <Trash2 size={14} /> {labels.hapus}
                    </button>
                  {/if}
                {/snippet}
                {#snippet trigger({ toggle })}
                  <Button variant="ghost" size="icon" class="text-text-muted hover:text-text-primary" onclick={toggle} aria-label={t('actionsFor', { name: rule.name })}>
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
      <span class="text-sm text-text-primary font-medium">{t('rulesSelected', { count: selectedCount })}</span>
      <div class="flex items-center gap-2">
        {#if canEdit}
          <Button variant="secondary" size="sm" onclick={() => { onbulkactivate([...selectedIds]); clearSelection(); }}>
            <Power size={14} /> {labels.activate}
          </Button>
          <Button variant="secondary" size="sm" onclick={() => { onbulkdeactivate([...selectedIds]); clearSelection(); }}>
            <PowerOff size={14} /> {labels.deactivate}
          </Button>
        {/if}
        {#if canDelete}
          <Button variant="danger" size="sm" onclick={() => { onbulkdelete([...selectedIds]); clearSelection(); }}>
            <Trash2 size={14} /> {labels.hapus}
          </Button>
        {/if}
        <Button variant="ghost" size="sm" onclick={clearSelection}>{labels.cancel}</Button>
      </div>
    </div>
  {/if}
{/if}
