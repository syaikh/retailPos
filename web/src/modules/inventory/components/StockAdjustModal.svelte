<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    stockAdjustProduct = null as {name: string, stock: number} | null,
    stockAdjustForm = $bindable({
      product_id: null as number | null,
      quantity_change: 0,
      notes: ''
    }),
    adjustingStock = false,
    onSubmit,
    onCancel,
  } = $props();

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    onSubmit();
  }
</script>

<Modal bind:open title="Adjust Stock" size="sm">
  <form
    onsubmit={handleSubmit}
    class="space-y-4"
  >
    {#if stockAdjustProduct}
      <div>
        <p class="text-sm text-text-muted mb-2">Product: <span class="text-text-primary font-medium">{stockAdjustProduct.name}</span></p>
        <p class="text-sm text-text-muted">Current Stock: <span class="text-text-primary">{stockAdjustProduct.stock ?? 0}</span></p>
      </div>
    {/if}
    <div>
      <label for="adjust-qty" class="block text-sm font-medium text-text-secondary mb-2">Quantity Change</label>
      <Input
        id="adjust-qty"
        type="number"
        bind:value={stockAdjustForm.quantity_change}
        placeholder="e.g., +10 or -5"
        required
      />
      <p class="text-xs text-text-muted mt-1">Positive to add stock, negative to reduce.</p>
    </div>
    <div>
      <label for="adjust-notes" class="block text-sm font-medium text-text-secondary mb-2">Notes <span class="text-destructive">*</span></label>
      <Input
        id="adjust-notes"
        type="text"
        bind:value={stockAdjustForm.notes}
        placeholder="Reason for adjustment (required)"
        required
      />
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={adjustingStock} onclick={onCancel}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={adjustingStock} onclick={onSubmit}>
      {#if adjustingStock}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      Adjust Stock
    </Button>
  {/snippet}
</Modal>
