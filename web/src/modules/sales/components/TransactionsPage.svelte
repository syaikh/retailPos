<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch } from '$shared/api/http-client';
  import { toast } from '$shared/stores/toast.svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta } from '$shared/utils/jakartaTime';
  import { debounce } from '$shared/utils/debounce';
  import TransactionFilters from './TransactionFilters.svelte';
  import TransactionTable from './TransactionTable.svelte';
  import TransactionDrawer from './TransactionDrawer.svelte';

  let loading = $state(true);
  let salesData = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let isSearching = $state(false);

  let startDate = $state(getDateNDaysAgoInJakarta(30));
  let endDate = $state(getTodayInJakarta());

  let showTransactionDrawer = $state(false);
  let selectedTransaction = $state(null);

  let paymentMethodOptions: { code: string; name: string }[] = $state([]);
  let selectedPaymentMethods: string[] = $state([]);
  let showPaymentDropdown = $state(false);
  let showExportDropdown = $state(false);

  const SLIDER_MAX_BOUND = 50000000;

  let sliderMin = $state<number | null>(null);
  let sliderMax = $state<number | null>(null);

  let appliedPaymentMethods: string[] = $state([]);
  let appliedSliderMin = $state<number | null>(null);
  let appliedSliderMax = $state<number | null>(null);

  let sortBy = $state('created_at');
  let sortDir = $state('DESC');

  let showDatePicker = $state(false);
  let selectedDateRange = $state('last30d');

  function sanitizeSearch(q: string): string {
    let s = q.trim();
    if (/^INV-/i.test(s)) s = s.slice(4).trim();
    return s;
  }

  const doSearch = debounce(() => {
    offset = 0;
    isSearching = true;
    fetchSales(true);
  }, 300);

  async function fetchSales(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const params = new URLSearchParams({
        startDate,
        endDate,
        limit: limit.toString(),
        offset: offset.toString(),
        search: sanitizeSearch(searchQuery),
        sortBy,
        sortDir,
      });
      if (appliedPaymentMethods.length > 0) {
        params.set('paymentMethods', appliedPaymentMethods.join(','));
      }
      if (appliedSliderMin !== null && appliedSliderMin > 0) {
        params.set('minTotal', appliedSliderMin.toString());
      }
      if (appliedSliderMax !== null && appliedSliderMax < SLIDER_MAX_BOUND) {
        params.set('maxTotal', appliedSliderMax.toString());
      }
      const res = await apiFetch(`/api/sales?${params.toString()}`);
      if (res.ok) {
        const data = await res.json();
        salesData = data.data || [];
        total = data.total || 0;
      }
    } catch {
      toast.error('Failed to load transactions');
    } finally {
      if (!isSearch) loading = false;
      isSearching = false;
    }
  }

  function handlePageChange(newOffset: number, newLimit: number) {
    offset = newOffset;
    limit = newLimit;
    fetchSales(false);
  }

  function toggleSort(column: string) {
    if (sortBy === column) {
      sortDir = sortDir === 'ASC' ? 'DESC' : 'ASC';
    } else {
      sortBy = column;
      sortDir = 'ASC';
    }
    offset = 0;
    fetchSales(false);
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
      showPaymentDropdown = false;
      showExportDropdown = false;
    }
  }

  function handleWindowClick(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (showDatePicker && !target.closest('.date-picker-container') && !target.closest('.date-picker-trigger')) {
      showDatePicker = false;
    }
    if (showPaymentDropdown && !target.closest('.payment-dropdown-container')) {
      showPaymentDropdown = false;
    }
    if (showExportDropdown && !target.closest('.export-dropdown-container')) {
      showExportDropdown = false;
    }
  }

  function handleApplyDateRange() {
    offset = 0;
    fetchSales(false);
  }

  onMount(async () => {
    await fetchSales(false);
    const res = await apiFetch('/api/payment-methods');
    if (res.ok) {
      const data = await res.json();
      paymentMethodOptions = (data.data || data || []).filter((m: any) => m.is_active !== false);
    }
  });
</script>

<svelte:window onkeydown={handleKeydown} onclick={handleWindowClick} />

<div class="space-y-5">
  <TransactionFilters
    bind:searchQuery
    bind:startDate
    bind:endDate
    bind:selectedPaymentMethods
    bind:showDatePicker
    bind:showPaymentDropdown
    bind:showExportDropdown
    bind:sliderMin
    bind:sliderMax
    bind:appliedPaymentMethods
    bind:appliedSliderMin
    bind:appliedSliderMax
    bind:isSearching
    bind:selectedDateRange
    {paymentMethodOptions}
    {loading}
    onsearch={doSearch}
    onapplyfilters={() => { offset = 0; fetchSales(false); }}
    oncancelfilters={() => {}}
    onresetfilters={() => { offset = 0; fetchSales(false); }}
    onapplydaterange={handleApplyDateRange}
  />

  <TransactionTable
    {salesData}
    {loading}
    {total}
    {limit}
    {offset}
    bind:sortBy
    bind:sortDir
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
