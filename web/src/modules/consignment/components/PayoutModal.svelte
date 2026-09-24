<script lang="ts">
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import { Button, Modal, Input, SelectSearch } from "$shared/ui";
  import { t } from "$shared/i18n";
  import { createPayout, listPaymentMethods } from "../services/consignment-service";
  import type { Settlement } from "../types";
  import { formatCurrency } from "../lib/format";
  import FormattedNumberInput from "./FormattedNumberInput.svelte";

  let {
    settlement,
    show = $bindable(),
    onclose,
    onpaid,
  }: {
    settlement: Settlement | null;
    show: boolean;
    onclose: () => void;
    onpaid: () => void;
  } = $props();

  let paying = $state(false);
  let paymentMethods = $state<{ value: number; label: string }[]>([]);
  let form = $state({
    payment_method_id: undefined as number | undefined,
    amount: 0,
    reference_number: "",
    notes: "",
  });

  $effect(() => {
    if (show && settlement) {
      form = {
        payment_method_id: undefined,
        amount: settlement.total_payable,
        reference_number: "",
        notes: "",
      };
      loadPaymentMethods();
    }
  });

  async function loadPaymentMethods() {
    try {
      const methods = await listPaymentMethods();
      paymentMethods = methods.map((m) => ({
        value: m.id,
        label: m.name || m.code,
      }));
    } catch {
      paymentMethods = [];
    }
  }

  async function submit() {
    if (!settlement) return;
    if (!form.payment_method_id) {
      toast.error("Please select a payment method");
      return;
    }
    if (form.amount <= 0) {
      toast.error("Amount must be greater than zero");
      return;
    }
    paying = true;
    try {
      const payout = await createPayout(settlement.id, {
        payment_method_id: form.payment_method_id,
        amount: form.amount,
        reference_number: form.reference_number || undefined,
        notes: form.notes || undefined,
      });
      toast.success(
        t("consignmentPayoutRecorded", { number: payout.payout_number }),
      );
      show = false;
      onpaid();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, "Failed to record payout"));
    } finally {
      paying = false;
    }
  }
</script>

<Modal bind:open={show} title="Record Payment" size="md">
  <div class="space-y-4">
    <div
      class="rounded-xl bg-surface-subtle/60 border border-border-default px-4 py-3 text-sm flex justify-between"
    >
      <span class="text-text-secondary">Outstanding</span>
      <span class="font-semibold text-text-primary"
        >{formatCurrency(settlement?.total_payable)}</span
      >
    </div>
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span>Payment Method <span class="text-danger">*</span></span>
      <SelectSearch
        bind:value={form.payment_method_id}
        options={paymentMethods}
        placeholder="Select payment method"
        searchPlaceholder="Search method..."
        notFoundText="No methods found"
      />
    </label>
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span>Amount <span class="text-danger">*</span></span>
      <FormattedNumberInput bind:value={form.amount} class="h-9 text-sm" />
    </label>
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span>Reference</span>
      <Input
        type="text"
        bind:value={form.reference_number}
        placeholder="Optional reference number"
        class="h-9 text-sm"
      />
    </label>
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span>Notes</span>
      <Input
        tag="textarea"
        bind:value={form.notes}
        rows={2}
        placeholder="Optional notes"
        class="text-sm"
      />
    </label>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (show = false)}>Cancel</Button>
      <Button onclick={submit} disabled={paying}>
        {paying ? "Saving..." : "Pay"}
      </Button>
    </div>
  {/snippet}
</Modal>
