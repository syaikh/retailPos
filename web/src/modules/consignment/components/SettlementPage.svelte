<script lang="ts">
  import { onMount } from "svelte";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import {
    Button,
    Modal,
    Input,
    EmptyState,
    Badge,
    Pagination,
    SearchBar,
  } from "$shared/ui";
  import { Wallet, Banknote } from "lucide-svelte";
  import { labels, t } from "$shared/i18n";
  import {
    getSettlementPreview,
    createSettlement,
    listSettlements,
    getSettlement,
  } from "../services/consignment-service";
  import type { Arrangement, Settlement } from "../types";
  import PayoutModal from "./PayoutModal.svelte";
  import { SETTLEMENT_STATUS_LABELS, SETTLEMENT_PAID } from "../types";
  import { formatCurrency, formatDateTime } from "../lib/format";

  const {
    arrangement,
    canSettle,
    canPay,
    initialTab = "all",
    onsettled,
  }: {
    arrangement: Arrangement;
    canSettle: boolean;
    canPay: boolean;
    initialTab?: "all" | "pending" | "paid";
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

  let showDetailModal = $state(false);
  let detailSettlement = $state<Settlement | null>(null);
  let loadingDetail = $state(false);

  let historyTab = $state<"all" | "pending" | "paid">(initialTab);
  let historySearch = $state("");
  let previewSearch = $state("");

  let pageLimit = $state(20);
  let pageOffset = $state(0);

  const filteredSettlements = $derived.by(() => {
    let result = settlements;
    if (historyTab === "pending")
      result = result.filter((s) => s.status === "pending_payment");
    if (historyTab === "paid")
      result = result.filter((s) => s.status === "paid");
    if (historySearch.trim()) {
      const q = historySearch.toLowerCase();
      result = result.filter((s) =>
        (s.items || []).some((item) =>
          item.product_name?.toLowerCase().includes(q),
        ),
      );
    }
    return result;
  });

  const pagedSettlements = $derived(
    filteredSettlements.slice(pageOffset, pageOffset + pageLimit),
  );

  const filteredPreviewItems = $derived.by(() => {
    if (!previewSearch.trim()) return preview?.items || [];
    const q = previewSearch.toLowerCase();
    return (preview?.items || []).filter((item) =>
      item.product_name?.toLowerCase().includes(q),
    );
  });

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
      const st = await createSettlement({
        supplier_id: arrangement.supplier_id,
      });
      toast.success(
        t("consignmentSettlementCreated", { number: st.settlement_number }),
      );
      showCreateModal = false;
      await load();
      await loadPreview();
      onsettled?.();
    } catch (e: unknown) {
      toast.error(
        getApiErrorMessage(e, labels.consignmentCreateSettlementError),
      );
    } finally {
      creating = false;
    }
  }

  function openPayout(st: Settlement) {
    if (st.status === SETTLEMENT_PAID) return;
    payoutSettlement = st;
    showPayoutModal = true;
  }

  async function openDetail(settlementId: number) {
    loadingDetail = true;
    showDetailModal = true;
    try {
      detailSettlement = await getSettlement(settlementId);
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentLoadError));
      showDetailModal = false;
    } finally {
      loadingDetail = false;
    }
  }

  onMount(() => {
    load();
    loadPreview();
  });

  function handlePageChange(newOffset: number, newLimit: number) {
    pageOffset = newOffset;
    pageLimit = newLimit;
  }
</script>

<div class="space-y-4">
  <div class="card">
    <div
      class="flex items-center gap-3 px-4 py-3 border-b border-border/50"
    >
      <h2 class="font-semibold text-text-primary whitespace-nowrap">
        {labels.consignmentUnsettled}
      </h2>
      {#if !previewLoading && (preview?.items?.length ?? 0) > 0}
        <SearchBar
          bind:value={previewSearch}
          placeholder="Search product..."
          class="flex-1 max-w-xs"
        />
      {/if}
      <div class="ml-auto whitespace-nowrap">
        {#if canSettle}
          <Button
            variant="secondary"
            size="sm"
            onclick={openCreate}
            disabled={!preview || (preview.items?.length ?? 0) === 0}
          >
            <Wallet class="w-4 h-4" />
            {labels.consignmentCreateSettlement}
          </Button>
        {/if}
      </div>
    </div>

    {#if previewLoading}
      <div class="p-8 text-center text-sm text-text-secondary">
        {labels.loading}
      </div>
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
            <tr
              class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50"
            >
              <th class="px-4 py-3">{labels.consignmentProduct}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentQty}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentUnitPrice}</th
              >
              <th class="px-4 py-3 text-right">{labels.consignmentSubtotal}</th>
              <th class="px-4 py-3 text-right"
                >{labels.consignmentStoreShare}</th
              >
            </tr>
          </thead>
          <tbody>
            {#each filteredPreviewItems as item, i (i)}
              <tr class="border-b border-border/40">
                <td class="px-4 py-3 font-medium text-text-primary"
                  >{item.product_name}</td
                >
                <td class="px-4 py-3 text-right text-text-primary"
                  >{item.quantity}</td
                >
                <td class="px-4 py-3 text-right text-text-secondary"
                  >{formatCurrency(item.unit_price)}</td
                >
                <td class="px-4 py-3 text-right text-text-primary"
                  >{formatCurrency(item.subtotal)}</td
                >
                <td class="px-4 py-3 text-right text-text-primary"
                  >{formatCurrency(item.store_share)}</td
                >
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <div
        class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50 flex flex-wrap gap-6 text-sm"
      >
        <span class="text-text-secondary"
          >{labels.consignmentTotalSales}:
          <span class="font-semibold text-text-primary"
            >{formatCurrency(preview.total_sale_value)}</span
          ></span
        >
        <span class="text-text-secondary"
          >{labels.consignmentStoreShare}:
          <span class="font-semibold text-text-primary"
            >{formatCurrency(preview.total_store_share)}</span
          ></span
        >
        <span class="text-text-secondary"
          >{labels.consignmentPayableToSupplier}:
          <span class="font-semibold text-primary"
            >{formatCurrency(preview.total_payable)}</span
          ></span
        >
      </div>
    {/if}
  </div>

  <div class="card">
    <div
      class="flex items-center gap-3 px-4 py-3 border-b border-border/50"
    >
      <h2 class="font-semibold text-text-primary whitespace-nowrap">
        {labels.consignmentSettlementHistory}
      </h2>
      {#if !loading && settlements.length > 0}
        <SearchBar
          bind:value={historySearch}
          placeholder="Search product..."
          class="flex-1 max-w-xs"
        />
      {/if}
    </div>
    {#if !loading && settlements.length > 0}
      <div class="flex gap-2 px-4 pt-3">
        <Button
          variant={historyTab === "all" ? "secondary" : "ghost"}
          size="sm"
          onclick={() => (historyTab = "all")}
        >All</Button>
        <Button
          variant={historyTab === "pending" ? "secondary" : "ghost"}
          size="sm"
          onclick={() => (historyTab = "pending")}
        >Pending</Button>
        <Button
          variant={historyTab === "paid" ? "secondary" : "ghost"}
          size="sm"
          onclick={() => (historyTab = "paid")}
        >Paid</Button>
      </div>
    {/if}
    {#if loading}
      <div class="p-8 text-center text-sm text-text-secondary">
        {labels.loading}
      </div>
    {:else if settlements.length === 0}
      <EmptyState
        icon={Banknote}
        title={labels.consignmentNoSettlements}
        subtitle={labels.consignmentNoSettlementsSubtitle}
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-muted/50">
            <tr
              class="text-left text-xs uppercase tracking-wider text-text-secondary"
            >
              <th class="p-4">{labels.consignmentSettlementNo}</th>
              <th class="p-4">{labels.consignmentDate}</th>
              <th class="p-4 text-right">{labels.consignmentTotal}</th>
              <th class="p-4">{labels.consignmentStatus}</th>
              <th class="p-4 text-right">{labels.actions}</th>
            </tr>
          </thead>
          <tbody>
            {#each pagedSettlements as st (st.id || st)}
              <tr
                class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer"
                onclick={() => openDetail(st.id)}
                role="button"
                tabindex="0"
                onkeydown={(e) => {
                  if (e.key === "Enter" || e.key === " ") openDetail(st.id);
                }}
              >
                <td class="p-4 font-medium text-text-primary"
                  >{st.settlement_number}</td
                >
                <td class="p-4 text-text-secondary"
                  >{formatDateTime(st.created_at)}</td
                >
                <td class="p-4 text-right text-text-primary"
                  >{formatCurrency(st.total_payable)}</td
                >
                <td class="p-4">
                  <Badge
                    variant={st.status === SETTLEMENT_PAID
                      ? "success"
                      : "warning"}
                  >
                    {labels[SETTLEMENT_STATUS_LABELS[st.status]] || st.status}
                  </Badge>
                </td>
                <td class="p-4 text-right">
                  {#if canPay && st.status !== SETTLEMENT_PAID}
                    <Button
                      variant="secondary"
                      size="sm"
                      onclick={(e) => {
                        e.stopPropagation();
                        openPayout(st);
                      }}
                    >
                      {labels.consignmentPay}
                    </Button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination
          total={settlements.length}
          limit={pageLimit}
          offset={pageOffset}
          onPageChange={handlePageChange}
        />
      </div>
    {/if}
  </div>
</div>

<Modal
  bind:open={showCreateModal}
  title={labels.consignmentConfirmSettlement}
  size="md"
>
  <p class="text-sm text-text-secondary">
    {t("consignmentSettlementConfirmText", {
      count: preview?.items?.length ?? 0,
      total: formatCurrency(preview?.total_payable ?? 0),
    })}
  </p>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showCreateModal = false)}
        >{labels.cancel}</Button
      >
      <Button onclick={submitCreate} disabled={creating}>
        {creating ? labels.saving : labels.consignmentCreateSettlement}
      </Button>
    </div>
  {/snippet}
</Modal>

<PayoutModal
  bind:show={showPayoutModal}
  settlement={payoutSettlement}
  onclose={() => (showPayoutModal = false)}
  onpaid={async () => {
    showPayoutModal = false;
    await load();
    await loadPreview();
    onsettled?.();
  }}
/>

<Modal
  bind:open={showDetailModal}
  title={detailSettlement?.settlement_number ||
    labels.consignmentSettlementDetail}
  size="xl"
>
  {#if loadingDetail}
    <div class="p-8 text-center text-sm text-text-secondary">
      {labels.loading}
    </div>
  {:else if detailSettlement}
    <div class="space-y-4">
      <div class="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
        <div>
          <div class="text-text-secondary text-xs uppercase tracking-wider">
            {labels.consignmentDate}
          </div>
          <div class="font-medium text-text-primary">
            {formatDateTime(detailSettlement.created_at)}
          </div>
        </div>
        <div>
          <div class="text-text-secondary text-xs uppercase tracking-wider">
            {labels.consignmentStatus}
          </div>
          <Badge
            variant={detailSettlement.status === SETTLEMENT_PAID
              ? "success"
              : "warning"}
          >
            {labels[SETTLEMENT_STATUS_LABELS[detailSettlement.status]] ||
              detailSettlement.status}
          </Badge>
        </div>
        <div>
          <div class="text-text-secondary text-xs uppercase tracking-wider">
            {labels.consignmentTotal}
          </div>
          <div class="font-semibold text-primary">
            {formatCurrency(detailSettlement.total_payable)}
          </div>
        </div>
      </div>

      <div>
        <h3 class="font-semibold text-text-primary mb-2">
          {labels.consignmentItems}
        </h3>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead class="bg-muted/50">
              <tr
                class="text-left text-xs uppercase tracking-wider text-text-secondary"
              >
                <th class="p-4">{labels.consignmentProduct}</th>
                <th class="p-4 text-right">{labels.consignmentQty}</th>
                <th class="p-4 text-right">{labels.consignmentUnitPrice}</th>
                <th class="p-4 text-right">{labels.consignmentSubtotal}</th>
                <th class="p-4 text-right">{labels.consignmentStoreShare}</th>
              </tr>
            </thead>
            <tbody>
              {#each detailSettlement.items as item, i (i)}
                <tr class="border-t border-border/40">
                  <td class="p-4 font-medium text-text-primary">
                    {item.product_name || `Product #${item.product_id}`}
                  </td>
                  <td class="p-4 text-right text-text-primary"
                    >{item.quantity}</td
                  >
                  <td class="p-4 text-right text-text-secondary"
                    >{formatCurrency(item.unit_price)}</td
                  >
                  <td class="p-4 text-right text-text-primary"
                    >{formatCurrency(item.subtotal)}</td
                  >
                  <td class="p-4 text-right text-text-primary"
                    >{formatCurrency(item.store_share)}</td
                  >
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

      <div>
        <h3 class="font-semibold text-text-primary mb-2">
          {labels.consignmentPayouts}
        </h3>
        {#if detailSettlement.payouts.length === 0}
          <EmptyState
            icon={Banknote}
            title={labels.consignmentNoPayouts}
            subtitle={labels.consignmentNoPayoutsSubtitle}
          />
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead class="bg-muted/50">
                <tr
                  class="text-left text-xs uppercase tracking-wider text-text-secondary"
                >
                  <th class="p-4">{labels.consignmentPayoutNo}</th>
                  <th class="p-4">{labels.consignmentPaymentMethod}</th>
                  <th class="p-4 text-right">{labels.consignmentAmount}</th>
                  <th class="p-4">{labels.consignmentReference}</th>
                  <th class="p-4">{labels.consignmentPaidBy}</th>
                  <th class="p-4">{labels.consignmentPaidAt}</th>
                </tr>
              </thead>
              <tbody>
                {#each detailSettlement.payouts as payout (payout.id)}
                  <tr class="border-t border-border/40">
                    <td class="p-4 font-medium text-text-primary"
                      >{payout.payout_number}</td
                    >
                    <td class="p-4 text-text-secondary"
                      >{payout.payment_method_name || "-"}</td
                    >
                    <td class="p-4 text-right text-text-primary"
                      >{formatCurrency(payout.amount)}</td
                    >
                    <td class="p-4 text-text-secondary"
                      >{payout.reference_number || "-"}</td
                    >
                    <td class="p-4 text-text-secondary"
                      >{payout.paid_by_username || "-"}</td
                    >
                    <td class="p-4 text-text-secondary"
                      >{formatDateTime(payout.paid_at)}</td
                    >
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>
  {/if}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showDetailModal = false)}>
        {labels.close}
      </Button>
      {#if detailSettlement && canPay && detailSettlement.status !== SETTLEMENT_PAID}
        <Button
          onclick={() => {
            showDetailModal = false;
            openPayout(detailSettlement!);
          }}
        >
          {labels.consignmentPay}
        </Button>
      {/if}
    </div>
  {/snippet}
</Modal>
