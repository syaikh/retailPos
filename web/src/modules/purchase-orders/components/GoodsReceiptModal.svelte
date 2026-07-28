<script lang="ts">
  import { onMount } from 'svelte';
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { getPurchaseOrderById } from '../services/po-service';
  import { toast } from '$shared/stores/toast.svelte';

  const store = usePurchaseOrderStore();
  let { poId } = $props();
  let po = $state<any>(null);
  let items = $state<any[]>([]);
  let saving = $state(false);
  let deliveryOrderNumber = $state('');
  let notes = $state('');

  onMount(async () => {
    po = await getPurchaseOrderById(poId);
    if (po) {
      items = po.items.map((item: any) => ({
        purchase_order_item_id: item.id,
        product_id: item.product_id,
        qty_good: 0,
        qty_damaged: 0,
        product_name: item.product_name,
        sku: item.sku,
        qty_ordered: item.qty_ordered,
        qty_received: item.qty_received,
        unit_cost: item.unit_cost,
      }));
    }
  });

  function updateItem(index: number, field: string, value: any) {
    items = items.map((item: any, i: number) => i === index ? { ...item, [field]: value } : item);
  }

  function getRemainingQty(item: any) {
    return (item.qty_ordered || 0) - (item.qty_received || 0);
  }

  function getTotalGood() {
    return items.reduce((sum: number, item: any) => sum + (item.qty_good || 0), 0);
  }

  async function handleSubmit() {
    saving = true;
    try {
      const validItems = items.filter((item: any) => item.qty_good > 0 || item.qty_damaged > 0);
      if (validItems.length === 0) {
        toast.error('Please enter receiving quantities');
        return;
      }
      await store.receive({
        purchase_order_id: poId,
        delivery_order_number: deliveryOrderNumber,
        notes,
        items: validItems,
      });
      toast.success('Goods receipt created');
      po = null;
      items = [];
      deliveryOrderNumber = '';
      notes = '';
    } catch (e: any) {
      toast.error(e.message || 'Failed to create goods receipt');
    } finally {
      saving = false;
    }
  }

  function handleClose() {
    po = null;
    items = [];
  }
</script>

<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
  <div class="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-y-auto">
    <div class="p-6 border-b flex justify-between items-center">
      <h2 class="text-xl font-bold">Receive Goods - {po?.po_number || ''}</h2>
      <button onclick={handleClose} class="text-gray-400 hover:text-gray-600">&times;</button>
    </div>

    <div class="p-6 space-y-6">
      {#if !po}
        <div class="p-8 text-center text-gray-500">Loading...</div>
      {:else}
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Delivery Order Number</label>
            <input type="text" bind:value={deliveryOrderNumber} placeholder="DO-001" class="w-full px-3 py-2 border rounded-lg" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Notes</label>
            <input type="text" bind:value={notes} placeholder="Receiving notes" class="w-full px-3 py-2 border rounded-lg" />
          </div>
        </div>

        <div>
          <h3 class="text-lg font-medium mb-2">Receiving Items</h3>
          <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Product</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">SKU</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Ordered</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Remaining</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Qty Good</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Qty Damaged</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200">
              {#each items as item, index}
                <tr>
                  <td class="px-4 py-3 text-sm">{item.product_name}</td>
                  <td class="px-4 py-3 text-sm text-gray-500">{item.sku || '-'}</td>
                  <td class="px-4 py-3 text-sm">{item.qty_ordered}</td>
                  <td class="px-4 py-3 text-sm">{getRemainingQty(item)}</td>
                  <td class="px-4 py-3">
                    <input type="number" min="0" max={getRemainingQty(item)} bind:value={item.qty_good} class="w-20 px-2 py-1 border rounded" />
                  </td>
                  <td class="px-4 py-3">
                    <input type="number" min="0" bind:value={item.qty_damaged} class="w-20 px-2 py-1 border rounded" />
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <div class="flex justify-between items-center pt-4 border-t">
          <div class="text-sm text-gray-600">
            Total Good: {getTotalGood()}
          </div>
          <div class="flex gap-2">
            <button onclick={handleClose} class="px-4 py-2 border rounded-lg hover:bg-gray-50">Cancel</button>
            <button onclick={handleSubmit} disabled={saving || getTotalGood() === 0} class="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50">
              {saving ? 'Saving...' : 'Create Goods Receipt'}
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
