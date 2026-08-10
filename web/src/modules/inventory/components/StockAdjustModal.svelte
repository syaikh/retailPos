<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { Loader2 } from 'lucide-svelte';
  import { labels } from '$shared/i18n';

  let {
    open = $bindable(false),
    stockAdjustProduct = $bindable(null as {name: string, stock: number} | null),
    productId = $bindable(null as number | null),
    quantityChange = $bindable(0),
    notes = $bindable(''),
    adjustingStock = false,
    onSubmit,
    onCancel,
  } = $props();

  let fieldErrors = $state<Record<string, string>>({});

  function validate(): boolean {
    const errors: Record<string, string> = {};
    if (quantityChange === 0) errors.quantity = labels.quantityChangeMustBeNonZero;
    if (!notes.trim()) errors.notes = labels.notesRequiredProvideReason;
    fieldErrors = errors;
    return Object.keys(errors).length === 0;
  }

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!validate()) return;
    onSubmit();
  }
</script>

<Modal bind:open title={labels.adjustStock} size="sm">
  <form
    onsubmit={handleSubmit}
    class="space-y-4"
  >
    {#if stockAdjustProduct}
      <div>
        <p class="text-sm text-text-muted mb-2">{labels.product}: <span class="text-text-primary font-medium">{stockAdjustProduct.name}</span></p>
        <p class="text-sm text-text-muted">{labels.stokSaatIni} <span class="text-text-primary">{stockAdjustProduct.stock ?? 0}</span></p>
      </div>
    {/if}
    <div>
      <label for="adjust-qty" class="block text-sm font-medium text-text-secondary mb-2">{labels.quantityChange}</label>
      <Input
        id="adjust-qty"
        type="number"
        bind:value={quantityChange}
        placeholder={labels.adjustQtyPlaceholder}
        error={fieldErrors.quantity}
        required
      />
      <p class="text-xs text-text-muted mt-1">{labels.positiveToAddNegativeToReduce}</p>
    </div>
    <div>
      <label for="adjust-notes" class="block text-sm font-medium text-text-secondary mb-2">{labels.notes} <span class="text-destructive">*</span></label>
      <Input
        id="adjust-notes"
        type="text"
        bind:value={notes}
        placeholder={labels.reasonForAdjustment}
        error={fieldErrors.notes}
        required
      />
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={adjustingStock} onclick={onCancel}>{labels.cancel}</Button>
    <Button variant="primary" class="px-5" disabled={adjustingStock} onclick={onSubmit}>
      {#if adjustingStock}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      {labels.adjustStock}
    </Button>
  {/snippet}
</Modal>
