<script lang="ts">
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { Button, Badge } from '$shared/ui';
  import GoodsReceiptModal from './GoodsReceiptModal.svelte';

  const store = usePurchaseOrderStore();
  let showReceiptModal = $state(false);

  $effect(() => {
    if (store.selectedPO) {
      store.loadReceipts(store.selectedPO.id);
    }
  });

  function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'danger' | 'primary' | 'muted' {
    switch (status) {
      case 'draft':
        return 'muted';
      case 'confirmed':
        return 'primary';
      case 'partial_received':
        return 'warning';
      case 'fully_received':
        return 'success';
      case 'cancelled':
      case 'rejected':
        return 'danger';
      case 'waiting_approval':
        return 'default';
      default:
        return 'muted';
    }
  }

  function formatCurrency(value: number): string {
    return value.toLocaleString('id-ID');
  }

  function formatDate(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    });
  }
</script>

{#if store.selectedPO}
  <div class="space-y-5">
    <div class="flex items-center justify-between gap-3">
      <h1 class="text-2xl font-bold text-text-primary">Purchase Order {store.selectedPO.po_number}</h1>
      <div class="flex items-center gap-2">
        {#if store.selectedPO.status === 'confirmed' || store.selectedPO.status === 'partial_received'}
          <Button variant="primary" onclick={() => showReceiptModal = true}>Receive Goods</Button>
        {/if}
        <Button variant="secondary" onclick={() => store.selectedPO = null}>Close</Button>
      </div>
    </div>

    <div class="card overflow-hidden">
      <div class="px-6 py-4 border-b border-border">
        <h2 class="text-base font-semibold text-text-primary">Information</h2>
      </div>
      <div class="p-6">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <span class="text-sm text-text-muted">Status</span>
            <div class="mt-1">
              <Badge variant={getStatusVariant(store.selectedPO.status)} size="sm">{store.selectedPO.status.replace(/_/g, ' ')}</Badge>
            </div>
          </div>
          <div>
            <span class="text-sm text-text-muted">Supplier</span>
            <div class="mt-1 text-sm font-medium text-text-primary">{(store.selectedPO as any).supplier_name || 'N/A'}</div>
          </div>
          <div>
            <span class="text-sm text-text-muted">Expected Date</span>
            <div class="mt-1 text-sm text-text-secondary">{formatDate(store.selectedPO.expected_date)}</div>
          </div>
          <div>
            <span class="text-sm text-text-muted">Payment Term</span>
            <div class="mt-1 text-sm text-text-secondary">{store.selectedPO.payment_term || '-'}</div>
          </div>
          <div class="sm:col-span-2">
            <span class="text-sm text-text-muted">Delivery Address</span>
            <div class="mt-1 text-sm text-text-secondary">{store.selectedPO.delivery_address || '-'}</div>
          </div>
          <div class="sm:col-span-2">
            <span class="text-sm text-text-muted">Notes</span>
            <div class="mt-1 text-sm text-text-secondary">{store.selectedPO.notes || '-'}</div>
          </div>
        </div>
      </div>
    </div>

    <div class="card overflow-hidden">
      <div class="px-6 py-4 border-b border-border">
        <h2 class="text-base font-semibold text-text-primary">Items</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[600px]">
          <thead class="bg-muted/50">
            <tr class="border-b text-left text-xs text-text-muted">
              <th class="px-4 py-3 font-semibold" scope="col">Product</th>
              <th class="px-4 py-3 font-semibold" scope="col">SKU</th>
              <th class="px-4 py-3 font-semibold text-right" scope="col">Qty Ordered</th>
              <th class="px-4 py-3 font-semibold text-right" scope="col">Qty Received</th>
              <th class="px-4 py-3 font-semibold text-right" scope="col">Unit Cost</th>
              <th class="px-4 py-3 font-semibold text-right" scope="col">Subtotal</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each store.selectedPO.items as item}
              <tr>
                <td class="px-4 py-3 text-sm">{item.product_name}</td>
                <td class="px-4 py-3 text-sm text-text-muted tabular-nums">{item.sku || '-'}</td>
                <td class="px-4 py-3 text-sm text-text-secondary text-right tabular-nums">{item.qty_ordered}</td>
                <td class="px-4 py-3 text-sm text-text-secondary text-right tabular-nums">{item.qty_received}</td>
                <td class="px-4 py-3 text-sm text-text-secondary text-right tabular-nums">{formatCurrency(item.unit_cost)}</td>
                <td class="px-4 py-3 text-sm text-text-secondary text-right tabular-nums">{formatCurrency(item.subtotal)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <div class="card overflow-hidden">
      <div class="px-6 py-4 border-b border-border">
        <h2 class="text-base font-semibold text-text-primary">Receiving History</h2>
      </div>
      <div class="p-6">
        {#if store.receipts.length === 0}
          <p class="text-text-muted text-sm">No receipts yet</p>
        {:else}
          <div class="space-y-4">
            {#each store.receipts as receipt}
              <div class="border border-border rounded-xl p-4">
                <div class="flex items-center justify-between gap-4">
                  <div>
                    <div class="font-medium text-text-primary">{receipt.gr_number}</div>
                    <div class="text-sm text-text-muted mt-0.5">Received at: {new Date(receipt.received_at).toLocaleString('id-ID')}</div>
                    {#if receipt.delivery_order_number}
                      <div class="text-sm text-text-muted mt-0.5">DO: {receipt.delivery_order_number}</div>
                    {/if}
                  </div>
                  <div class="text-right">
                    <div class="text-sm font-medium text-text-primary">{receipt.items.length} item(s)</div>
                    <div class="text-sm text-text-muted mt-0.5">
                      Good: {receipt.items.reduce((sum: number, i: any) => sum + i.qty_good, 0)} | 
                      Damaged: {receipt.items.reduce((sum: number, i: any) => sum + i.qty_damaged, 0)}
                    </div>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>

  {#if showReceiptModal && store.selectedPO}
    <GoodsReceiptModal poId={store.selectedPO.id!} bind:open={showReceiptModal} />
  {/if}
{:else}
  <div class="card p-8 text-center text-text-muted">Select a purchase order to view details</div>
{/if}
