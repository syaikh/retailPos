<script lang="ts">
  import { onMount } from 'svelte';
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { getProducts } from '$modules/product/services/product-service';
  import { getSuppliers } from '$modules/supplier/services/supplier-service';
  import { Button, Input, Modal } from '$shared/ui';
  import { toast } from '$shared/stores/toast.svelte';
  import { Loader2 } from 'lucide-svelte';

  const store = usePurchaseOrderStore();

  let {
    open = $bindable(false),
  }: {
    open?: boolean;
  } = $props();

  let saving = $state(false);

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

  $effect(() => {
    if (store.selectedPO) {
      po = { ...store.selectedPO, items: store.selectedPO.items?.map((item: any) => ({ ...item })) || [] };
      open = true;
    }
  });

  onMount(async () => {
    const [prodRes, suppRes] = await Promise.all([
      getProducts({ limit: 100 } as any),
      getSuppliers({ limit: 100 } as any),
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
      open = false;
    } catch (e: any) {
      toast.error(e.message || 'Failed to save purchase order');
    } finally {
      saving = false;
    }
  }

  function handleClose() {
    open = false;
    store.selectedPO = null;
  }
</script>

<Modal bind:open title={store.selectedPO ? 'Edit Purchase Order' : 'Create Purchase Order'} size="xl">
  <div class="space-y-6">
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1.5">Supplier</label>
        <Input tag="select" bind:value={po.supplier_id}>
          <option value={0}>Select supplier</option>
          {#each suppliers as s}
            <option value={s.id}>{s.name}</option>
          {/each}
        </Input>
      </div>
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1.5">Expected Date</label>
        <Input type="date" bind:value={po.expected_date} />
      </div>
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1.5">Payment Term</label>
        <Input type="text" bind:value={po.payment_term} placeholder="e.g. Net 30" />
      </div>
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1.5">Supplier Reference Number</label>
        <Input type="text" bind:value={po.supplier_reference_number} />
      </div>
      <div class="sm:col-span-2">
        <label class="block text-sm font-medium text-text-secondary mb-1.5">Delivery Address</label>
        <Input tag="textarea" bind:value={po.delivery_address} rows={2} />
      </div>
      <div class="sm:col-span-2">
        <label class="block text-sm font-medium text-text-secondary mb-1.5">Notes</label>
        <Input tag="textarea" bind:value={po.notes} rows={2} />
      </div>
    </div>

    <div>
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-base font-semibold text-text-primary">Items</h3>
        <Button variant="secondary" size="sm" onclick={addItem}>Add Item</Button>
      </div>

      {#if po.items.length === 0}
        <p class="text-text-muted text-sm">No items added</p>
      {:else}
        <div class="overflow-x-auto border border-border rounded-xl">
          <table class="w-full min-w-[600px]">
            <thead class="bg-muted/50">
              <tr class="border-b text-left text-xs text-text-muted">
                <th class="px-3 py-2 font-semibold" scope="col">Product</th>
                <th class="px-3 py-2 font-semibold" scope="col">Qty</th>
                <th class="px-3 py-2 font-semibold" scope="col">Unit Cost</th>
                <th class="px-3 py-2 font-semibold" scope="col">Discount</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">Subtotal</th>
                <th class="px-3 py-2 w-[50px]" scope="col"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each po.items as item, index}
                <tr>
                  <td class="px-3 py-2">
                    <Input tag="select" bind:value={item.product_id} class="text-sm">
                      <option value={0}>Select product</option>
                      {#each products as p}
                        <option value={p.id}>{p.name} ({p.sku})</option>
                      {/each}
                    </Input>
                  </td>
                  <td class="px-3 py-2">
                    <Input type="number" bind:value={item.qty_ordered} min={1} class="w-20 text-sm" />
                  </td>
                  <td class="px-3 py-2">
                    <Input type="number" bind:value={item.unit_cost} min={0} class="w-28 text-sm" />
                  </td>
                  <td class="px-3 py-2">
                    <Input type="number" bind:value={item.discount_amount} min={0} class="w-24 text-sm" />
                  </td>
                  <td class="px-3 py-3 text-sm text-text-secondary text-right tabular-nums">{calculateSubtotal(item).toLocaleString('id-ID')}</td>
                  <td class="px-3 py-2">
                    <Button variant="ghost" size="icon" onclick={() => removeItem(index)} aria-label="Remove item" class="text-danger hover:text-danger-light">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                    </Button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      <div class="text-base font-semibold text-text-primary tabular-nums">
        Total: {getTotalSubtotal().toLocaleString('id-ID')}
      </div>
      <div class="flex items-center gap-2">
        <Button variant="secondary" onclick={handleClose}>Cancel</Button>
        <Button variant="primary" onclick={handleSubmit} disabled={saving || po.items.length === 0 || po.supplier_id === 0}>
          {#if saving}
            <Loader2 size={16} class="animate-spin" />
          {/if}
          {saving ? 'Saving...' : (store.selectedPO ? 'Update' : 'Create Draft')}
        </Button>
      </div>
    </div>
  {/snippet}
</Modal>
