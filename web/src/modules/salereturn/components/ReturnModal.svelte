<script lang="ts">
  import { Modal, Button, Input } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { formatDateTimeInJakarta } from "$shared/utils/jakartaTime";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import ReturnItemRow from "./ReturnItemRow.svelte";
  import ReturnSummary from "./ReturnSummary.svelte";
  import { ineligibleText } from "../lib/labels";
  import {
    getReturnableSale,
    processReturn,
  } from "../services/salereturn-service";
  import type { ProcessReturnRequest, ReturnLineSelection } from "../types";

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
      show = false;
      onreturned(ret.status);
    } catch (e) {
      toast.error(getApiErrorMessage(e, labels.returnFailed));
    } finally {
      submitting = false;
    }
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
  {/snippet}
</Modal>
