<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/router';
import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
import { useShiftStore } from '../stores/shift-store.svelte';
import ShiftDetailDrawer from './ShiftDetailDrawer.svelte';
import { Button, CurrencyInput, Input, Modal, Badge, Dropdown, CashBreakdown, Pagination, SortableHeader } from '$shared/ui';
import { useRBAC } from '$shared/composables/useRBAC.svelte';
import { useAuthStore } from '$modules/auth';
  import {
  Clock,
  Plus,
  Lock,
  ChevronDown,
  Loader2,
  Download,
} from 'lucide-svelte';
  import type { Shift } from '../types';

  const store = useShiftStore();
  const rbac = useRBAC();
  const authStore = useAuthStore();

  if (rbac.isCashier && authStore.user?.id) {
    store.userIdFilter = authStore.user.id;
  }

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
      alert('Failed to export shifts');
    }
  }



  async function handleOpenShift() {
    if (openingBalance <= 0) return;
    isSubmitting = true;
    try {
      await store.doOpenShift(null, openingBalance);
      showOpenModal = false;
      openingBalance = 0;
      prevFilters = '';
      goto('/pos');
    } catch (e: any) {
      alert(e?.response?.data?.error || 'Failed to open shift');
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
      alert(e?.response?.data?.error || 'Failed to close shift');
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
      alert(e?.response?.data?.error || 'Failed to review shift');
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
      alert(e?.response?.data?.error || 'Failed to audit shift');
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
    }
  }

  onMount(() => {
    store.loadActiveShift();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-5">
  <!-- Toolbar -->
  <div class="card p-4">
    <div class="flex items-center gap-4">
      {#if !rbac.isCashier}
      <Dropdown placement="bottom-start" items={[
        { label: 'All Status', checked: store.statusFilter === '', onclick: () => { store.statusFilter = ''; } },
        { label: 'Open', checked: store.statusFilter === 'open', onclick: () => { store.statusFilter = 'open'; } },
        { label: 'Closed', checked: store.statusFilter === 'closed', onclick: () => { store.statusFilter = 'closed'; } },
      ]}>
        {#snippet trigger({ toggle })}
          <button
            type="button"
            class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {store.statusFilter !== '' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
            onclick={toggle}
          >
            <span>{store.statusFilter === '' ? 'All Status' : store.statusFilter === 'open' ? 'Open' : 'Closed'}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
        {/snippet}
      </Dropdown>

      <Dropdown placement="bottom-start" items={[
        { label: 'All Review Status', checked: store.needsReviewFilter === null, onclick: () => { store.needsReviewFilter = null; } },
        { label: 'Needs Review', checked: store.needsReviewFilter === true, onclick: () => { store.needsReviewFilter = true; } },
        { label: 'Reviewed', checked: store.needsReviewFilter === false, onclick: () => { store.needsReviewFilter = false; } },
      ]}>
        {#snippet trigger({ toggle })}
          <button
            type="button"
            class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {store.needsReviewFilter !== null ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
            onclick={toggle}
          >
            <span>{store.needsReviewFilter === null ? 'Review Status' : store.needsReviewFilter ? 'Needs Review' : 'Reviewed'}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
        {/snippet}
      </Dropdown>

      <Dropdown placement="bottom-start" items={[
        { label: 'All Discrepancies', checked: store.discrepancyFilter === '', onclick: () => { store.discrepancyFilter = ''; } },
        { label: 'Balanced', checked: store.discrepancyFilter === 'balanced', onclick: () => { store.discrepancyFilter = 'balanced'; } },
        { label: 'Surplus', checked: store.discrepancyFilter === 'surplus', onclick: () => { store.discrepancyFilter = 'surplus'; } },
        { label: 'Shortage', checked: store.discrepancyFilter === 'shortage', onclick: () => { store.discrepancyFilter = 'shortage'; } },
      ]}>
        {#snippet trigger({ toggle })}
          <button
            type="button"
            class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {store.discrepancyFilter !== '' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
            onclick={toggle}
          >
            <span>{store.discrepancyFilter === '' ? 'Discrepancy' : store.discrepancyFilter === 'balanced' ? 'Balanced' : store.discrepancyFilter === 'surplus' ? 'Surplus' : 'Shortage'}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
        {/snippet}
      </Dropdown>
      {/if}

      <div class="ml-auto flex items-center gap-2">
        <Dropdown placement="bottom-end" items={[
          { label: 'Export CSV', onclick: () => downloadExport('csv') },
          { label: 'Export XLSX', onclick: () => downloadExport('xlsx') },
        ]}>
          {#snippet trigger({ toggle })}
            <Button variant="secondary" class="shrink-0 px-3" onclick={toggle}>
              <Download size={14} />
              Export
              <ChevronDown size={14} />
            </Button>
          {/snippet}
        </Dropdown>
        {#if store.activeShift}
          <Button variant="danger" onclick={() => { showCloseModal = true; closingBalance = store.activeShift?.opening_balance || 0; }}>
            <Lock size={16} class="mr-2" />
            Close Shift
          </Button>
        {:else}
          <Button variant="primary" onclick={() => { showOpenModal = true; openingBalance = 0; }}>
            <Plus size={16} class="mr-2" />
            Open Shift
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
          <h3 class="font-semibold text-text-primary">Active Shift</h3>
          <p class="text-sm text-text-muted">Opened {formatDateTime(store.activeShift.opened_at)}</p>
        </div>
        <Badge variant="success" class="ml-auto">Open</Badge>
      </div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div>
          <p class="text-xs text-text-muted">Opening Balance</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.opening_balance)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Total Sales</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.total_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Cash Sales</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.cash_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Transactions</p>
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
            <th class="text-left px-3 py-3 font-semibold text-text-secondary align-top {rbac.isCashier ? 'w-[170px]' : 'w-[150px]'}">
              <SortableHeader label="OPENED AT" column="opened_at" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} />
            </th>
            {#if !rbac.isCashier && authStore.user}
            <th class="text-left px-3 py-3 font-semibold text-text-secondary align-top w-[120px]">CASHIER</th>
            {/if}
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[110px]">
              <SortableHeader label="OPENING (RP)" column="opening_balance" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} align="right" />
            </th>
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[110px]">
              <SortableHeader label="CASH SALES (RP)" column="cash_sales" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} align="right" />
            </th>
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[110px]">
              <SortableHeader label="TOTAL SALES (RP)" column="total_sales" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} align="right" />
            </th>
            <th class="text-center px-3 py-3 font-semibold text-text-secondary align-top w-[50px]">TXN</th>
            <th class="text-right px-3 py-3 font-semibold text-text-secondary align-top w-[110px]">
              <SortableHeader label="DISCREPANCY" column="discrepancy" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} align="right" />
            </th>
            <th class="text-center px-3 py-3 font-semibold text-text-secondary align-top w-[80px]">STATUS</th>
            <th class="text-left px-3 py-3 font-semibold text-text-secondary align-top w-[150px]">
              <SortableHeader label="CLOSED AT" column="closed_at" sortColumn={store.sortBy} sortDirection={store.sortDir} onsort={(col) => { store.sortBy = col; store.page = 0; }} />
            </th>
          </tr>
        </thead>
        <tbody>
          {#if store.loading}
            <tr>
              <td colspan={rbac.isCashier ? 8 : 9} class="px-4 py-12 text-center text-text-muted">
                <Loader2 size={20} class="animate-spin mx-auto mb-2" />
                Loading shifts...
              </td>
            </tr>
          {:else if store.shifts.length === 0}
            <tr>
              <td colspan={rbac.isCashier ? 8 : 9} class="px-4 py-12 text-center text-text-muted">
                No shifts found
              </td>
            </tr>
          {:else}
            {#each store.shifts as shift}
              <tr
                class="border-b border-border/50 hover:bg-surface-hover cursor-pointer transition-colors"
                onclick={() => openDetail(shift)}
              >
                <td class="px-3 py-3 text-text-primary text-xs whitespace-nowrap">{formatDateTime(shift.opened_at)}</td>
                {#if !rbac.isCashier && authStore.user}
                <td class="px-3 py-3 text-text-primary text-xs truncate" title={shift.username || '-'}>{shift.username || '-'}</td>
                {/if}
                <td class="px-3 py-3 text-right text-text-primary text-xs tabular-nums">{formatNumber(shift.opening_balance)}</td>
                <td class="px-3 py-3 text-right text-text-primary text-xs tabular-nums">{formatNumber(shift.cash_sales)}</td>
                <td class="px-3 py-3 text-right font-medium text-text-primary text-xs tabular-nums">{formatNumber(shift.total_sales)}</td>
                <td class="px-3 py-3 text-center text-text-secondary text-xs">{shift.transaction_count}</td>
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
                    <Badge variant="success" size="sm">Open</Badge>
                  {:else}
                    <Badge variant={shift.needs_review ? 'warning' : 'muted'} size="sm">Closed</Badge>
                  {/if}
                </td>
                <td class="px-3 py-3 text-text-secondary text-xs whitespace-nowrap">{formatDateTime(shift.closed_at)}</td>
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
<Modal bind:open={showOpenModal} title="Open Shift" size="sm">
  <form onsubmit={(e) => { e.preventDefault(); handleOpenShift(); }} class="space-y-4">
    <div>
      <label for="opening-balance" class="block text-sm font-medium text-text-secondary mb-2">
        Opening Balance (Rp)
      </label>
      <CurrencyInput id="opening-balance" bind:value={openingBalance} placeholder="0" required />
      <p class="text-xs text-text-muted mt-1">Amount of cash in the drawer at shift start</p>
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isSubmitting} onclick={() => { showOpenModal = false; }}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={isSubmitting || openingBalance <= 0} onclick={handleOpenShift}>
      {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      Open Shift
    </Button>
  {/snippet}
</Modal>

<!-- Close Shift Modal -->
<Modal bind:open={showCloseModal} title="Close Shift" size="xl" panelClass="!max-h-none">
  {#if store.activeShift}
    <div class="space-y-6">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4 bg-surface-secondary rounded-lg p-4">
        <div>
          <p class="text-xs text-text-muted">Opening Balance</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.opening_balance)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Cash Sales</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.cash_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Non-Cash Sales</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.non_cash_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Transactions</p>
          <p class="text-lg font-bold text-text-primary">{store.activeShift.transaction_count}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Total Sales</p>
          <p class="text-lg font-bold text-text-primary">{formatMoney(store.activeShift.total_sales)}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Expected Cash</p>
          <p class="text-lg font-bold text-primary">{formatMoney(store.activeShift.opening_balance + store.activeShift.cash_sales)}</p>
        </div>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleCloseShift(); }} class="space-y-4">
        <div>
          <label for="closing-balance" class="block text-sm font-medium text-text-secondary mb-2">
            Closing Balance (Rp)
          </label>
          <CashBreakdown bind:total={closingBalance} />
          {#if closingBalance > 0 && store.activeShift}
            {@const expected = store.activeShift.opening_balance + store.activeShift.cash_sales}
            {@const disc = closingBalance - expected}
            <p class="text-xs mt-1 {disc === 0 ? 'text-success' : 'text-danger'}">
              {disc === 0 ? 'Balanced' : `Discrepancy: ${disc > 0 ? '+' : ''}${formatMoney(disc)}`}
            </p>
          {/if}
        </div>
        <div>
          <label for="close-notes" class="block text-sm font-medium text-text-secondary mb-2">Notes (optional)</label>
          <Input
            id="close-notes"
            type="text"
            bind:value={closeNotes}
            placeholder="Optional notes"
          />
        </div>
      </form>
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isSubmitting} onclick={() => { showCloseModal = false; }}>Cancel</Button>
    <Button variant="danger" class="px-5" disabled={isSubmitting || closingBalance <= 0} onclick={handleCloseShift}>
      {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      Close Shift
    </Button>
  {/snippet}
</Modal>

<ShiftDetailDrawer
  bind:showDetailDrawer
  {selectedShift}
  isManager={rbac.isManager}
  onreview={handleReview}
  onaudit={() => selectedShift && openAuditModal(selectedShift)}
/>

<!-- Audit Modal -->
<Modal bind:open={showAuditModal} title="Surprise Audit" size="sm">
  {#if selectedShift}
    <div class="space-y-4">
      <div class="bg-surface-secondary rounded-lg p-4 space-y-2">
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">Cashier</span>
          <span class="text-text-primary font-medium">{selectedShift.username || '-'}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">Opening Balance</span>
          <span class="text-text-primary font-medium">{formatMoney(selectedShift.opening_balance)}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">Cash Sales (system)</span>
          <span class="text-text-primary font-medium">{formatMoney(selectedShift.cash_sales)}</span>
        </div>
      </div>

      {#if auditResult}
        <div class="bg-surface-secondary rounded-lg p-4 space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-text-muted">Expected Cash</span>
            <span class="text-text-primary font-medium">{formatMoney(auditResult.expected_cash)}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-text-muted">Actual Balance</span>
            <span class="text-text-primary font-medium">{formatMoney(auditResult.actual_balance)}</span>
          </div>
          <div class="flex justify-between text-sm border-t border-border pt-2">
            <span class="text-text-muted">Difference</span>
            <span class="text-sm font-bold {auditResult.off_by === 0 ? 'text-success' : 'text-danger'}">
              {auditResult.off_by > 0 ? '+' : ''}{formatMoney(auditResult.off_by)}
            </span>
          </div>
        </div>
      {:else}
        <form onsubmit={(e) => { e.preventDefault(); handleAudit(); }} class="space-y-4">
          <div>
            <label for="audit-balance" class="block text-sm font-medium text-text-secondary mb-2">
              Actual Cash in Drawer (Rp)
            </label>
            <CurrencyInput id="audit-balance" bind:value={auditActualBalance} placeholder="0" required />
          </div>
        </form>
      {/if}
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={() => { showAuditModal = false; }}>Close</Button>
    {#if !auditResult}
      <Button variant="primary" class="px-5" disabled={isSubmitting || auditActualBalance <= 0} onclick={handleAudit}>
        {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
        Submit Audit
      </Button>
    {/if}
  {/snippet}
</Modal>
