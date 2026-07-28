<script lang="ts">
  import { onMount } from 'svelte';
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { getProducts } from '$modules/product/services/product-service';
  import { getSuppliers } from '$modules/supplier/services/supplier-service';
  import { toast } from '$shared/stores/toast.svelte';

  const store = usePurchaseOrderStore();
  let po = $state<any>(store.selectedPO || {
    supplier_id: 0,
    expected_date: '',
    payment_term: '',
    delivery_address: '',
    supplier_reference_number: '',
    notes: '',
    items: [],
  });
  let products = $state<any[]>([]);
  let suppliers = $state<any[]>([]);
  let saving = $state(false);

  onMount(async () => {
    const [prodRes, suppRes] = await Promise.all([
      getProducts({ limit: 100 }),
      getSuppliers({ limit: 100 }),
    ]);
    products = prodRes.data || [];
    suppliers = suppRes.data || [];
  });

  function addItem() {
    po.items = [...po.items, { product_id: 0, qty_ordered: 1, unit_cost: 0, discount_amount: 0 }];
  }

  function removeItem(index: number) {
    po.items = po.items.filter((_: any, i: number) => i !== index);
  }

  function updateItem(index: number, field: string, value: any) {
    po.items = po.items.map((item: any, i: number) => i === index ? { ...item, [field]: value } : item);
  }

  function calculateSubtotal(item: any) {
    return item.qty_ordered * item.unit_cost - item.discount_amount;
  }

  function getTotalSubtotal() {
    return po.items.reduce((sum: number, item: any) => sum + calculateSubtotal(item), 0);
  }

  async function handleSubmit() {
    saving = true;
    try {
      if (store.selectedPO) {
        await store.update(store.selectedPO.id, po);
        toast.success('Purchase order updated');
      } else {
        await store.create(po);
        toast.success('Purchase order created');
      }
    } catch (e: any) {
      toast.error(e.message || 'Failed to save purchase order');
    } finally {
      saving = false;
    }
  }

  function handleClose() {
    store.selectedPO = null;
  }
</script>

<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
  <div class="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-y-auto">
    <div class="p-6 border-b flex justify-between items-center">
      <h2 class="text-xl font-bold">{store.selectedPO ? 'Edit Purchase Order' : 'Create Purchase Order'}</h2>
      <button onclick={handleClose} class="text-gray-400 hover:text-gray-600">&times;</button>
    </div>

    <div class="p-6 space-y-6">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Supplier</label>
          <select bind:value={po.supplier_id} class="w-full px-3 py-2 border rounded-lg">
            <option value={0}>Select supplier</option>
            {#each suppliers as s}
              <option value={s.id}>{s.name}</option>
            {/each}
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Expected Date</label>
          <input type="date" bind:value={po.expected_date} class="w-full px-3 py-2 border rounded-lg" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Payment Term</label>
          <input type="text" bind:value={po.payment_term} placeholder="e.g. Net 30" class="w-full px-3 py-2 border rounded-lg" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Supplier Reference Number</label>
          <input type="text" bind:value={po.supplier_reference_number} class="w-full px-3 py-2 border rounded-lg" />
        </div>
        <div class="col-span-2">
          <label class="block text-sm font-medium text-gray-700 mb-1">Delivery Address</label>
          <textarea bind:value={po.delivery_address} rows="2" class="w-full px-3 py-2 border rounded-lg"></textarea>
        </div>
        <div class="col-span-2">
          <label class="block text-sm font-medium text-gray-700 mb-1">Notes</label>
          <textarea bind:value={po.notes} rows="2" class="w-full px-3 py-2 border rounded-lg"></textarea>
        </div>
      </div>

      <div>
        <div class="flex justify-between items-center mb-2">
          <h3 class="text-lg font-medium">Items</h3>
          <button onclick={addItem} class="px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 text-sm">
            Add Item
          </button>
        </div>

        {#if po.items.length === 0}
          <p class="text-gray-500 text-sm">No items added</p>
        {:else}
          <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Product</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Qty</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Unit Cost</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Discount</th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Subtotal</th>
                <th class="px-4 py-2"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200">
              {#each po.items as item, index}
                <tr>
                  <td class="px-4 py-2">
                    <select bind:value={item.product_id} class="w-full px-2 py-1 border rounded">
                      <option value={0}>Select product</option>
                      {#each products as p}
                        <option value={p.id}>{p.name} ({p.sku})</option>
                      {/each}
                    </select>
                  </td>
                  <td class="px-4 py-2">
                    <input type="number" bind:value={item.qty_ordered} min="1" class="w-20 px-2 py-1 border rounded" />
                  </td>
                  <td class="px-4 py-2">
                    <input type="number" bind:value={item.unit_cost} min="0" class="w-32 px-2 py-1 border rounded" />
                  </td>
                  <td class="px-4 py-2">
                    <input type="number" bind:value={item.discount_amount} min="0" class="w-24 px-2 py-1 border rounded" />
                  </td>
                  <td class="px-4 py-2 text-sm">{calculateSubtotal(item).toLocaleString('id-ID')}</td>
                  <td class="px-4 py-2">
                    <button onclick={() => removeItem(index)} class="text-red-600 hover:text-red-800 text-sm">Remove</button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>

      <div class="flex justify-between items-center pt-4 border-t">
        <div class="text-lg font-medium">
          Total: {getTotalSubtotal().toLocaleString('id-ID')}
        </div>
        <div class="flex gap-2">
          <button onclick={handleClose} class="px-4 py-2 border rounded-lg hover:bg-gray-50">Cancel</button>
          <button onclick={handleSubmit} disabled={saving || po.items.length === 0 || po.supplier_id === 0} class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
            {saving ? 'Saving...' : (store.selectedPO ? 'Update' : 'Create Draft')}
          </button>
        </div>
      </div>
    </div>
  </div>
</div>
