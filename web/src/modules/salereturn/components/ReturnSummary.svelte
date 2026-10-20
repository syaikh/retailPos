<script lang="ts">
  import { labels } from "$shared/i18n";
  import { formatCurrency } from "$shared/utils/currency";
  import type { ReturnableItem, ReturnLineSelection } from "../types";

  let {
    items,
    selected = {},
    threshold = 0,
  }: {
    items: ReturnableItem[];
    selected?: Record<number, ReturnLineSelection>;
    threshold?: number;
  } = $props();

  const chosen = $derived(
    items.filter((it) => selected[it.sale_item_id]?.checked),
  );
  const totalUnits = $derived(
    chosen.reduce(
      (sum, it) => sum + (selected[it.sale_item_id]?.quantity ?? 0),
      0,
    ),
  );
  // unit_price is the tax-inclusive line value, so unit_price × qty is the
  // same refund figure the server computes for a refund outcome; a
  // replacement outcome pays nothing out, so this stays an estimate.
  const estimatedRefund = $derived(
    chosen.reduce(
      (sum, it) =>
        sum + it.unit_price * (selected[it.sale_item_id]?.quantity ?? 0),
      0,
    ),
  );
  const needsApproval = $derived(estimatedRefund > threshold);
</script>

<div
  class="rounded-xl border border-border bg-surface-subtle/30 px-4 py-3 space-y-1"
>
  <div class="flex items-center justify-between text-sm">
    <span class="text-text-secondary"
      >{labels.returnSelectedCount.replace("{count}", String(totalUnits))}</span
    >
    <span class="font-semibold text-text-primary"
      >{formatCurrency(estimatedRefund)}</span
    >
  </div>
  <p class="text-xs text-text-secondary">{labels.returnEstimatedRefund}</p>
  {#if needsApproval}
    <p class="text-xs text-warning-light">
      {labels.returnApprovalNeeded.replace(
        "{amount}",
        formatCurrency(threshold),
      )}
    </p>
  {/if}
</div>
