<script lang="ts">
  import { onMount } from 'svelte';
  import { Pagination, Skeleton } from '$shared/ui';
  import { User, Search } from 'lucide-svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { labels } from '$shared/i18n';
  import { useSalesStore } from '../stores/sales-store.svelte';
  import { getSalesLookup } from '../services/sales-service';
  import type { SaleLookupSummary } from '../types';
  import TransactionFilters from './TransactionFilters.svelte';
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

  let searchQuery = $state('');
  let startDate = $state(getDateNDaysAgoInJakarta(30));
  let endDate = $state(getTodayInJakarta());
  let dateRange = $state('last30d');
  let paymentMethods = $state<string[]>([]);
  let minTotal = $state<number | null>(null);
  let maxTotal = $state<number | null>(null);
  let sortBy = $state('created_at');
  let sortDir = $state<'asc' | 'desc'>('desc');
  let page = $state(0);
  let pageSize = $state(20);

  let loading = $state(false);
  let total = $state(0);
  let data = $state<SaleLookupSummary[]>([]);

  let prevKey = '';

  function buildFilters() {
    return {
      startDate,
      endDate,
      limit: pageSize,
      offset: page * pageSize,
      search: searchQuery,
      sortBy,
      sortDir,
      paymentMethods,
      minTotal: minTotal ?? undefined,
      maxTotal: maxTotal ?? undefined,
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

  $effect(() => {
    const key = JSON.stringify(buildFilters());
    if (key === prevKey) return;
    prevKey = key;
    load();
  });

  function handlePageChange(newOffset: number, newLimit: number) {
    page = Math.floor(newOffset / newLimit);
    pageSize = newLimit;
  }

  function toggleSort(col: string) {
    if (sortBy === col) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = col;
      sortDir = 'asc';
    }
    page = 0;
  }

  function formatDateTime(dateStr: string) {
    return formatDateTimeInJakarta(dateStr);
  }

  onMount(() => {
    store.loadPaymentMethods();
  });
</script>

<div class="space-y-5">
  <TransactionFilters
    bind:searchQuery
    bind:startDate
    bind:endDate
    bind:selectedPaymentMethods={paymentMethods}
    bind:sliderMin={minTotal}
    bind:sliderMax={maxTotal}
    bind:selectedDateRange={dateRange}
    paymentMethodOptions={store.paymentMethodOptions}
    showExport={false}
    showPaymentMethods={false}
    showAmountRange={false}
    searchPlaceholder={labels.searchByInvoiceNumber}
  />

  <p class="text-xs text-text-muted flex items-center gap-1.5 px-1">
    <Search size={13} class="shrink-0" />
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
    {:else if data.length === 0}
      <div class="px-4 py-12 text-center" role="status">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <User size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">{labels.noTransactionsFound}</p>
        <p class="text-text-muted text-sm mt-1">{labels.tryAdjustingSearchOrDateRange}</p>
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
                  <span class="text-sm font-medium text-text-primary">{sale.invoice_number}</span>
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

      <div class="p-4 bg-surface-subtle/30 border-t border-border/50">
        <Pagination
          {total}
          limit={pageSize}
          offset={page * pageSize}
          onPageChange={handlePageChange}
        />
      </div>
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
