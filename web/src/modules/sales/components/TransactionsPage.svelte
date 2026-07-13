<script lang="ts">
  import { onMount } from 'svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta } from '$shared/utils/jakartaTime';
  import { useSalesStore } from '../stores/sales-store.svelte';
  import { createQueryManager } from '../lib/query-manager';
  import TransactionFilters from './TransactionFilters.svelte';
  import TransactionTable from './TransactionTable.svelte';
  import TransactionDrawer from './TransactionDrawer.svelte';

  const store = useSalesStore();

  store.startDate = getDateNDaysAgoInJakarta(30);
  store.endDate = getTodayInJakarta();
  store.dateRange = 'last30d';

  let showDatePicker = $state(false);
  let showTransactionDrawer = $state(false);
  let selectedTransaction = $state(null);

  let prevFilters = '';

  const qm = createQueryManager({
    getFilters: () => store.currentFilters,
    fetch: async (filters, signal) => {
      await store.load(filters, signal);
    },
  });

  $effect(() => {
    const current = {
      searchQuery: store.searchQuery,
      paymentMethods: store.paymentMethods,
      minTotal: store.minTotal,
      maxTotal: store.maxTotal,
      dateRange: store.dateRange,
      startDate: store.startDate,
      endDate: store.endDate,
      page: store.page,
      pageSize: store.pageSize,
      sortBy: store.sortBy,
      sortDir: store.sortDir,
    };

    const json = JSON.stringify(current);
    if (json === prevFilters) return;

    const changed = new Set<string>();
    if (prevFilters) {
      try {
        const prev = JSON.parse(prevFilters);
        for (const key of Object.keys(current)) {
          if (JSON.stringify((current as Record<string, unknown>)[key]) !== JSON.stringify((prev as Record<string, unknown>)[key])) {
            changed.add(key);
          }
        }
      } catch {
        Object.keys(current).forEach(k => changed.add(k));
      }
    } else {
      Object.keys(current).forEach(k => changed.add(k));
    }

    const paginationKeys = new Set(['page', 'pageSize', 'sortBy', 'sortDir']);
    const hasFilterChange = [...changed].some(k => !paginationKeys.has(k));
    if (hasFilterChange && prevFilters) {
      store.page = 0;
    }

    prevFilters = json;
    qm.notify(store.currentFilters, changed);
  });

  function handlePageChange(newOffset: number, newLimit: number) {
    store.page = Math.floor(newOffset / newLimit);
    store.pageSize = newLimit;
  }

  function toggleSort(column: string) {
    if (store.sortBy === column) {
      store.sortDir = store.sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      store.sortBy = column;
      store.sortDir = 'asc';
    }
    store.page = 0;
  }

  function openTransactionDetails(transaction: any) {
    selectedTransaction = transaction;
    showTransactionDrawer = true;
  }

  function closeTransactionDrawer() {
    showTransactionDrawer = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      showTransactionDrawer = false;
      showDatePicker = false;
    }
  }

  onMount(async () => {
    await store.loadPaymentMethods();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-5">
  <TransactionFilters
    bind:searchQuery={store.searchQuery}
    bind:startDate={store.startDate}
    bind:endDate={store.endDate}
    bind:selectedPaymentMethods={store.paymentMethods}
    bind:showDatePicker
    bind:sliderMin={store.minTotal}
    bind:sliderMax={store.maxTotal}
    bind:selectedDateRange={store.dateRange}
    paymentMethodOptions={store.paymentMethodOptions}
  />

  <TransactionTable
    salesData={store.salesData}
    loading={store.loading}
    total={store.total}
    limit={store.pageSize}
    offset={store.page * store.pageSize}
    bind:sortBy={store.sortBy}
    bind:sortDir={store.sortDir}
    ontogglesort={toggleSort}
    onpagechange={handlePageChange}
    onrowclick={openTransactionDetails}
  />

  <TransactionDrawer
    {selectedTransaction}
    bind:showTransactionDrawer
    onclose={closeTransactionDrawer}
  />
</div>

<style>
  :global(input[type="date"]::-webkit-calendar-picker-indicator) {
    filter: invert(1);
    cursor: pointer;
  }
</style>
