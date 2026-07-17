<script lang="ts">
  import { Button, Skeleton, SortableHeader } from '$shared/ui';
  import { Pencil, Trash2, DollarSign } from 'lucide-svelte';
  import type { PricingRule, PricingType, PricingMethod } from '../types';

  let {
    rules = [],
    loading = false,
    searchQuery = '',
    sortBy = 'name',
    sortDir = 'asc',
    canEdit = false,
    canDelete = false,
    pricingTypes = [],
    pricingMethods = [],
    onsort = (_col: string) => {},
    onedit = (_rule: PricingRule) => {},
    ondelete = (_rule: PricingRule) => {},
  }: {
    rules: PricingRule[];
    loading: boolean;
    searchQuery: string;
    sortBy: string;
    sortDir: 'asc' | 'desc';
    canEdit: boolean;
    canDelete: boolean;
    pricingTypes: { value: string; label: string }[];
    pricingMethods: { value: string; label: string }[];
    onsort?: (col: string) => void;
    onedit?: (rule: PricingRule) => void;
    ondelete?: (rule: PricingRule) => void;
  } = $props();

  function targetLabel(rule: PricingRule): string {
    if (rule.product_id) return `Product #${rule.product_id}`;
    if (rule.category_id) return `Category #${rule.category_id}`;
    if (rule.brand_id) return `Brand #${rule.brand_id}`;
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
      case 'fixed_price': return `Rp ${formatPrice(rule.pricing_value)}`;
      case 'discount_percent': return `${rule.pricing_value}%`;
      case 'discount_amount': return `-Rp ${formatPrice(rule.pricing_value)}`;
      case 'markup_percent': return `+${rule.pricing_value}%`;
      default: return String(rule.pricing_value);
    }
  }
</script>

{#if loading}
  <div class="overflow-x-auto">
  <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label="Loading pricing rules">
    <colgroup>
      <col style="width: 20%;" />
      <col style="width: 10%;" />
      <col style="width: 12%;" />
      <col style="width: 15%;" />
      <col style="width: 10%;" />
      <col style="width: 8%;" />
      <col style="width: 8%;" />
      <col style="width: 17%;" />
    </colgroup>
    <thead><tr><th>Nama</th><th>Tipe</th><th>Target</th><th>Metode & Nilai</th><th>Min Qty</th><th>Prioritas</th><th>Status</th><th>Aksi</th></tr></thead>
    <tbody>{#each Array(5) as _}<tr>{#each Array(8) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
  </table>
  </div>
{:else if rules.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-gray-400" role="status">
    <DollarSign class="w-12 h-12 mb-3" aria-hidden="true" />
    <p>Belum ada pricing rules</p>
  </div>
{:else}
  <div class="overflow-x-auto">
  <table class="w-full min-w-[900px]" style="table-layout: fixed;" role="grid" aria-label="Pricing rules">
    <colgroup>
      <col style="width: 20%;" />
      <col style="width: 10%;" />
      <col style="width: 12%;" />
      <col style="width: 15%;" />
      <col style="width: 10%;" />
      <col style="width: 8%;" />
      <col style="width: 8%;" />
      <col style="width: 17%;" />
    </colgroup>
    <thead class="bg-muted/50">
      <tr class="border-b text-left text-sm text-text-muted">
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label="NAMA" column="name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label="TIPE" column="pricing_type" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold" scope="col">TARGET</th>
        <th class="px-4 py-3 font-semibold" scope="col">METODE & NILAI</th>
        <th class="px-4 py-3 font-semibold text-right">
          <SortableHeader label="MIN QTY" column="minimum_quantity" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
        </th>
        <th class="px-4 py-3 font-semibold text-right">
          <SortableHeader label="PRIORITAS" column="priority" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
        </th>
        <th class="px-4 py-3 font-semibold">
          <SortableHeader label="STATUS" column="is_active" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="px-4 py-3 font-semibold text-right" scope="col">AKSI</th>
      </tr>
    </thead>
    <tbody>
      {#each rules as rule (rule.id)}
        <tr class="border-b border-border hover:bg-surface-hover/50 transition-colors">
          <td class="px-4 py-3 font-medium truncate">{rule.name}</td>
          <td class="px-4 py-3"><span class="inline-flex items-center rounded-md bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700 truncate">{pricingTypes.find(t => t.value === rule.pricing_type)?.label || rule.pricing_type}</span></td>
          <td class="px-4 py-3 text-xs truncate">{targetLabel(rule)}</td>
          <td class="px-4 py-3 text-xs">
            <span class="text-text-muted">{methodLabel(rule.pricing_method)}</span>
            <span class="font-medium tabular-nums">{valueLabel(rule)}</span>
          </td>
          <td class="px-4 py-3 text-right tabular-nums">{rule.minimum_quantity}{rule.maximum_quantity ? `-${rule.maximum_quantity}` : ''}</td>
          <td class="px-4 py-3 text-right tabular-nums">{rule.priority}</td>
          <td class="px-4 py-3">
            {#if rule.is_active}
              <span class="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Aktif</span>
            {:else}
              <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600">Nonaktif</span>
            {/if}
          </td>
          <td class="px-4 py-3 text-right">
            <div class="flex items-center justify-end gap-1" role="group" aria-label="Actions for {rule.name}">
              {#if canEdit}
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light" onclick={() => onedit(rule)} aria-label="Edit {rule.name}"><Pencil class="w-4 h-4" /></Button>
              {/if}
              {#if canDelete}
                <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger hover:bg-danger-subtle" onclick={() => ondelete(rule)} aria-label="Delete {rule.name}"><Trash2 class="w-4 h-4" /></Button>
              {/if}
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
  </div>
{/if}
