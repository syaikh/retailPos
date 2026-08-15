<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { Button, Modal, Input, SelectSearch, EmptyState } from '$shared/ui';
  import { Plus, Trash2 } from 'lucide-svelte';
  import { labels } from '$shared/i18n';
  import { getProductOptions } from '$modules/product/services/product-service';
  import { setTerms } from '../services/consignment-service';
  import type { Arrangement, Term, SetTermsPayload } from '../types';
  import {
    SHARE_TYPE_PERCENTAGE,
    SHARE_TYPE_FIXED_AMOUNT,
    SHARE_TYPE_LABELS,
  } from '../types';
  import { formatCurrency } from '../lib/format';

  let {
    arrangement,
    canUpdate,
    onsaved,
  }: {
    arrangement: Arrangement;
    canUpdate: boolean;
    onsaved?: () => void;
  } = $props();

  interface TermRow {
    product_id?: number;
    price: number;
    store_share_type: string;
    store_share_value: number;
  }

  let terms = $state<Term[]>([]);
  let loading = $state(true);
  let productOptions = $state<{ value: number; label: string }[]>([]);
  let showAddModal = $state(false);
  let saving = $state(false);
  let newRow = $state<TermRow>({
    product_id: undefined,
    price: 0,
    store_share_type: SHARE_TYPE_PERCENTAGE,
    store_share_value: 20,
  });

  async function load() {
    loading = true;
    try {
      terms = arrangement.terms || [];
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

  function openAdd() {
    newRow = {
      product_id: undefined,
      price: 0,
      store_share_type: SHARE_TYPE_PERCENTAGE,
      store_share_value: 20,
    };
    showAddModal = true;
  }

  async function submitAdd() {
    if (!newRow.product_id) {
      toast.error(labels.consignmentSelectProductError);
      return;
    }
    if (newRow.store_share_type === SHARE_TYPE_PERCENTAGE && (newRow.store_share_value <= 0 || newRow.store_share_value >= 100)) {
      toast.error(labels.consignmentPercentRange);
      return;
    }
    if (newRow.store_share_type === SHARE_TYPE_FIXED_AMOUNT && newRow.store_share_value <= 0) {
      toast.error(labels.consignmentShareGreaterThanZero);
      return;
    }
    saving = true;
    try {
      // The backend replaces ALL terms on save, so merge the existing rows
      // with the newly added one — otherwise "Tambah Term" wipes everything.
      const payload: SetTermsPayload[] = terms.map((t) => ({
        product_id: t.product_id,
        price: t.price,
        store_share_type: t.store_share_type,
        store_share_value: t.store_share_value,
      }));
      payload.push({
        product_id: newRow.product_id as number,
        price: newRow.price,
        store_share_type: newRow.store_share_type,
        store_share_value: newRow.store_share_value,
      });
      const saved = await setTerms(arrangement.id, payload);
      terms = saved;
      toast.success(labels.consignmentTermsSaved);
      showAddModal = false;
      onsaved?.();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || e.message || labels.consignmentTermsSaveError);
    } finally {
      saving = false;
    }
  }

  function shareLabel(t: Term): string {
    if (t.store_share_type === SHARE_TYPE_PERCENTAGE) return `${t.store_share_value}%`;
    return formatCurrency(t.store_share_value);
  }

  onMount(() => {
    load();
    loadProducts();
  });
</script>

<div class="card">
  <div class="flex items-center justify-between px-4 py-3 border-b border-border/50">
    <h2 class="font-semibold text-text-primary">{labels.consignmentTermsHeader}</h2>
    {#if canUpdate}
      <Button variant="secondary" size="sm" onclick={openAdd}>
        <Plus class="w-4 h-4" /> {labels.consignmentAddTerm}
      </Button>
    {/if}
  </div>

  {#if loading}
    <div class="p-8 text-center text-sm text-text-secondary">{labels.loading}</div>
  {:else if terms.length === 0}
    <EmptyState
      icon={Plus}
      title={labels.consignmentNoTerms}
      subtitle={labels.consignmentNoTermsSubtitle}
    />
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50">
            <th class="px-4 py-3">{labels.consignmentProduct}</th>
            <th class="px-4 py-3 text-right">{labels.consignmentPrice}</th>
            <th class="px-4 py-3">{labels.consignmentStoreShare}</th>
          </tr>
        </thead>
        <tbody>
          {#each terms as t}
            <tr class="border-b border-border/40">
              <td class="px-4 py-3">
                <div class="font-medium text-text-primary">{t.product_name}</div>
                <div class="text-xs text-text-secondary">{t.product_sku}</div>
              </td>
              <td class="px-4 py-3 text-right text-text-primary">{formatCurrency(t.price)}</td>
              <td class="px-4 py-3 text-text-secondary">
                {labels[SHARE_TYPE_LABELS[t.store_share_type]]} — {shareLabel(t)}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<Modal bind:open={showAddModal} title={labels.consignmentAddTerm} size="md">
  {#snippet children()}
    <div class="space-y-4">
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentProduct} <span class="text-danger">*</span></span>
        <SelectSearch
          bind:value={newRow.product_id}
          options={productOptions}
          placeholder={labels.consignmentSelectProduct}
          searchPlaceholder={labels.consignmentSearchProduct}
          notFoundText={labels.consignmentProductNotFound}
        />
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentPrice} (Rp) <span class="text-danger">*</span></span>
        <Input type="number" min="0" bind:value={newRow.price} class="h-9 text-sm" />
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentShareType} <span class="text-danger">*</span></span>
        <Input tag="select" bind:value={newRow.store_share_type} class="h-9 text-sm">
          {#snippet children()}
            <option value={SHARE_TYPE_PERCENTAGE}>{labels[SHARE_TYPE_LABELS[SHARE_TYPE_PERCENTAGE]]}</option>
            <option value={SHARE_TYPE_FIXED_AMOUNT}>{labels[SHARE_TYPE_LABELS[SHARE_TYPE_FIXED_AMOUNT]]}</option>
          {/snippet}
        </Input>
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{newRow.store_share_type === SHARE_TYPE_PERCENTAGE ? labels.consignmentSharePercentLabel : labels.consignmentShareFixedLabel} <span class="text-danger">*</span></span>
        <Input type="number" min="0" bind:value={newRow.store_share_value} class="h-9 text-sm" />
      </label>
      <p class="text-xs text-text-muted">{labels.consignmentTermsNote}</p>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showAddModal = false)}>{labels.cancel}</Button>
      <Button onclick={submitAdd} disabled={saving}>
        {saving ? labels.saving : labels.save}
      </Button>
    </div>
  {/snippet}
</Modal>