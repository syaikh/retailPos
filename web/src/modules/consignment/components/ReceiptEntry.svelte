<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { Button, Modal, Input, SelectSearch, EmptyState } from '$shared/ui';
  import { Plus, Trash2, Truck } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';
  import { getProductOptions } from '$modules/product/services/product-service';
  import { createReceipt, listReceipts } from '../services/consignment-service';
  import type { Arrangement, Receipt } from '../types';
  import { formatCurrency, formatDateTime } from '../lib/format';

  let {
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
    brought_qty: number;
    rejected_qty: number;
    notes: string;
    conflict?: string;
  }

  let receipts = $state<Receipt[]>([]);
  let loading = $state(true);
  let productOptions = $state<{ value: number; label: string }[]>([]);
  let showEntryModal = $state(false);
  let submitting = $state(false);
  let lines = $state<Line[]>([]);
  let entryNotes = $state('');
  let termByProduct = $state<Record<number, { price: number; store_share_type: string; store_share_value: number }>>({});

  async function load() {
    loading = true;
    try {
      receipts = await listReceipts(arrangement.supplier_id);
    } catch {
      receipts = [];
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

  function openEntry() {
    lines = [{ product_id: undefined, brought_qty: 1, rejected_qty: 0, notes: '' }];
    entryNotes = '';
    termByProduct = {};
    (arrangement.terms || []).forEach((t) => {
      termByProduct[t.product_id] = {
        price: t.price,
        store_share_type: t.store_share_type,
        store_share_value: t.store_share_value,
      };
    });
    showEntryModal = true;
  }

  function addLine() {
    lines = [...lines, { product_id: undefined, brought_qty: 1, rejected_qty: 0, notes: '' }];
  }

  function removeLine(index: number) {
    lines = lines.filter((_, i) => i !== index);
  }

  function acceptedQty(line: Line): number {
    const a = Math.max(0, line.brought_qty - (line.rejected_qty || 0));
    return a;
  }

  async function submitEntry() {
    if (lines.length === 0) {
      toast.error(labels.consignmentAddAtLeastOneLine);
      return;
    }
    const items = lines
      .filter((l) => l.product_id && acceptedQty(l) > 0)
      .map((l) => ({
        product_id: l.product_id!,
        accepted_qty: acceptedQty(l),
        notes: l.notes || undefined,
      }));
    if (items.length === 0) {
      toast.error(labels.consignmentEnterAcceptedQty);
      return;
    }
    submitting = true;
    try {
      const rec = await createReceipt({
        arrangement_id: arrangement.id,
        notes: entryNotes || undefined,
        items,
      });
      toast.success(t('consignmentReceiptRecorded', { number: rec.receipt_number }));
      showEntryModal = false;
      await load();
      oncreated?.();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || e.message || labels.consignmentRecordReceiptError);
    } finally {
      submitting = false;
    }
  }

  onMount(() => {
    load();
    loadProducts();
  });
</script>

<div class="space-y-4">
  <div class="card">
    <div class="flex items-center justify-between px-4 py-3 border-b border-border/50">
      <h2 class="font-semibold text-text-primary">{labels.consignmentReceiptHistory}</h2>
      {#if canCreate}
        <Button variant="secondary" size="sm" onclick={openEntry}>
          <Plus class="w-4 h-4" /> {labels.consignmentRecordReceipt}
        </Button>
      {/if}
    </div>

    {#if loading}
      <div class="p-8 text-center text-sm text-text-secondary">{labels.loading}</div>
    {:else if receipts.length === 0}
      <EmptyState
        icon={Truck}
        title={labels.consignmentNoReceipts}
        subtitle={labels.consignmentNoReceiptsSubtitle}
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50">
              <th class="px-4 py-3">{labels.consignmentReceiptNo}</th>
              <th class="px-4 py-3">{labels.consignmentDate}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentItems}</th>
              <th class="px-4 py-3">{labels.consignmentTotalValue}</th>
            </tr>
          </thead>
          <tbody>
            {#each receipts as r}
              <tr class="border-b border-border/40">
                <td class="px-4 py-3 font-medium text-text-primary">{r.receipt_number}</td>
                <td class="px-4 py-3 text-text-secondary">{formatDateTime(r.received_at)}</td>
                <td class="px-4 py-3 text-right text-text-secondary">{t('consignmentItemCount', { count: r.items?.length ?? 0 })}</td>
                <td class="px-4 py-3 text-text-primary">
                  {formatCurrency((r.items || []).reduce((s, i) => s + i.accepted_qty * i.price, 0))}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>

<Modal bind:open={showEntryModal} title={labels.consignmentRecordReceipt} size="lg">
  {#snippet children()}
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <span class="text-sm font-medium text-text-secondary">{labels.consignmentItemLines}</span>
        <Button variant="secondary" size="sm" onclick={addLine}>
          <Plus class="w-4 h-4" /> {labels.consignmentAddLine}
        </Button>
      </div>

      {#each lines as line, i (i)}
        <div class="rounded-xl border border-border-default p-3 space-y-3">
          <div class="grid grid-cols-1 md:grid-cols-[1fr_auto_auto_auto] gap-3 items-end">
            <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
              <span>{labels.consignmentProduct} <span class="text-danger">*</span></span>
              <SelectSearch
                bind:value={line.product_id}
                options={productOptions}
                placeholder={labels.consignmentSelectProduct}
                searchPlaceholder={labels.consignmentSearchProduct}
                notFoundText={labels.consignmentProductNotFound}
              />
            </label>
            <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
              <span>{labels.consignmentBrought}</span>
              <Input type="number" min="0" bind:value={line.brought_qty} class="h-9 w-24 text-sm" />
            </label>
            <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
              <span>{labels.consignmentRejected}</span>
              <Input type="number" min="0" bind:value={line.rejected_qty} class="h-9 w-24 text-sm" />
            </label>
            {#if lines.length > 1}
              <Button variant="ghost" size="sm" aria-label={labels.consignmentDeleteLine} onclick={() => removeLine(i)}>
                <Trash2 class="w-4 h-4" />
              </Button>
            {/if}
          </div>
          <div class="flex flex-wrap items-center gap-4 text-xs">
            <span class="text-text-secondary">
              {labels.consignmentAccepted} <span class="font-semibold text-text-primary">{acceptedQty(line)}</span>
            </span>
            {#if line.product_id && termByProduct[line.product_id]}
              <span class="text-text-secondary">
                {labels.consignmentTabTerms}: <span class="font-medium text-text-primary">{formatCurrency(termByProduct[line.product_id].price)}</span> {labels.consignmentPerUnit}
              </span>
            {:else if line.product_id}
              <span class="text-amber-600">{labels.consignmentNoTermsWarning}</span>
            {/if}
            {#if line.conflict}
              <span class="text-danger">{line.conflict}</span>
            {/if}
          </div>
        </div>
      {/each}

      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.notes}</span>
        <Input tag="textarea" bind:value={entryNotes} rows={2} placeholder={labels.consignmentReceiptNotesPlaceholder} class="text-sm" />
      </label>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showEntryModal = false)}>{labels.cancel}</Button>
      <Button onclick={submitEntry} disabled={submitting}>
        {submitting ? labels.saving : labels.save}
      </Button>
    </div>
  {/snippet}
</Modal>