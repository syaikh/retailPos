<script lang="ts">
  import { fly } from 'svelte/transition';
  import { Button, CurrencyInput } from '$shared/ui';
  import { X, Check, User, ChevronRight } from 'lucide-svelte';
  import { tick } from 'svelte';

  const denominations = [5000, 10000, 20000, 50000, 100000];
  let exactMode = $state(false);
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
    exactMode = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      close();
      return;
    }
    if (e.key === 'F7' && paymentMethod.toUpperCase() === 'CASH') {
      e.preventDefault();
      exactMode = true;
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
    if (paymentMethod.toUpperCase() !== 'CASH') {
      cashReceived = totalAmount;
      exactMode = false;
    } else if (!exactMode && cashReceived === 0) {
      cashReceived = 0;
    }
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
      class="relative z-[55] w-full max-w-4xl max-h-[85vh] flex flex-col rounded-2xl border border-border-default bg-bg-card shadow-modal p-5"
    >
      <div class="flex items-center justify-between shrink-0 mb-3">
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

      <div class="flex-1 min-h-0 overflow-hidden">
        <div class="grid grid-cols-12 gap-5 h-full">

          <!-- LEFT: Item table -->
          <div class="col-span-7 flex flex-col min-h-0 rounded-lg border border-border/50 bg-surface/50">
            <div class="grid grid-cols-12 gap-1 px-4 py-2 text-[10px] font-semibold text-text-muted uppercase tracking-wider border-b border-border/30">
              <span class="col-span-4">Item</span>
              <span class="col-span-2 text-right">Harga</span>
              <span class="col-span-1 text-center">Qty</span>
              <span class="col-span-2 text-right">Diskon</span>
              <span class="col-span-3 text-right">Subtotal</span>
            </div>
            <div class="flex-1 min-h-0 overflow-y-auto p-1">
              {#each cart as item}
                <div class="grid grid-cols-12 gap-1 items-center px-3 py-2 text-xs rounded-lg hover:bg-surface-hover transition-colors">
                  <span class="col-span-4 truncate text-text-secondary font-medium">{item.name}</span>
                  <span class="col-span-2 text-right text-text-muted tabular-nums">
                    {#if item.discount && item.discount > 0}
                      <span class="line-through">{item.original_price.toLocaleString('id-ID')}</span>
                      <br><span class="text-text-secondary">{item.price.toLocaleString('id-ID')}</span>
                    {:else}
                      {item.original_price.toLocaleString('id-ID')}
                    {/if}
                  </span>
                  <span class="col-span-1 text-center text-text-secondary tabular-nums">{item.quantity}</span>
                  <span class="col-span-2 text-right tabular-nums {item.discount && item.discount > 0 ? 'text-red-400' : 'text-text-muted'}">
                    {item.discount && item.discount > 0 ? (item.discount * item.quantity).toLocaleString('id-ID') : '—'}
                  </span>
                  <span class="col-span-3 text-right font-semibold text-text-primary tabular-nums">
                    {(item.price * item.quantity).toLocaleString('id-ID')}
                  </span>
                </div>
              {/each}
            </div>
          </div>

          <!-- RIGHT: Payment details -->
          <div class="col-span-5 flex flex-col min-h-0 gap-3">

            <!-- Total -->
            <div class="text-center pb-2 border-b border-border/30">
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

            <!-- Payment Method -->
            <div class="grid grid-cols-3 gap-1.5">
              {#each paymentOptions as opt}
                <button
                  class="py-2 rounded-xl border text-[11px] font-medium transition-all {paymentMethod === opt.id ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
                  onclick={() => paymentMethod = opt.id}
                >
                  {opt.label}
                </button>
              {/each}
            </div>

            <!-- Customer -->
            <button
              class="w-full flex items-center gap-2 text-xs text-text-secondary hover:text-text-primary py-1.5 px-2 rounded-lg hover:bg-surface-hover transition-colors"
              onclick={() => onselectcustomer()}
            >
              <User size={14} class="shrink-0 text-text-muted" />
              <span class="truncate">{selectedCustomerLabel}</span>
              <ChevronRight size={12} class="shrink-0 text-text-muted ml-auto" />
            </button>

            <!-- Cash / Non-cash -->
            {#if paymentMethod.toUpperCase() === 'CASH'}
              <div>
                <label for="cash-received-input" class="text-[11px] text-text-muted mb-1.5 font-medium block">
                  Cash Received
                </label>
                <CurrencyInput id="cash-received-input" bind:value={cashReceived} placeholder="0" />
              </div>

              <div class="grid grid-cols-5 gap-1.5">
                {#each denominations as denom}
                  <button
                    class="py-1.5 rounded-xl border text-[11px] font-semibold transition-all {exactMode ? 'border-border text-text-muted/40 opacity-40 cursor-not-allowed' : cashReceived > 0 && cashReceived % denom === 0 && cashReceived < denom * 2 ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-secondary hover:border-primary-light hover:text-primary-light hover:bg-primary-subtle/30'}"
                    disabled={exactMode}
                    onclick={() => {
                      if (exactMode) return;
                      exactMode = false;
                      cashReceived += denom;
                    }}
                  >
                    {denom >= 1000000 ? `${denom / 1000000}jt` : denom >= 1000 ? `${denom / 1000}rb` : String(denom)}
                  </button>
                {/each}
              </div>

              <div class="flex gap-1.5">
                <button
                  class="flex-1 py-1.5 rounded-xl border text-[11px] font-semibold transition-all {exactMode ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-secondary hover:border-primary-light hover:text-primary-light hover:bg-primary-subtle/30'}"
                  onclick={() => {
                    exactMode = true;
                    cashReceived = totalAmount;
                  }}
                >
                  Tepat [F7]
                </button>
                <button
                  class="flex-1 py-1.5 rounded-xl border border-danger/30 text-[11px] font-semibold text-danger hover:bg-danger-subtle/30 transition-all"
                  onclick={() => {
                    exactMode = false;
                    cashReceived = 0;
                  }}
                >
                  Reset
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
              <div class="text-center py-3 rounded-lg bg-surface/50 border border-border/50">
                <p class="text-[11px] text-text-muted mb-1">Total yang harus dibayar</p>
                <p class="text-2xl font-extrabold text-text-primary">Rp {totalAmount.toLocaleString('id-ID')}</p>
                <p class="text-[11px] text-text-muted mt-1">Konfirmasi setelah pembayaran berhasil</p>
              </div>
            {/if}

            <!-- Spacer to push actions to bottom -->
            <div class="flex-1 min-h-2"></div>

            <!-- Actions -->
            <div class="flex gap-2 pt-2 border-t border-border/30">
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
      </div>
    </div>
  </div>
{/if}

<style></style>
