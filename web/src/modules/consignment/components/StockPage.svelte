<script lang="ts">
  import { onMount } from 'svelte';
  import { EmptyState, Badge } from '$shared/ui';
  import { Package } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';
  import { listStock } from '../services/consignment-service';
  import type { Arrangement, StockRow } from '../types';

  let {
    arrangement,
  }: {
    arrangement: Arrangement;
  } = $props();

  let rows = $state<StockRow[]>([]);
  let loading = $state(true);

  async function load() {
    loading = true;
    try {
      rows = await listStock(arrangement.supplier_id);
    } catch {
      rows = [];
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    load();
  });
</script>

<div class="card">
  <div class="px-4 py-3 border-b border-border/50">
    <h2 class="font-semibold text-text-primary">{t('consignmentStockFor', { name: arrangement.supplier_name || '' })}</h2>
  </div>

  {#if loading}
    <div class="p-8 text-center text-sm text-text-secondary">{labels.loading}</div>
  {:else if rows.length === 0}
    <EmptyState
      icon={Package}
      title={labels.consignmentNoStock}
      subtitle={labels.consignmentNoStockSubtitle}
    />
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50">
            <th class="px-4 py-3">{labels.consignmentProduct}</th>
            <th class="px-4 py-3 text-right">{labels.consignmentAvailableStock}</th>
            <th class="px-4 py-3 text-right">{labels.consignmentPendingReturnQty}</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as r}
            <tr class="border-b border-border/40">
              <td class="px-4 py-3">
                <div class="font-medium text-text-primary">{r.product_name}</div>
                <div class="text-xs text-text-secondary">{r.product_sku}</div>
              </td>
              <td class="px-4 py-3 text-right">
                <Badge variant={r.available_qty > 0 ? 'success' : 'muted'}>{r.available_qty}</Badge>
              </td>
              <td class="px-4 py-3 text-right text-text-secondary">{r.pending_return_qty}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>