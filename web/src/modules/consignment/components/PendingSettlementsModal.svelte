<script lang="ts">
  import { Button, Modal, EmptyState } from "$shared/ui";
  import { labels } from "$shared/i18n";
  import { listSettlements } from "../services/consignment-service";
  import type { Settlement } from "../types";
  import { formatCurrency } from "../lib/format";
  import PayoutModal from "./PayoutModal.svelte";

  let {
    show = $bindable(),
  }: {
    show: boolean;
  } = $props();

  let settlements = $state<Settlement[]>([]);
  let loading = $state(false);
  let selectedSettlement = $state<Settlement | null>(null);
  let showPayoutModal = $state(false);

  $effect(() => {
    if (show) {
      load();
    }
  });

  async function load() {
    loading = true;
    try {
      settlements = await listSettlements(undefined, "pending_payment");
    } catch {
      settlements = [];
    } finally {
      loading = false;
    }
  }

  function openPayout(st: Settlement) {
    selectedSettlement = st;
    showPayoutModal = true;
  }

  function formatStatus(status: string) {
    if (status === "pending_payment")
      return labels.settlementStatusPendingPayment;
    if (status === "paid") return labels.settlementStatusPaid;
    return status;
  }
</script>

<Modal bind:open={show} title={labels.consignmentPendingSettlements} size="lg">
  <div class="space-y-3">
    {#if loading}
      <div class="p-8 text-center text-sm text-text-secondary">
        {labels.loading}
      </div>
    {:else if settlements.length === 0}
      <EmptyState
        title={labels.consignmentNoPendingSettlements}
        subtitle={labels.consignmentNoPendingSettlementsSubtitle}
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr
              class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50"
            >
              <th class="px-4 py-3">{labels.consignmentSettlementNo}</th>
              <th class="px-4 py-3 text-right">{labels.consignmentTotal}</th>
              <th class="px-4 py-3">{labels.consignmentStatus}</th>
              <th class="px-4 py-3 text-right">{labels.actions}</th>
            </tr>
          </thead>
          <tbody>
            {#each settlements as st (st.id)}
              <tr class="border-b border-border/40 hover:bg-surface-hover/50">
                <td class="px-4 py-3 font-medium text-text-primary">
                  {st.settlement_number}
                </td>
                <td class="px-4 py-3 text-right text-text-primary">
                  {formatCurrency(st.total_payable)}
                </td>
                <td class="px-4 py-3">
                  <span
                    class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-warning/10 text-warning"
                  >
                    {formatStatus(st.status)}
                  </span>
                </td>
                <td class="px-4 py-3 text-right">
                  <Button size="sm" onclick={() => openPayout(st)}>
                    {labels.consignmentPay}
                  </Button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
  {#snippet footer()}
    <div class="flex justify-end w-full">
      <Button variant="secondary" onclick={() => (show = false)}
        >{labels.close}</Button
      >
    </div>
  {/snippet}
</Modal>

<PayoutModal
  bind:show={showPayoutModal}
  settlement={selectedSettlement}
  onclose={() => (showPayoutModal = false)}
  onpaid={async () => {
    showPayoutModal = false;
    await load();
  }}
/>
