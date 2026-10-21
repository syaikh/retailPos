<script lang="ts">
  import { Modal, Button, Input } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { formatDateTimeInJakarta } from "$shared/utils/jakartaTime";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import { printReceiptWithToast } from "$shared/services/print-service";
  import ReturnItemRow from "./ReturnItemRow.svelte";
  import ReturnSummary from "./ReturnSummary.svelte";
  import {
    ineligibleText,
    returnConditionLabel,
    returnReasonLabel,
  } from "../lib/labels";
  import { buildReturnReceipt } from "../lib/receipt";
  import {
    getReturnableSale,
    processReturn,
  } from "../services/salereturn-service";
  import type {
    ProcessReturnRequest,
    ReturnLineSelection,
    SaleReturn,
  } from "../types";

  let {
    saleId = null,
    show = $bindable(false),
    onreturned = () => {},
  }: {
    saleId?: number | null;
    show?: boolean;
    onreturned?: (returnStatus: string) => void;
  } = $props();

  let data = $state<Awaited<ReturnType<typeof getReturnableSale>>>(null);
  let loading = $state(false);
  let submitting = $state(false);
  let reason = $state("damaged");
  let notes = $state("");
  let selected = $state<Record<number, ReturnLineSelection>>({});
  // A just-completed return: the modal stays open on it so the cashier can
  // hand the customer their receipt (Phase 4.4), mirroring the POS's
  // post-checkout print. Pending returns have nothing to print yet.
  let completed = $state<SaleReturn | null>(null);
  let printing = $state(false);

  // (Re)load the pre-flight snapshot every time the modal opens: quantities,
  // eligibility and stock are all live server state, so a stale snapshot
  // would only surface as a failed submit.
  $effect(() => {
    if (show && saleId) {
      load(saleId);
    }
  });

  async function load(id: number) {
    loading = true;
    data = null;
    selected = {};
    reason = "damaged";
    notes = "";
    completed = null;
    try {
      data = await getReturnableSale(id);
    } catch (e) {
      toast.error(getApiErrorMessage(e, labels.returnLoadFailed));
    } finally {
      loading = false;
    }
  }

  function toggle(id: number, checked: boolean) {
    selected = {
      ...selected,
      [id]: { checked, quantity: selected[id]?.quantity ?? 1 },
    };
  }

  function setQuantity(id: number, quantity: number) {
    const prev = selected[id];
    if (!prev) return;
    selected = { ...selected, [id]: { ...prev, quantity } };
  }

  const selectedItems = $derived(
    (data?.items ?? []).filter((it) => selected[it.sale_item_id]?.checked),
  );
  // A fully-consignment sale is eligible by design (PRD §10.1 lists no
  // consignment entry-point guard) yet has no selectable line, so the modal
  // would otherwise open on a permanently disabled submit with no reason.
  const hasSelectable = $derived(
    (data?.items ?? []).some(
      (it) => !it.consignment && it.returnable_quantity > 0,
    ),
  );
  const canSubmit = $derived(
    !submitting &&
      !loading &&
      data?.eligible === true &&
      selectedItems.length > 0,
  );

  async function submit() {
    if (!data || selectedItems.length === 0) return;
    const request: ProcessReturnRequest = {
      reason,
      condition: "damaged",
      items: selectedItems.map((it) => ({
        sale_item_id: it.sale_item_id,
        quantity: selected[it.sale_item_id]?.quantity ?? 1,
      })),
    };
    if (notes.trim()) request.notes = notes.trim();

    submitting = true;
    try {
      const ret = await processReturn(data.sale_id, request);
      toast.success(
        ret.status === "pending_approval"
          ? labels.returnSuccessPending
          : labels.returnSuccess,
      );
      if (ret.status === "completed") {
        // Stay open on the result view so the receipt can be printed now or
        // reprinted later from the transaction drawer.
        completed = ret;
      } else {
        show = false;
      }
      onreturned(ret.status);
    } catch (e) {
      toast.error(getApiErrorMessage(e, labels.returnFailed));
    } finally {
      submitting = false;
    }
  }

  /** Print the just-completed return's receipt (Phase 4.4). Failure surfaces
   * as the shared print-agent toast; the result view stays open so the cashier
   * can retry. */
  async function printReturnReceipt() {
    if (!completed || !data) return;
    printing = true;
    try {
      await printReceiptWithToast(
        buildReturnReceipt(completed, data.invoice_number),
      );
    } finally {
      printing = false;
    }
  }

  function close() {
    completed = null;
    show = false;
  }
</script>

<Modal
  bind:open={show}
  title="{labels.returnItems} — {data?.invoice_number ?? ''}"
  size="lg"
>
  {#if loading}
    <p class="py-8 text-center text-sm text-text-secondary">
      {labels.loading}
    </p>
  {:else if completed}
    <div class="space-y-4">
      <div
        class="rounded-xl border border-success/30 bg-success/10 px-4 py-3 text-sm text-success-light"
      >
        {labels.returnSuccess}
      </div>
      <div class="space-y-2">
        <div class="flex items-center justify-between text-sm">
          <span class="text-text-secondary">{labels.returnReceiptNumber}</span>
          <span class="text-text-primary">{completed.return_number}</span>
        </div>
        <div class="flex items-center justify-between text-sm">
          <span class="text-text-secondary">{labels.returnReason}</span>
          <span class="text-text-primary"
            >{returnReasonLabel(completed.reason)}</span
          >
        </div>
        <div class="flex items-center justify-between text-sm">
          <span class="text-text-secondary">{labels.returnCondition}</span>
          <span class="text-text-primary"
            >{returnConditionLabel(completed.condition)}</span
          >
        </div>
        <div class="border-t border-border-default"></div>
        {#each completed.items as it (it.id)}
          <div class="flex items-center justify-between text-sm">
            <span class="text-text-primary"
              >{it.product_name} × {it.quantity}</span
            >
            {#if completed.return_type === "refund"}
              <span class="text-text-secondary"
                >{it.subtotal.toLocaleString("id-ID")}</span
              >
            {:else}
              <span class="text-text-secondary">{labels.returnReplacement}</span
              >
            {/if}
          </div>
        {/each}
        {#if completed.return_type === "refund"}
          <div class="flex items-center justify-between text-sm font-semibold">
            <span class="text-text-secondary">{labels.returnTotalRefund}</span>
            <span class="text-text-primary"
              >{completed.total_refund_amount.toLocaleString("id-ID")}</span
            >
          </div>
        {/if}
      </div>
    </div>
  {:else if !data}
    <p class="py-8 text-center text-sm text-text-secondary">
      {labels.returnLoadFailed}
    </p>
  {:else if !data.eligible}
    <p
      class="rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning-light"
    >
      {ineligibleText(data.ineligible_reason, data.window_days)}
    </p>
  {:else}
    <div class="space-y-5">
      <div class="rounded-xl bg-surface-subtle/30 px-4 py-3">
        <p class="text-xs font-semibold text-text-secondary">
          {labels.returnOriginalSale}
        </p>
        <p class="mt-1 text-sm text-text-primary">
          {formatDateTimeInJakarta(data.created_at)}
        </p>
      </div>

      <div class="space-y-2">
        <p class="text-sm font-semibold text-text-primary">
          {labels.returnSelectItems}
        </p>
        {#if !hasSelectable}
          <p
            class="rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning-light"
          >
            {labels.returnNoSelectableItems}
          </p>
        {/if}
        {#each data.items as item (item.sale_item_id)}
          <ReturnItemRow
            {item}
            selected={selected[item.sale_item_id]?.checked ?? false}
            quantity={selected[item.sale_item_id]?.quantity ?? 1}
            ontoggle={(checked) => toggle(item.sale_item_id, checked)}
            onquantity={(qty) => setQuantity(item.sale_item_id, qty)}
          />
        {/each}
      </div>

      <div class="space-y-1">
        <label
          for="return-reason"
          class="text-xs font-semibold text-text-secondary"
          >{labels.returnReason}</label
        >
        <select
          id="return-reason"
          class="w-full rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary transition-colors duration-200 focus:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary-default/20"
          bind:value={reason}
        >
          <option value="damaged">{labels.returnReasonCodeDamaged}</option>
          <option value="defective">{labels.returnReasonCodeDefective}</option>
          <option value="wrong_item">{labels.returnReasonCodeWrongItem}</option>
          <option value="other">{labels.returnReasonCodeOther}</option>
        </select>
        <p class="text-[11px] text-text-muted">
          {labels.returnCondition}: {labels.returnConditionDamaged} — {labels.returnConditionOnly}
        </p>
      </div>

      <div class="space-y-1">
        <label
          for="return-notes"
          class="text-xs font-semibold text-text-secondary"
          >{labels.returnNotesOptional}</label
        >
        <Input
          tag="textarea"
          id="return-notes"
          class="min-h-[60px] resize-none"
          bind:value={notes}
        />
      </div>

      <ReturnSummary
        items={data.items}
        {selected}
        threshold={data.approval_threshold}
      />
    </div>
  {/if}
  {#snippet footer()}
    {#if completed}
      <Button
        variant="secondary"
        class="rounded-xl px-4 h-11 text-sm font-semibold"
        onclick={close}>{labels.close}</Button
      >
      <Button
        variant="primary"
        class="rounded-xl px-4 h-11 text-sm font-semibold"
        disabled={printing}
        onclick={printReturnReceipt}>{labels.returnPrintReceipt}</Button
      >
    {:else}
      <Button
        variant="secondary"
        class="rounded-xl px-4 h-11 text-sm font-semibold"
        onclick={() => (show = false)}>{labels.cancel}</Button
      >
      {#if data?.eligible}
        <Button
          variant="primary"
          class="rounded-xl px-4 h-11 text-sm font-semibold"
          disabled={!canSubmit}
          onclick={submit}
        >
          {#if selectedItems.length === 0}
            {labels.returnSelectItems}
          {:else}
            {labels.returnSubmit}
          {/if}
        </Button>
      {/if}
    {/if}
  {/snippet}
</Modal>
