<script lang="ts">
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import { Button, Modal, EmptyState } from "$shared/ui";
  import { t } from "$shared/i18n";
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
    if (status === "pending_payment") return "Pending Payment";
    if (status === "paid") return "Paid";
    return status;
  }
</script>

<Modal bind:open={show} title="Pending Settlements" size="lg">
  <div class="space-y-3">
    {#if loading}
      <div class="p-8 text-center text-sm text-text-secondary">Loading...</div>
    {:else if settlements.length === 0}
      <EmptyState
        title="No pending settlements"
        subtitle="All settlements have been paid."
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr
              class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50"
            >
              <th class="px-4 py-3">Settlement No</th>
              <th class="px-4 py-3 text-right">Amount</th>
              <th class="px-4 py-3">Status</th>
              <th class="px-4 py-3 text-right">Action</th>
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
                  <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-warning/10 text-warning">
                    {formatStatus(st.status)}
                  </span>
                </td>
                <td class="px-4 py-3 text-right">
                  <Button size="sm" onclick={() => openPayout(st)}>
                    Pay
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
      <Button variant="secondary" onclick={() => (show = false)}>Close</Button>
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
