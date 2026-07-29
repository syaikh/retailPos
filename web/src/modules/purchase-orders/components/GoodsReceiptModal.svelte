<script lang="ts">
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { getPurchaseOrderById } from '../services/po-service';
  import { Button, Input, Modal } from '$shared/ui';
  import { toast } from '$shared/stores/toast.svelte';
  import { Loader2 } from 'lucide-svelte';

  const store = usePurchaseOrderStore();
  let { poId, open = $bindable(false) }: { poId: number | null | undefined; open?: boolean } = $props();
  let po = $state<any>(null);
  let items = $state<any[]>([]);
  let saving = $state(false);
  let deliveryOrderNumber = $state('');
  let notes = $state('');

  $effect(() => {
    if (!open || !poId) return;
    po = null;
    items = [];
    getPurchaseOrderById(poId).then(result => {
      po = result;
      if (result) {
        items = result.items.map((item: any) => ({
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
      handleClose();
    } catch (e: any) {
      toast.error(e.message || 'Failed to create goods receipt');
    } finally {
      saving = false;
    }
  }

  function handleClose() {
    open = false;
    po = null;
    items = [];
    deliveryOrderNumber = '';
    notes = '';
  }
</script>

<Modal bind:open title="Receive Goods - {po?.po_number || ''}" size="xl">
  {#if !po}
    <div class="flex items-center justify-center py-8">
      <Loader2 size={24} class="animate-spin text-primary-light" />
    </div>
  {:else}
    <div class="space-y-6">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
            <span>Delivery Order Number</span>
            <Input type="text" bind:value={deliveryOrderNumber} placeholder="DO-001" />
          </label>
        </div>
        <div>
          <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
            <span>Notes</span>
            <Input type="text" bind:value={notes} placeholder="Receiving notes" />
          </label>
        </div>
      </div>

      <div>
        <h3 class="text-base font-semibold text-text-primary mb-3">Receiving Items</h3>
        <div class="overflow-x-auto border border-border rounded-xl">
          <table class="w-full min-w-[600px]">
            <thead class="bg-muted/50">
              <tr class="border-b text-left text-xs text-text-muted">
                <th class="px-3 py-2 font-semibold" scope="col">Product</th>
                <th class="px-3 py-2 font-semibold" scope="col">SKU</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">Ordered</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">Remaining</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">Qty Good</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">Qty Damaged</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each items as item, index}
                <tr>
                  <td class="px-3 py-3 text-sm">{item.product_name}</td>
                  <td class="px-3 py-3 text-sm text-text-muted tabular-nums">{item.sku || '-'}</td>
                  <td class="px-3 py-3 text-sm text-text-secondary text-right tabular-nums">{item.qty_ordered}</td>
                  <td class="px-3 py-3 text-sm text-text-secondary text-right tabular-nums">{getRemainingQty(item)}</td>
                  <td class="px-3 py-3">
                    <Input type="number" min={0} max={getRemainingQty(item)} bind:value={item.qty_good} class="w-20 text-sm ml-auto" />
                  </td>
                  <td class="px-3 py-3">
                    <Input type="number" min={0} bind:value={item.qty_damaged} class="w-20 text-sm ml-auto" />
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      <div class="text-sm text-text-secondary">
        Total Good: <span class="font-semibold text-text-primary tabular-nums">{getTotalGood()}</span>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="secondary" onclick={handleClose}>Cancel</Button>
        <Button variant="primary" onclick={handleSubmit} disabled={saving || getTotalGood() === 0}>
          {#if saving}
            <Loader2 size={16} class="animate-spin" />
          {/if}
          {saving ? 'Saving...' : 'Create Goods Receipt'}
        </Button>
      </div>
    </div>
  {/snippet}
</Modal>
