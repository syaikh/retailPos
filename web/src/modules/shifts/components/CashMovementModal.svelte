<script lang="ts">
  import { Button, CurrencyInput, Input, Modal } from '$shared/ui';
  import { labels } from '$shared/i18n';
  import { Loader2 } from 'lucide-svelte';
  import { recordCashMovement } from '../services/shift-service';
  import type { CashMovement } from '../types';

  let {
    open = $bindable(),
    shiftId,
    onrecord = () => {},
  }: {
    open: boolean;
    shiftId: number;
    onrecord?: (movement: CashMovement) => void;
  } = $props();

  let movementType = $state<'cash_drop' | 'paid_in' | 'paid_out'>('paid_in');
  let amount = $state(0);
  let description = $state('');
  let isSubmitting = $state(false);

  const typeOptions = [
    { value: 'paid_in' as const, label: labels.paidIn, desc: labels.paidInDesc, icon: '📥' },
    { value: 'paid_out' as const, label: labels.paidOut, desc: labels.paidOutDesc, icon: '📤' },
    { value: 'cash_drop' as const, label: labels.cashDrop, desc: labels.cashDropDesc, icon: '🏦' },
  ];

  const isValid = $derived(amount > 0);

  function reset() {
    movementType = 'paid_in';
    amount = 0;
    description = '';
  }

  async function handleSubmit() {
    if (!isValid || isSubmitting) return;
    isSubmitting = true;
    try {
      const movement = await recordCashMovement(shiftId, movementType, amount, description || null);
      onrecord(movement);
      open = false;
      reset();
    } catch (e: any) {
      alert(e?.response?.data?.error || labels.cashMovementFailed);
    } finally {
      isSubmitting = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !isSubmitting) {
      open = false;
      reset();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<Modal bind:open title={labels.recordCashMovement} size="sm">
  <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-4">
    <div>
      <label class="block text-sm font-medium text-text-secondary mb-2">{labels.cashMovementType}</label>
      <div class="grid grid-cols-3 gap-2">
        {#each typeOptions as opt}
          <button
            type="button"
            class="flex flex-col items-center gap-1 p-3 rounded-xl border-2 transition-all text-center
              {movementType === opt.value
                ? 'border-primary bg-primary/5 text-primary'
                : 'border-border hover:border-border-strong text-text-muted hover:text-text-secondary'}"
            onclick={() => movementType = opt.value}
          >
            <span class="text-lg">{opt.icon}</span>
            <span class="text-xs font-medium">{opt.label}</span>
          </button>
        {/each}
      </div>
      <p class="text-xs text-text-muted mt-1.5">
        {typeOptions.find(o => o.value === movementType)?.desc}
      </p>
    </div>

    <div>
      <label for="cash-movement-amount" class="block text-sm font-medium text-text-secondary mb-2">
        {labels.cashMovementAmount}
      </label>
      <CurrencyInput id="cash-movement-amount" bind:value={amount} placeholder="0" required />
    </div>

    <div>
      <label for="cash-movement-desc" class="block text-sm font-medium text-text-secondary mb-2">
        {labels.cashMovementDescription}
      </label>
      <Input id="cash-movement-desc" bind:value={description} placeholder={labels.cashMovementDescription} />
    </div>
  </form>

  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isSubmitting} onclick={() => { open = false; reset(); }}>
      {labels.cancel}
    </Button>
    <Button variant="primary" class="px-5" disabled={isSubmitting || !isValid} onclick={handleSubmit}>
      {#if isSubmitting}<Loader2 size={16} class="animate-spin mr-2" />{/if}
      {labels.save}
    </Button>
  {/snippet}
</Modal>
