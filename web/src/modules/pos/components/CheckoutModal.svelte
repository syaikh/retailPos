<script lang="ts">
  import { fly } from 'svelte/transition';
  import { Button } from '$shared/ui';
  import { X, Check, Search } from 'lucide-svelte';
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

  function handleCashInput(e: Event) {
    const raw = (e.target as HTMLInputElement).value.replace(/[^0-9]/g, '');
    cashReceived = raw ? parseInt(raw, 10) : 0;
  }

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
      aria-label="Pembayaran Selesai"
      class="relative z-[55] w-full max-w-xl rounded-2xl border border-border-default bg-bg-card shadow-modal p-6"
    >
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-xl font-bold text-text-primary">Pembayaran Selesai</h2>
        <button
          onclick={close}
          class="w-9 h-9 flex items-center justify-center rounded-xl text-text-muted hover:text-text-primary hover:bg-surface-hover/50 transition-colors"
          title="Tutup [F3 / Esc]"
          aria-label="Tutup"
        >
          <X size={20} />
        </button>
      </div>

      <div class="mb-6 text-center">
        {#if taxAmount > 0}
          <div class="flex justify-center gap-6 text-xs text-text-muted mb-2">
            <span>DPP: {dppDisplay.toLocaleString('id-ID')}</span>
            <span>PPN 11%: {taxAmount.toLocaleString('id-ID')}</span>
          </div>
        {/if}
        <p class="text-sm text-text-muted mb-1 font-medium">Total Tagihan</p>
        <p class="text-4xl font-extrabold text-purple-400">
          {totalAmount.toLocaleString('id-ID')}
        </p>
      </div>

      <p class="text-xs text-text-muted mb-2 font-medium">Metode Pembayaran</p>
      <div class="grid grid-cols-3 gap-2 mb-6">
        {#each paymentOptions as opt}
          <button
            class="flex flex-col items-center gap-1 py-2.5 rounded-xl border text-xs font-medium transition-all {paymentMethod === opt.id ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
            onclick={() => paymentMethod = opt.id}
          >
            <opt.icon size={18} />
            {opt.label}
          </button>
        {/each}
      </div>

      <div class="mb-4">
        <p class="text-xs text-text-muted mb-1 font-medium">Customer</p>
        <Button variant="secondary" class="w-full justify-between text-sm" onclick={() => onselectcustomer()}>
          <span class="truncate">{selectedCustomerLabel}</span>
          <Search size={14} />
        </Button>
      </div>

      {#if paymentMethod === 'Cash'}
        <div class="mb-4">
          <label for="cash-received-input" class="text-xs text-text-muted mb-1.5 font-medium block">
            Cash Received [F6] = Total
          </label>
          <input
            id="cash-received-input"
            type="text"
            inputmode="numeric"
            value={cashReceived || ''}
            oninput={handleCashInput}
            class="text-lg font-bold text-text-primary [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
            placeholder="0"
          />
        </div>

        <div class="flex flex-wrap gap-2 mb-4">
          {#each quickCashPresets as preset}
            <button
              class="px-3.5 py-1.5 rounded-xl border border-border text-xs font-semibold text-text-secondary hover:border-primary-light hover:text-primary-light transition-colors"
              onclick={() => cashReceived = preset}
            >
              Rp {preset.toLocaleString('id-ID')}
            </button>
          {/each}
          <button
            class="px-3.5 py-1.5 rounded-xl border border-primary-light text-xs font-semibold text-primary-light hover:bg-primary-subtle transition-colors"
            onclick={() => cashReceived = totalAmount}
          >
            Tepat ({totalAmount.toLocaleString('id-ID')})
          </button>
        </div>

        <div
          class="flex items-center justify-between p-4 rounded-xl
            {changeDue >= 0
              ? 'bg-success-subtle border border-success-default/20'
              : 'bg-danger-subtle border border-danger-default/20'}"
        >
          <span class="text-sm font-medium text-text-secondary">Uang Kembali</span>
          <span
            class="text-2xl font-extrabold
              {changeDue >= 0 ? 'text-emerald-400' : 'text-danger-light'}"
          >
            {Math.abs(changeDue).toLocaleString('id-ID')}
            {#if changeDue < 0}
              <span class="text-xs font-semibold text-danger-light ml-1">(kurang)</span>
            {/if}
          </span>
        </div>
      {:else}
        <div class="mb-4">
          <label for="card-ewallet-amount-input" class="text-xs text-text-muted mb-1.5 font-medium block">
            Jumlah Bayar
          </label>
          <input
            id="card-ewallet-amount-input"
            type="text"
            inputmode="numeric"
            value={cashReceived || ''}
            oninput={handleCashInput}
            class="text-lg font-bold text-text-primary [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
            placeholder="0"
          />
        </div>
      {/if}

      <div class="flex gap-3 mt-6">
        <Button
          variant="secondary"
          class="flex-1"
          onclick={close}
        >
          Batal [F3]
        </Button>
        <Button
          variant="success"
          class="flex-1"
          disabled={cart.length === 0 || changeDue < 0}
          onclick={onfinalize}
        >
          <Check size={16} />
          Selesai &amp; Cetak [Enter]
        </Button>
      </div>
    </div>
  </div>
{/if}

<style></style>
