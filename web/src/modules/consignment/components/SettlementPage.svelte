<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { Button, Modal, Input, EmptyState, Badge, SelectSearch } from '$shared/ui';
  import { Wallet, Banknote } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';
  import {
    getSettlementPreview,
    createSettlement,
    listSettlements,
    listPaymentMethods,
    createPayout,
  } from '../services/consignment-service';
  import type { Arrangement, Settlement } from '../types';
  import { SETTLEMENT_STATUS_LABELS, SETTLEMENT_PAID } from '../types';
  import { formatCurrency, formatDateTime } from '../lib/format';

  let {
    arrangement,
    canSettle,
    canPay,
    onsettled,
  }: {
    arrangement: Arrangement;
    canSettle: boolean;
    canPay: boolean;
    onsettled?: () => void;
  } = $props();

  let preview = $state<Settlement | null>(null);
  let settlements = $state<Settlement[]>([]);
  let loading = $state(true);
  let previewLoading = $state(false);
  let showCreateModal = $state(false);
  let creating = $state(false);

  let showPayoutModal = $state(false);
  let payoutSettlement = $state<Settlement | null>(null);
  let paying = $state(false);
  let paymentMethods = $state<{ value: number; label: string }[]>([]);
  let payoutForm = $state({ payment_method_id: undefined as number | undefined, amount: 0, reference_number: '', notes: '' });

  async function loadPreview() {
    previewLoading = true;
    try {
      preview = await getSettlementPreview(arrangement.supplier_id);
    } catch {
      preview = null;
    } finally {
      previewLoading = false;
    }
  }

  async function load() {
    loading = true;
    try {
      settlements = await listSettlements(arrangement.supplier_id);
    } catch {
      settlements = [];
    } finally {
      loading = false;
    }
  }

  async function loadPaymentMethods() {
    try {
      const methods = await listPaymentMethods();
      paymentMethods = methods.map((m) => ({
        value: m.id,
        label: m.code ? `${m.name} (${m.code})` : m.name,
      }));
    } catch {
      paymentMethods = [];
    }
  }

  function openCreate() {
    showCreateModal = true;
  }

  async function submitCreate() {
    if (!preview || (preview.items?.length ?? 0) === 0) {
      toast.error(labels.consignmentNoUnsettledSalesError);
      return;
    }
    creating = true;
    try {
      const st = await createSettlement({ supplier_id: arrangement.supplier_id });
      toast.success(t('consignmentSettlementCreated', { number: st.settlement_number }));
      showCreateModal = false;
      await load();
      await loadPreview();
      onsettled?.();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || e.message || labels.consignmentCreateSettlementError);
    } finally {
      creating = false;
    }
  }

  function openPayout(st: Settlement) {
    if (st.status === SETTLEMENT_PAID) return;
    payoutSettlement = st;
    payoutForm = {
      payment_method_id: undefined,
      amount: st.total_payable,
      reference_number: '',
      notes: '',
    };
    showPayoutModal = true;
  }

  async function submitPayout() {
    if (!payoutSettlement) return;
    if (!payoutForm.payment_method_id) {
      toast.error(labels.consignmentSelectPaymentMethodError);
      return;
    }
    if (payoutForm.amount <= 0) {
      toast.error(labels.consignmentAmountGreaterThanZero);
      return;
    }
    paying = true;
    try {
      const payout = await createPayout(payoutSettlement.id, {
        payment_method_id: payoutForm.payment_method_id,
        amount: payoutForm.amount,
        reference_number: payoutForm.reference_number || undefined,
        notes: payoutForm.notes || undefined,
      });
      toast.success(t('consignmentPayoutRecorded', { number: payout.payout_number }));
      showPayoutModal = false;
      await load();
      await loadPreview();
      onsettled?.();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || e.message || labels.consignmentRecordPayoutError);
    } finally {
      paying = false;
    }
  }

  onMount(() => {
    load();
    loadPreview();
    loadPaymentMethods();
  });
</script>

<div class="space-y-4">
  <div class="card">
    <div class="flex items-center justify-between px-4 py-3 border-b border-border/50">
      <h2 class="font-semibold text-text-primary">{labels.consignmentUnsettled}</h2>
      {#if canSettle}
        <Button variant="secondary" size="sm" onclick={openCreate} disabled={!preview || (preview.items?.length ?? 0) === 0}>
          <Wallet class="w-4 h-4" /> {labels.consignmentCreateSettlement}
        </Button>
      {/if}
    </div>

    {#if previewLoading}
      <div class="p-8 text-center text-sm text-text-secondary">{labels.loading}</div>
    {:else if !preview || (preview.items?.length ?? 0) === 0}
      <EmptyState
        icon={Wallet}
        title={labels.consignmentNoUnsettled}
        subtitle={labels.consignmentNoUnsettledSubtitle}
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50">
              <th class="px-4 py-3">{labels.consignmentProduct}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentQty}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentUnitPrice}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentSubtotal}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentStoreShare}</th>
            </tr>
          </thead>
          <tbody>
            {#each preview.items as item}
              <tr class="border-b border-border/40">
                <td class="px-4 py-3 font-medium text-text-primary">{item.product_name}</td>
                <td class="px-4 py-3 text-right text-text-primary">{item.quantity}</td>
                <td class="px-4 py-3 text-right text-text-secondary">{formatCurrency(item.unit_price)}</td>
                <td class="px-4 py-3 text-right text-text-primary">{formatCurrency(item.subtotal)}</td>
                <td class="px-4 py-3 text-right text-text-primary">{formatCurrency(item.store_share)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50 flex flex-wrap gap-6 text-sm">
        <span class="text-text-secondary">{labels.consignmentTotalSales}: <span class="font-semibold text-text-primary">{formatCurrency(preview.total_sale_value)}</span></span>
        <span class="text-text-secondary">{labels.consignmentStoreShare}: <span class="font-semibold text-text-primary">{formatCurrency(preview.total_store_share)}</span></span>
        <span class="text-text-secondary">{labels.consignmentPayableToSupplier}: <span class="font-semibold text-primary">{formatCurrency(preview.total_payable)}</span></span>
      </div>
    {/if}
  </div>

  <div class="card">
    <div class="px-4 py-3 border-b border-border/50">
      <h2 class="font-semibold text-text-primary">{labels.consignmentSettlementHistory}</h2>
    </div>
    {#if loading}
      <div class="p-8 text-center text-sm text-text-secondary">{labels.loading}</div>
    {:else if settlements.length === 0}
      <EmptyState icon={Banknote} title={labels.consignmentNoSettlements} subtitle={labels.consignmentNoSettlementsSubtitle} />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50">
              <th class="px-4 py-3">{labels.consignmentSettlementNo}</th>
              <th class="px-4 py-3">{labels.consignmentDate}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentTotal}</th>
              <th class="px-4 py-3">{labels.consignmentStatus}</th>
              <th class="px-4 py-3 text-right">{labels.actions}</th>
            </tr>
          </thead>
          <tbody>
            {#each settlements as st}
              <tr class="border-b border-border/40">
                <td class="px-4 py-3 font-medium text-text-primary">{st.settlement_number}</td>
                <td class="px-4 py-3 text-text-secondary">{formatDateTime(st.created_at)}</td>
                <td class="px-4 py-3 text-right text-text-primary">{formatCurrency(st.total_payable)}</td>
                <td class="px-4 py-3">
                  <Badge variant={st.status === SETTLEMENT_PAID ? 'success' : 'warning'}>
                    {labels[SETTLEMENT_STATUS_LABELS[st.status]] || st.status}
                  </Badge>
                </td>
                <td class="px-4 py-3 text-right">
                  {#if canPay && st.status !== SETTLEMENT_PAID}
                    <Button variant="secondary" size="sm" onclick={() => openPayout(st)}>
                      {labels.consignmentPay}
                    </Button>
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

<Modal bind:open={showCreateModal} title={labels.consignmentConfirmSettlement} size="md">
  {#snippet children()}
    <p class="text-sm text-text-secondary">
      {t('consignmentSettlementConfirmText', { count: preview?.items?.length ?? 0, total: formatCurrency(preview?.total_payable ?? 0) })}
    </p>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showCreateModal = false)}>{labels.cancel}</Button>
      <Button onclick={submitCreate} disabled={creating}>
        {creating ? labels.saving : labels.consignmentCreateSettlement}
      </Button>
    </div>
  {/snippet}
</Modal>

<Modal bind:open={showPayoutModal} title={labels.consignmentRecordPayment} size="md">
  {#snippet children()}
    <div class="space-y-4">
      <div class="rounded-xl bg-surface-subtle/60 border border-border-default px-4 py-3 text-sm flex justify-between">
        <span class="text-text-secondary">{labels.consignmentOutstanding}</span>
        <span class="font-semibold text-text-primary">{formatCurrency(payoutSettlement?.total_payable)}</span>
      </div>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentPaymentMethod} <span class="text-danger">*</span></span>
        <SelectSearch
          bind:value={payoutForm.payment_method_id}
          options={paymentMethods}
          placeholder={labels.consignmentSelectPaymentMethod}
          searchPlaceholder={labels.consignmentSearchMethod}
          notFoundText={labels.consignmentNotFound}
        />
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentAmount} <span class="text-danger">*</span></span>
        <Input type="number" min="1" bind:value={payoutForm.amount} class="h-9 text-sm" />
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.consignmentReference}</span>
        <Input type="text" bind:value={payoutForm.reference_number} placeholder={labels.consignmentReferencePlaceholder} class="h-9 text-sm" />
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.notes}</span>
        <Input tag="textarea" bind:value={payoutForm.notes} rows={2} placeholder={labels.consignmentNotesPlaceholder} class="text-sm" />
      </label>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showPayoutModal = false)}>{labels.cancel}</Button>
      <Button onclick={submitPayout} disabled={paying}>
        {paying ? labels.saving : labels.consignmentPay}
      </Button>
    </div>
  {/snippet}
</Modal>