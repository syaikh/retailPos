<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { useShiftStore } from '../stores/shift-store.svelte';
  import { Button, Input, Modal, Badge } from '$shared/ui';
  import {
    Clock,
    Plus,
    Lock,
    ChevronLeft,
    ChevronRight,
    ArrowUpDown,
    ArrowUp,
    ArrowDown,
    Loader2,
    Calendar,
    DollarSign,
    Receipt,
    AlertTriangle,
  } from 'lucide-svelte';
  import type { Shift } from '../types';

  const store = useShiftStore();

  let showOpenModal = $state(false);
  let showCloseModal = $state(false);
  let openingBalance = $state(0);
  let closingBalance = $state(0);
  let closeNotes = $state('');
  let isSubmitting = $state(false);
  let selectedShift = $state<Shift | null>(null);
  let showDetailModal = $state(false);

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
    store.page;
    store.pageSize;
    store.sortBy;
    store.sortDir;
    loadShifts();
  });

  function toggleSort(column: string) {
    if (store.sortBy === column) {
      store.sortDir = store.sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      store.sortBy = column;
      store.sortDir = 'desc';
    }
    store.page = 0;
  }

  async function handleOpenShift() {
    if (openingBalance < 0) return;
    isSubmitting = true;
    try {
      await store.doOpenShift(null, openingBalance);
      showOpenModal = false;
      openingBalance = 0;
      loadShifts();
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
      loadShifts();
    } catch (e: any) {
      alert(e?.response?.data?.error || 'Failed to close shift');
    } finally {
      isSubmitting = false;
    }
  }

  function openDetail(shift: Shift) {
    selectedShift = shift;
    showDetailModal = true;
  }

  function formatDateTime(dateStr: string | null) {
    if (!dateStr) return '-';
    const d = new Date(dateStr);
    return d.toLocaleDateString('id-ID', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  function formatMoney(amount: number) {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      showOpenModal = false;
      showCloseModal = false;
      showDetailModal = false;
    }
  }

  onMount(() => {
    store.loadActiveShift();
    loadShifts();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-5">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-bold text-text-primary">Shift Management</h1>
      <p class="text-sm text-text-muted mt-1">Manage cashier shifts and balances</p>
    </div>
    <div class="flex gap-3">
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

  <!-- Filter -->
  <div class="flex items-center gap-3">
    <select
      bind:value={store.statusFilter}
      class="h-9 px-3 rounded-lg border border-border bg-surface text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
    >
      <option value="">All Status</option>
      <option value="open">Open</option>
      <option value="closed">Closed</option>
    </select>
  </div>

  <!-- Table -->
  <div class="bg-surface rounded-xl border border-border overflow-hidden">
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border bg-surface-secondary">
            <th class="text-left px-4 py-3 font-medium text-text-secondary">
              <button class="flex items-center gap-1 hover:text-text-primary" onclick={() => toggleSort('opened_at')}>
                Opened At
                {#if store.sortBy === 'opened_at'}
                  {#if store.sortDir === 'asc'}<ArrowUp size={14} />{:else}<ArrowDown size={14} />{/if}
                {:else}
                  <ArrowUpDown size={14} />
                {/if}
              </button>
            </th>
            <th class="text-left px-4 py-3 font-medium text-text-secondary">Cashier</th>
            <th class="text-left px-4 py-3 font-medium text-text-secondary">Store</th>
            <th class="text-right px-4 py-3 font-medium text-text-secondary">Opening</th>
            <th class="text-right px-4 py-3 font-medium text-text-secondary">Cash Sales</th>
            <th class="text-right px-4 py-3 font-medium text-text-secondary">Total Sales</th>
            <th class="text-center px-4 py-3 font-medium text-text-secondary">TXN</th>
            <th class="text-right px-4 py-3 font-medium text-text-secondary">Discrepancy</th>
            <th class="text-center px-4 py-3 font-medium text-text-secondary">Status</th>
            <th class="text-left px-4 py-3 font-medium text-text-secondary">
              <button class="flex items-center gap-1 hover:text-text-primary" onclick={() => toggleSort('closed_at')}>
                Closed At
                {#if store.sortBy === 'closed_at'}
                  {#if store.sortDir === 'asc'}<ArrowUp size={14} />{:else}<ArrowDown size={14} />{/if}
                {:else}
                  <ArrowUpDown size={14} />
                {/if}
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          {#if store.loading}
            <tr>
              <td colspan="10" class="px-4 py-12 text-center text-text-muted">
                <Loader2 size={20} class="animate-spin mx-auto mb-2" />
                Loading shifts...
              </td>
            </tr>
          {:else if store.shifts.length === 0}
            <tr>
              <td colspan="10" class="px-4 py-12 text-center text-text-muted">
                No shifts found
              </td>
            </tr>
          {:else}
            {#each store.shifts as shift}
              <tr
                class="border-b border-border/50 hover:bg-surface-hover cursor-pointer transition-colors"
                onclick={() => openDetail(shift)}
              >
                <td class="px-4 py-3 text-text-primary">{formatDateTime(shift.opened_at)}</td>
                <td class="px-4 py-3 text-text-primary">{shift.username || '-'}</td>
                <td class="px-4 py-3 text-text-secondary">{shift.store_name || '-'}</td>
                <td class="px-4 py-3 text-right text-text-primary">{formatMoney(shift.opening_balance)}</td>
                <td class="px-4 py-3 text-right text-text-primary">{formatMoney(shift.cash_sales)}</td>
                <td class="px-4 py-3 text-right font-medium text-text-primary">{formatMoney(shift.total_sales)}</td>
                <td class="px-4 py-3 text-center text-text-secondary">{shift.transaction_count}</td>
                <td class="px-4 py-3 text-right">
                  {#if shift.discrepancy !== null}
                    <span class="{shift.discrepancy === 0 ? 'text-success' : 'text-danger'}">
                      {shift.discrepancy > 0 ? '+' : ''}{formatMoney(shift.discrepancy)}
                    </span>
                  {:else}
                    <span class="text-text-muted">-</span>
                  {/if}
                </td>
                <td class="px-4 py-3 text-center">
                  {#if shift.status === 'open'}
                    <Badge variant="success">Open</Badge>
                  {:else}
                    <Badge variant="secondary">Closed</Badge>
                  {/if}
                </td>
                <td class="px-4 py-3 text-text-secondary">{formatDateTime(shift.closed_at)}</td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    {#if store.total > 0}
      <div class="flex items-center justify-between px-4 py-3 border-t border-border">
        <p class="text-sm text-text-muted">
          Showing {store.offset + 1}–{Math.min(store.offset + store.pageSize, store.total)} of {store.total}
        </p>
        <div class="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            disabled={store.page === 0}
            onclick={() => { store.page--; }}
          >
            <ChevronLeft size={16} />
          </Button>
          <span class="text-sm text-text-secondary px-2">
            Page {store.page + 1} of {Math.ceil(store.total / store.pageSize)}
          </span>
          <Button
            variant="secondary"
            size="sm"
            disabled={(store.page + 1) * store.pageSize >= store.total}
            onclick={() => { store.page++; }}
          >
            <ChevronRight size={16} />
          </Button>
        </div>
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
      <Input
        id="opening-balance"
        type="number"
        bind:value={openingBalance}
        placeholder="Enter opening balance"
        min="0"
        required
      />
      <p class="text-xs text-text-muted mt-1">Amount of cash in the drawer at shift start</p>
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isSubmitting} onclick={() => { showOpenModal = false; }}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={isSubmitting} onclick={handleOpenShift}>
      {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      Open Shift
    </Button>
  {/snippet}
</Modal>

<!-- Close Shift Modal -->
<Modal bind:open={showCloseModal} title="Close Shift" size="sm">
  {#if store.activeShift}
    <div class="space-y-4">
      <div class="bg-surface-secondary rounded-lg p-4 space-y-2">
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">Opening Balance</span>
          <span class="text-text-primary font-medium">{formatMoney(store.activeShift.opening_balance)}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">Cash Sales</span>
          <span class="text-text-primary font-medium">{formatMoney(store.activeShift.cash_sales)}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">Non-Cash Sales</span>
          <span class="text-text-primary font-medium">{formatMoney(store.activeShift.non_cash_sales)}</span>
        </div>
        <div class="flex justify-between text-sm border-t border-border pt-2">
          <span class="text-text-muted">Total Sales</span>
          <span class="text-text-primary font-bold">{formatMoney(store.activeShift.total_sales)}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-text-muted">Transactions</span>
          <span class="text-text-primary font-medium">{store.activeShift.transaction_count}</span>
        </div>
        <div class="flex justify-between text-sm border-t border-border pt-2">
          <span class="text-text-muted">Expected Cash</span>
          <span class="text-primary font-bold">{formatMoney(store.activeShift.opening_balance + store.activeShift.cash_sales)}</span>
        </div>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleCloseShift(); }} class="space-y-4">
        <div>
          <label for="closing-balance" class="block text-sm font-medium text-text-secondary mb-2">
            Closing Balance (Rp)
          </label>
          <Input
            id="closing-balance"
            type="number"
            bind:value={closingBalance}
            placeholder="Enter cash counted"
            min="0"
            required
          />
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
    <Button variant="danger" class="px-5" disabled={isSubmitting} onclick={handleCloseShift}>
      {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      Close Shift
    </Button>
  {/snippet}
</Modal>

<!-- Detail Modal -->
<Modal bind:open={showDetailModal} title="Shift Detail" size="md">
  {#if selectedShift}
    <div class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <p class="text-xs text-text-muted">Cashier</p>
          <p class="text-sm font-medium text-text-primary">{selectedShift.username || '-'}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Store</p>
          <p class="text-sm font-medium text-text-primary">{selectedShift.store_name || '-'}</p>
        </div>
        <div>
          <p class="text-xs text-text-muted">Status</p>
          <Badge variant={selectedShift.status === 'open' ? 'success' : 'secondary'}>
            {selectedShift.status === 'open' ? 'Open' : 'Closed'}
          </Badge>
        </div>
        <div>
          <p class="text-xs text-text-muted">Opened At</p>
          <p class="text-sm font-medium text-text-primary">{formatDateTime(selectedShift.opened_at)}</p>
        </div>
        {#if selectedShift.closed_at}
          <div>
            <p class="text-xs text-text-muted">Closed At</p>
            <p class="text-sm font-medium text-text-primary">{formatDateTime(selectedShift.closed_at)}</p>
          </div>
        {/if}
      </div>

      <div class="border-t border-border pt-4 space-y-3">
        <div class="flex justify-between">
          <span class="text-sm text-text-muted">Opening Balance</span>
          <span class="text-sm font-medium text-text-primary">{formatMoney(selectedShift.opening_balance)}</span>
        </div>
        {#if selectedShift.closing_balance !== null}
          <div class="flex justify-between">
            <span class="text-sm text-text-muted">Closing Balance</span>
            <span class="text-sm font-medium text-text-primary">{formatMoney(selectedShift.closing_balance)}</span>
          </div>
        {/if}
        <div class="flex justify-between">
          <span class="text-sm text-text-muted">Cash Sales</span>
          <span class="text-sm font-medium text-text-primary">{formatMoney(selectedShift.cash_sales)}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-sm text-text-muted">Non-Cash Sales</span>
          <span class="text-sm font-medium text-text-primary">{formatMoney(selectedShift.non_cash_sales)}</span>
        </div>
        <div class="flex justify-between border-t border-border pt-2">
          <span class="text-sm font-medium text-text-primary">Total Sales</span>
          <span class="text-sm font-bold text-text-primary">{formatMoney(selectedShift.total_sales)}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-sm text-text-muted">Transactions</span>
          <span class="text-sm font-medium text-text-primary">{selectedShift.transaction_count}</span>
        </div>
        {#if selectedShift.discrepancy !== null}
          <div class="flex justify-between border-t border-border pt-2">
            <span class="text-sm font-medium text-text-primary">Discrepancy</span>
            <span class="text-sm font-bold {selectedShift.discrepancy === 0 ? 'text-success' : 'text-danger'}">
              {selectedShift.discrepancy > 0 ? '+' : ''}{formatMoney(selectedShift.discrepancy)}
            </span>
          </div>
        {/if}
        {#if selectedShift.notes}
          <div class="border-t border-border pt-2">
            <p class="text-xs text-text-muted">Notes</p>
            <p class="text-sm text-text-primary">{selectedShift.notes}</p>
          </div>
        {/if}
      </div>
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={() => { showDetailModal = false; }}>Close</Button>
  {/snippet}
</Modal>
