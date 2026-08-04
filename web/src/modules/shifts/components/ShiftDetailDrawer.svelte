<script lang="ts">
  import { Badge, Button, Drawer } from '$shared/ui';
  import { CheckCircle, Clock } from 'lucide-svelte';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import type { Shift } from '../types';

  let {
    selectedShift,
    showDetailDrawer = $bindable(),
    canReview = false,
    canAudit = false,
    onreview = () => {},
    onaudit = () => {},
  }: {
    selectedShift: Shift | null;
    showDetailDrawer: boolean;
    canReview?: boolean;
    canAudit?: boolean;
    onreview?: () => void;
    onaudit?: () => void;
  } = $props();

  function formatMoney(amount: number) {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount);
  }

  function formatDateTime(dateStr: string | null) {
    if (!dateStr) return '-';
    return formatDateTimeInJakarta(dateStr);
  }
</script>

<Drawer bind:open={showDetailDrawer} width={480} ariaLabel="Shift detail">
  {#if selectedShift}
    <div class="space-y-4">
      <div class="flex items-center gap-3 mb-2">
        <h2 class="text-lg font-bold text-text-primary">Shift Detail</h2>
        <Badge variant={selectedShift.status === 'open' ? 'success' : 'muted'}>
          {selectedShift.status === 'open' ? 'Open' : 'Closed'}
        </Badge>
      </div>

      <div class="rounded-2xl bg-surface-default border border-border overflow-hidden">
        <div class="px-4 py-3 grid grid-cols-2 gap-x-6 gap-y-3">
          <div>
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Cashier</span>
            <p class="text-text-secondary text-sm mt-0.5">{selectedShift.username || '-'}</p>
          </div>
          <div>
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Store</span>
            <p class="text-text-secondary text-sm mt-0.5">{selectedShift.store_name || '-'}</p>
          </div>
          <div>
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Opened At</span>
            <p class="text-text-secondary text-sm mt-0.5">{formatDateTime(selectedShift.opened_at)}</p>
          </div>
          {#if selectedShift.closed_at}
            <div>
              <span class="text-xs font-semibold tracking-wide text-text-muted/80">Closed At</span>
              <p class="text-text-secondary text-sm mt-0.5">{formatDateTime(selectedShift.closed_at)}</p>
            </div>
          {/if}
        </div>
      </div>

      <div class="rounded-2xl bg-surface-default border border-border overflow-hidden">
        <div class="px-4 py-2 border-b border-border/60 flex items-center gap-1.5">
          <span class="text-base leading-none">💰</span>
          <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Financial Summary</h4>
        </div>
        <div class="p-4 space-y-3">
          <div class="flex justify-between">
            <span class="text-sm text-text-muted">Opening Balance</span>
            <span class="text-sm font-medium text-text-primary">{formatMoney(selectedShift.opening_balance)}</span>
          </div>
          {#if selectedShift.closing_balance != null}
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
          {#if selectedShift.discrepancy != null}
            <div class="flex justify-between border-t border-border pt-2">
              <span class="text-sm font-medium text-text-primary">Discrepancy</span>
              <span class="text-sm font-bold {selectedShift.discrepancy === 0 ? 'text-success' : 'text-danger'}">
                {selectedShift.discrepancy > 0 ? '+' : ''}{formatMoney(selectedShift.discrepancy)}
              </span>
            </div>
          {/if}
        </div>
      </div>

      {#if selectedShift.notes}
        <div class="rounded-2xl bg-surface-default border border-border overflow-hidden">
          <div class="px-4 py-2 border-b border-border/60 flex items-center gap-1.5">
            <span class="text-base leading-none">📝</span>
            <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Notes</h4>
          </div>
          <div class="px-4 py-3">
            <p class="text-text-secondary text-sm whitespace-pre-wrap break-words">{selectedShift.notes}</p>
          </div>
        </div>
      {/if}
    </div>
  {/if}

  {#snippet footer()}
    {#if selectedShift}
      <div class="flex flex-col gap-3">
        {#if selectedShift.needs_review && canReview}
          <Button variant="primary" class="w-full rounded-xl h-11" onclick={onreview}>
            <CheckCircle size={16} class="mr-2" />
            Review & Approve
          </Button>
        {/if}
        {#if selectedShift.status === 'closed' && canAudit}
          <Button variant="secondary" class="w-full rounded-xl h-11" onclick={onaudit}>
            <Clock size={16} class="mr-2" />
            Surprise Audit
          </Button>
        {/if}
      </div>
    {/if}
  {/snippet}
</Drawer>
