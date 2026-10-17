<script lang="ts">
  import { onMount } from "svelte";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import {
    Button,
    Modal,
    Input,
    NumberInput,
    EmptyState,
    Badge,
  } from "$shared/ui";
  import { Plus, ClipboardCheck, CheckCircle2 } from "lucide-svelte";
  import { labels, t } from "$shared/i18n";
  import {
    listReconciliations,
    getReconciliationCountSheet,
    getReconciliation,
    createReconciliation,
    saveReconciliationCount,
    postReconciliation,
    fetchAllPages,
  } from "../services/consignment-service";
  import type {
    Arrangement,
    Reconciliation,
    ReconciliationCountSheetRow,
  } from "../types";
  import {
    RECONCILE_STATUS_LABELS,
    RECONCILE_STATUS_DRAFT,
    RECONCILE_STATUS_COUNTED,
    RECONCILE_STATUS_POSTED,
    RECONCILE_REASON_LABELS,
    RECONCILE_REASONS,
    reconcileReasonCreatesLiability,
  } from "../types";
  import { formatDateTime, formatCurrency } from "../lib/format";

  const {
    arrangement,
    canReconcile,
    oncreated,
  }: {
    arrangement: Arrangement;
    canReconcile: boolean;
    oncreated?: () => void;
  } = $props();

  let records = $state<Reconciliation[]>([]);
  let loading = $state(true);
  let loadSeq = 0;

  let starting = $state(false);
  let editorOpen = $state(false);
  let editorRecord = $state<Reconciliation | null>(null);
  let sheet = $state<ReconciliationCountSheetRow[]>([]);
  let counts = $state<Record<number, { physical: number; reason: string }>>({});
  let saving = $state(false);

  let postOpen = $state(false);
  let posting = $state(false);

  const editorItems = $derived(
    sheet.map((row) => {
      const entry = counts[row.product_id] ?? {
        physical: row.system_available,
        reason: "",
      };
      const variance = entry.physical - row.system_available;
      return { row, entry, variance };
    }),
  );
  const editorShortfall = $derived(
    editorItems.reduce((s, i) => (i.variance < 0 ? s - i.variance : s), 0),
  );
  const editorSurplus = $derived(
    editorItems.reduce((s, i) => (i.variance > 0 ? s + i.variance : s), 0),
  );

  async function load() {
    const seq = ++loadSeq;
    loading = true;
    try {
      const all = await fetchAllPages((p) =>
        listReconciliations(arrangement.supplier_id, undefined, p),
      );
      if (seq !== loadSeq) return;
      records = all.filter((r) => r.arrangement_id === arrangement.id);
    } catch {
      if (seq !== loadSeq) return;
      records = [];
    } finally {
      if (seq === loadSeq) loading = false;
    }
  }

  function storeShare(row: ReconciliationCountSheetRow, qty: number): number {
    if (row.store_share_type === "percentage") {
      return Math.trunc((row.price * qty * row.store_share_value) / 100);
    }
    return Math.trunc(row.store_share_value) * qty;
  }

  // Mirrors the server pricing: only lost/stolen shortfalls become a store
  // liability, valued at the supplier entitlement the store failed to protect.
  function previewLiability(productId: number): number {
    const item = editorItems.find((i) => i.row.product_id === productId);
    if (!item || item.variance >= 0) return 0;
    if (!reconcileReasonCreatesLiability(item.entry.reason)) return 0;
    const qty = -item.variance;
    const entitlement = item.row.price * qty;
    return Math.max(0, entitlement - storeShare(item.row, qty));
  }

  async function startCount() {
    starting = true;
    try {
      const rec = await createReconciliation({
        arrangement_id: arrangement.id,
      });
      toast.success(
        t("reconcileCreated", { number: rec.reconciliation_number }),
      );
      await load();
      await openEditor(rec.id);
      oncreated?.();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.reconcileCreateError));
    } finally {
      starting = false;
    }
  }

  async function openEditor(id: number) {
    editorRecord = null;
    sheet = [];
    counts = {};
    try {
      const [sheetRows, rec] = await Promise.all([
        getReconciliationCountSheet(arrangement.id),
        getReconciliation(id),
      ]);
      const existing = new Map((rec.items ?? []).map((i) => [i.product_id, i]));
      sheet = sheetRows;
      const next: Record<number, { physical: number; reason: string }> = {};
      for (const row of sheetRows) {
        const item = existing.get(row.product_id);
        next[row.product_id] = {
          physical: item ? item.physical_available : row.system_available,
          reason: item?.reason ?? "",
        };
      }
      counts = next;
      editorRecord = rec;
      editorOpen = true;
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.reconcileCreateError));
    }
  }

  function reasonMissing(): boolean {
    return editorItems.some((i) => i.variance !== 0 && !i.entry.reason);
  }

  async function saveCount() {
    if (!editorRecord) return;
    if (reasonMissing()) {
      toast.error(labels.reconcileReasonRequired);
      return;
    }
    saving = true;
    try {
      await saveReconciliationCount(editorRecord.id, {
        items: editorItems.map((i) => ({
          product_id: i.row.product_id,
          physical_available: i.entry.physical,
          reason: i.variance !== 0 ? i.entry.reason : undefined,
        })),
      });
      toast.success(labels.reconcileSaved);
      editorOpen = false;
      editorRecord = null;
      await load();
      oncreated?.();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.reconcileSaveError));
    } finally {
      saving = false;
    }
  }

  function askPost(rec: Reconciliation) {
    editorRecord = rec;
    postOpen = true;
  }

  async function confirmPost() {
    if (!editorRecord) return;
    posting = true;
    try {
      await postReconciliation(editorRecord.id);
      toast.success(labels.reconcilePosted);
      postOpen = false;
      editorRecord = null;
      await load();
      oncreated?.();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.reconcilePostError));
    } finally {
      posting = false;
    }
  }

  onMount(load);
</script>

<div class="space-y-4">
  <div class="card">
    <div
      class="flex items-center justify-between px-4 py-3 border-b border-border/50"
    >
      <div>
        <h2 class="font-semibold text-text-primary">{labels.reconcileTitle}</h2>
        <p class="text-xs text-text-secondary">{labels.reconcileSubtitle}</p>
      </div>
      {#if canReconcile}
        <Button
          variant="secondary"
          size="sm"
          onclick={startCount}
          disabled={starting}
        >
          <Plus class="w-4 h-4" />
          {labels.reconcileNew}
        </Button>
      {/if}
    </div>

    {#if loading}
      <div class="p-8 text-center text-sm text-text-secondary">
        {labels.loading}
      </div>
    {:else if records.length === 0}
      <EmptyState
        icon={ClipboardCheck}
        title={labels.reconcileNoRecords}
        subtitle={labels.reconcileNoRecordsSubtitle}
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-muted/50">
            <tr
              class="text-left text-xs uppercase tracking-wider text-text-secondary"
            >
              <th class="p-4">{labels.reconcileNumber}</th>
              <th class="p-4">{labels.consignmentDate}</th>
              <th class="p-4 text-right">{labels.reconcileTotalShortfall}</th>
              <th class="p-4 text-right">{labels.reconcileTotalSurplus}</th>
              <th class="p-4 text-right">{labels.consignmentLiability}</th>
              <th class="p-4">{labels.consignmentStatus}</th>
              <th class="p-4 text-right">{labels.actions}</th>
            </tr>
          </thead>
          <tbody>
            {#each records as rec (rec.id)}
              <tr
                class="border-t border-border hover:bg-surface-hover/50 transition-colors"
              >
                <td class="p-4 font-medium text-text-primary"
                  >{rec.reconciliation_number}</td
                >
                <td class="p-4 text-text-secondary"
                  >{formatDateTime(rec.created_at)}</td
                >
                <td class="p-4 text-right">{rec.total_shortfall}</td>
                <td class="p-4 text-right">{rec.total_surplus}</td>
                <td class="p-4 text-right"
                  >{rec.total_liability
                    ? formatCurrency(rec.total_liability)
                    : "-"}</td
                >
                <td class="p-4">
                  <Badge
                    variant={rec.status === RECONCILE_STATUS_POSTED
                      ? "success"
                      : rec.status === RECONCILE_STATUS_COUNTED
                        ? "warning"
                        : "muted"}
                  >
                    {labels[RECONCILE_STATUS_LABELS[rec.status]] || rec.status}
                  </Badge>
                </td>
                <td class="p-4 text-right">
                  {#if canReconcile}
                    {#if rec.status === RECONCILE_STATUS_DRAFT}
                      <Button
                        variant="secondary"
                        size="sm"
                        onclick={() => openEditor(rec.id)}
                      >
                        {labels.edit}
                      </Button>
                    {:else if rec.status === RECONCILE_STATUS_COUNTED}
                      <Button
                        variant="primary"
                        size="sm"
                        onclick={() => askPost(rec)}
                      >
                        <CheckCircle2 class="w-4 h-4" />
                        {labels.reconcilePost}
                      </Button>
                    {/if}
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>

<Modal
  bind:open={editorOpen}
  title={editorRecord?.reconciliation_number || labels.reconcileCountSheet}
  size="lg"
>
  <div class="space-y-4">
    <p class="text-sm text-text-secondary">{labels.reconcileLiabilityHint}</p>
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr
            class="text-left text-xs uppercase tracking-wider text-text-secondary"
          >
            <th class="p-2">{labels.reconcileProduct}</th>
            <th class="p-2 text-right">{labels.reconcileSystem}</th>
            <th class="p-2 text-right">{labels.reconcilePhysical}</th>
            <th class="p-2 text-right">{labels.reconcileVariance}</th>
            <th class="p-2">{labels.reconcileReason}</th>
            <th class="p-2 text-right">{labels.reconcileLiability}</th>
          </tr>
        </thead>
        <tbody>
          {#each editorItems as item (item.row.product_id)}
            <tr class="border-t border-border">
              <td class="p-2">
                <div class="font-medium text-text-primary">
                  {item.row.product_name}
                </div>
                <div class="text-xs text-text-secondary">
                  {item.row.product_sku}
                </div>
              </td>
              <td class="p-2 text-right">{item.row.system_available}</td>
              <td class="p-2 text-right">
                <NumberInput
                  min="0"
                  bind:value={counts[item.row.product_id].physical}
                  class="h-8 w-24 text-sm"
                />
              </td>
              <td class="p-2 text-right">
                <span
                  class={item.variance < 0
                    ? "text-danger"
                    : item.variance > 0
                      ? "text-warning"
                      : "text-text-secondary"}
                >
                  {item.variance > 0 ? "+" : ""}{item.variance}
                </span>
              </td>
              <td class="p-2">
                {#if item.variance !== 0}
                  <Input
                    tag="select"
                    bind:value={counts[item.row.product_id].reason}
                    class="h-8 text-sm"
                  >
                    <option value="">{labels.reconcileSelectReason}</option>
                    {#each RECONCILE_REASONS as reason (reason)}
                      <option value={reason}
                        >{labels[RECONCILE_REASON_LABELS[reason]]}</option
                      >
                    {/each}
                  </Input>
                {:else}
                  <span class="text-text-muted">—</span>
                {/if}
              </td>
              <td class="p-2 text-right">
                {previewLiability(item.row.product_id)
                  ? formatCurrency(previewLiability(item.row.product_id))
                  : "-"}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <div class="flex gap-6 text-sm">
      <span
        >{labels.reconcileTotalShortfall}:
        <strong class="text-danger">{editorShortfall}</strong></span
      >
      <span
        >{labels.reconcileTotalSurplus}:
        <strong class="text-warning">{editorSurplus}</strong></span
      >
    </div>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (editorOpen = false)}
        >{labels.cancel}</Button
      >
      <Button onclick={saveCount} disabled={saving}>
        {saving ? labels.saving : labels.reconcileSaveCount}
      </Button>
    </div>
  {/snippet}
</Modal>

<Modal bind:open={postOpen} title={labels.reconcilePostConfirm} size="sm">
  <p class="text-sm text-text-secondary">
    {t("reconcilePostConfirmText", {
      shortfall: editorRecord?.total_shortfall ?? 0,
      surplus: editorRecord?.total_surplus ?? 0,
    })}
  </p>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (postOpen = false)}
        >{labels.cancel}</Button
      >
      <Button variant="danger" onclick={confirmPost} disabled={posting}>
        {posting ? labels.saving : labels.reconcilePost}
      </Button>
    </div>
  {/snippet}
</Modal>
