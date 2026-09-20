<script lang="ts">
  import { onMount } from "svelte";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import {
    Button,
    Modal,
    Input,
    NumberInput,
    SelectSearch,
    EmptyState,
    Pagination,
  } from "$shared/ui";
  import { Plus, Trash2, RotateCcw, Copy, Check } from "lucide-svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { labels, t } from "$shared/i18n";
  import {
    listReturns,
    createReturn,
    listPendingReturns,
    listStock,
    getReturn,
  } from "../services/consignment-service";
  import type {
    Arrangement,
    ConsignmentReturn,
    PendingReturn,
    StockRow,
  } from "../types";
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
    fromPending?: boolean;
  }

  let returns = $state<ConsignmentReturn[]>([]);
  let openPending = $state<PendingReturn[]>([]);
  let stockRows = $state<StockRow[]>([]);
  let loading = $state(true);
  let showModal = $state(false);
  let submitting = $state(false);
  let stockOptions = $state<{ value: number; label: string }[]>([]);
  let lines = $state<Line[]>([]);

  let selectedReturn = $state<ConsignmentReturn | null>(null);
  let showDetailModal = $state(false);
  let loadingDetail = $state(false);

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
      const stock = await listStock(arrangement.supplier_id);
      stockRows = stock;
      stockOptions = stock
        .filter(
          (s) => s.available_qty > 0 && s.arrangement_id === arrangement.id,
        )
        .map((s) => ({
          value: s.product_id,
          label: s.product_sku
            ? `${s.product_name} (${s.product_sku}) — ${labels.consignmentAvailableStock} ${s.available_qty}`
            : `${s.product_name} — ${labels.consignmentAvailableStock} ${s.available_qty}`,
        }));
    } catch {
      stockOptions = [];
    }
  }

  function openModal() {
    lines = [{ product_id: undefined, qty: 1, reason: "other" }];
    showModal = true;
  }

  function returnAllPending() {
    lines = openPending.map((pr) => ({
      product_id: pr.product_id,
      qty: pr.qty,
      reason: pr.reason,
      pending_return_id: pr.id,
      fromPending: true,
    }));
    showModal = true;
  }

  function addLine() {
    lines = [...lines, { product_id: undefined, qty: 1, reason: "other" }];
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
      }));
    if (items.length === 0) {
      toast.error(labels.consignmentEnterOneLine);
      return;
    }
    for (const item of items) {
      if (item.pending_return_id) {
        const pr = openPending.find((p) => p.id === item.pending_return_id);
        if (pr && item.qty > pr.qty) {
          toast.error(t("consignmentQtyExceedsStock", { max: pr.qty }));
          return;
        }
      } else {
        const max =
          stockRows.find(
            (s) =>
              s.product_id === item.product_id &&
              s.arrangement_id === arrangement.id,
          )?.available_qty ?? 0;
        if (item.qty > max) {
          toast.error(t("consignmentQtyExceedsStock", { max }));
          return;
        }
      }
    }
    submitting = true;
    try {
      const ret = await createReturn({
        arrangement_id: arrangement.id,
        items,
      });
      toast.success(
        t("consignmentReturnRecorded", { number: ret.return_number }),
      );
      showModal = false;
      await load();
      oncreated?.();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentRecordReturnError));
    } finally {
      submitting = false;
    }
  }

  onMount(() => {
    load();
    loadProducts();
  });

  let showCopied = $state(new SvelteSet<string>());

  function copySku(sku: string) {
    navigator.clipboard.writeText(sku).then(() => {
      const next = new SvelteSet(showCopied);
      next.add(sku);
      showCopied = next;
      toast.success(labels.copiedToClipboard);
      setTimeout(() => {
        const removed = new SvelteSet(next);
        removed.delete(sku);
        showCopied = removed;
      }, 2000);
    });
  }

  function handlePageChange(newOffset: number, newLimit: number) {
    pageOffset = newOffset;
    pageLimit = newLimit;
  }

  async function loadDetail(id: number) {
    loadingDetail = true;
    try {
      selectedReturn = await getReturn(id);
      showDetailModal = true;
    } catch {
      toast.error(labels.consignmentRecordReturnError);
    } finally {
      loadingDetail = false;
    }
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
        {#if openPending.length > 0}
          <Button variant="primary" size="sm" onclick={returnAllPending}>
            <RotateCcw class="w-4 h-4" />
            {labels.consignmentReturnAllPending}
          </Button>
        {/if}
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
                class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer"
                role="button"
                tabindex="0"
                onclick={() => loadDetail(r.id)}
                onkeydown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    loadDetail(r.id);
                  }
                }}
              >
                <td class="p-4 font-medium text-text-primary"
                  >{r.return_number}</td
                >
                <td class="p-4 text-text-secondary"
                  >{formatDateTime(r.returned_at)}</td
                >
                <td class="p-4 text-right text-text-primary">{r.total_items}</td
                >
                <td class="p-4 text-right text-text-primary font-medium">
                  {r.total_qty}
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

<Modal bind:open={showModal} title={labels.consignmentRecordReturn} size="2xl">
  <div class="space-y-4">
    {#if lines.length > 0 && lines.some((l) => l.fromPending)}
      <div
        class="flex items-center gap-2 px-3 py-2 rounded-lg bg-primary/5 border border-primary/20 text-sm text-primary"
      >
        <RotateCcw class="w-4 h-4 shrink-0" />
        <span
          >{t("consignmentOpenPendingNotice", {
            count: lines.filter((l) => l.fromPending).length,
          })}</span
        >
      </div>
    {/if}

    <div class="flex items-center justify-between">
      <span class="text-sm font-medium text-text-secondary"
        >{labels.consignmentItemLines}</span
      >
      <Button variant="secondary" size="sm" onclick={addLine}>
        <Plus class="w-4 h-4" />
        {labels.consignmentAddLine}
      </Button>
    </div>

    <div class="overflow-x-auto rounded-xl border border-border-default">
      <table class="w-full text-sm" style="min-width: 700px;">
        <thead class="bg-muted/50">
          <tr
            class="text-left text-xs uppercase tracking-wider text-text-secondary"
          >
            <th class="px-3 py-3 w-[260px]">{labels.consignmentProduct}</th>
            <th class="px-3 py-3 w-20">{labels.consignmentQty}</th>
            <th class="px-3 py-3 w-32">{labels.consignmentReason}</th>
            <th class="px-3 py-3 w-44">{labels.consignmentLinkPendingReturn}</th
            >
            <th class="px-3 py-3 w-10"></th>
          </tr>
        </thead>
        <tbody>
          {#each lines as line, i (i)}
            {@const otherSelectedIds = new Set(
              lines
                .filter((l, j) => j !== i && l.product_id)
                .map((l) => l.product_id!),
            )}
            {@const rowOptions = stockOptions.filter(
              (o) => !otherSelectedIds.has(o.value),
            )}
            {@const rowPending = openPending.filter(
              (pr) => !otherSelectedIds.has(pr.product_id),
            )}
            <tr
              class="border-t border-border/40 {line.fromPending
                ? 'bg-primary/5'
                : ''}"
            >
              <td class="px-3 py-3 w-[260px]">
                {#if line.fromPending}
                  <span class="font-medium text-text-primary text-xs">
                    {stockOptions
                      .find((o) => o.value === line.product_id)
                      ?.label?.split(" — ")[0] || `#${line.product_id}`}
                  </span>
                {:else if rowOptions.length === 0}
                  <span class="text-text-muted text-xs"
                    >{labels.consignmentAllProductsAssigned}</span
                  >
                {:else}
                  <SelectSearch
                    bind:value={line.product_id}
                    options={rowOptions}
                    placeholder={labels.consignmentSelectProduct}
                    searchPlaceholder={labels.consignmentSearchProduct}
                    notFoundText={labels.consignmentNoStockAvailable}
                    disabled={!!line.pending_return_id}
                    size="sm"
                  />
                {/if}
              </td>
              <td class="px-3 py-3 w-20">
                <NumberInput
                  min="1"
                  bind:value={line.qty}
                  class="h-8 w-16 text-xs"
                />
              </td>
              <td class="px-3 py-3 w-32">
                {#if line.fromPending}
                  <span class="text-text-primary text-xs">
                    {labels[RETURN_REASON_LABELS[line.reason]] || line.reason}
                  </span>
                {:else}
                  <Input
                    tag="select"
                    bind:value={line.reason}
                    class="h-8 text-xs"
                  >
                    {#each RETURN_REASONS as reason (reason)}
                      <option value={reason}
                        >{labels[RETURN_REASON_LABELS[reason]]}</option
                      >
                    {/each}
                  </Input>
                {/if}
              </td>
              <td class="px-3 py-3 w-44">
                {#if line.fromPending}
                  <span class="text-xs text-text-primary">
                    PR-{line.pending_return_id}
                  </span>
                {:else}
                  <Input
                    tag="select"
                    bind:value={line.pending_return_id}
                    class="h-8 text-xs"
                    onchange={(e: Event) => {
                      const val = (e.target as HTMLSelectElement).value;
                      const prId = val ? Number(val) : undefined;
                      if (prId) {
                        const pr = openPending.find((p) => p.id === prId);
                        if (pr) {
                          line.product_id = pr.product_id;
                          line.reason = pr.reason;
                        }
                      } else {
                        line.product_id = undefined;
                        line.reason = "other";
                      }
                    }}
                  >
                    <option value={undefined}>—</option>
                    {#each rowPending as pr (pr.id || pr)}
                      <option value={pr.id}>
                        {pr.product_name} ×{pr.qty}
                      </option>
                    {/each}
                  </Input>
                {/if}
              </td>
              <td class="px-3 py-3 w-10">
                {#if lines.length > 1}
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label={labels.consignmentDeleteLine}
                    onclick={() => removeLine(i)}
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </Button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
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

<Modal
  bind:open={showDetailModal}
  title={selectedReturn?.return_number || labels.consignmentReturnDetail}
  size="xl"
>
  {#if loadingDetail}
    <div class="p-8 text-center text-sm text-text-secondary">
      {labels.loading}
    </div>
  {:else if selectedReturn}
    <div class="space-y-4">
      <div class="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
        <div>
          <div class="text-text-secondary text-xs uppercase tracking-wider">
            {labels.consignmentReturnNo}
          </div>
          <div class="font-medium text-text-primary">
            {selectedReturn.return_number}
          </div>
        </div>
        <div>
          <div class="text-text-secondary text-xs uppercase tracking-wider">
            {labels.consignmentDate}
          </div>
          <div class="font-medium text-text-primary">
            {formatDateTime(selectedReturn.returned_at)}
          </div>
        </div>
        <div>
          <div class="text-text-secondary text-xs uppercase tracking-wider">
            {labels.consignmentReturnedBy}
          </div>
          <div class="font-medium text-text-primary">
            {selectedReturn.returned_by_username || "—"}
          </div>
        </div>
      </div>

      <div>
        <div class="text-text-secondary text-xs uppercase tracking-wider mb-2">
          {labels.consignmentReturnItem}
        </div>
        <div class="overflow-x-auto rounded-xl border border-border-default">
          <table class="w-full text-sm">
            <thead class="bg-muted/50">
              <tr
                class="text-left text-xs uppercase tracking-wider text-text-secondary"
              >
                <th class="px-4 py-3">{labels.consignmentProduct}</th>
                <th class="px-4 py-3 text-right">{labels.consignmentQty}</th>
                <th class="px-4 py-3">{labels.consignmentReason}</th>
              </tr>
            </thead>
            <tbody>
              {#each selectedReturn.items as item (item.id)}
                <tr class="border-t border-border/40">
                  <td class="px-4 py-3">
                    <div class="font-medium text-text-primary">
                      {item.product_name || `#${item.product_id}`}
                    </div>
                    <div class="text-xs text-text-secondary flex items-center gap-1">
                      {item.product_sku || ""}
                      {#if item.product_sku}
                        <button
                          type="button"
                          onclick={() => copySku(item.product_sku ?? "")}
                          class="p-0.5 text-text-muted hover:text-text-primary"
                          title={labels.copiedToClipboard}
                        >
                          {#if showCopied.has(item.product_sku)}
                            <Check size={10} class="text-success" />
                          {:else}
                            <Copy size={10} />
                          {/if}
                        </button>
                      {/if}
                    </div>
                  </td>
                  <td
                    class="px-4 py-3 text-right text-text-primary font-medium"
                  >
                    {item.qty}
                  </td>
                  <td class="px-4 py-3 text-text-secondary">
                    {labels[RETURN_REASON_LABELS[item.reason]] || item.reason}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  {/if}
  {#snippet footer()}
    <div class="flex justify-end">
      <Button variant="secondary" onclick={() => (showDetailModal = false)}>
        {labels.cancel}
      </Button>
    </div>
  {/snippet}
</Modal>
