<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { goto } from '$app/router';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { formatDuration } from '$shared/utils/duration';
import { useShiftStore } from '../stores/shift-store.svelte';
  import ShiftDetailDrawer from './ShiftDetailDrawer.svelte';
  import CashMovementModal from './CashMovementModal.svelte';
  import { Button, CurrencyInput, Input, Modal, Badge, Dropdown, CashBreakdown, Pagination, SortableHeader, Skeleton } from '$shared/ui';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';
  import { useAuthStore } from '$modules/auth';
  import { settingsStore } from '$shared/stores/settings.svelte';
  import { labels } from '$shared/i18n';
  import { getActiveStores } from '$modules/stores/services/stores-service';
  import type { Store } from '$modules/stores/types';
  import {
  Clock,
  Plus,
  Lock,
  ChevronDown,
  Loader2,
  Download,
  ArrowLeftRight,
} from 'lucide-svelte';
  import type { Shift } from '../types';

  const store = useShiftStore();
  const rbac = useRBAC();
  const authStore = useAuthStore();

  // @ownership-only — data-scope: cashier hanya melihat shift milik sendiri.
  const isCashier = $derived(rbac.isCashier);

  $effect(() => {
    if (isCashier && authStore.user?.id) {
      store.userIdFilter = authStore.user.id;
    }
  });

  let showOpenModal = $state(false);
  let showCloseModal = $state(false);
  let openingBalance = $state(0);
  let closingBalance = $state(0);
  let closeNotes = $state('');
  let isSubmitting = $state(false);
  let selectedShift = $state<Shift | null>(null);
  let showDetailDrawer = $state(false);
  let showAuditModal = $state(false);
  let auditActualBalance = $state(0);
  let auditResult = $state<{ expected_cash: number; actual_balance: number; off_by: number } | null>(null);
  let showCashMovementModal = $state(false);
  let showExpected = $state(false);
  let stores = $state<Store[]>([]);
  let selectedStoreId = $state<number | null>(null);

  let prevFilters = '';

  function loadShifts() {
    const filters = store.currentFilters;
    const json = JSON.stringify(filters);
    if (json !== prevFilters) {
      prevFilters = json;
      store.loadShifts(filters);
    }
  }

  $effect(() => {
    store.statusFilter;
    store.needsReviewFilter;
    store.discrepancyFilter;
    store.userIdFilter;
    store.sortBy;
    store.sortDir;
    store.page = 0;
    loadShifts();
  });

  async function downloadExport(format: 'csv' | 'xlsx') {
    try {
      const blob = await store.doExport(format);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `shifts.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (e) {
      alert(labels.failedToExportShifts);
    }
  }



  async function handleOpenShift() {
    if (openingBalance <= 0) return;
    if (stores.length > 1 && !selectedStoreId) return;
    isSubmitting = true;
    try {
      await store.doOpenShift(selectedStoreId, openingBalance);
      showOpenModal = false;
      openingBalance = 0;
      selectedStoreId = null;
      prevFilters = '';
      goto('/pos');
    } catch (e: any) {
      alert(e?.response?.data?.error || labels.failedToOpenShift);
    } finally {
      isSubmitting = false;
    }
  }

  async function handleCloseShift() {
    if (closingBalance < 0) return;
    if (!store.activeShift) return;
    isSubmitting = true;
    try {
      await store.doCloseShift(store.activeShift.id, closingBalance, closeNotes || null);
      showCloseModal = false;
      closingBalance = 0;
      closeNotes = '';
      prevFilters = '';
      store.loadShifts(store.currentFilters);
    } catch (e: any) {
      alert(e?.response?.data?.error || labels.failedToCloseShift);
    } finally {
      isSubmitting = false;
    }
  }

  async function handleReview() {
    if (!selectedShift) return;
    isSubmitting = true;
    try {
      await store.doReviewShift(selectedShift.id);
      selectedShift = null;
      showDetailDrawer = false;
      prevFilters = '';
      store.loadShifts(store.currentFilters);
    } catch (e: any) {
      alert(e?.response?.data?.error || labels.failedToReviewShift);
    } finally {
      isSubmitting = false;
    }
  }

  function openDetail(shift: Shift) {
    selectedShift = shift;
    showDetailDrawer = true;
  }

  function formatDateTime(dateStr: string | null) {
    if (!dateStr) return '-';
    return formatDateTimeInJakarta(dateStr);
  }

  function formatMoney(amount: number) {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount);
  }

  function formatNumber(amount: number) {
    return amount.toLocaleString('id-ID');
  }

  async function handleAudit() {
    if (!selectedShift || auditActualBalance <= 0) return;
    isSubmitting = true;
    auditResult = null;
    try {
      const result = await store.doAuditShift(selectedShift.id, auditActualBalance);
      auditResult = { expected_cash: result.expected_cash, actual_balance: result.actual_balance, off_by: result.off_by };
    } catch (e: any) {
      alert(e?.response?.data?.error || labels.failedToAuditShift);
    } finally {
      isSubmitting = false;
    }
  }

  function openAuditModal(shift: Shift) {
    selectedShift = shift;
    auditActualBalance = 0;
    auditResult = null;
    showAuditModal = true;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      showOpenModal = false;
      showCloseModal = false;
      showDetailDrawer = false;
      showAuditModal = false;
      showCashMovementModal = false;
      showExpected = false;
    }
  }

  let pollInterval: ReturnType<typeof setInterval> | null = null;

  function startPolling() {
    stopPolling();
    if (store.activeShift) {
      pollInterval = setInterval(() => store.loadActiveShift(), 30000);
    }
  }

  function stopPolling() {
    if (pollInterval) {
      clearInterval(pollInterval);
      pollInterval = null;
    }
  }

  $effect(() => {
    const _ = store.activeShift;
    if (store.activeShift) {
      startPolling();
    } else {
      stopPolling();
    }
  });

  $effect(() => {
    if (showOpenModal) {
      getActiveStores().then((s) => {
        stores = s;
        if (s.length === 1) selectedStoreId = s[0].id;
      }).catch(() => { stores = []; });
    } else {
      selectedStoreId = null;
    }
  });

  onDestroy(() => {
    stopPolling();
  });

  onMount(() => {
    store.loadActiveShift();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-5">
  <!-- Toolbar -->
  <div class="card p-4">
    <div class="flex items-center gap-4">
      {#if !isCashier}
      <Dropdown placement="bottom-start" items={[
        { label: labels.allStatus, checked: store.statusFilter === '', onclick: () => { store.statusFilter = ''; } },
        { label: labels.open, checked: store.statusFilter === 'open', onclick: () => { store.statusFilter = 'open'; } },
        { label: labels.closed, checked: store.statusFilter === 'closed', onclick: () => { store.statusFilter = 'closed'; } },
      ]}>
        {#snippet trigger({ toggle })}
          <button
            type="button"
            class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {store.statusFilter !== '' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
            onclick={toggle}
          >
            <span>{store.statusFilter === '' ? labels.allStatus : store.statusFilter === 'open' ? labels.open : labels.closed}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
        {/snippet}
      </Dropdown>

      <Dropdown placement="bottom-start" items={[
        { label: labels.allStatus, checked: store.needsReviewFilter === null, onclick: () => { store.needsReviewFilter = null; } },
        { label: labels.needsReview, checked: store.needsReviewFilter === true, onclick: () => { store.needsReviewFilter = true; } },
        { label: labels.reviewed, checked: store.needsReviewFilter === false, onclick: () => { store.needsReviewFilter = false; } },
      ]}>
        {#snippet trigger({ toggle })}
          <button
            type="button"
            class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {store.needsReviewFilter !== null ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
            onclick={toggle}
          >
            <span>{store.needsReviewFilter === null ? labels.reviewStatus : store.needsReviewFilter ? labels.needsReview : labels.reviewed}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
        {/snippet}
      </Dropdown>

      <Dropdown placement="bottom-start" items={[
        { label: labels.allStatus, checked: store.discrepancyFilter === '', onclick: () => { store.discrepancyFilter = ''; } },
        { label: labels.balanced, checked: store.discrepancyFilter === 'balanced', onclick: () => { store.discrepancyFilter = 'balanced'; } },
        { label: labels.surplus, checked: store.discrepancyFilter === 'surplus', onclick: () => { store.discrepancyFilter = 'surplus'; } },
        { label: labels.shortage, checked: store.discrepancyFilter === 'shortage', onclick: () => { store.discrepancyFilter = 'shortage'; } },
      ]}>
        {#snippet trigger({ toggle })}
          <button
            type="button"
            class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {store.discrepancyFilter !== '' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
            onclick={toggle}
          >
            <span>{store.discrepancyFilter === '' ? labels.discrepancy : store.discrepancyFilter === 'balanced' ? labels.balanced : store.discrepancyFilter === 'surplus' ? labels.surplus : labels.shortage}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
        {/snippet}
      </Dropdown>
      {/if}

      <div class="ml-auto flex items-center gap-2">
        <Dropdown placement="bottom-end" items={[
          { label: labels.exportCsv, onclick: () => downloadExport('csv') },
          { label: labels.exportXlsx, onclick: () => downloadExport('xlsx') },
        ]}>
          {#snippet trigger({ toggle })}
            <Button variant="secondary" class="shrink-0 px-3" onclick={toggle}>
              <Download size={14} />
              {labels.export}
              <ChevronDown size={14} />
            </Button>
          {/snippet}
        </Dropdown>
        {#if store.activeShift}
          {#if rbac.can(Permissions.shift.cashMovement)}
            <Button variant="secondary" onclick={() => { showCashMovementModal = true; }}>
              <ArrowLeftRight size={16} class="mr-2" />
              {labels.cashMovements}
            </Button>
          {/if}
          <Button variant="danger" onclick={() => { showCloseModal = true; closingBalance = store.activeShift?.opening_balance || 0; }}>
            <Lock size={16} class="mr-2" />
            {labels.closeShift}
          </Button>
        {:else}
          <Button variant="primary" onclick={() => { showOpenModal = true; openingBalance = 0; }}>
            <Plus size={16} class="mr-2" />
            {labels.openShift}
          </Button>
        {/if}
      </div>
    </div>
  </div>

  <!-- Active Shift Banner -->
  {#if store.activeShift}
    <div class="bg-gradient-to-r from-primary-subtle to-primary-subtle/50 border border-primary/20 rounded-xl p-5">
      <div class="flex items-center gap-3 mb-3">
        <div class="w-10 h-10 rounded-xl bg-primary/20 flex items-center justify-center">
          <Clock size={20} class="text-primary" />
        </div>
        <div>
          <h3 class="font-semibold text-text-primary">{labels.activeShift}</h3>
          <p class="text-sm text-text-muted">{labels.openedOn.replace('{date}', formatDateTime(store.activeShift.opened_at))}</p>
        </div>
        <Badge variant="success" class="ml-auto">{labels.open}</Badge>
      </div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div>
          <p class="text-xs text-text-muted">{labels.openingBalance}</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.opening_balance)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">{labels.totalSales}</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.total_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">{labels.cashSales}</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.cash_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">{labels.transactions}</p>
          <p class="text-lg font-bold text-text-primary">{store.activeShift.transaction_count}</p>
        </div>
      </div>
    </div>
  {/if}

  <!-- Table -->
  <div class="bg-surface rounded-xl border border-border overflow-hidden">
    <div class="overflow-x-auto">
      <table class="w-full text-sm table-fixed">
        <thead>
          <tr class="border-b border-border bg-surface-secondary">
            <th class="text-left px-3 py-3 font-semibold text-text-secondary align-top {isCashier ? 'w-[170px]' : 'w-[135px]'}">
              <SortableHeader label={labels.openedAtLabel} column="opened_at" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} />
            </th>
                {#if !isCashier && authStore.user}
            <th class="text-left px-3 py-3 font-semibold text-text-secondary align-top w-[95px]">{labels.cashier}</th>
            {/if}
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[110px]">
              <SortableHeader label={labels.openingRp} column="opening_balance" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} align="right" />
            </th>
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[150px]">
              <SortableHeader label={labels.cashSalesRp} column="cash_sales" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} align="right" />
            </th>
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[150px]">
              <SortableHeader label={labels.totalSalesRp} column="total_sales" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} align="right" />
            </th>
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[50px]">{labels.txn}</th>
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[115px]">
              <SortableHeader label={labels.discrepancyLabel} column="discrepancy" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} align="right" />
            </th>
            <th class="text-center px-3 py-3 font-semibold text-text-secondary align-top w-[80px]">{labels.status}</th>
            <th class="text-left px-3 py-3 font-semibold text-text-secondary align-top w-[145px]">
              <SortableHeader label={labels.closedAtLabel} column="closed_at" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} />
            </th>
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[80px]">{labels.duration}</th>
          </tr>
        </thead>
        <tbody>
          {#if store.loading}
            {#each Array(5) as _}
              <tr>{#each Array(isCashier ? 9 : 10) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>
            {/each}
          {:else if store.shifts.length === 0}
            <tr>
              <td colspan={isCashier ? 9 : 10} class="px-4 py-12 text-center text-text-muted">
                {labels.noShifts}
              </td>
            </tr>
          {:else}
            {#each store.shifts as shift}
              <tr
                class="border-b border-border/50 hover:bg-surface-hover cursor-pointer transition-colors"
                onclick={() => openDetail(shift)}
              >
                <td class="px-3 py-3 text-text-primary text-xs whitespace-nowrap">{formatDateTime(shift.opened_at)}</td>
            {#if !isCashier && authStore.user}
                <td class="px-3 py-3 text-text-primary text-xs truncate" title={shift.username || '-'}>{shift.username || '-'}</td>
                {/if}
                <td class="px-3 py-3 text-right text-text-primary text-xs tabular-nums">{formatNumber(shift.opening_balance)}</td>
                <td class="px-3 py-3 text-right text-text-primary text-xs tabular-nums">{formatNumber(shift.cash_sales)}</td>
                <td class="px-3 py-3 text-right font-medium text-text-primary text-xs tabular-nums">{formatNumber(shift.total_sales)}</td>
                <td class="px-3 py-3 text-right text-text-secondary text-xs tabular-nums">{shift.transaction_count}</td>
                <td class="px-3 py-3 text-right text-xs">
                  {#if shift.discrepancy != null}
                    <span class="{shift.discrepancy === 0 ? 'text-success' : 'text-danger'} tabular-nums">
                      {shift.discrepancy > 0 ? '+' : ''}{formatNumber(shift.discrepancy)}
                    </span>
                  {:else}
                    <span class="text-text-muted">-</span>
                  {/if}
                </td>
                <td class="px-3 py-3 text-center">
                  {#if shift.status === 'open'}
                    <Badge variant="success" size="sm">{labels.open}</Badge>
                  {:else}
                    <Badge variant={shift.needs_review ? 'warning' : 'muted'} size="sm">{labels.closed}</Badge>
                  {/if}
                </td>
                <td class="px-3 py-3 text-text-secondary text-xs whitespace-nowrap">{formatDateTime(shift.closed_at)}</td>
                <td class="px-3 py-3 text-right text-text-secondary text-xs tabular-nums">{formatDuration(shift.opened_at, shift.closed_at)}</td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    {#if !store.loading && store.shifts.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination
          total={store.total}
          limit={store.pageSize}
          offset={store.offset}
          onPageChange={(newOffset, newLimit) => {
            store.page = Math.floor(newOffset / newLimit);
            store.pageSize = newLimit;
            prevFilters = "";
            loadShifts();
          }}
        />
      </div>
    {/if}
  </div>
</div>

<!-- Open Shift Modal -->
<Modal bind:open={showOpenModal} title={labels.openShift} size="sm">
  <form onsubmit={(e) => { e.preventDefault(); handleOpenShift(); }} class="space-y-4">
    {#if stores.length > 1}
      <div>
        <label for="store-select" class="block text-sm font-medium text-text-secondary mb-2">
          {labels.store}
        </label>
        <select
          id="store-select"
          bind:value={selectedStoreId}
          class="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
        >
          <option value={null}>{labels.selectStore}</option>
          {#each stores as s}
            <option value={s.id}>{s.name}</option>
          {/each}
        </select>
      </div>
    {/if}
    <div>
      <label for="opening-balance" class="block text-sm font-medium text-text-secondary mb-2">
        {labels.openingBalanceRp}
      </label>
      <CurrencyInput id="opening-balance" bind:value={openingBalance} placeholder="0" required />
      <p class="text-xs text-text-muted mt-1">{labels.cashInDrawerAtStart}</p>
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isSubmitting} onclick={() => { showOpenModal = false; }}>{labels.cancel}</Button>
    <Button variant="primary" class="px-5" disabled={isSubmitting || openingBalance <= 0 || (stores.length > 1 && !selectedStoreId)} onclick={handleOpenShift}>
      {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      {labels.openShift}
    </Button>
  {/snippet}
</Modal>

<!-- Close Shift Modal -->
<Modal bind:open={showCloseModal} title={labels.closeShift} size="xl" panelClass="!max-h-none">
  {#if store.activeShift}
    <div class="space-y-6">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4 bg-surface-secondary rounded-lg p-4">
        <div>
          <p class="text-xs text-text-muted">{labels.openingBalance}</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.opening_balance)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">{labels.cashSales}</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.cash_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">{labels.nonCashSales}</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.non_cash_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">{labels.transactions}</p>
          <p class="text-lg font-bold text-text-primary">{store.activeShift.transaction_count}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">{labels.totalSales}</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.total_sales)}</p>
        </div>
        {#if !settingsStore.shiftBlindClose || showExpected}
          <div>
            <p class="text-xs text-text-muted">{labels.expectedCash}</p>
            <p class="text-lg font-bold text-primary">{formatMoney(store.activeShift.opening_balance + store.activeShift.cash_sales)}</p>
          </div>
        {:else}
          <div>
            <button
              type="button"
              class="text-xs text-primary underline hover:text-primary/80"
              onclick={() => { showExpected = true; }}
            >
              {labels.showExpected}
            </button>
          </div>
        {/if}
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleCloseShift(); }} class="space-y-4">
        <div>
          <label for="closing-balance" class="block text-sm font-medium text-text-secondary mb-2">
            {labels.closingBalanceRp}
          </label>
          <CashBreakdown bind:total={closingBalance} />
          {#if closingBalance > 0 && store.activeShift}
            {@const expected = store.activeShift.opening_balance + store.activeShift.cash_sales}
            {@const disc = closingBalance - expected}
            <p class="text-xs mt-1 {disc === 0 ? 'text-success' : 'text-danger'}">
              {disc === 0 ? labels.balanced : `${labels.discrepancy}: ${disc > 0 ? '+' : ''}${formatMoney(disc)}`}
            </p>
          {/if}
        </div>
        <div>
          <label for="close-notes" class="block text-sm font-medium text-text-secondary mb-2">{labels.optionalNotes}</label>
          <Input
            id="close-notes"
            type="text"
            bind:value={closeNotes}
            placeholder={labels.optionalNotes}
          />
        </div>
      </form>
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isSubmitting} onclick={() => { showCloseModal = false; showExpected = false; }}>{labels.cancel}</Button>
    <Button variant="danger" class="px-5" disabled={isSubmitting || closingBalance <= 0} onclick={handleCloseShift}>
      {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      {labels.closeShift}
    </Button>
  {/snippet}
</Modal>

<ShiftDetailDrawer
  bind:showDetailDrawer
  {selectedShift}
  canReview={rbac.can(Permissions.shift.review)}
  canAudit={rbac.can(Permissions.shift.audit)}
  onreview={handleReview}
  onaudit={() => selectedShift && openAuditModal(selectedShift)}
/>

<!-- Audit Modal -->
<Modal bind:open={showAuditModal} title={labels.surpriseAudit} size="sm">
  {#if selectedShift}
    <div class="space-y-4">
      <div class="bg-surface-secondary rounded-lg p-4 space-y-2">
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">{labels.cashier}</span>
          <span class="text-text-primary font-medium">{selectedShift.username || '-'}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">{labels.openingBalance}</span>
          <span class="text-text-primary font-medium">{formatMoney(selectedShift.opening_balance)}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">{labels.cashSalesSystem}</span>
          <span class="text-text-primary font-medium">{formatMoney(selectedShift.cash_sales)}</span>
        </div>
      </div>

      {#if auditResult}
        <div class="bg-surface-secondary rounded-lg p-4 space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-text-muted">{labels.expectedCash}</span>
            <span class="text-text-primary font-medium">{formatMoney(auditResult.expected_cash)}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-text-muted">{labels.actualBalance}</span>
            <span class="text-text-primary font-medium">{formatMoney(auditResult.actual_balance)}</span>
          </div>
          <div class="flex justify-between text-sm border-t border-border pt-2">
            <span class="text-text-muted">{labels.difference}</span>
            <span class="text-sm font-bold {auditResult.off_by === 0 ? 'text-success' : 'text-danger'}">
              {auditResult.off_by > 0 ? '+' : ''}{formatMoney(auditResult.off_by)}
            </span>
          </div>
        </div>
      {:else}
        <form onsubmit={(e) => { e.preventDefault(); handleAudit(); }} class="space-y-4">
          <div>
            <label for="audit-balance" class="block text-sm font-medium text-text-secondary mb-2">
              {labels.actualCashInDrawer}
            </label>
            <CurrencyInput id="audit-balance" bind:value={auditActualBalance} placeholder="0" required />
          </div>
        </form>
      {/if}
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={() => { showAuditModal = false; }}>{labels.close}</Button>
    {#if !auditResult}
      <Button variant="primary" class="px-5" disabled={isSubmitting || auditActualBalance <= 0} onclick={handleAudit}>
        {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
        {labels.submitAudit}
      </Button>
    {/if}
  {/snippet}
</Modal>

{#if store.activeShift}
  <CashMovementModal
    bind:open={showCashMovementModal}
    shiftId={store.activeShift.id}
    onrecord={() => { prevFilters = ''; store.loadShifts(store.currentFilters); }}
  />
{/if}
