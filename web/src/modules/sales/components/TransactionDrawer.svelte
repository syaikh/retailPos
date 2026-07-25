<script lang="ts">
  import { Badge, Button, Drawer } from '$shared/ui';
  import { Printer, Download } from 'lucide-svelte';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte';
  import { downloadInvoice } from '$modules/sales/lib/invoicePdf';
  import { toast } from '$shared/stores/toast.svelte';
  import apiClient from '$shared/api/http-client';

  let {
    selectedTransaction = null,
    showTransactionDrawer = $bindable(false),
    onclose = () => {},
    onprint = () => {},
    ondownload = () => {},
  } = $props();

  let detailLoading = $state(false);
  let transactionDetail = $state<any>(null);

  const displayTransaction = $derived(transactionDetail || selectedTransaction);

  $effect(() => {
    if (showTransactionDrawer && selectedTransaction?.id) {
      detailLoading = true;
      transactionDetail = null;
      apiClient.get(`/sales/${selectedTransaction.id}`)
        .then(r => {
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

  function statusVariant(s: string) {
    return s === 'completed' ? 'success' : s === 'refunded' ? 'danger' : 'warning';
  }

  function getPaymentMethodVariant(method = '') {
    if (!method) return 'muted';
    const m = method.toLowerCase();
    if (m === 'cash') return 'success';
    if (m === 'qris' || m === 'e_wallet') return 'default';
    if (m === 'card') return 'primary';
    if (m === 'transfer') return 'muted';
    return 'muted';
  }

  const formatDateTime = (date: Date) => {
    const isoStr = date instanceof Date ? date.toISOString() : String(date);
    return formatDateTimeInJakarta(isoStr);
  };

  function printTransactionReceipt() {
    if (!displayTransaction || !displayTransaction.items) return;
    const taxAmount = displayTransaction.tax || 0;
    const paymentLines = displayTransaction.payments && displayTransaction.payments.length > 0
      ? displayTransaction.payments.map((p: any) => `${p.payment_method_code}: Rp ${(p.amount || 0).toLocaleString('id-ID')}`).join(', ')
      : (displayTransaction.payment_method || '—');
    const cashReceived = displayTransaction.payments?.find((p: any) => p.payment_method_code === 'CASH')?.amount || displayTransaction.total_amount;
    printReceiptStore.set({
      invoice_number: displayTransaction.invoice_number,
      created_at: displayTransaction.created_at,
      items: displayTransaction.items.map((item: any) => ({
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
      payments: displayTransaction.payments?.map((p: any) => ({
        method: p.payment_method_code,
        amount: p.amount,
        reference_number: p.reference_number,
      })),
      cashReceived,
      changeDue: 0,
      customer_name: displayTransaction.customer_name || undefined,
      total_savings: (displayTransaction.items || []).reduce((sum: number, item: any) => {
        if (item.original_price && item.original_price > item.unit_price) {
          return sum + (item.original_price - item.unit_price) * item.quantity;
        }
        return sum;
      }, 0),
    });
    setTimeout(() => window.print(), 300);
  }

  async function downloadInvoiceHandler() {
    if (!displayTransaction) return;
    const ok = await downloadInvoice(displayTransaction, formatDateTime);
    if (ok) {
      toast.success('Invoice downloaded');
    } else {
      toast.error('Failed to download invoice');
    }
  }

  function handleClose() {
    showTransactionDrawer = false;
    transactionDetail = null;
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

<Drawer bind:open={showTransactionDrawer} width={520} ariaLabel="Transaction details" onclose={() => onclose()}>
  {#if displayTransaction}
    <div class="flex items-center gap-3 mb-4">
      <h2 class="text-lg font-bold text-text-primary">Transaction Details</h2>
      <span class="inline-flex items-center px-2.5 py-0.5 text-xs font-medium rounded-full {statusVariant(displayTransaction.status) === 'success' ? 'bg-success/20 text-success' : statusVariant(displayTransaction.status) === 'warning' ? 'bg-warning/20 text-warning' : 'bg-info/20 text-info'}">
        {displayTransaction.status || 'completed'}
      </span>
      {#if detailLoading}
        <span class="text-xs text-text-muted">Loading...</span>
      {/if}
    </div>

    <div class="grid grid-cols-2 gap-x-8 gap-y-4">
      <div class="space-y-3">
        <div>
          <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Invoice Number</p>
          <p class="text-sm font-semibold text-text-primary font-mono">{displayTransaction.invoice_number}</p>
        </div>
        <div>
          <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Date & Time</p>
          <p class="text-sm text-text-primary">{formatDateTime(new Date(displayTransaction.created_at))}</p>
        </div>
        <div>
          <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Customer</p>
          <p class="text-sm text-text-primary">{displayTransaction.customer_name || 'Walk-in / General'}</p>
        </div>
      </div>
      <div class="space-y-3">
        <div>
          <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Payment Method</p>
          <div class="mt-1 space-y-1">
            {#if displayTransaction.payments && displayTransaction.payments.length > 0}
              {#each displayTransaction.payments as payment}
                <div class="flex items-center justify-between gap-2 text-sm">
                  <span class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full {getPaymentMethodVariant(payment.payment_method_code) === 'success' ? 'bg-success/20 text-success' : getPaymentMethodVariant(payment.payment_method_code) === 'muted' ? 'bg-muted/20 text-muted' : 'bg-primary/20 text-primary'}">
                    {payment.payment_method_code || '—'}
                  </span>
                  <span class="font-medium text-text-primary">{(payment.amount || 0).toLocaleString('id-ID')}</span>
                </div>
                {#if payment.reference_number}
                  <p class="text-[10px] text-text-muted ml-1">Ref: {payment.reference_number}</p>
                {/if}
              {/each}
            {:else}
              <span class="inline-flex items-center px-2.5 py-1 text-xs font-medium rounded-full {getPaymentMethodVariant(displayTransaction.payment_method) === 'success' ? 'bg-success/20 text-success' : getPaymentMethodVariant(displayTransaction.payment_method) === 'muted' ? 'bg-muted/20 text-muted' : 'bg-primary/20 text-primary'}">
                {displayTransaction.payment_method || '—'}
              </span>
            {/if}
          </div>
        </div>
      </div>
    </div>

    {#if displayTransaction.items && displayTransaction.items.length > 0}
      <div>
        <p class="text-sm font-semibold text-text-secondary mb-3">Items</p>
        <div class="border border-border rounded-lg">
          <div class="max-h-80 overflow-y-auto">
            <table class="w-full text-sm">
              <thead class="sticky top-0 bg-surface-subtle z-10">
                <tr>
                  <th class="text-left py-3 px-4 font-semibold text-text-primary">Description</th>
                  <th class="text-center py-3 px-4 font-semibold text-text-primary w-20">Qty</th>
                  <th class="text-right py-3 px-4 font-semibold text-text-primary w-28">Price</th>
                  <th class="text-right py-3 px-4 font-semibold text-text-primary w-32">Subtotal</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                {#each displayTransaction.items as item}
                  <tr class="hover:bg-surface/50">
                    <td class="py-3 px-4 text-text-primary">
                      <div>{item.name}</div>
                      {#if item.pricing_rule_name}
                        <div class="text-[10px] text-primary-light mt-0.5 font-medium">{item.pricing_rule_name}</div>
                      {/if}
                    </td>
                    <td class="py-3 px-4 text-center text-text-secondary">{item.quantity}</td>
                    <td class="py-3 px-4 text-right text-text-secondary">
                      {#if item.original_price && item.original_price > item.unit_price}
                        <span class="line-through text-text-muted text-[10px] block">{item.original_price.toLocaleString('id-ID')}</span>
                      {/if}
                      <span>{(item.unit_price || 0).toLocaleString('id-ID')}</span>
                    </td>
                    <td class="py-3 px-4 text-right font-medium text-text-primary">{(item.unit_price * item.quantity).toLocaleString('id-ID')}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
          <div class="bg-surface-subtle/50 border-t border-border">
            {#if displayTransaction.items?.some((item: any) => item.original_price && item.original_price > item.unit_price)}
              {@const totalSavings = displayTransaction.items.reduce((sum: number, item: any) => {
                if (item.original_price && item.original_price > item.unit_price) {
                  return sum + (item.original_price - item.unit_price) * item.quantity;
                }
                return sum;
              }, 0)}
              {#if totalSavings > 0}
                <div class="flex justify-between items-center py-2 px-4 text-sm">
                  <span class="text-green-600 dark:text-green-400">Hemat</span>
                  <span class="text-green-600 dark:text-green-400 font-medium">-{totalSavings.toLocaleString('id-ID')}</span>
                </div>
              {/if}
            {/if}
            {#if displayTransaction.tax && displayTransaction.tax > 0}
              <div class="flex justify-between items-center py-2 px-4 text-sm">
                <span class="text-text-muted">Subtotal (DPP)</span>
                <span class="text-text-secondary">{((displayTransaction.total_amount || 0) - displayTransaction.tax).toLocaleString('id-ID')}</span>
              </div>
              <div class="flex justify-between items-center py-2 px-4 text-sm border-t border-border/50">
                <span class="text-text-muted">PPN 11%</span>
                <span class="text-text-secondary">{(displayTransaction.tax || 0).toLocaleString('id-ID')}</span>
              </div>
            {/if}
            <div class="flex justify-between items-center py-3 px-4 border-t border-border/50">
              <span class="font-bold text-text-primary">TOTAL</span>
              <span class="font-bold text-lg text-text-primary">Rp {(displayTransaction.total_amount || 0).toLocaleString('id-ID')}</span>
            </div>
          </div>
        </div>
      </div>
    {/if}
  {/if}

  {#snippet footer()}
    <div class="grid grid-cols-[auto_1fr_1fr] gap-3">
      <Button variant="secondary" class="rounded-xl px-4 h-11 text-sm font-semibold whitespace-nowrap" onclick={handleClose}>
        Close
      </Button>
      <Button variant="secondary" class="rounded-xl px-4 h-11 text-sm font-semibold flex items-center gap-1.5 whitespace-nowrap" onclick={handlePrint}>
        <Printer size={15} class="mr-1.5" />
        Print Receipt
      </Button>
      <Button variant="primary" class="rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm flex items-center gap-1.5 whitespace-nowrap" onclick={handleDownload}>
        <Download size={15} class="mr-1.5" />
        Download Invoice
      </Button>
    </div>
  {/snippet}
</Drawer>
