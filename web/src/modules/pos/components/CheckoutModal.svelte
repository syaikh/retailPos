<script lang="ts">
  import { fly } from 'svelte/transition';
  import { Button, CurrencyInput } from '$shared/ui';
  import { X, Check, User, ChevronRight } from 'lucide-svelte';
  import { tick } from 'svelte';

  const quickCashPresets = [50000, 100000, 200000, 500000, 1000000];
  let dialogEl: HTMLDivElement | undefined = $state();
  let previousFocus: HTMLElement | null = null;

  let {
    showCheckoutModal = $bindable(false),
    cart = [],
    totalAmount = 0,
    subtotal = 0,
    taxAmount = 0,
    dppDisplay = 0,
    paymentMethod = $bindable('Cash'),
    paymentOptions = [],
    selectedCustomerLabel = '',
    cashReceived = $bindable(0),
    changeDue = 0,
    checkingOut = false,
    onfinalize = () => {},
    onselectcustomer = () => {},
  }: {
    showCheckoutModal: boolean;
    cart: any[];
    totalAmount: number;
    subtotal: number;
    taxAmount: number;
    dppDisplay: number;
    paymentMethod: string;
    paymentOptions: any[];
    selectedCustomerLabel: string;
    cashReceived: number;
    changeDue: number;
    checkingOut: boolean;
    onfinalize?: () => void;
    onselectcustomer?: () => void;
  } = $props();

  let totalSavings = $derived(
    cart.reduce((sum, item) => {
      if (item.discount && item.discount > 0) {
        return sum + item.discount * item.quantity;
      }
      return sum;
    }, 0)
  );

  let hasDiscountedItems = $derived(cart.some(item => item.discount && item.discount > 0));

  function close() {
    showCheckoutModal = false;
    cashReceived = 0;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
      return;
    }
    if (e.key === 'F7' && paymentMethod === 'Cash') {
      e.preventDefault();
      cashReceived = totalAmount;
      return;
    }
    if (e.key === 'Tab' && dialogEl) {
      const focusable = dialogEl.querySelectorAll<HTMLElement>(
        'button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])'
      );
      if (focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }
  }

  $effect(() => {
    cashReceived = paymentMethod === 'Cash' ? 0 : totalAmount;
  });

  $effect(() => {
    if (showCheckoutModal) {
      previousFocus = document.activeElement as HTMLElement;
      tick().then(() => {
        const firstFocusable = dialogEl?.querySelector<HTMLElement>(
          'button:not([disabled]), input:not([disabled])'
        );
        firstFocusable?.focus();
      });
    } else if (previousFocus) {
      previousFocus.focus();
      previousFocus = null;
    }
  });
</script>

{#if showCheckoutModal}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center print-modal-overlay"
    transition:fly={{ y: 40, duration: 300 }}
    onkeydown={handleKeydown}
    role="none"
  >
    <div
      class="absolute inset-0 bg-black/70 backdrop-blur-sm"
      onclick={close}
      role="presentation"
    ></div>

    <div
      bind:this={dialogEl}
      role="dialog"
      aria-modal="true"
      aria-label="Pembayaran"
      class="relative z-[55] w-full max-w-md max-h-[95vh] overflow-y-auto rounded-2xl border border-border-default bg-bg-card shadow-modal p-5"
    >
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-lg font-bold text-text-primary">Pembayaran</h2>
        <button
          onclick={close}
          class="w-8 h-8 flex items-center justify-center rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover/50 transition-colors"
          title="Tutup [F3 / Esc]"
          aria-label="Tutup"
        >
          <X size={18} />
        </button>
      </div>

      <div class="mb-3 text-center">
        {#if taxAmount > 0}
          <div class="flex justify-center gap-4 text-[11px] text-text-muted mb-1">
            <span>DPP: {dppDisplay.toLocaleString('id-ID')}</span>
            <span>PPN: {taxAmount.toLocaleString('id-ID')}</span>
          </div>
        {/if}
        {#if hasDiscountedItems}
          <div class="text-[11px] text-green-500 mb-1">
            Hemat: {totalSavings.toLocaleString('id-ID')}
          </div>
        {/if}
        <p class="text-3xl font-extrabold text-purple-400">
          Rp {totalAmount.toLocaleString('id-ID')}
        </p>
      </div>

      <div class="mb-3 rounded-lg border border-border/50 bg-surface/50 px-3 py-2">
        <div class="space-y-1 max-h-40 overflow-y-auto">
          {#each cart as item}
            <div class="flex items-center justify-between text-xs">
              <span class="text-text-secondary truncate max-w-[60%]">{item.name} × {item.quantity}</span>
              <span class="text-text-primary font-medium">
                {#if item.discount && item.discount > 0}
                  <span class="line-through text-text-muted mr-1">{(item.original_price * item.quantity).toLocaleString('id-ID')}</span>
                {/if}
                {(item.price * item.quantity).toLocaleString('id-ID')}
              </span>
            </div>
          {/each}
        </div>
      </div>

      <div class="grid grid-cols-3 gap-1.5 mb-3">
        {#each paymentOptions as opt}
          <button
            class="py-2 rounded-xl border text-[11px] font-medium transition-all {paymentMethod === opt.id ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
            onclick={() => paymentMethod = opt.id}
          >
            {opt.label}
          </button>
        {/each}
      </div>

      <button
        class="w-full flex items-center gap-2 text-xs text-text-secondary hover:text-text-primary py-1.5 px-2 rounded-lg hover:bg-surface-hover transition-colors mb-3"
        onclick={() => onselectcustomer()}
      >
        <User size={14} class="shrink-0 text-text-muted" />
        <span class="truncate">{selectedCustomerLabel}</span>
        <ChevronRight size={12} class="shrink-0 text-text-muted ml-auto" />
      </button>

      {#if paymentMethod === 'Cash'}
        <div class="mb-3">
          <label for="cash-received-input" class="text-[11px] text-text-muted mb-1.5 font-medium block">
            Cash Received [F7]
          </label>
          <CurrencyInput id="cash-received-input" bind:value={cashReceived} placeholder="0" />
        </div>

        <div class="grid grid-cols-3 gap-1.5 mb-3">
          {#each quickCashPresets as preset}
            <button
              class="py-1.5 rounded-xl border border-border text-[11px] font-semibold text-text-secondary hover:border-primary-light hover:text-primary-light hover:bg-primary-subtle/30 transition-all {cashReceived === preset ? 'border-primary bg-primary-subtle text-primary-light' : ''}"
              onclick={() => cashReceived = preset}
            >
              {preset >= 1000000 ? `${preset / 1000000}jt` : `${preset / 1000}rb`}
            </button>
          {/each}
          <button
            class="py-1.5 rounded-xl border border-primary-light text-[11px] font-semibold text-primary-light hover:bg-primary-subtle transition-all"
            onclick={() => cashReceived = totalAmount}
          >
            Tepat [F7]
          </button>
        </div>

        <div
          class="flex items-center justify-between px-3 py-1.5 rounded-xl
            {changeDue >= 0
              ? 'bg-success-subtle border border-success-default/20'
              : 'bg-danger-subtle border border-danger-default/20'}"
        >
          <span class="text-xs font-medium text-text-secondary">Kembali</span>
          <span
            class="text-xl font-extrabold
              {changeDue >= 0 ? 'text-emerald-400' : 'text-danger-light'}"
          >
            Rp {Math.abs(changeDue).toLocaleString('id-ID')}
            {#if changeDue < 0}
              <span class="text-[10px] font-semibold text-danger-light ml-1">(kurang)</span>
            {/if}
          </span>
        </div>
      {:else}
        <div class="mb-3 text-center py-3 rounded-lg bg-surface/50 border border-border/50">
          <p class="text-[11px] text-text-muted mb-1">Total yang harus dibayar</p>
          <p class="text-2xl font-extrabold text-text-primary">Rp {totalAmount.toLocaleString('id-ID')}</p>
          <p class="text-[11px] text-text-muted mt-1">Konfirmasi setelah pembayaran berhasil</p>
        </div>
      {/if}

      <div class="flex gap-2 mt-4">
        <Button
          variant="secondary"
          class="flex-1 py-2"
          onclick={close}
        >
          Batal [F3]
        </Button>
        <Button
          variant="success"
          class="flex-1 py-2"
          disabled={cart.length === 0 || changeDue < 0}
          onclick={onfinalize}
        >
          <Check size={14} />
          Selesai [Enter]
        </Button>
      </div>
    </div>
  </div>
{/if}

<style></style>
