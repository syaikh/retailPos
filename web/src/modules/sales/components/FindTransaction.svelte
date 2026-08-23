<script lang="ts">
  import { onMount } from 'svelte';
  import { SearchBar, Button, Skeleton } from '$shared/ui';
  import { User, Copy, Check } from 'lucide-svelte';
  import { getTodayInJakarta, formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { labels, t } from '$shared/i18n';

  let copiedInvoice = $state<string | null>(null);

  async function copyInvoice(invoice: string) {
    try {
      await navigator.clipboard.writeText(invoice);
      copiedInvoice = invoice;
      setTimeout(() => { copiedInvoice = null; }, 1500);
    } catch {
      // Clipboard unavailable (non-secure context) — leave feedback unset.
    }
  }
  import { useSalesStore } from '../stores/sales-store.svelte';
  import { getSalesLookup } from '../services/sales-service';
  import type { SaleLookupSummary } from '../types';
  import TransactionDrawer from './TransactionDrawer.svelte';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';

  const store = useSalesStore();
  const rbac = useRBAC();

  let selectedLookupId = $state<number | null>(null);
  let showLookupDrawer = $state(false);

  function openLookupDetail(id: number) {
    selectedLookupId = id;
    showLookupDrawer = true;
  }

  function handleLookupClose() {
    showLookupDrawer = false;
    selectedLookupId = null;
  }

  // Search-only: the cashier enters a receipt number to pull a single
  // cross-cashier (redacted) transaction. No default store-wide table is
  // loaded — a receipt is the proof of purchase, so lookup is intentional.
  let searchQuery = $state('');
  let sortBy = $state('created_at');
  let sortDir = $state<'asc' | 'desc'>('desc');
  let page = $state(0);
  let pageSize = $state(20);

  let loading = $state(false);
  let total = $state(0);
  let data = $state<SaleLookupSummary[]>([]);

  // Invoice lookup is date-independent: when a search term is present we widen
  // the window to the epoch so old receipts can still be found (the receipt
  // number itself is the gate, not recency).
  function buildFilters() {
    const hasSearch = searchQuery.trim().length > 0;
    return {
      startDate: hasSearch ? '2000-01-01' : getTodayInJakarta(),
      endDate: getTodayInJakarta(),
      limit: pageSize,
      offset: page * pageSize,
      search: hasSearch ? searchQuery.trim() : undefined,
      sortBy,
      sortDir,
    };
  }

  async function load() {
    loading = true;
    try {
      const res = await getSalesLookup(buildFilters());
      data = res.data;
      total = res.total;
    } catch {
      data = [];
      total = 0;
    } finally {
      loading = false;
    }
  }

  async function runSearch() {
    if (!searchQuery.trim()) return;
    page = 0;
    await load();
  }

  function toggleSort(col: string) {
    if (sortBy === col) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = col;
      sortDir = 'asc';
    }
    page = 0;
    // Re-query: the auto-load $effect was removed (search-only), so sorting
    // must explicitly re-run the lookup. runSearch guards the empty-query case.
    runSearch();
  }

  function formatDateTime(dateStr: string) {
    return formatDateTimeInJakarta(dateStr);
  }

  onMount(() => {
    store.loadPaymentMethods();
  });
</script>

<div class="space-y-5">
  <div class="flex items-center gap-2">
    <SearchBar
      bind:value={searchQuery}
      placeholder={labels.searchByInvoiceNumber}
      onsubmit={runSearch}
      class="flex-1"
    />
    <Button onclick={runSearch} disabled={!searchQuery.trim()}>{labels.search}</Button>
  </div>

  <p class="text-xs text-text-muted flex items-center gap-1.5 px-1">
    {labels.lookupRedactedNotice}
  </p>

  <div class="card p-0 overflow-hidden">
    {#if loading}
      <div aria-busy="true" aria-label={labels.loadingTransactions}>
        <div class="divide-y divide-border">
        {#each { length: 5 } as _}
          <div class="flex items-center gap-4 px-4 py-3.5">
            <Skeleton width="w-32" height="h-4" />
            <Skeleton width="w-24" height="h-4" />
            <Skeleton width="w-24" height="h-6" rounded="rounded-full" class="ml-auto" />
            <Skeleton width="w-28" height="h-4" />
          </div>
        {/each}
        </div>
      </div>
    {:else if !searchQuery.trim()}
      <div class="px-4 py-12 text-center" role="status">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <User size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">{labels.findTransactionHint}</p>
      </div>
    {:else if data.length === 0}
      <div class="px-4 py-12 text-center" role="status">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <User size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">{t('noResultsFor', { query: searchQuery.trim() })}</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full min-w-[760px]">
          <thead class="bg-muted/50">
            <tr>
              <th class="text-left p-4 font-semibold">
                <button class="flex items-center gap-1 hover:text-text-primary" onclick={() => toggleSort('invoice_number')}>
                  {labels.invoiceLabel}
                </button>
              </th>
              <th class="text-left p-4 font-semibold">
                <button class="flex items-center gap-1 hover:text-text-primary" onclick={() => toggleSort('created_at')}>
                  {labels.dateLabel}
                </button>
              </th>
              <th class="text-left p-4 font-semibold">{labels.cashierLabel}</th>
              <th class="text-right p-4 font-semibold">
                <button class="flex items-center gap-1 hover:text-text-primary ml-auto" onclick={() => toggleSort('total_amount')}>
                  {labels.totalRp}
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            {#each data as sale (sale.id)}
              <tr
                class="border-t border-border cursor-pointer hover:bg-surface/50 transition-colors"
                role="button"
                tabindex="0"
                aria-label={labels.viewTransactionDetail}
                onclick={() => openLookupDetail(sale.id)}
                onkeydown={(e: KeyboardEvent) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    openLookupDetail(sale.id);
                  }
                }}
              >
                <td class="p-4">
                  <span class="flex items-center gap-1.5">
                    <span class="text-sm font-medium text-text-primary">{sale.invoice_number}</span>
                    <button
                      type="button"
                      class="p-0.5 hover:text-primary transition-colors w-5 h-5 flex items-center justify-center shrink-0"
                      title={labels.copyInvoiceNumber}
                      aria-label={labels.copyInvoiceNumber}
                      onclick={(e) => { e.stopPropagation(); copyInvoice(sale.invoice_number); }}
                    >
                      {#if copiedInvoice === sale.invoice_number}
                        <Check size={13} class="text-primary" />
                      {:else}
                        <Copy size={13} class="text-text-muted hover:text-primary" />
                      {/if}
                    </button>
                  </span>
                </td>
                <td class="p-4 text-sm text-text-secondary">
                  {formatDateTime(sale.created_at)}
                </td>
                <td class="p-4 text-sm text-text-secondary">
                  {sale.cashier_name || labels.walkInGeneral}
                </td>
                <td class="p-4 text-right text-sm font-semibold text-text-primary">
                  {(sale.total_amount || 0).toLocaleString('id-ID')}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      {#if total > pageSize}
        <div class="p-4 bg-surface-subtle/30 border-t border-border/50 text-center text-sm text-text-muted">
          {labels.refineSearch}
        </div>
      {/if}
    {/if}
  </div>

  <TransactionDrawer
    selectedTransaction={selectedLookupId ? { id: selectedLookupId } : null}
    bind:showTransactionDrawer={showLookupDrawer}
    mode="lookup"
    canReprint={rbac.can(Permissions.sale.receiptPrint)}
    onclose={handleLookupClose}
  />
</div>
