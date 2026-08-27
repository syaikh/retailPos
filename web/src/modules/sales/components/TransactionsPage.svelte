<script lang="ts">
  import { onMount } from 'svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime';
  import { useSalesStore } from '../stores/sales-store.svelte';
  import { createQueryManager } from '../lib/query-manager';
  import { useAuthStore } from '$modules/auth';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { useShiftStore } from '$modules/shifts';
  import { goto, subscribe as subscribeRoute } from '$app/router';
  import { toast } from '$shared/stores/toast.svelte';
  import { labels } from '$shared/i18n';
  import { RefreshCw } from 'lucide-svelte';
  import { useWebSocket } from '$shared/api/websocket';
  import { Button } from '$shared/ui';
  import TransactionFilters from './TransactionFilters.svelte';
  import TransactionTable from './TransactionTable.svelte';
  import TransactionDrawer from './TransactionDrawer.svelte';
  import FindTransaction from './FindTransaction.svelte';
  import { Permissions } from '$shared/constants/permissions';
  import { getSaleById, getSaleLookupDetail } from '../services/sales-service';

  const store = useSalesStore();
  const authStore = useAuthStore();
  const rbac = useRBAC();
  const shiftStore = useShiftStore();

  // @ownership-only — data-scope: cashier hanya melihat transaksi milik sendiri.
  if (rbac.isCashier && authStore.user?.id) {
    store.cashierId = authStore.user.id;
  } else {
    store.cashierId = null;
  }

  store.startDate = getDateNDaysAgoInJakarta(30);
  store.endDate = getTodayInJakarta();
  store.dateRange = 'last30d';

  // Default scope is the cashier's own sales ("My Transactions"). The
  // cross-cashier "Find Transaction" tab is only offered to holders of
  // sale.lookup.
  let activeTab = $state<'mine' | 'lookup'>('mine');
  const canLookup = $derived(rbac.can(Permissions.sale.lookup));
  // report.view holders already see every cashier's sales in "My Transactions",
  // so the tab bar (My Transactions + Find Transaction) is redundant for them.
  const canAccessAll = $derived(rbac.can(Permissions.report.view));

  let showDatePicker = $state(false);
  let showTransactionDrawer = $state(false);
  let selectedTransaction = $state(null);
  // Deep-link mode for notifications ("/transactions?txn=<id>"): 'history' uses
  // the owner-scoped endpoint, 'lookup' the cross-cashier redacted endpoint.
  let drawerMode = $state<'history' | 'lookup'>('history');

  // Refresh + freshness state (Part B).
  let lastUpdated = $state<Date | null>(null);
  let refreshing = $state(false);
  // Manager all-sales websocket banner (Phase 2).
  let newTxnCount = $state(0);
  let newTxnSince = $state<Date | null>(null);

  let prevFilters = '';

  const qm = createQueryManager({
    getFilters: () => store.currentFilters,
    fetch: async (filters, signal) => {
      await store.load(filters, signal);
    },
  });

  $effect(() => {
    // Cashier tanpa shift aktif tidak memuat data transaksi (akan di-redirect ke /shifts).
    if (rbac.isCashier && !shiftStore.activeShift) return;
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

  // Stamp the last successful fetch time (covers filters, pagination, refresh).
  $effect(() => {
    store.salesData;
    lastUpdated = new Date();
  });

  // Manager all-sales view: count store-wide new sales pushed over websocket.
  // Own-scope (cashier) tabs must not count — a sale from another cashier is not
  // in their list, so a banner there would be wrong.
  $effect(() => {
    const ws = useWebSocket();
    const unsub = ws.on('sale_created', () => {
      if (!canAccessAll) return;
      newTxnCount += 1;
      if (!newTxnSince) newTxnSince = new Date();
    });
    return () => unsub();
  });

  function jakartaHHMM(d: Date = new Date()): string {
    const shifted = new Date(d.getTime() + JAKARTA_OFFSET_MS);
    const h = String(shifted.getUTCHours()).padStart(2, '0');
    const m = String(shifted.getUTCMinutes()).padStart(2, '0');
    return `${h}:${m} WIB`;
  }

  function refresh() {
    refreshing = true;
    store.page = 0;
    store.load(store.currentFilters).finally(() => {
      refreshing = false;
      lastUpdated = new Date();
      // The refreshed view already includes any new sales, so clear the banner.
      newTxnCount = 0;
      newTxnSince = null;
    });
  }

  function viewNew() {
    refresh();
    newTxnCount = 0;
    newTxnSince = null;
  }

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

  // Open the detail drawer for a transaction referenced by the ?txn=<id> query
  // param (e.g. arriving from the "new transaction" notification). Prefers a
  // transaction already in the loaded list, then owner-scoped detail, then the
  // cross-cashier lookup detail as a fallback.
  async function openTxnFromQuery() {
    const txn = new URLSearchParams(window.location.search).get('txn');
    const id = Number(txn);
    if (!txn || !Number.isInteger(id) || id <= 0) return;

    let opened = false;
    const existing = store.salesData.find((t: any) => t.id === id);
    if (existing) {
      drawerMode = 'history';
      activeTab = 'mine';
      openTransactionDetails(existing);
      opened = true;
    } else {
      const sale = await getSaleById(id);
      if (sale) {
        drawerMode = 'history';
        activeTab = 'mine';
        openTransactionDetails(sale);
        opened = true;
      } else {
        const lookup = await getSaleLookupDetail(id);
        if (lookup) {
          drawerMode = 'lookup';
          // A cross-cashier (foreign) sale belongs in the Find Transaction
          // context, not "My Transactions". Switch to that tab when the
          // caller holds sale.lookup; otherwise fall back to the default
          // "mine" tab (the drawer simply will not open without the
          // permission, matching prior behaviour).
          if (canLookup) activeTab = 'lookup';
          openTransactionDetails(lookup);
          opened = true;
        }
      }
    }

    // On the manager all-sales view, opening a "new transaction" notification
    // must reconcile the banner + table: viewing the sale marks it as seen, so
    // reload the list (pulling the sale in) and clear the "N new" banner.
    // The cashier own-scope tab has no banner, so it is left untouched.
    if (opened && canAccessAll) {
      refresh();
    }
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
    // @display-only — flow guard navigasi UX: cashier tanpa shift aktif diarahkan ke /shifts.
    await shiftStore.loadActiveShift();
    if (rbac.isCashier && !shiftStore.activeShift) {
      toast.error(labels.toastMustOpenShiftFirst);
      goto('/shifts');
      return;
    }
    await store.loadPaymentMethods();

    // Deep-link from a "new transaction" notification: open its detail drawer.
    await openTxnFromQuery();
  });

  // Re-open the deep-linked transaction whenever the route gains a ?txn=<id>
  // (e.g. clicking the notification while already on this page).
  $effect(() => {
    const unsubscribe = subscribeRoute(() => {
      openTxnFromQuery();
    });
    return () => unsubscribe();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-5">
  {#if !canAccessAll}
    <div class="flex items-center gap-1 border-b border-border">
      <button
        class="px-4 py-2.5 text-sm font-semibold border-b-2 -mb-px transition-colors {activeTab === 'mine' ? 'border-primary-default text-text-primary' : 'border-transparent text-text-muted hover:text-text-secondary'}"
        onclick={() => (activeTab = 'mine')}
      >
        {labels.myTransactions}
      </button>
      {#if canLookup}
        <button
          class="px-4 py-2.5 text-sm font-semibold border-b-2 -mb-px transition-colors {activeTab === 'lookup' ? 'border-primary-default text-text-primary' : 'border-transparent text-text-muted hover:text-text-secondary'}"
          onclick={() => (activeTab = 'lookup')}
        >
          {labels.findTransaction}
        </button>
      {/if}
    </div>
  {/if}

  {#if activeTab === 'mine'}
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

    {#if canAccessAll && newTxnCount > 0}
      <div class="flex items-center justify-between gap-3 rounded-lg border border-primary/30 bg-primary/5 px-4 py-2.5 text-sm">
        <span class="text-text-secondary">
          <span class="font-semibold text-primary">{newTxnCount}</span>
          {labels.newTransactionsSince} {jakartaHHMM(newTxnSince ?? new Date())}
        </span>
        <Button variant="secondary" size="sm" onclick={viewNew}>{labels.view}</Button>
      </div>
    {/if}

    <div class="flex items-center justify-between px-1 py-1">
      <span class="text-xs text-text-muted">
        {store.total} {labels.transaction} · {labels.updated} {jakartaHHMM(lastUpdated ?? new Date())}
      </span>
      <Button variant="ghost" size="icon" onclick={refresh} disabled={refreshing} aria-label={labels.refresh} title={labels.refresh}>
        <RefreshCw size={16} class={refreshing ? 'animate-spin' : ''} />
      </Button>
    </div>

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
  {:else}
    <FindTransaction />
  {/if}

  <!-- Deep-link drawer (notification "?txn=<id>") is mounted regardless of the
       active tab: a foreign sale resolves to lookup mode and switches to the
       Find Transaction tab, while an own sale stays on "My Transactions". -->
  <TransactionDrawer
    {selectedTransaction}
    mode={drawerMode}
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
