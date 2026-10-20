<script lang="ts">
  import { Modal, Button, Badge } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { onMount } from "svelte";
  import { formatCurrency } from "$shared/utils/currency";
  import { formatDateTimeInJakarta } from "$shared/utils/jakartaTime";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import { useWebSocket } from "$shared/api/websocket";
  import ApprovalModal from "./ApprovalModal.svelte";
  import { returnStatusLabel, returnStatusVariant } from "../lib/labels";
  import { listPendingApprovals } from "../services/salereturn-service";
  import type { SaleReturn } from "../types";

  let {
    show = $bindable(false),
    ondecided = () => {},
  }: {
    show?: boolean;
    ondecided?: () => void;
  } = $props();

  let items = $state<SaleReturn[]>([]);
  let loading = $state(false);
  let activeId = $state<number | null>(null);
  let showReview = $state(false);

  const ws = useWebSocket();

  // The queue is fetched on every open; approval/rejection from the review
  // modal refreshes it so the decided return drops off the list.
  $effect(() => {
    if (show) load();
  });

  // While the queue is open, a live event keeps it current without the manager
  // reopening it: a new request adds a row, a resolution drops one. Guarded on
  // `show` so a closed modal does not fire redundant API calls.
  onMount(() => {
    const unsubRequested = ws.on("return_approval_requested", () => {
      if (show) load();
    });
    const unsubResolved = ws.on("return_approval_resolved", () => {
      if (show) load();
    });
    return () => {
      unsubRequested();
      unsubResolved();
    };
  });

  async function load() {
    loading = true;
    try {
      items = await listPendingApprovals();
    } catch (e) {
      items = [];
      toast.error(getApiErrorMessage(e, labels.returnApprovalLoadFailed));
    } finally {
      loading = false;
    }
  }
</script>

<!-- Persistent while the review modal is stacked on top: both are Modal
     instances and each binds Escape on window, so a single Escape would
     otherwise close the review AND the queue behind it. The queue is also
     the modal the reviewer returns to, so dismissing it by accident is
     never what they meant. -->
<Modal
  bind:open={show}
  title={labels.returnApprovals}
  size="xl"
  persistent={showReview}
>
  {#if loading}
    <p class="py-8 text-center text-sm text-text-secondary">
      {labels.loading}
    </p>
  {:else if items.length === 0}
    <p class="py-8 text-center text-sm text-text-secondary">
      {labels.returnApprovalsEmpty}
    </p>
  {:else}
    <ul class="space-y-2">
      {#each items as ret (ret.id)}
        <li
          class="flex items-center gap-3 rounded-xl border border-border bg-surface-subtle/30 px-4 py-3"
        >
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm font-semibold text-text-primary"
                >{ret.return_number}</span
              >
              <Badge variant={returnStatusVariant(ret.status)} size="sm"
                >{returnStatusLabel(ret.status)}</Badge
              >
            </div>
            <p class="mt-1 text-xs text-text-secondary">
              {labels.returnSaleRef} #{ret.original_sale_id} ·
              {formatDateTimeInJakarta(ret.created_at)}
              {#if ret.expires_at}
                · {labels.returnExpires}
                {formatDateTimeInJakarta(ret.expires_at)}
              {/if}
            </p>
          </div>
          <div class="shrink-0 text-right">
            <p class="text-sm font-semibold text-text-primary">
              {formatCurrency(ret.total_refund_amount)}
            </p>
            <Button
              variant="secondary"
              size="sm"
              class="mt-1.5"
              onclick={() => {
                activeId = ret.id;
                showReview = true;
              }}>{labels.returnApprovalReview}</Button
            >
          </div>
        </li>
      {/each}
    </ul>
  {/if}
  {#snippet footer()}
    <Button
      variant="secondary"
      class="rounded-xl px-4 h-11 text-sm font-semibold"
      onclick={() => (show = false)}>{labels.close}</Button
    >
  {/snippet}
</Modal>

<ApprovalModal
  returnId={activeId}
  bind:show={showReview}
  ondecided={() => {
    load();
    ondecided();
  }}
/>
