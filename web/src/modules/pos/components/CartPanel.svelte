<script lang="ts">
  import { slide } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import { Badge, Button } from '$shared/ui';
  import { ShoppingCart, X, Minus, Plus, Wallet, Printer, Search } from 'lucide-svelte';

  let {
    cart = [],
    totalAmount = 0,
    totalItems = 0,
    subtotal = 0,
    taxAmount = 0,
    dppDisplay = 0,
    paymentMethod = $bindable('Cash'),
    paymentOptions = [],
    selectedCustomerLabel = '',
    lastSale = null,
    checkingOut = false,
    class: className = '',
    onupdateqty = (id: number, delta: number) => {},
    onremovefromcart = (id: number) => {},
    onclearcart = () => {},
    oncheckout = () => {},
    onprintreceipt = () => {},
    onselectcustomer = () => {},
  }: {
    cart: any[];
    totalAmount: number;
    totalItems: number;
    subtotal: number;
    taxAmount: number;
    dppDisplay: number;
    paymentMethod: string;
    paymentOptions: any[];
    selectedCustomerLabel: string;
    lastSale: any;
    checkingOut: boolean;
    class?: string;
    onupdateqty?: (id: number, delta: number) => void;
    onremovefromcart?: (id: number) => void;
    onclearcart?: () => void;
    oncheckout?: () => void;
    onprintreceipt?: () => void;
    onselectcustomer?: () => void;
  } = $props();
</script>

<div class={`card flex flex-col overflow-hidden p-0 sticky top-0 h-[calc(100vh-120px)] max-h-[800px] ${className}`}>
  <div class="px-4 py-3.5 border-b border-border flex items-center justify-between shrink-0">
    <div class="flex items-center gap-2">
      <ShoppingCart size={18} class="text-primary-light" />
      <span class="font-semibold text-text-primary">Cart</span>
      {#if totalItems > 0}
        <Badge variant="primary" size="sm">{totalItems}</Badge>
      {/if}
    </div>
    {#if cart.length > 0}
      <div class="flex items-center gap-1">
        <kbd class="px-1 py-0.5 text-[10px] font-medium text-danger/60 bg-danger-subtle/30 rounded border border-danger/20 select-none">ALT+DEL</kbd>
        <Button onclick={onclearcart} variant="ghost" size="icon" class="text-xs text-danger hover:bg-danger-subtle" title="Clear cart [ALT+DEL]" aria-label="Clear cart">
          <X size={14} />
        </Button>
      </div>
    {/if}
  </div>

  {#if cart.length === 0}
    <div class="flex-1 p-4 flex flex-col items-center justify-center text-center overflow-y-auto">
      <div class="empty-state-icon bg-surface w-20 h-20 rounded-2xl mb-4 border border-dashed border-border-strong flex items-center justify-center animate-pulse">
        <ShoppingCart size={32} class="text-text-muted opacity-50" />
      </div>
      <p class="text-text-secondary font-medium">Your cart is empty</p>
      <p class="text-text-muted text-xs mt-1">Add products to start selling</p>
    </div>
  {:else}
    <div class="flex-1 overflow-y-auto divide-y divide-border overflow-x-hidden">
      {#each cart as item (item.id)}
        <div
          class="flex items-start gap-3 px-4 py-3 hover:bg-surface-hover/50 transition-colors"
          animate:flip={{ duration: 300 }}
          transition:slide={{ duration: 250 }}
        >
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-text-primary truncate">{item.name}</p>
            <p class="text-xs text-text-muted mt-0.5">
              {item.price.toLocaleString('id-ID')} × {item.quantity}
              = <span class="text-text-secondary font-medium">{(item.price * item.quantity).toLocaleString('id-ID')}</span>
            </p>
          </div>
          <div class="flex items-center gap-1.5 shrink-0 mt-0.5">
            <button
              class="w-8 h-8 rounded-lg bg-surface hover:bg-surface-hover text-text-secondary flex items-center justify-center transition-colors border border-border active:scale-95"
              onclick={() => onupdateqty(item.id, -1)}
              aria-label="Decrease quantity"
            >
              <Minus size={14} />
            </button>
            <input
              type="number"
              min="1"
              max={item.stock || 999}
              bind:value={item.quantity}
              oninput={() => {
                const maxStock = item.stock || 999;
                if (item.quantity > maxStock) {
                  item.quantity = maxStock;
                }
              }}
              onblur={() => {
                if (item.quantity <= 0) onremovefromcart(item.id);
                else {
                  const maxStock = item.stock || 999;
                  if (item.quantity > maxStock) {
                    item.quantity = maxStock;
                  }
                }
              }}
              onkeydown={(e) => {
                if (e.key === 'Enter') {
                  e.target.blur();
                }
              }}
              class="w-12 text-center text-sm font-semibold text-text-primary bg-surface border border-border rounded-xl px-1 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary"
            />
            <button
              class="w-8 h-8 rounded-lg bg-surface hover:bg-surface-hover text-text-secondary flex items-center justify-center transition-colors border border-border active:scale-95"
              onclick={() => onupdateqty(item.id, 1)}
              aria-label="Increase quantity"
            >
              <Plus size={14} />
            </button>
            <button
              class="w-8 h-8 rounded-lg hover:bg-danger-subtle text-text-muted hover:text-danger flex items-center justify-center transition-colors ml-1 active:scale-95"
              onclick={() => onremovefromcart(item.id)}
              aria-label="Remove item"
            >
              <X size={14} />
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <div class="border-t border-border p-4 space-y-3 bg-bg-secondary shrink-0">
    {#if taxAmount > 0}
      <div class="flex justify-between text-xs text-text-muted">
        <span>DPP</span>
        <span>{dppDisplay.toLocaleString('id-ID')}</span>
      </div>
      <div class="flex justify-between text-xs text-text-muted">
        <span>PPN 11%</span>
        <span>{taxAmount.toLocaleString('id-ID')}</span>
      </div>
    {/if}
    <div class="flex justify-between font-bold text-text-primary">
      <span>Total</span>
      <span class="text-white text-base">{totalAmount.toLocaleString('id-ID')}</span>
    </div>

    <div>
      <p class="text-xs text-text-muted mb-2 font-medium">Payment method</p>
      <div class="grid grid-cols-3 gap-2">
        {#each paymentOptions as opt}
          <button
            class="flex flex-col items-center gap-1 py-2 rounded-xl border text-xs font-medium transition-all {paymentMethod === opt.id ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
            onclick={() => paymentMethod = opt.id}
          >
            <opt.icon size={16} />
            {opt.label}
          </button>
        {/each}
      </div>
    </div>

    <div>
      <p class="text-xs text-text-muted mb-1 font-medium">Customer</p>
      <Button variant="secondary" class="w-full justify-between" onclick={onselectcustomer}>
        <span class="truncate">{selectedCustomerLabel}</span>
        <Search size={14} />
      </Button>
    </div>

    <Button
      variant="success"
      class="w-full py-3"
      onclick={oncheckout}
      disabled={checkingOut || cart.length === 0}
    >
      {#if checkingOut}
        <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
        Processing…
      {:else}
        <Wallet size={16} />
        Bayar [F4] · Rp {totalAmount.toLocaleString('id-ID')}
      {/if}
    </Button>

    <Button
      variant="ghost"
      class="w-full py-2 mt-2"
      onclick={onprintreceipt}
      disabled={!lastSale || !lastSale.invoice_number}
    >
      <Printer size={16} />
      {#if lastSale && lastSale.invoice_number}
        Print Last Receipt · {lastSale.invoice_number}
      {:else}
        Print Last Receipt
      {/if}
    </Button>
  </div>
</div>

<style></style>
