<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { Button, Modal, Input, SelectSearch, EmptyState, Badge } from '$shared/ui';
  import { Plus, RotateCcw } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';
  import { getProductOptions } from '$modules/product/services/product-service';
  import { listStock, listPendingReturns, createPendingReturn } from '../services/consignment-service';
  import type { Arrangement, PendingReturn, StockRow } from '../types';
  import {
    PENDING_RETURN_STATUS_LABELS,
    RETURN_REASON_LABELS,
    RETURN_REASONS,
  } from '../types';
  import { formatDateTime } from '../lib/format';

  let {
    arrangement,
    canCreate,
    oncreated,
  }: {
    arrangement: Arrangement;
    canCreate: boolean;
    oncreated?: () => void;
  } = $props();

  let pendingReturns = $state<PendingReturn[]>([]);
  let stockRows = $state<StockRow[]>([]);
  let loading = $state(true);
  let showModal = $state(false);
  let submitting = $state(false);
  let productOptions = $state<{ value: number; label: string }[]>([]);
  let stockOptions = $state<{ value: number; label: string }[]>([]);
  let form = $state({ product_id: undefined as number | undefined, qty: 1, reason: 'damaged', notes: '' });

  async function load() {
    loading = true;
    try {
      const [prs, stock] = await Promise.all([
        listPendingReturns(arrangement.supplier_id),
        listStock(arrangement.supplier_id),
      ]);
      pendingReturns = prs;
      stockRows = stock;
      stockOptions = stock
        .filter((s) => s.available_qty > 0)
        .map((s) => ({
          value: s.product_id,
          label: s.product_sku ? `${s.product_name} (${s.product_sku}) — ${labels.consignmentAvailableStock} ${s.available_qty}` : `${s.product_name} — ${labels.consignmentAvailableStock} ${s.available_qty}`,
        }));
    } catch {
      pendingReturns = [];
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

  function maxQtyFor(productId?: number): number {
    const row = stockRows.find((s) => s.product_id === productId);
    return row?.available_qty ?? 0;
  }

  function openModal() {
    form = { product_id: undefined, qty: 1, reason: 'damaged', notes: '' };
    showModal = true;
  }

  async function submit() {
    if (!form.product_id) {
      toast.error(labels.consignmentSelectProductError);
      return;
    }
    if (form.qty <= 0) {
      toast.error(labels.consignmentQtyGreaterThanZero);
      return;
    }
    const max = maxQtyFor(form.product_id);
    if (form.qty > max) {
      toast.error(t('consignmentQtyExceedsStock', { max }));
      return;
    }
    submitting = true;
    try {
      await createPendingReturn({
        product_id: form.product_id,
        qty: form.qty,
        reason: form.reason,
        notes: form.notes || undefined,
      });
      toast.success(labels.consignmentPendingReturnRecorded);
      showModal = false;
      await load();
      oncreated?.();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || e.message || labels.consignmentRecordPendingReturnError);
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
      <h2 class="font-semibold text-text-primary">{labels.consignmentPendingReturns}</h2>
      {#if canCreate}
        <Button variant="secondary" size="sm" onclick={openModal}>
          <Plus class="w-4 h-4" /> {labels.consignmentRecordPendingReturn}
        </Button>
      {/if}
    </div>

    {#if loading}
      <div class="p-8 text-center text-sm text-text-secondary">{labels.loading}</div>
    {:else if pendingReturns.length === 0}
      <EmptyState
        icon={RotateCcw}
        title={labels.consignmentNoPendingReturns}
        subtitle={labels.consignmentNoPendingReturnsSubtitle}
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50">
              <th class="px-4 py-3">{labels.consignmentProduct}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentQty}</th>
              <th class="px-4 py-3">{labels.consignmentReason}</th>
              <th class="px-4 py-3">{labels.consignmentStatus}</th>
              <th class="px-4 py-3">{labels.consignmentDate}</th>
            </tr>
          </thead>
          <tbody>
            {#each pendingReturns as pr}
              <tr class="border-b border-border/40">
                <td class="px-4 py-3">
                  <div class="font-medium text-text-primary">{pr.product_name}</div>
                  <div class="text-xs text-text-secondary">{pr.product_sku}</div>
                </td>
                <td class="px-4 py-3 text-right text-text-primary">{pr.qty}</td>
                <td class="px-4 py-3 text-text-secondary">{labels[RETURN_REASON_LABELS[pr.reason]] || pr.reason}</td>
                <td class="px-4 py-3">
                  <Badge variant={pr.status === 'open' ? 'warning' : 'success'}>
                    {labels[PENDING_RETURN_STATUS_LABELS[pr.status]] || pr.status}
                  </Badge>
                </td>
                <td class="px-4 py-3 text-text-secondary">{formatDateTime(pr.created_at)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>

<Modal bind:open={showModal} title={labels.consignmentRecordPendingReturn} size="md">
  {#snippet children()}
    <div class="space-y-4">
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentProductFromStock} <span class="text-danger">*</span></span>
        <SelectSearch
          bind:value={form.product_id}
          options={stockOptions.length ? stockOptions : productOptions}
          placeholder={labels.consignmentSelectProduct}
          searchPlaceholder={labels.consignmentSearchProduct}
          notFoundText={labels.consignmentNoStockAvailable}
          onchange={(v) => {
            form.product_id = v;
            const max = maxQtyFor(v);
            if (form.qty > max && max > 0) form.qty = max;
          }}
        />
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentQty} <span class="text-danger">*</span></span>
        <Input type="number" min="1" bind:value={form.qty} class="h-9 text-sm" />
        {#if form.product_id && maxQtyFor(form.product_id) > 0}
          <span class="text-xs text-text-muted">{t('consignmentMax', { max: maxQtyFor(form.product_id) })}</span>
        {/if}
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentReason} <span class="text-danger">*</span></span>
        <Input tag="select" bind:value={form.reason} class="h-9 text-sm">
          {#snippet children()}
            {#each RETURN_REASONS as reason}
              <option value={reason}>{labels[RETURN_REASON_LABELS[reason]]}</option>
            {/each}
          {/snippet}
        </Input>
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.notes}</span>
        <Input tag="textarea" bind:value={form.notes} rows={2} placeholder={labels.consignmentNotesPlaceholder} class="text-sm" />
      </label>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showModal = false)}>{labels.cancel}</Button>
      <Button onclick={submit} disabled={submitting}>
        {submitting ? labels.saving : labels.save}
      </Button>
    </div>
  {/snippet}
</Modal>