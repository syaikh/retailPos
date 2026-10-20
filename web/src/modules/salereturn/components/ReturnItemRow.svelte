<script lang="ts">
  import { Badge } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { formatCurrency } from "$shared/utils/currency";
  import { Minus, Plus } from "lucide-svelte";
  import type { ReturnableItem } from "../types";

  let {
    item,
    selected = false,
    quantity = 1,
    ontoggle,
    onquantity,
  }: {
    item: ReturnableItem;
    selected?: boolean;
    quantity?: number;
    ontoggle?: (selected: boolean) => void;
    onquantity?: (quantity: number) => void;
  } = $props();

  // Consignment lines and fully consumed lines are never selectable:
  // ProcessReturn rejects consignment outright and the over-return guard
  // rejects exhausted lines, so refusing them here avoids a failed submit.
  const selectable = $derived(
    !item.consignment && item.returnable_quantity > 0,
  );

  const showsReplacement = $derived(
    selected && item.stock_available >= quantity,
  );

  function changeQuantity(delta: number) {
    const next = Math.min(
      Math.max(1, quantity + delta),
      item.returnable_quantity,
    );
    onquantity?.(next);
  }
</script>

<div
  class="rounded-xl border px-4 py-3 {selected
    ? 'border-primary-default/40 bg-primary-subtle/20'
    : 'border-border bg-surface-subtle/30'}"
>
  <div class="flex items-start gap-3">
    <input
      type="checkbox"
      class="mt-1 h-4 w-4 shrink-0 rounded border-border text-primary-default focus:ring-primary-default/30"
      checked={selected}
      disabled={!selectable}
      onchange={(e) => ontoggle?.(e.currentTarget.checked)}
      aria-label="{labels.returnSelectItems}: {item.product_name}"
    />
    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-sm font-medium text-text-primary"
          >{item.product_name}</span
        >
        {#if item.consignment}
          <Badge variant="warning" size="sm"
            >{labels.returnConsignmentNotSupported}</Badge
          >
        {:else if item.returnable_quantity <= 0}
          <Badge variant="muted" size="sm">{labels.returnAlreadyReturned}</Badge
          >
        {/if}
      </div>
      <div
        class="mt-1 flex flex-wrap items-center gap-3 text-xs text-text-secondary"
      >
        <span>{labels.returnSoldQty}: {item.quantity}</span>
        <span>{labels.returnAlreadyReturned}: {item.returned_quantity}</span>
        <span>{labels.returnAvailableQty}: {item.returnable_quantity}</span>
        <span>{formatCurrency(item.unit_price)}</span>
      </div>
      {#if selected}
        <div class="mt-2 flex flex-wrap items-center gap-3 text-xs">
          <div class="flex items-center gap-1.5">
            <button
              type="button"
              class="flex h-6 w-6 items-center justify-center rounded-md border border-border text-text-secondary transition-colors hover:bg-surface-default disabled:opacity-40"
              disabled={quantity <= 1}
              onclick={() => changeQuantity(-1)}
              aria-label="{labels.qty} −"
            >
              <Minus class="h-3 w-3" />
            </button>
            <span
              class="w-6 text-center text-sm font-semibold text-text-primary"
              aria-live="polite">{quantity}</span
            >
            <button
              type="button"
              class="flex h-6 w-6 items-center justify-center rounded-md border border-border text-text-secondary transition-colors hover:bg-surface-default disabled:opacity-40"
              disabled={quantity >= item.returnable_quantity}
              onclick={() => changeQuantity(1)}
              aria-label="{labels.qty} +"
            >
              <Plus class="h-3 w-3" />
            </button>
          </div>
          {#if showsReplacement}
            <Badge variant="success" size="sm">
              {labels.returnReplacement} ({labels.returnStock}: {item.stock_available})
            </Badge>
          {:else}
            <Badge variant="muted" size="sm">{labels.returnRefund}</Badge>
          {/if}
          <span class="font-medium text-text-primary"
            >{formatCurrency(item.unit_price * quantity)}</span
          >
        </div>
      {/if}
    </div>
  </div>
</div>
