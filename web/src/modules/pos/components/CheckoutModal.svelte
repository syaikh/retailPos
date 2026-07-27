<script lang="ts">
  import { fly } from 'svelte/transition';
  import { Button, CurrencyInput } from '$shared/ui';
  import { X, Check, User, ChevronRight, Plus, Trash2 } from 'lucide-svelte';
  import { tick } from 'svelte';
  import type { PaymentAllocation } from '../types';

  const denominations = [5000, 10000, 20000, 50000, 100000];
  let dialogEl: HTMLDivElement | undefined = $state();
  let previousFocus: HTMLElement | null = null;

  interface AllocationRow {
    id: string;
    methodCode: string;
    amount: number;
    referenceNumber: string;
  }

  let {
    showCheckoutModal = $bindable(false),
    cart = [],
    totalAmount = 0,
    subtotal = 0,
    taxAmount = 0,
    dppDisplay = 0,
    paymentOptions = [],
    selectedCustomerLabel = '',
    checkingOut = false,
    onfinalize = (payments: PaymentAllocation[]) => {},
    onselectcustomer = () => {},
  }: {
    showCheckoutModal: boolean;
    cart: any[];
    totalAmount: number;
    subtotal: number;
    taxAmount: number;
    dppDisplay: number;
    paymentOptions: Array<{ id: string; label: string; icon?: any; requiresReference?: boolean }>;
    selectedCustomerLabel: string;
    checkingOut: boolean;
    onfinalize?: (payments: PaymentAllocation[]) => void;
    onselectcustomer?: () => void;
  } = $props();

  let allocations = $state<AllocationRow[]>([]);
  let nextId = $state(1);

  const totalAllocated = $derived(allocations.reduce((sum, a) => sum + a.amount, 0));
  const remainingBalance = $derived(totalAmount - totalAllocated);
  const canComplete = $derived(remainingBalance === 0 && allocations.length > 0);
  const cashAllocation = $derived(allocations.find(a => a.methodCode === 'CASH'));

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
    allocations = [];
    nextId = 1;
  }

  function generateRefNumber(methodCode: string): string {
    const now = new Date();
    const dd = String(now.getDate()).padStart(2, '0');
    const mm = String(now.getMonth() + 1).padStart(2, '0');
    const yy = String(now.getFullYear()).slice(2);
    const rand = String(Math.floor(100000 + Math.random() * 900000));
    if (methodCode === 'CARD' || methodCode === 'EDC') return `EDC/${dd}${mm}${yy}/${rand}`;
    if (methodCode === 'E_WALLET') return `EW/${dd}${mm}${yy}/${rand}`;
    return `REF/${dd}${mm}${yy}/${rand}`;
  }

  function addAllocation(methodCode: string) {
    const existing = allocations.find(a => a.methodCode === methodCode);
    if (existing) {
      const input = document.getElementById(`alloc-amount-${existing.id}`);
      input?.focus();
      return;
    }
    const opt = paymentOptions.find(o => o.id === methodCode);
    const allocAmount = remainingBalance > 0 ? remainingBalance : 0;
    allocations = [...allocations, {
      id: `a${nextId++}`,
      methodCode,
      amount: allocAmount,
      referenceNumber: opt?.requiresReference ? generateRefNumber(methodCode) : '',
    }];
  }

  function removeAllocation(id: string) {
    allocations = allocations.filter(a => a.id !== id);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      close();
      return;
    }
    if (e.key === 'F7' && cashAllocation) {
      e.preventDefault();
      allocations = allocations.map(a =>
        a.methodCode === 'CASH' ? { ...a, amount: totalAmount } : a
      );
      return;
    }
    if (e.key === 'Enter' && canComplete) {
      e.preventDefault();
      handleFinalize();
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
      allocations = [{ id: 'a1', methodCode: 'CASH', amount: 0, referenceNumber: '' }];
      nextId = 2;
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

  function handleFinalize() {
    if (!canComplete) return;
    const payments: PaymentAllocation[] = allocations.map(a => ({
      payment_method_code: a.methodCode,
      amount: a.amount,
      reference_number: a.referenceNumber || undefined,
    }));
    onfinalize(payments);
  }
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
      class="relative z-[55] w-full max-w-4xl h-dvh max-h-[calc(100vh-2rem)] flex flex-col rounded-2xl border border-border-default bg-bg-card shadow-modal p-5"
    >
      <div class="flex items-center justify-between shrink-0 mb-3">
        <h2 class="text-lg font-bold text-text-primary">Pembayaran</h2>
        <button
          onclick={close}
          class="w-8 h-8 flex items-center justify-center rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover/50 transition-colors"
          title="Tutup [Esc]"
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

            <!-- Fixed top: Total + Payment grid + Customer -->
            <div class="shrink-0 space-y-3">
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

              <!-- Payment Method Grid -->
              <div class="grid grid-cols-3 gap-1.5">
                {#each paymentOptions as opt}
                  {@const isUsed = allocations.some(a => a.methodCode === opt.id)}
                  <button
                    class="py-2 rounded-xl border text-[11px] font-medium transition-all {isUsed ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
                    onclick={() => addAllocation(opt.id)}
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
            </div>

            <!-- Scrollable: Allocations List -->
            {#if allocations.length > 0}
              <div class="flex items-center justify-between px-1">
                <span class="text-[10px] font-semibold text-text-muted uppercase tracking-wider">Alokasi Pembayaran</span>
                <button
                  class="text-[10px] text-danger hover:text-danger-light transition-colors"
                  onclick={() => { allocations = []; }}
                >
                  Hapus semua
                </button>
              </div>
            {/if}
            <div class="flex-1 min-h-0 overflow-y-auto space-y-2">
              {#each allocations as alloc (alloc.id)}
                {@const opt = paymentOptions.find(o => o.id === alloc.methodCode)}
                {@const isCash = alloc.methodCode === 'CASH'}
                <div class="rounded-xl border border-border/50 bg-surface/50 p-2.5 space-y-2">
                  <div class="flex items-center justify-between">
                    <span class="text-[11px] font-semibold text-text-primary px-2 py-0.5 rounded-lg bg-primary-subtle text-primary-light">
                      {opt?.label || alloc.methodCode}
                    </span>
                    <button
                      class="w-6 h-6 flex items-center justify-center rounded-md text-text-muted hover:text-danger hover:bg-danger-subtle/30 transition-colors"
                      onclick={() => removeAllocation(alloc.id)}
                      title="Hapus pembayaran ini"
                      aria-label="Hapus pembayaran"
                    >
                      <Trash2 size={12} />
                    </button>
                  </div>

                  <div>
                    <label for="alloc-amount-{alloc.id}" class="text-[10px] text-text-muted mb-1 block">
                      Jumlah
                    </label>
                    <CurrencyInput
                      id="alloc-amount-{alloc.id}"
                      bind:value={alloc.amount}
                      placeholder="0"
                    />
                  </div>

                  {#if !isCash && opt?.requiresReference}
                    <div>
                      <label for="alloc-ref-{alloc.id}" class="text-[10px] text-text-muted mb-1 block">
                        No. Referensi
                      </label>
                      <input
                        id="alloc-ref-{alloc.id}"
                        type="text"
                        bind:value={alloc.referenceNumber}
                        placeholder="Masukkan no. referensi"
                        class="w-full px-2 py-1.5 rounded-lg border border-border bg-surface text-xs text-text-primary placeholder:text-text-muted outline-none focus:border-primary-light transition-colors"
                      />
                    </div>
                  {/if}

                  {#if isCash}
                    <div class="grid grid-cols-5 gap-1">
                      {#each denominations as denom}
                        <button
                          class="py-1 rounded-lg border text-[10px] font-semibold transition-all border-border text-text-secondary hover:border-primary-light hover:text-primary-light hover:bg-primary-subtle/30"
                          onclick={() => { alloc.amount += denom; allocations = allocations; }}
                        >
                          {denom >= 1000000 ? `${denom / 1000000}jt` : denom >= 1000 ? `${denom / 1000}rb` : String(denom)}
                        </button>
                      {/each}
                    </div>
                    <div class="flex gap-1 mt-1">
                      <button
                        class="flex-1 py-1 rounded-lg border text-[10px] font-semibold transition-all border-border text-text-secondary hover:border-primary-light hover:text-primary-light hover:bg-primary-subtle/30"
                        onclick={() => { alloc.amount = totalAmount; allocations = allocations; }}
                      >
                        Tepat [F7]
                      </button>
                      <button
                        class="flex-1 py-1 rounded-lg border border-danger/30 text-[10px] font-semibold text-danger hover:bg-danger-subtle/30 transition-all"
                        onclick={() => { alloc.amount = 0; allocations = allocations; }}
                      >
                        Reset
                      </button>
                    </div>
                  {/if}
                </div>
              {/each}

              {#if allocations.length === 0}
                <div class="text-center py-4 text-[11px] text-text-muted">
                  Pilih metode pembayaran di samping untuk menambahkan pembayaran
                </div>
              {/if}
            </div>

            <!-- Fixed bottom: Actions -->
            <div class="shrink-0">
              <div class="flex gap-2 pt-2 border-t border-border/30">
                <Button
                  variant="secondary"
                  class="flex-1 py-2"
                  onclick={close}
                >
                  Batal [Esc]
                </Button>
                <Button
                  variant="success"
                  class="flex-1 py-2"
                  disabled={cart.length === 0 || !canComplete}
                  onclick={handleFinalize}
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
  </div>
{/if}

<style></style>
