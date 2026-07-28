<script lang="ts">
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import GoodsReceiptModal from './GoodsReceiptModal.svelte';

  const store = usePurchaseOrderStore();
  let showReceiptModal = $state(false);

  $effect(() => {
    if (store.selectedPO) {
      store.loadReceipts(store.selectedPO.id);
    }
  });

  function getStatusBadge(status: string) {
    const colors: Record<string, string> = {
      draft: 'bg-gray-100 text-gray-700',
      confirmed: 'bg-blue-100 text-blue-700',
      partial_received: 'bg-yellow-100 text-yellow-700',
      fully_received: 'bg-green-100 text-green-700',
      cancelled: 'bg-red-100 text-red-700',
    };
    return colors[status] || 'bg-gray-100 text-gray-700';
  }
</script>

{#if store.selectedPO}
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Purchase Order {store.selectedPO.po_number}</h1>
      <div class="flex gap-2">
        {#if store.selectedPO.status === 'confirmed' || store.selectedPO.status === 'partial_received'}
          <button onclick={() => showReceiptModal = true} class="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700">
            Receive Goods
          </button>
        {/if}
        <button onclick={() => store.selectedPO = null} class="px-4 py-2 border rounded-lg hover:bg-gray-50">Close</button>
      </div>
    </div>

    <div class="bg-white rounded-lg shadow mb-6">
      <div class="p-6 border-b">
        <h2 class="text-lg font-medium mb-4">Information</h2>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <span class="text-sm text-gray-500">Status</span>
            <div class="mt-1">
              <span class="px-2 py-1 text-xs font-medium rounded-full {getStatusBadge(store.selectedPO.status)}">
                {store.selectedPO.status}
              </span>
            </div>
          </div>
          <div>
            <span class="text-sm text-gray-500">Supplier</span>
            <div class="mt-1 text-sm font-medium">{store.selectedPO.supplier_name || 'N/A'}</div>
          </div>
          <div>
            <span class="text-sm text-gray-500">Expected Date</span>
            <div class="mt-1 text-sm">{store.selectedPO.expected_date || '-'}</div>
          </div>
          <div>
            <span class="text-sm text-gray-500">Payment Term</span>
            <div class="mt-1 text-sm">{store.selectedPO.payment_term || '-'}</div>
          </div>
          <div class="col-span-2">
            <span class="text-sm text-gray-500">Delivery Address</span>
            <div class="mt-1 text-sm">{store.selectedPO.delivery_address || '-'}</div>
          </div>
          <div class="col-span-2">
            <span class="text-sm text-gray-500">Notes</span>
            <div class="mt-1 text-sm">{store.selectedPO.notes || '-'}</div>
          </div>
        </div>
      </div>
    </div>

    <div class="bg-white rounded-lg shadow mb-6">
      <div class="p-6 border-b">
        <h2 class="text-lg font-medium mb-4">Items</h2>
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Product</th>
              <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">SKU</th>
              <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Qty Ordered</th>
              <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Qty Received</th>
              <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Unit Cost</th>
              <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Subtotal</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200">
            {#each store.selectedPO.items as item}
              <tr>
                <td class="px-4 py-3 text-sm">{item.product_name}</td>
                <td class="px-4 py-3 text-sm text-gray-500">{item.sku || '-'}</td>
                <td class="px-4 py-3 text-sm">{item.qty_ordered}</td>
                <td class="px-4 py-3 text-sm">{item.qty_received}</td>
                <td class="px-4 py-3 text-sm">{item.unit_cost.toLocaleString('id-ID')}</td>
                <td class="px-4 py-3 text-sm">{item.subtotal.toLocaleString('id-ID')}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <div class="bg-white rounded-lg shadow">
      <div class="p-6 border-b">
        <h2 class="text-lg font-medium mb-4">Receiving History</h2>
        {#if store.receipts.length === 0}
          <p class="text-gray-500 text-sm">No receipts yet</p>
        {:else}
          <div class="space-y-4">
            {#each store.receipts as receipt}
              <div class="border rounded-lg p-4">
                <div class="flex justify-between items-start">
                  <div>
                    <div class="font-medium">{receipt.gr_number}</div>
                    <div class="text-sm text-gray-500">Received at: {new Date(receipt.received_at).toLocaleString('id-ID')}</div>
                    {#if receipt.delivery_order_number}
                      <div class="text-sm text-gray-500">DO: {receipt.delivery_order_number}</div>
                    {/if}
                  </div>
                  <div class="text-right">
                    <div class="text-sm font-medium">{receipt.items.length} item(s)</div>
                    <div class="text-sm text-gray-500">
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

  {#if showReceiptModal}
    <GoodsReceiptModal poId={store.selectedPO.id} on:close={() => showReceiptModal = false} />
  {/if}
{:else}
  <div class="p-6 text-center text-gray-500">Select a purchase order to view details</div>
{/if}
