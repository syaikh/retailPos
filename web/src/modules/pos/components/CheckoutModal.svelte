<script lang="ts">
  import { fly } from "svelte/transition";
  import { Button, CurrencyInput } from "$shared/ui";
  import { X, Check, User, ChevronRight, Trash2, Plus } from "lucide-svelte";
  import { tick } from "svelte";
  import type {
    DisplayCartItem,
    PaymentAllocation,
    PaymentOption,
  } from "../types";
  import { labels, t, paymentMethodLabel } from "$shared/i18n";

  const denominations = [5000, 10000, 20000, 50000, 100000];
  // Mirrors internal/sale.MaxPaymentsPerSale so the "+" action cannot queue a
  // payload the backend would reject at Confirm time.
  const MAX_PAYMENTS_PER_SALE = 10;
  let dialogEl: HTMLDivElement | undefined = $state();
  let previousFocus: HTMLElement | null = null;
  let wasConfirmStep = false;

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
    taxAmount = 0,
    dppDisplay = 0,
    paymentOptions = [],
    selectedCustomerLabel = "",
    onfinalize = (_payments: PaymentAllocation[]) => {},
    onselectcustomer = () => {},
  }: {
    showCheckoutModal: boolean;
    cart: DisplayCartItem[];
    totalAmount: number;
    taxAmount: number;
    dppDisplay: number;
    paymentOptions: PaymentOption[];
    selectedCustomerLabel: string;
    onfinalize?: (payments: PaymentAllocation[]) => void;
    onselectcustomer?: () => void;
  } = $props();

  let allocations = $state<AllocationRow[]>([]);
  let nextId = $state(1);
  let confirmStep = $state(false);

  const totalAllocated = $derived(
    allocations.reduce((sum, a) => sum + a.amount, 0),
  );
  const remainingBalance = $derived(totalAmount - totalAllocated);
  const cashTotal = $derived(
    allocations
      .filter((a) => a.methodCode === "CASH")
      .reduce((sum, a) => sum + a.amount, 0),
  );
  const nonCashTotal = $derived(totalAllocated - cashTotal);
  const overTenderOnCash = $derived(
    remainingBalance < 0 && nonCashTotal <= totalAmount,
  );
  const changeDue = $derived(
    remainingBalance < 0 && overTenderOnCash ? Math.abs(remainingBalance) : 0,
  );
  const confirmRows = $derived(
    allocations.map((a) => ({
      id: a.id,
      label: paymentMethodLabel(
        a.methodCode,
        paymentOptions.find((o) => o.id === a.methodCode)?.label,
      ),
      amount: a.amount,
    })),
  );
  const canComplete = $derived(
    allocations.length > 0 &&
      (remainingBalance === 0 || (remainingBalance < 0 && overTenderOnCash)) &&
      allocations.every((a) => {
        const opt = paymentOptions.find((o) => o.id === a.methodCode);
        return !(opt?.requiresReference && !a.referenceNumber?.trim());
      }),
  );
  const cashAllocation = $derived(
    allocations.find((a) => a.methodCode === "CASH"),
  );

  const totalSavings = $derived(
    cart.reduce((sum, item) => {
      if (item.discount && item.discount > 0) {
        return sum + item.discount * item.quantity;
      }
      return sum;
    }, 0),
  );

  const hasDiscountedItems = $derived(
    cart.some((item) => item.discount && item.discount > 0),
  );

  function close() {
    showCheckoutModal = false;
    allocations = [];
    nextId = 1;
    confirmStep = false;
  }

  function generateRefNumber(methodCode: string): string {
    const shifted = new Date(Date.now() + 7 * 60 * 60 * 1000);
    const dd = String(shifted.getUTCDate()).padStart(2, "0");
    const mm = String(shifted.getUTCMonth() + 1).padStart(2, "0");
    const yy = String(shifted.getUTCFullYear()).slice(2);
    const rand = String(Math.floor(100000 + Math.random() * 900000));
    if (methodCode === "CARD" || methodCode === "EDC")
      return `EDC/${dd}${mm}${yy}/${rand}`;
    if (methodCode === "E_WALLET") return `EW/${dd}${mm}${yy}/${rand}`;
    return `REF/${dd}${mm}${yy}/${rand}`;
  }

  function addAllocation(methodCode: string) {
    const existing = allocations.find((a) => a.methodCode === methodCode);
    if (existing) {
      const input = document.getElementById(`alloc-amount-${existing.id}`);
      input?.focus();
      return;
    }
    // S-D: do not add a redundant method once the total is fully allocated.
    if (remainingBalance <= 0) return;
    const opt = paymentOptions.find((o) => o.id === methodCode);
    const allocAmount = remainingBalance;
    const newId = `a${nextId++}`;
    allocations = [
      ...allocations,
      {
        id: newId,
        methodCode,
        amount: allocAmount,
        referenceNumber: opt?.requiresReference
          ? generateRefNumber(methodCode)
          : "",
      },
    ];
    tick().then(() =>
      document.getElementById(`alloc-amount-${newId}`)?.focus(),
    );
  }

  function removeAllocation(id: string) {
    allocations = allocations.filter((a) => a.id !== id);
  }

  function addAnotherAllocation(methodCode: string) {
    if (methodCode === "CASH") return;
    if (remainingBalance <= 0) return;
    if (allocations.length >= MAX_PAYMENTS_PER_SALE) return;
    const opt = paymentOptions.find((o) => o.id === methodCode);
    const newId = `a${nextId++}`;
    allocations = [
      ...allocations,
      {
        id: newId,
        methodCode,
        amount: remainingBalance,
        referenceNumber: opt?.requiresReference
          ? generateRefNumber(methodCode)
          : "",
      },
    ];
    tick().then(() =>
      document.getElementById(`alloc-amount-${newId}`)?.focus(),
    );
  }

  function resetAmounts() {
    allocations = allocations.map((a) => ({ ...a, amount: 0 }));
  }

  function openConfirm() {
    if (!canComplete) return;
    confirmStep = true;
  }

  function splitEqually() {
    if (allocations.length === 0) return;
    const base = Math.floor(totalAmount / allocations.length);
    const remainder = totalAmount - base * allocations.length;
    allocations = allocations.map((a, i) => ({
      ...a,
      amount: i === 0 ? base + remainder : base,
    }));
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      if (confirmStep) {
        confirmStep = false;
        return;
      }
      close();
      return;
    }
    // Number-key shortcuts: add the Nth payment method (ignored while typing in a field)
    if (
      !confirmStep &&
      /^[1-9]$/.test(e.key) &&
      dialogEl &&
      !(document.activeElement instanceof HTMLInputElement)
    ) {
      const idx = parseInt(e.key, 10) - 1;
      const opt = paymentOptions[idx];
      if (opt) {
        e.preventDefault();
        addAllocation(opt.id);
      }
      return;
    }
    if (!confirmStep && e.key === "F7" && cashAllocation) {
      e.preventDefault();
      allocations = allocations.map((a) =>
        a.methodCode === "CASH" ? { ...a, amount: totalAmount } : a,
      );
      return;
    }
    if (e.key === "Enter" && canComplete && !confirmStep) {
      e.preventDefault();
      openConfirm();
      return;
    }
    if (e.key === "Tab" && dialogEl) {
      const focusable = dialogEl.querySelectorAll<HTMLElement>(
        'button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
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
      allocations = [
        {
          id: "a1",
          methodCode: "CASH",
          amount: totalAmount,
          referenceNumber: "",
        },
      ];
      nextId = 2;
      tick().then(() => {
        const firstFocusable = dialogEl?.querySelector<HTMLElement>(
          "button:not([disabled]), input:not([disabled])",
        );
        firstFocusable?.focus();
      });
    } else {
      confirmStep = false;
      if (previousFocus) {
        previousFocus.focus();
        previousFocus = null;
      }
    }
  });

  $effect(() => {
    if (confirmStep && showCheckoutModal) {
      tick().then(() => document.getElementById("confirm-finalize")?.focus());
    } else if (wasConfirmStep && showCheckoutModal) {
      // Backing out of the confirm view unmounts the focused element, which
      // would leave focus on <body> and silence the wrapper-bound shortcuts
      // (Enter / F7 / digits / Escape). Put focus back inside the dialog.
      tick().then(() => document.getElementById("checkout-done")?.focus());
    }
    wasConfirmStep = confirmStep;
  });

  function handleFinalize() {
    if (!canComplete) return;
    const payments: PaymentAllocation[] = allocations.map((a) => ({
      payment_method_code: a.methodCode,
      amount: a.amount,
      reference_number: a.referenceNumber || undefined,
    }));
    onfinalize(payments);
  }
</script>

{#if showCheckoutModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center print-modal-overlay"
    transition:fly={{ y: 40, duration: 300 }}
    onkeydown={handleKeydown}
    role="none"
  >
    <div
      class="absolute inset-0 bg-black/70 backdrop-blur-sm"
      onclick={() => (confirmStep ? (confirmStep = false) : close())}
      role="presentation"
    ></div>

    <div
      bind:this={dialogEl}
      role="dialog"
      aria-modal="true"
      aria-label={labels.payment}
      tabindex="-1"
      class="relative z-[55] w-full max-w-4xl h-dvh max-h-[calc(100vh-2rem)] flex flex-col rounded-2xl border border-border-default bg-bg-card shadow-modal p-5"
    >
      {#if confirmStep}
        <div
          class="flex-1 flex flex-col items-center justify-center gap-5 text-center"
        >
          <h2 class="text-lg font-bold text-text-primary">
            {labels.confirmPayment}
          </h2>
          <div
            class="w-full max-w-xs max-h-[50vh] overflow-y-auto rounded-xl border border-border/50 bg-surface/50 p-4 space-y-2 text-left"
          >
            <div class="flex justify-between text-xs">
              <span class="text-text-muted">{labels.total}</span>
              <span class="font-semibold text-text-primary tabular-nums"
                >{totalAmount.toLocaleString("id-ID")}</span
              >
            </div>
            {#each confirmRows as row (row.id)}
              <div class="flex justify-between text-xs">
                <span class="text-text-muted">{row.label}</span>
                <span class="font-semibold text-text-primary tabular-nums"
                  >{row.amount.toLocaleString("id-ID")}</span
                >
              </div>
            {/each}
            {#if changeDue > 0}
              <div
                class="flex justify-between text-xs border-t border-border/30 pt-2"
              >
                <span class="text-text-muted">{labels.changeDue}</span>
                <span class="font-bold text-success tabular-nums"
                  >{changeDue.toLocaleString("id-ID")}</span
                >
              </div>
            {/if}
          </div>
          <div class="flex gap-2 w-full max-w-xs">
            <Button
              variant="secondary"
              class="flex-1 py-2"
              onclick={() => (confirmStep = false)}
            >
              {labels.cancel}
            </Button>
            <Button
              variant="success"
              class="flex-1 py-2"
              id="confirm-finalize"
              onclick={handleFinalize}
            >
              <Check size={14} />
              {labels.confirmEnter}
            </Button>
          </div>
        </div>
      {:else}
        <div class="flex items-center justify-between shrink-0 mb-3">
          <h2 class="text-lg font-bold text-text-primary">
            {allocations.length > 1 ? labels.paymentSplit : labels.payment}
          </h2>
          <button
            onclick={close}
            class="w-8 h-8 flex items-center justify-center rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover/50 transition-colors"
            title={`${labels.close} [Esc]`}
            aria-label={labels.close}
          >
            <X size={18} />
          </button>
        </div>

        <div class="flex-1 min-h-0 overflow-hidden">
          <div class="grid grid-cols-12 gap-5 h-full">
            <!-- LEFT: Item table -->
            <div
              class="col-span-7 flex flex-col min-h-0 rounded-lg border border-border/50 bg-surface/50"
            >
              <div
                class="grid grid-cols-12 gap-1 px-4 py-2 text-[10px] font-semibold text-text-muted uppercase tracking-wider border-b border-border/30"
              >
                <span class="col-span-4">{labels.item}</span>
                <span class="col-span-2 text-right">{labels.price}</span>
                <span class="col-span-1 text-center">{labels.qty}</span>
                <span class="col-span-2 text-right">{labels.discount}</span>
                <span class="col-span-3 text-right">{labels.subtotal}</span>
              </div>
              <div class="flex-1 min-h-0 overflow-y-auto p-1">
                {#each cart as item, i (i)}
                  <div
                    class="grid grid-cols-12 gap-1 items-center px-3 py-2 text-xs rounded-lg hover:bg-surface-hover transition-colors"
                  >
                    <span
                      class="col-span-4 truncate text-text-secondary font-medium"
                      >{item.name}</span
                    >
                    <span
                      class="col-span-2 text-right text-text-muted tabular-nums"
                    >
                      {#if item.discount && item.discount > 0}
                        <span class="line-through"
                          >{item.original_price.toLocaleString("id-ID")}</span
                        >
                        <br /><span class="text-text-secondary"
                          >{item.price.toLocaleString("id-ID")}</span
                        >
                      {:else}
                        {item.original_price.toLocaleString("id-ID")}
                      {/if}
                    </span>
                    <span
                      class="col-span-1 text-center text-text-secondary tabular-nums"
                      >{item.quantity}</span
                    >
                    <span
                      class="col-span-2 text-right tabular-nums {item.discount &&
                      item.discount > 0
                        ? 'text-danger-light'
                        : 'text-text-muted'}"
                    >
                      {item.discount && item.discount > 0
                        ? (item.discount * item.quantity).toLocaleString(
                            "id-ID",
                          )
                        : "—"}
                    </span>
                    <span
                      class="col-span-3 text-right font-semibold text-text-primary tabular-nums"
                    >
                      {(item.price * item.quantity).toLocaleString("id-ID")}
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
                    <div
                      class="flex justify-center gap-4 text-[11px] text-text-muted mb-1"
                    >
                      <span
                        >{labels.dpp}: {dppDisplay.toLocaleString(
                          "id-ID",
                        )}</span
                      >
                      <span
                        >{labels.ppn}: {taxAmount.toLocaleString("id-ID")}</span
                      >
                    </div>
                  {/if}
                  {#if hasDiscountedItems}
                    <div class="text-[11px] text-green-500 mb-1">
                      {t("savingsWithAmount", {
                        amount: totalSavings.toLocaleString("id-ID"),
                      })}
                    </div>
                  {/if}
                  <p class="text-3xl font-extrabold text-primary">
                    {labels.currencySymbol}
                    {totalAmount.toLocaleString("id-ID")}
                  </p>
                </div>

                <!-- Payment Method Grid -->
                <div class="grid grid-cols-3 gap-1.5">
                  {#each paymentOptions as opt (opt.id || opt)}
                    {@const isUsed = allocations.some(
                      (a) => a.methodCode === opt.id,
                    )}
                    <button
                      disabled={remainingBalance <= 0 && !isUsed}
                      class="py-2 rounded-xl border text-[11px] font-medium transition-all {isUsed
                        ? 'border-primary bg-primary-subtle text-primary-light'
                        : remainingBalance <= 0
                          ? 'border-border text-text-muted/40 cursor-not-allowed'
                          : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
                      onclick={() => addAllocation(opt.id)}
                    >
                      {paymentMethodLabel(opt.id, opt.label)}
                    </button>
                  {/each}
                </div>

                {#if allocations.length >= 2}
                  <button
                    class="w-full py-1.5 rounded-lg border border-dashed border-border-strong text-[11px] font-medium text-text-secondary hover:border-primary-light hover:text-primary-light transition-colors"
                    onclick={splitEqually}
                  >
                    ⟳ {labels.splitEqually}
                  </button>
                {/if}

                <!-- Customer -->
                <button
                  class="w-full flex items-center gap-2 text-xs text-text-secondary hover:text-text-primary py-1.5 px-2 rounded-lg hover:bg-surface-hover transition-colors"
                  onclick={() => onselectcustomer()}
                >
                  <User size={14} class="shrink-0 text-text-muted" />
                  <span class="truncate">{selectedCustomerLabel}</span>
                  <ChevronRight
                    size={12}
                    class="shrink-0 text-text-muted ml-auto"
                  />
                </button>
              </div>

              <!-- Scrollable: Allocations List -->
              {#if allocations.length > 0}
                <div class="flex items-center justify-between px-1">
                  <span
                    class="text-[10px] font-semibold text-text-muted uppercase tracking-wider"
                    >{labels.paymentAllocation}</span
                  >
                  <div class="flex items-center gap-3">
                    <button
                      class="text-[10px] text-text-muted hover:text-text-secondary transition-colors"
                      onclick={resetAmounts}
                    >
                      {labels.resetAmounts}
                    </button>
                    <button
                      class="text-[10px] text-danger hover:text-danger-light transition-colors"
                      onclick={() => {
                        allocations = [];
                      }}
                    >
                      {labels.removeAll}
                    </button>
                  </div>
                </div>
              {/if}
              <div class="flex-1 min-h-0 overflow-y-auto space-y-2">
                {#each allocations as alloc (alloc.id)}
                  {@const opt = paymentOptions.find(
                    (o) => o.id === alloc.methodCode,
                  )}
                  {@const isCash = alloc.methodCode === "CASH"}
                  <div
                    class="rounded-xl border border-border/50 bg-surface/50 p-2.5 space-y-2"
                  >
                    <div class="flex items-center justify-between">
                      <div class="flex items-center gap-1.5">
                        <span
                          class="text-[11px] font-semibold text-text-primary px-2 py-0.5 rounded-lg bg-primary-subtle text-primary-light"
                        >
                          {paymentMethodLabel(alloc.methodCode, opt?.label)}
                        </span>
                        <span class="text-[10px] text-success"
                          >&#10003; {labels.added}</span
                        >
                      </div>
                      <div class="flex items-center gap-1">
                        {#if !isCash}
                          <button
                            class="w-6 h-6 flex items-center justify-center rounded-md text-text-muted hover:text-primary hover:bg-primary-subtle/50 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                            onclick={() =>
                              addAnotherAllocation(alloc.methodCode)}
                            disabled={remainingBalance <= 0 ||
                              allocations.length >= MAX_PAYMENTS_PER_SALE}
                            title={labels.addAnotherPayment}
                            aria-label={labels.addAnotherPayment}
                          >
                            <Plus size={12} />
                          </button>
                        {/if}
                        <button
                          class="w-6 h-6 flex items-center justify-center rounded-md text-text-muted hover:text-danger hover:bg-danger-subtle/30 transition-colors"
                          onclick={() => removeAllocation(alloc.id)}
                          title={labels.removeThisPayment}
                          aria-label={labels.removePayment}
                        >
                          <Trash2 size={12} />
                        </button>
                      </div>
                    </div>

                    <div>
                      <label
                        for="alloc-amount-{alloc.id}"
                        class="text-[10px] text-text-muted mb-1 block"
                      >
                        {labels.amount}
                      </label>
                      <CurrencyInput
                        id="alloc-amount-{alloc.id}"
                        bind:value={alloc.amount}
                        placeholder="0"
                      />
                    </div>

                    {#if !isCash && opt?.requiresReference}
                      <div>
                        <label
                          for="alloc-ref-{alloc.id}"
                          class="text-[10px] text-text-muted mb-1 block"
                        >
                          {labels.referenceNumber}
                          <span class="text-danger">*</span>
                        </label>
                        <input
                          id="alloc-ref-{alloc.id}"
                          type="text"
                          bind:value={alloc.referenceNumber}
                          placeholder={labels.referenceNumberPlaceholder}
                          class="w-full px-2 py-1.5 rounded-lg border text-xs text-text-primary placeholder:text-text-muted outline-none focus:border-primary-light transition-colors {alloc.referenceNumber?.trim()
                            ? 'border-border bg-surface'
                            : 'border-danger bg-danger-subtle/20'}"
                        />
                      </div>
                    {/if}

                    {#if isCash}
                      <div class="grid grid-cols-5 gap-1">
                        {#each denominations as denom (denom)}
                          <button
                            class="py-1 rounded-lg border text-[10px] font-semibold transition-all border-border text-text-secondary hover:border-primary-light hover:text-primary-light hover:bg-primary-subtle/30"
                            onclick={() => {
                              alloc.amount += denom;
                              allocations = allocations;
                            }}
                          >
                            {denom >= 1000000
                              ? `${denom / 1000000}${labels.denomMillion}`
                              : denom >= 1000
                                ? `${denom / 1000}${labels.denomThousand}`
                                : String(denom)}
                          </button>
                        {/each}
                      </div>
                      <div class="flex gap-1 mt-1">
                        <button
                          class="flex-1 py-1 rounded-lg border text-[10px] font-semibold transition-all border-border text-text-secondary hover:border-primary-light hover:text-primary-light hover:bg-primary-subtle/30"
                          onclick={() => {
                            alloc.amount = totalAmount;
                            allocations = allocations;
                          }}
                        >
                          {labels.exactAmountShortcut}
                        </button>
                        <button
                          class="flex-1 py-1 rounded-lg border border-danger/30 text-[10px] font-semibold text-danger hover:bg-danger-subtle/30 transition-all"
                          onclick={() => {
                            alloc.amount = 0;
                            allocations = allocations;
                          }}
                        >
                          {labels.reset}
                        </button>
                      </div>
                    {/if}
                  </div>
                {/each}

                {#if allocations.length === 0}
                  <div class="text-center py-4 text-[11px] text-text-muted">
                    {labels.selectPaymentMethodHint}
                  </div>
                {/if}
              </div>

              <!-- Fixed bottom: Summary + Actions -->
              <div class="shrink-0">
                <!-- Summary bar: Total | Paid | Remaining (U1) -->
                <div
                  class="grid grid-cols-3 gap-2 px-1 pb-2 mb-2 border-b border-border/30 text-center"
                >
                  <div>
                    <p
                      class="text-[10px] uppercase tracking-wider text-text-muted"
                    >
                      {labels.total}
                    </p>
                    <p
                      class="text-sm font-semibold text-text-primary tabular-nums"
                    >
                      {totalAmount.toLocaleString("id-ID")}
                    </p>
                  </div>
                  <div>
                    <p
                      class="text-[10px] uppercase tracking-wider text-text-muted"
                    >
                      {labels.paid}
                    </p>
                    <p
                      class="text-sm font-semibold text-text-primary tabular-nums"
                    >
                      {totalAllocated.toLocaleString("id-ID")}
                    </p>
                  </div>
                  <div>
                    <p
                      class="text-[10px] uppercase tracking-wider text-text-muted"
                    >
                      {labels.remaining}
                    </p>
                    {#if remainingBalance > 0}
                      <p class="text-sm font-bold text-danger tabular-nums">
                        {remainingBalance.toLocaleString("id-ID")} &#9888;
                      </p>
                    {:else if remainingBalance < 0 && overTenderOnCash}
                      <p class="text-sm font-bold text-success tabular-nums">
                        {labels.changeDue}
                        {changeDue.toLocaleString("id-ID")}
                      </p>
                    {:else if remainingBalance < 0}
                      <p class="text-sm font-bold text-danger tabular-nums">
                        {labels.overpaid}
                        {Math.abs(remainingBalance).toLocaleString("id-ID")}
                      </p>
                    {:else}
                      <p class="text-sm font-bold text-success tabular-nums">
                        &#10003;
                      </p>
                    {/if}
                  </div>
                </div>
                <!-- Actions -->
                <div class="flex gap-2 pt-2">
                  <Button
                    variant="secondary"
                    class="flex-1 py-2"
                    onclick={close}
                  >
                    {labels.cancelEsc}
                  </Button>
                  <Button
                    variant="success"
                    class="flex-1 py-2"
                    id="checkout-done"
                    disabled={cart.length === 0 || !canComplete}
                    onclick={openConfirm}
                  >
                    <Check size={14} />
                    {labels.doneEnter}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style></style>
