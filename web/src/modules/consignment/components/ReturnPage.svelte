<script lang="ts">
  import { onMount } from "svelte";
  import { toast } from "$shared/stores/toast.svelte";
  import {
    Button,
    Modal,
    Input,
    NumberInput,
    SelectSearch,
    EmptyState,
    Pagination,
  } from "$shared/ui";
  import { Plus, Trash2, RotateCcw } from "lucide-svelte";
  import { labels, t } from "$shared/i18n";
  import { getProductOptions } from "$modules/product/services/product-service";
  import {
    listReturns,
    createReturn,
    listPendingReturns,
  } from "../services/consignment-service";
  import type { Arrangement, ConsignmentReturn, PendingReturn } from "../types";
  import { RETURN_REASON_LABELS, RETURN_REASONS } from "../types";
  import { formatDateTime } from "../lib/format";

  const {
    arrangement,
    canCreate,
    oncreated,
  }: {
    arrangement: Arrangement;
    canCreate: boolean;
    oncreated?: () => void;
  } = $props();

  interface Line {
    product_id?: number;
    qty: number;
    reason: string;
    pending_return_id?: number;
    notes: string;
  }

  let returns = $state<ConsignmentReturn[]>([]);
  let openPending = $state<PendingReturn[]>([]);
  let loading = $state(true);
  let showModal = $state(false);
  let submitting = $state(false);
  let productOptions = $state<{ value: number; label: string }[]>([]);
  let lines = $state<Line[]>([]);
  let returnNotes = $state("");

  let pageLimit = $state(20);
  let pageOffset = $state(0);
  const pagedReturns = $derived(
    returns.slice(pageOffset, pageOffset + pageLimit),
  );

  async function load() {
    loading = true;
    try {
      const [rts, prs] = await Promise.all([
        listReturns(arrangement.supplier_id),
        listPendingReturns(arrangement.supplier_id),
      ]);
      returns = rts;
      openPending = prs.filter((p) => p.status === "open");
    } catch {
      returns = [];
    } finally {
      loading = false;
    }
  }

  async function loadProducts() {
    try {
      const opts = await getProductOptions();
      productOptions = opts.map((p) => ({
        value: p.id,
        label: p.sku ? `${p.name} (${p.sku})` : p.name,
      }));
    } catch {
      productOptions = [];
    }
  }

  function openModal() {
    lines = [{ product_id: undefined, qty: 1, reason: "other", notes: "" }];
    returnNotes = "";
    showModal = true;
  }

  function addLine() {
    lines = [
      ...lines,
      { product_id: undefined, qty: 1, reason: "other", notes: "" },
    ];
  }

  function removeLine(index: number) {
    lines = lines.filter((_, i) => i !== index);
  }

  async function submit() {
    const items = lines
      .filter((l) => l.product_id && l.qty > 0)
      .map((l) => ({
        product_id: l.product_id!,
        qty: l.qty,
        reason: l.reason,
        pending_return_id: l.pending_return_id || undefined,
        notes: l.notes || undefined,
      }));
    if (items.length === 0) {
      toast.error(labels.consignmentEnterOneLine);
      return;
    }
    submitting = true;
    try {
      const ret = await createReturn({
        arrangement_id: arrangement.id,
        notes: returnNotes || undefined,
        items,
      });
      toast.success(
        t("consignmentReturnRecorded", { number: ret.return_number }),
      );
      showModal = false;
      await load();
      oncreated?.();
    } catch (e: unknown) {
      const raw =
        e instanceof Error ? e.message : labels.consignmentRecordReturnError;
      toast.error(raw);
    } finally {
      submitting = false;
    }
  }

  onMount(() => {
    load();
    loadProducts();
  });

  function handlePageChange(newOffset: number, newLimit: number) {
    pageOffset = newOffset;
    pageLimit = newLimit;
  }
</script>

<div class="space-y-4">
  <div class="card">
    <div
      class="flex items-center justify-between px-4 py-3 border-b border-border/50"
    >
      <h2 class="font-semibold text-text-primary">
        {labels.consignmentReturns}
      </h2>
      {#if canCreate}
        <Button variant="secondary" size="sm" onclick={openModal}>
          <Plus class="w-4 h-4" />
          {labels.consignmentRecordReturn}
        </Button>
      {/if}
    </div>

    {#if openPending.length > 0}
      <div
        class="px-4 py-3 bg-amber-50/60 border-b border-amber-200 text-sm text-amber-800"
      >
        {t("consignmentOpenPendingNotice", { count: openPending.length })}
      </div>
    {/if}

    {#if loading}
      <div class="p-8 text-center text-sm text-text-secondary">
        {labels.loading}
      </div>
    {:else if returns.length === 0}
      <EmptyState
        icon={RotateCcw}
        title={labels.consignmentNoReturns}
        subtitle={labels.consignmentNoReturnsSubtitle}
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-muted/50">
            <tr
              class="text-left text-xs uppercase tracking-wider text-text-secondary"
            >
              <th class="p-4">{labels.consignmentReturnNo}</th>
              <th class="p-4">{labels.consignmentDate}</th>
              <th class="p-4 text-right">{labels.consignmentReturnItem}</th>
              <th class="p-4 text-right">{labels.consignmentQty}</th>
            </tr>
          </thead>
          <tbody>
            {#each pagedReturns as r (r.id || r)}
              <tr
                class="border-t border-border hover:bg-surface-hover/50 transition-colors"
              >
                <td class="p-4 font-medium text-text-primary"
                  >{r.return_number}</td
                >
                <td class="p-4 text-text-secondary"
                  >{formatDateTime(r.returned_at)}</td
                >
                <td class="p-4 text-right text-text-secondary"
                  >{r.items?.length ?? 0}</td
                >
                <td class="p-4 text-right text-text-primary">
                  {(r.items || []).reduce((s, i) => s + i.qty, 0)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination
          total={returns.length}
          limit={pageLimit}
          offset={pageOffset}
          onPageChange={handlePageChange}
        />
      </div>
    {/if}
  </div>
</div>

<Modal bind:open={showModal} title={labels.consignmentRecordReturn} size="lg">
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <span class="text-sm font-medium text-text-secondary"
        >{labels.consignmentItemLines}</span
      >
      <Button variant="secondary" size="sm" onclick={addLine}>
        <Plus class="w-4 h-4" />
        {labels.consignmentAddLine}
      </Button>
    </div>

    {#each lines as line, i (i)}
      <div class="rounded-xl border border-border-default p-3 space-y-3">
        <div
          class="grid grid-cols-1 md:grid-cols-[1fr_auto_auto_auto] gap-3 items-end"
        >
          <label
            class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
          >
            <span
              >{labels.consignmentProduct}
              <span class="text-danger">*</span></span
            >
            <SelectSearch
              bind:value={line.product_id}
              options={productOptions}
              placeholder={labels.consignmentSelectProduct}
              searchPlaceholder={labels.consignmentSearchProduct}
              notFoundText={labels.consignmentProductNotFound}
            />
          </label>
          <label
            class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
          >
            <span>{labels.consignmentQty}</span>
            <NumberInput
              min="1"
              bind:value={line.qty}
              class="h-9 w-24 text-sm"
            />
          </label>
          <label
            class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
          >
            <span>{labels.consignmentReason}</span>
            <Input
              tag="select"
              bind:value={line.reason}
              class="h-9 w-36 text-sm"
            >
              {#each RETURN_REASONS as reason (reason)}
                <option value={reason}
                  >{labels[RETURN_REASON_LABELS[reason]]}</option
                >
              {/each}
            </Input>
          </label>
          {#if lines.length > 1}
            <Button
              variant="ghost"
              size="sm"
              aria-label={labels.consignmentDeleteLine}
              onclick={() => removeLine(i)}
            >
              <Trash2 class="w-4 h-4" />
            </Button>
          {/if}
        </div>

        <label
          class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
        >
          <span>{labels.consignmentLinkPendingReturn}</span>
          <Input
            tag="select"
            bind:value={line.pending_return_id}
            class="h-9 text-sm"
          >
            <option value={undefined}>{labels.consignmentNoLink}</option>
            {#each openPending as pr (pr.id || pr)}
              <option value={pr.id}>
                {pr.product_name} ×{pr.qty} ({labels[
                  RETURN_REASON_LABELS[pr.reason]
                ] || pr.reason})
              </option>
            {/each}
          </Input>
        </label>

        <label
          class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
        >
          <span>{labels.notes}</span>
          <Input
            type="text"
            bind:value={line.notes}
            placeholder={labels.consignmentNotesPlaceholder}
            class="h-9 text-sm"
          />
        </label>
      </div>
    {/each}

    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span>{labels.consignmentOverallNotes}</span>
      <Input
        tag="textarea"
        bind:value={returnNotes}
        rows={2}
        placeholder={labels.consignmentReturnNotesPlaceholder}
        class="text-sm"
      />
    </label>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showModal = false)}
        >{labels.cancel}</Button
      >
      <Button onclick={submit} disabled={submitting}>
        {submitting ? labels.saving : labels.save}
      </Button>
    </div>
  {/snippet}
</Modal>
