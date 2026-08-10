<script lang="ts">
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { getPurchaseOrderById } from '../services/po-service';
  import { Button, Input, Modal } from '$shared/ui';
  import { toast } from '$shared/stores/toast.svelte';
  import { labels, t } from '$shared/i18n';
  import { Loader2 } from 'lucide-svelte';

  const store = usePurchaseOrderStore();
  let { poId, open = $bindable(false), onReceiptCreated }: { poId: number | null | undefined; open?: boolean; onReceiptCreated?: () => void } = $props();
  let po = $state<any>(null);
  let items = $state<any[]>([]);
  let saving = $state(false);
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
        toast.error(labels.pleaseEnterReceivingQuantities);
        return;
      }
      const result = await store.receive({
        purchase_order_id: poId,
        notes,
        items: validItems,
      });
      const doNumber = result?.data?.delivery_order_number;
      toast.success(doNumber ? t('goodsReceiptCreatedWithNumber', { number: doNumber }) : labels.goodsReceiptCreated);
      onReceiptCreated?.();
      handleClose();
    } catch (e: any) {
      toast.error(e.message || labels.failedToCreateGoodsReceipt);
    } finally {
      saving = false;
    }
  }

  function handleClose() {
    open = false;
    po = null;
    items = [];
    notes = '';
  }
</script>

<Modal bind:open title={po ? `${labels.receiveGoods} - ${po.po_number || ''}` : labels.receiveGoods} size="xl">
  {#if !po}
    <div class="flex items-center justify-center py-8">
      <Loader2 size={24} class="animate-spin text-primary-light" />
    </div>
  {:else}
    <div class="space-y-6">
      <div>
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.notes}</span>
          <Input type="text" bind:value={notes} placeholder={labels.receivingNotes} />
        </label>
      </div>

      <div>
        <h3 class="text-base font-semibold text-text-primary mb-3">{labels.receivingItems}</h3>
        <div class="overflow-x-auto border border-border rounded-xl">
          <table class="w-full min-w-[600px]">
            <thead class="bg-muted/50">
              <tr class="border-b text-left text-xs text-text-muted">
                <th class="px-3 py-2 font-semibold" scope="col">{labels.product}</th>
                <th class="px-3 py-2 font-semibold" scope="col">{labels.sku}</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">{labels.ordered}</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">{labels.remaining}</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">{labels.qtyGood}</th>
                <th class="px-3 py-2 font-semibold text-right" scope="col">{labels.qtyDamaged}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each items as item, index}
                <tr>
                  <td class="px-3 py-3 text-sm">{item.product_name}</td>
                  <td class="px-3 py-3 text-sm text-text-muted tabular-nums">{item.sku || '-'}</td>
                  <td class="px-3 py-3 text-sm text-text-secondary text-right tabular-nums">{item.qty_ordered}</td>
                  <td class="px-3 py-3 text-sm text-text-secondary text-right tabular-nums">{getRemainingQty(item)}</td>
                  {#if getRemainingQty(item) <= 0}
                    <td class="px-3 py-3 text-sm text-text-muted text-right tabular-nums" colspan="2">{t('fullyReceivedWithQty', { qty: item.qty_received })}</td>
                  {:else}
                    <td class="px-3 py-3">
                      <Input type="number" min={0} max={getRemainingQty(item)} bind:value={item.qty_good} class="w-20 text-sm ml-auto" selectOnFocus oninput={() => { if (item.qty_good + item.qty_damaged > getRemainingQty(item)) item.qty_damaged = Math.max(0, getRemainingQty(item) - item.qty_good); }} />
                    </td>
                    <td class="px-3 py-3">
                      <Input type="number" min={0} max={getRemainingQty(item)} bind:value={item.qty_damaged} class="w-20 text-sm ml-auto" selectOnFocus oninput={() => { if (item.qty_good + item.qty_damaged > getRemainingQty(item)) item.qty_good = Math.max(0, getRemainingQty(item) - item.qty_damaged); }} />
                    </td>
                  {/if}
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
        {labels.totalGood}: <span class="font-semibold text-text-primary tabular-nums">{getTotalGood()}</span>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="secondary" onclick={handleClose}>{labels.cancel}</Button>
        <Button variant="primary" onclick={handleSubmit} disabled={saving || getTotalGood() === 0}>
          {#if saving}
            <Loader2 size={16} class="animate-spin" />
          {/if}
          {saving ? labels.saving : labels.createGoodsReceipt}
        </Button>
      </div>
    </div>
  {/snippet}
</Modal>
