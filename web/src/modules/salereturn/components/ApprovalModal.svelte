<script lang="ts">
  import { Modal, Button, Input, Badge } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { formatCurrency } from "$shared/utils/currency";
  import { formatDateTimeInJakarta } from "$shared/utils/jakartaTime";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import { returnStatusLabel, returnStatusVariant } from "../lib/labels";
  import {
    getReturnById,
    approveReturn,
    rejectReturn,
  } from "../services/salereturn-service";

  let {
    returnId = null,
    show = $bindable(false),
    ondecided = () => {},
  }: {
    returnId?: number | null;
    show?: boolean;
    ondecided?: () => void;
  } = $props();

  let detail = $state<Awaited<ReturnType<typeof getReturnById>>>(null);
  let loading = $state(false);
  let processing = $state(false);
  let note = $state("");
  let rejectReason = $state("");
  let mode = $state<"approve" | "reject">("approve");

  // Approval re-runs the stock/outcome decision inside its own transaction,
  // so the detail is re-fetched on every open rather than reused.
  $effect(() => {
    if (show && returnId) load(returnId);
  });

  async function load(id: number) {
    loading = true;
    detail = null;
    note = "";
    rejectReason = "";
    mode = "approve";
    try {
      detail = await getReturnById(id);
    } catch (e) {
      toast.error(getApiErrorMessage(e, labels.returnApprovalLoadFailed));
    } finally {
      loading = false;
    }
  }

  async function decide() {
    if (!detail) return;
    if (mode === "reject" && !rejectReason.trim()) {
      toast.error(labels.returnRejectReasonRequired);
      return;
    }
    processing = true;
    try {
      if (mode === "approve") {
        await approveReturn(detail.id, note.trim() || undefined);
        toast.success(labels.returnApproveSuccess);
      } else {
        await rejectReturn(detail.id, rejectReason.trim());
        toast.success(labels.returnRejectSuccess);
      }
      show = false;
      ondecided();
    } catch (e) {
      toast.error(getApiErrorMessage(e, labels.returnApprovalFailed));
    } finally {
      processing = false;
    }
  }
</script>

<Modal
  bind:open={show}
  title="{labels.returnApprovals} — {detail?.return_number ?? ''}"
  size="lg"
>
  {#if loading}
    <p class="py-8 text-center text-sm text-text-secondary">
      {labels.loading}
    </p>
  {:else if !detail}
    <p class="py-8 text-center text-sm text-text-secondary">
      {labels.returnApprovalLoadFailed}
    </p>
  {:else}
    <div class="space-y-5">
      <div class="flex items-center gap-2">
        <Badge variant={returnStatusVariant(detail.status)} size="sm"
          >{returnStatusLabel(detail.status)}</Badge
        >
        <span class="text-xs text-text-secondary">
          {labels.returnSaleRef} #{detail.original_sale_id}
        </span>
      </div>

      <div class="rounded-xl bg-surface-subtle/30 px-4 py-3 text-sm space-y-1">
        <div class="flex justify-between">
          <span class="text-text-secondary">{labels.returnReason}</span>
          <span class="text-text-primary">{detail.reason}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-text-secondary">{labels.dateAndTime}</span>
          <span class="text-text-primary"
            >{formatDateTimeInJakarta(detail.created_at)}</span
          >
        </div>
        {#if detail.expires_at}
          <div class="flex justify-between">
            <span class="text-text-secondary">{labels.returnExpires}</span>
            <span class="text-text-primary"
              >{formatDateTimeInJakarta(detail.expires_at)}</span
            >
          </div>
        {/if}
        <div class="flex justify-between border-t border-border/50 pt-1">
          <span class="font-semibold text-text-primary"
            >{labels.returnTotalRefund}</span
          >
          <span class="font-semibold text-text-primary"
            >{formatCurrency(detail.total_refund_amount)}</span
          >
        </div>
      </div>

      <div class="space-y-2">
        {#each detail.items as item (item.id)}
          <div
            class="rounded-xl border border-border bg-surface-subtle/30 px-4 py-2.5 flex items-center justify-between text-sm"
          >
            <div class="min-w-0">
              <p class="truncate font-medium text-text-primary">
                {item.product_name}
              </p>
              <p class="text-xs text-text-secondary">
                {labels.qty}: {item.quantity} × {formatCurrency(
                  item.unit_price,
                )}
              </p>
            </div>
            <span class="font-medium text-text-primary"
              >{formatCurrency(item.subtotal)}</span
            >
          </div>
        {/each}
      </div>

      {#if mode === "approve"}
        <div class="space-y-1">
          <label
            for="approval-note"
            class="text-xs font-semibold text-text-secondary"
            >{labels.returnNotesOptional}</label
          >
          <Input
            tag="textarea"
            id="approval-note"
            class="min-h-[60px] resize-none"
            bind:value={note}
          />
        </div>
      {:else}
        <div class="space-y-1">
          <label
            for="rejection-reason"
            class="text-xs font-semibold text-text-secondary"
            >{labels.returnRejectReason} *</label
          >
          <Input
            tag="textarea"
            id="rejection-reason"
            class="min-h-[60px] resize-none"
            bind:value={rejectReason}
          />
        </div>
      {/if}
    </div>
  {/if}
  {#snippet footer()}
    <Button
      variant="secondary"
      class="rounded-xl px-4 h-11 text-sm font-semibold"
      disabled={processing}
      onclick={() => (show = false)}>{labels.cancel}</Button
    >
    <Button
      variant="danger"
      class="rounded-xl px-4 h-11 text-sm font-semibold"
      disabled={processing || loading}
      onclick={() => {
        // First click reveals the mandatory reason field; the second
        // submits. Calling decide() on the first click would toast
        // "reason required" before the reviewer has typed anything.
        if (mode !== "reject") {
          mode = "reject";
          return;
        }
        decide();
      }}
    >
      {mode === "reject" ? labels.returnConfirmReject : labels.reject}
    </Button>
    <Button
      variant="success"
      class="rounded-xl px-4 h-11 text-sm font-semibold"
      disabled={processing || loading}
      onclick={() => {
        mode = "approve";
        decide();
      }}>{labels.approve}</Button
    >
  {/snippet}
</Modal>
