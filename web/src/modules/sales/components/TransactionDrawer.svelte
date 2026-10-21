<script lang="ts">
  import { Button, Drawer, Badge } from "$shared/ui";
  import { Printer, Download } from "lucide-svelte";
  import { formatDateTimeInJakarta } from "$shared/utils/jakartaTime";
  import { printReceiptWithToast } from "$shared/services/print-service";
  import { downloadInvoice } from "$modules/sales/lib/invoicePdf";
  import { toast } from "$shared/stores/toast.svelte";
  import apiClient from "$shared/api/http-client";
  import { labels } from "$shared/i18n";
  import {
    ReturnModal,
    getReturnableSale,
    getReturnById,
    listReturnsBySale,
    buildReturnReceipt,
    ineligibleText,
    returnStatusLabel,
    returnStatusVariant,
  } from "$modules/salereturn";
  import type { ReturnableSale, SaleReturn } from "$modules/salereturn";
  import type { Sale, SaleItem, PaymentInfo } from "../types";

  let {
    selectedTransaction = null,
    showTransactionDrawer = $bindable(false),
    mode = "history",
    canReprint = true,
    canReturn = false,
    onclose = () => {},
    onprint = () => {},
    ondownload = () => {},
    onreturned = () => {},
  } = $props();

  let detailLoading = $state(false);
  let transactionDetail = $state<Sale | null>(null);
  let returnable = $state<ReturnableSale | null>(null);
  let returnLoading = $state(false);
  let returnLoadFailed = $state(false);
  let returnRequestToken = 0;
  let showReturnModal = $state(false);
  let returns = $state<SaleReturn[]>([]);
  let returnsToken = 0;
  let returnPrinting = $state(false);

  const displayTransaction = $derived(transactionDetail || selectedTransaction);

  // In lookup mode the caller only knows the id, so the body must wait for the
  // detail fetch to populate `transactionDetail` (otherwise created_at is undefined
  // and formatting it throws). In history mode the caller passes a full object, so
  // it is ready immediately.
  const ready = $derived(
    mode === "lookup"
      ? transactionDetail !== null
      : displayTransaction !== null,
  );

  $effect(() => {
    if (showTransactionDrawer && selectedTransaction?.id) {
      detailLoading = true;
      transactionDetail = null;
      // In lookup mode (cross-cashier Find Transaction drill-down) the detail is
      // fetched from the redacted /sales/lookup/:id endpoint instead of the
      // owner-scoped /sales/:id endpoint.
      const url =
        mode === "lookup"
          ? `/sales/lookup/${selectedTransaction.id}`
          : `/sales/${selectedTransaction.id}`;
      apiClient
        .get(url)
        .then((r) => {
          transactionDetail = r.data?.data || r.data;
        })
        .catch(() => {
          transactionDetail = null;
        })
        .finally(() => {
          detailLoading = false;
        });
    }
  });

  // The data behind the disabled state of the "Return Items" button
  // (sale.return holders only). Fetched fresh on every open: the window, the
  // open shift and the already-returned quantities all move with time and
  // with other transactions.
  $effect(() => {
    if (showTransactionDrawer && canReturn && selectedTransaction?.id) {
      loadReturnable(selectedTransaction.id);
    }
  });

  // Reprint support (Phase 4.4): the returns already recorded against this
  // sale, so a completed one can be reprinted from here. The list rows are
  // headers only (the repository does not hydrate items), so printing fetches
  // the full detail first. Lookup mode is excluded on purpose: that endpoint
  // is redacted for cross-cashier views.
  $effect(() => {
    if (showTransactionDrawer && mode !== "lookup" && selectedTransaction?.id) {
      loadReturns(selectedTransaction.id);
    }
  });

  async function loadReturns(id: number) {
    // Same stale-response guard as loadReturnable: a re-open can leave an
    // older reply in flight. The rows are cleared up front too: a drawer
    // close via Escape or the backdrop never runs handleClose, so without
    // this the previous sale's returns (including a reprint button that
    // would build a receipt against the NEW sale's invoice number) flash
    // under the new sale until the list reply lands.
    const token = ++returnsToken;
    returns = [];
    try {
      const list = await listReturnsBySale(id);
      if (token === returnsToken) returns = list;
    } catch {
      // The section is a convenience; a failed list just means no reprint
      // shortcuts, and the modal's own print still covers the fresh return.
      if (token === returnsToken) returns = [];
    }
  }

  async function printReturnReceipt(r: SaleReturn) {
    // printReceiptWithToast surfaces its own failure toast and never throws,
    // so the detail fetch is the only rejection source — without this guard a
    // failed getReturnById would surface as an unhandled promise rejection
    // from the onclick handler with no user feedback. The in-flight flag also
    // stops a double click from firing two fetch/print cycles.
    if (returnPrinting) return;
    returnPrinting = true;
    try {
      const full = await getReturnById(r.id);
      if (!full || !displayTransaction) return;
      await printReceiptWithToast(
        buildReturnReceipt(full, displayTransaction.invoice_number),
      );
    } catch {
      toast.error(labels.returnLoadFailed);
    } finally {
      returnPrinting = false;
    }
  }

  async function loadReturnable(id: number) {
    // A drawer re-open can leave an older response in flight; ignoring any
    // reply that is not from the latest request keeps sale A's snapshot from
    // overwriting sale B's.
    const token = ++returnRequestToken;
    returnLoading = true;
    returnable = null;
    returnLoadFailed = false;
    try {
      const data = await getReturnableSale(id);
      if (token !== returnRequestToken) return;
      returnable = data;
    } catch {
      if (token !== returnRequestToken) return;
      // Distinguish a failed fetch from an ineligible sale: both disable the
      // button, but only one of them is explained by the reason text.
      returnable = null;
      returnLoadFailed = true;
    } finally {
      if (token === returnRequestToken) returnLoading = false;
    }
  }

  function statusVariant(s: string) {
    return s === "completed"
      ? "success"
      : s === "refunded"
        ? "danger"
        : "warning";
  }

  function getPaymentMethodVariant(method = "") {
    if (!method) return "muted";
    const m = method.toLowerCase();
    if (m === "cash") return "success";
    if (m === "qris" || m === "e_wallet") return "default";
    if (m === "card") return "primary";
    if (m === "transfer") return "muted";
    return "muted";
  }

  const formatDateTime = (date: Date | string | number | undefined | null) => {
    const d = date instanceof Date ? date : new Date(date as string);
    if (isNaN(d.getTime())) return "—";
    return formatDateTimeInJakarta(d.toISOString());
  };

  async function printTransactionReceipt() {
    if (!displayTransaction || !displayTransaction.items) return;
    const taxAmount = displayTransaction.tax || 0;
    const paymentLines =
      displayTransaction.payments && displayTransaction.payments.length > 0
        ? displayTransaction.payments
            .map(
              (p: PaymentInfo) =>
                `${p.payment_method_code}: ${labels.currencySymbol} ${(p.amount || 0).toLocaleString("id-ID")}`,
            )
            .join(", ")
        : displayTransaction.payment_method || "—";
    const cashReceived =
      displayTransaction.payments?.find(
        (p: PaymentInfo) => p.payment_method_code === "CASH",
      )?.amount || displayTransaction.total_amount;
    const payload = {
      invoice_number: displayTransaction.invoice_number,
      created_at: displayTransaction.created_at,
      items: displayTransaction.items.map((item: SaleItem) => ({
        name: item.name,
        quantity: item.quantity,
        unit_price: item.unit_price,
        original_price: item.original_price,
        pricing_rule_name: item.pricing_rule_name,
        pricing_type: item.pricing_type,
      })),
      total_amount: displayTransaction.total_amount,
      subtotal_dpp: displayTransaction.total_amount - taxAmount,
      tax: taxAmount,
      paymentMethod: paymentLines,
      payments: displayTransaction.payments?.map((p: PaymentInfo) => ({
        method: p.payment_method_code,
        amount: p.amount,
        reference_number: p.reference_number,
      })),
      cashReceived,
      changeDue: displayTransaction.change_due ?? 0,
      customer_name: displayTransaction.customer_name || undefined,
      total_savings: (displayTransaction.items || []).reduce(
        (sum: number, item: SaleItem) => {
          if (item.original_price && item.original_price > item.unit_price) {
            return (
              sum + (item.original_price - item.unit_price) * item.quantity
            );
          }
          return sum;
        },
        0,
      ),
    };
    await printReceiptWithToast(payload);
  }

  async function downloadInvoiceHandler() {
    if (!displayTransaction) return;
    const ok = await downloadInvoice(displayTransaction, formatDateTime);
    if (ok) {
      toast.success(labels.toastInvoiceDownloaded);
    } else {
      toast.error(labels.toastFailedToDownloadInvoice);
    }
  }

  function handleClose() {
    showTransactionDrawer = false;
    transactionDetail = null;
    returns = [];
    onclose();
  }

  function handlePrint() {
    printTransactionReceipt();
    onprint();
  }

  function handleDownload() {
    downloadInvoiceHandler();
    ondownload();
  }
</script>

<Drawer
  bind:open={showTransactionDrawer}
  width={520}
  ariaLabel={labels.transactionDetails}
  onclose={() => onclose()}
>
  {#if ready}
    <div class="flex items-center gap-3 mb-4">
      <h2 class="text-lg font-bold text-text-primary">
        {labels.transactionDetails}
      </h2>
      <span
        class="inline-flex items-center px-2.5 py-0.5 text-xs font-medium rounded-full {statusVariant(
          displayTransaction.status,
        ) === 'success'
          ? 'bg-success/20 text-success'
          : statusVariant(displayTransaction.status) === 'warning'
            ? 'bg-warning/20 text-warning'
            : 'bg-info/20 text-info'}"
      >
        {displayTransaction.status || "completed"}
      </span>
      {#if detailLoading}
        <span class="text-xs text-text-muted">{labels.loading}</span>
      {/if}
    </div>

    <div class="grid grid-cols-2 gap-x-8 gap-y-4">
      <div class="space-y-3">
        <div>
          <p
            class="text-xs font-medium text-text-muted uppercase tracking-wide"
          >
            {labels.invoiceNumber}
          </p>
          <p class="text-sm font-semibold text-text-primary font-mono">
            {displayTransaction.invoice_number}
          </p>
        </div>
        <div>
          <p
            class="text-xs font-medium text-text-muted uppercase tracking-wide"
          >
            {labels.dateAndTime}
          </p>
          <p class="text-sm text-text-primary">
            {formatDateTime(new Date(displayTransaction.created_at))}
          </p>
        </div>
        {#if mode !== "lookup"}
          <div>
            <p
              class="text-xs font-medium text-text-muted uppercase tracking-wide"
            >
              {labels.customer}
            </p>
            <p class="text-sm text-text-primary">
              {displayTransaction.customer_name || labels.walkInGeneral}
            </p>
          </div>
        {/if}
      </div>
      <div class="space-y-3">
        <div>
          <p
            class="text-xs font-medium text-text-muted uppercase tracking-wide"
          >
            {labels.paymentMethod}
          </p>
          <div class="mt-1 space-y-1">
            {#if displayTransaction.payments && displayTransaction.payments.length > 0}
              {#each displayTransaction.payments as payment, i (i)}
                <div class="flex items-center justify-between gap-2 text-sm">
                  <span
                    class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full {getPaymentMethodVariant(
                      payment.payment_method_code,
                    ) === 'success'
                      ? 'bg-success/20 text-success'
                      : getPaymentMethodVariant(payment.payment_method_code) ===
                          'muted'
                        ? 'bg-muted/20 text-muted'
                        : 'bg-primary/20 text-primary'}"
                  >
                    {payment.payment_method_code || "—"}
                  </span>
                  <span class="font-medium text-text-primary"
                    >{(payment.amount || 0).toLocaleString("id-ID")}</span
                  >
                </div>
                {#if payment.reference_number}
                  <p class="text-[10px] text-text-muted ml-1">
                    {labels.refLabel}{payment.reference_number}
                  </p>
                {/if}
              {/each}
            {:else}
              <span
                class="inline-flex items-center px-2.5 py-1 text-xs font-medium rounded-full {getPaymentMethodVariant(
                  displayTransaction.payment_method,
                ) === 'success'
                  ? 'bg-success/20 text-success'
                  : getPaymentMethodVariant(
                        displayTransaction.payment_method,
                      ) === 'muted'
                    ? 'bg-muted/20 text-muted'
                    : 'bg-primary/20 text-primary'}"
              >
                {displayTransaction.payment_method || "—"}
              </span>
            {/if}
          </div>
        </div>
      </div>
    </div>

    {#if displayTransaction.items && displayTransaction.items.length > 0}
      <div class="mt-5">
        <p class="text-sm font-semibold text-text-secondary mb-3">
          {labels.items}
        </p>
        <div class="border border-border rounded-lg">
          <div class="max-h-80 overflow-y-auto">
            <table class="w-full text-sm">
              <thead class="sticky top-0 bg-surface-subtle z-10">
                <tr>
                  <th
                    class="text-left py-3 px-4 font-semibold text-text-primary"
                    >{labels.description}</th
                  >
                  <th
                    class="text-right py-3 px-4 font-semibold text-text-primary w-20"
                    >{labels.qty}</th
                  >
                  <th
                    class="text-right py-3 px-4 font-semibold text-text-primary w-28"
                    >{labels.price}</th
                  >
                  <th
                    class="text-right py-3 px-4 font-semibold text-text-primary w-32"
                    >{labels.subTotal}</th
                  >
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                {#each displayTransaction.items as item, i (i)}
                  <tr class="hover:bg-surface/50">
                    <td class="py-3 px-4 text-text-primary">
                      <div>{item.name}</div>
                      {#if item.pricing_rule_name}
                        <div
                          class="text-[10px] text-primary-light mt-0.5 font-medium"
                        >
                          {item.pricing_rule_name}
                        </div>
                      {/if}
                    </td>
                    <td class="py-3 px-4 text-right text-text-secondary"
                      >{item.quantity}</td
                    >
                    <td class="py-3 px-4 text-right text-text-secondary">
                      {#if item.original_price && item.original_price > item.unit_price}
                        <span
                          class="line-through text-text-muted text-[10px] block"
                          >{item.original_price.toLocaleString("id-ID")}</span
                        >
                      {/if}
                      <span
                        >{(item.unit_price || 0).toLocaleString("id-ID")}</span
                      >
                    </td>
                    <td
                      class="py-3 px-4 text-right font-medium text-text-primary"
                      >{(item.unit_price * item.quantity).toLocaleString(
                        "id-ID",
                      )}</td
                    >
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
          <div class="bg-surface-subtle/50 border-t border-border">
            {#if displayTransaction.items?.some((item: SaleItem) => item.original_price && item.original_price > item.unit_price)}
              {@const totalSavings = displayTransaction.items.reduce(
                (sum: number, item: SaleItem) => {
                  if (
                    item.original_price &&
                    item.original_price > item.unit_price
                  ) {
                    return (
                      sum +
                      (item.original_price - item.unit_price) * item.quantity
                    );
                  }
                  return sum;
                },
                0,
              )}
              {#if totalSavings > 0}
                <div
                  class="flex justify-between items-center py-2 px-4 text-sm"
                >
                  <span class="text-green-600 dark:text-green-400"
                    >{labels.hemat}</span
                  >
                  <span class="text-green-600 dark:text-green-400 font-medium"
                    >-{totalSavings.toLocaleString("id-ID")}</span
                  >
                </div>
              {/if}
            {/if}
            {#if displayTransaction.tax && displayTransaction.tax > 0}
              <div class="flex justify-between items-center py-2 px-4 text-sm">
                <span class="text-text-muted"
                  >{labels.subTotal} ({labels.dpp})</span
                >
                <span class="text-text-secondary"
                  >{(
                    (displayTransaction.total_amount || 0) -
                    displayTransaction.tax
                  ).toLocaleString("id-ID")}</span
                >
              </div>
              <div
                class="flex justify-between items-center py-2 px-4 text-sm border-t border-border/50"
              >
                <span class="text-text-muted">{labels.ppn}</span>
                <span class="text-text-secondary"
                  >{(displayTransaction.tax || 0).toLocaleString("id-ID")}</span
                >
              </div>
            {/if}
            <div
              class="flex justify-between items-center py-3 px-4 border-t border-border/50"
            >
              <span class="font-bold text-text-primary"
                >{labels.totalLabel}</span
              >
              <span class="font-bold text-lg text-text-primary"
                >{labels.currencySymbol}
                {(displayTransaction.total_amount || 0).toLocaleString(
                  "id-ID",
                )}</span
              >
            </div>
          </div>
        </div>
      </div>

      {#if canReturn}
        <div class="mt-5">
          <Button
            variant="secondary"
            class="w-full rounded-xl h-11 text-sm font-semibold"
            disabled={returnLoading || !returnable?.eligible}
            title={returnable && !returnable.eligible
              ? ineligibleText(
                  returnable.ineligible_reason,
                  returnable.window_days,
                )
              : ""}
            onclick={() => (showReturnModal = true)}
          >
            {labels.returnItems}
          </Button>
          {#if !returnLoading && returnLoadFailed}
            <p class="mt-1.5 text-xs text-warning-light">
              {labels.returnLoadFailed}
            </p>
          {:else if !returnLoading && returnable && !returnable.eligible}
            <p class="mt-1.5 text-xs text-text-muted">
              {ineligibleText(
                returnable.ineligible_reason,
                returnable.window_days,
              )}
            </p>
          {/if}
        </div>
      {/if}
    {/if}

    {#if mode !== "lookup" && returns.length > 0}
      <div class="mt-5">
        <p class="text-sm font-semibold text-text-secondary mb-3">
          {labels.returnsOfSale}
        </p>
        <div class="border border-border rounded-lg divide-y divide-border">
          {#each returns as r (r.id)}
            <div
              class="flex items-center justify-between gap-2 px-4 py-3 text-sm"
            >
              <div class="min-w-0">
                <p class="font-mono text-text-primary">{r.return_number}</p>
                <p class="text-xs text-text-muted">
                  {formatDateTime(r.created_at)} ·
                  {r.return_type === "replacement"
                    ? labels.returnReplacement
                    : labels.returnRefund}
                </p>
              </div>
              <div class="flex items-center gap-2">
                <Badge variant={returnStatusVariant(r.status)}
                  >{returnStatusLabel(r.status)}</Badge
                >
                {#if r.status === "completed"}
                  <Button
                    variant="secondary"
                    class="rounded-xl h-9 px-3 text-sm font-semibold flex items-center gap-1.5 whitespace-nowrap"
                    aria-label={labels.returnPrintReceipt}
                    title={labels.returnPrintReceipt}
                    disabled={returnPrinting}
                    onclick={() => printReturnReceipt(r)}
                  >
                    <Printer size={15} />
                    {labels.print}
                  </Button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}

  {#snippet footer()}
    {#if mode === "lookup"}
      <div class="grid grid-cols-[auto_1fr] gap-3">
        <Button
          variant="secondary"
          class="rounded-xl px-4 h-11 text-sm font-semibold whitespace-nowrap"
          onclick={handleClose}
        >
          {labels.close}
        </Button>
        {#if canReprint}
          <Button
            variant="primary"
            class="rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm flex items-center gap-1.5 whitespace-nowrap"
            onclick={handlePrint}
          >
            <Printer size={15} class="mr-1.5" />
            {labels.printReceipt}
          </Button>
        {/if}
      </div>
    {:else}
      <div class="grid grid-cols-[auto_1fr_1fr] gap-3">
        <Button
          variant="secondary"
          class="rounded-xl px-4 h-11 text-sm font-semibold whitespace-nowrap"
          onclick={handleClose}
        >
          {labels.close}
        </Button>
        <Button
          variant="secondary"
          class="rounded-xl px-4 h-11 text-sm font-semibold flex items-center gap-1.5 whitespace-nowrap"
          onclick={handlePrint}
        >
          <Printer size={15} class="mr-1.5" />
          {labels.printReceipt}
        </Button>
        <Button
          variant="primary"
          class="rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm flex items-center gap-1.5 whitespace-nowrap"
          onclick={handleDownload}
        >
          <Download size={15} class="mr-1.5" />
          {labels.downloadInvoice}
        </Button>
      </div>
    {/if}
  {/snippet}
</Drawer>

<!-- Rendered as a SIBLING of the drawer, not inside it: both components bind
     Escape at different levels (the modal on window, the drawer on its
     panel), and nesting the modal in the panel would close the drawer too. -->
<ReturnModal
  saleId={selectedTransaction?.id ?? null}
  bind:show={showReturnModal}
  onreturned={() => {
    if (selectedTransaction?.id) loadReturns(selectedTransaction.id);
    onreturned();
  }}
/>
