<script lang="ts">
  import { Badge, Button, Drawer } from '$shared/ui';
  import { Printer, Download } from 'lucide-svelte';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte';
  import { downloadInvoice } from '$modules/sales/lib/invoicePdf';
  import { toast } from '$shared/stores/toast.svelte';

  let {
    selectedTransaction = null,
    showTransactionDrawer = $bindable(false),
    onclose = () => {},
    onprint = () => {},
    ondownload = () => {},
  } = $props();

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
    if (!selectedTransaction || !selectedTransaction.items) return;
    const taxAmount = selectedTransaction.tax || 0;
    printReceiptStore.set({
      invoice_number: selectedTransaction.invoice_number,
      created_at: selectedTransaction.created_at,
      items: selectedTransaction.items.map((item: any) => ({
        name: item.name,
        quantity: item.quantity,
        unit_price: item.unit_price,
      })),
      total_amount: selectedTransaction.total_amount,
      subtotal_dpp: selectedTransaction.total_amount - taxAmount,
      tax: taxAmount,
      paymentMethod: selectedTransaction.payment_method || '—',
      cashReceived: selectedTransaction.cash_received || selectedTransaction.total_amount,
      changeDue: selectedTransaction.change_due || 0,
      customer_name: selectedTransaction.customer_name || undefined,
    });
    setTimeout(() => window.print(), 300);
  }

  async function downloadInvoiceHandler() {
    if (!selectedTransaction) return;
    const ok = await downloadInvoice(selectedTransaction, formatDateTime);
    if (ok) {
      toast.success('Invoice downloaded');
    } else {
      toast.error('Failed to download invoice');
    }
  }

  function handleClose() {
    showTransactionDrawer = false;
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
  {#if selectedTransaction}
    <div class="flex items-center gap-3 mb-4">
      <h2 class="text-lg font-bold text-text-primary">Transaction Details</h2>
      <span class="inline-flex items-center px-2.5 py-0.5 text-xs font-medium rounded-full {statusVariant(selectedTransaction.status) === 'success' ? 'bg-success/20 text-success' : statusVariant(selectedTransaction.status) === 'warning' ? 'bg-warning/20 text-warning' : 'bg-info/20 text-info'}">
        {selectedTransaction.status || 'completed'}
      </span>
    </div>

    <div class="grid grid-cols-2 gap-x-8 gap-y-4">
      <div class="space-y-3">
        <div>
          <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Invoice Number</p>
          <p class="text-sm font-semibold text-text-primary font-mono">{selectedTransaction.invoice_number}</p>
        </div>
        <div>
          <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Date & Time</p>
          <p class="text-sm text-text-primary">{formatDateTime(new Date(selectedTransaction.created_at))}</p>
        </div>
        <div>
          <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Customer</p>
          <p class="text-sm text-text-primary">{selectedTransaction.customer_name || 'Walk-in / General'}</p>
        </div>
      </div>
      <div class="space-y-3">
        <div>
          <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Payment Method</p>
          <div class="mt-1">
            <span class="inline-flex items-center px-2.5 py-1 text-xs font-medium rounded-full {getPaymentMethodVariant(selectedTransaction.payment_method) === 'success' ? 'bg-success/20 text-success' : getPaymentMethodVariant(selectedTransaction.payment_method) === 'warning' ? 'bg-warning/20 text-warning' : 'bg-primary/20 text-primary'}">
              {selectedTransaction.payment_method || '—'}
            </span>
          </div>
        </div>
      </div>
    </div>

    {#if selectedTransaction.items && selectedTransaction.items.length > 0}
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
                {#each selectedTransaction.items as item}
                  <tr class="hover:bg-surface/50">
                    <td class="py-3 px-4 text-text-primary">{item.name}</td>
                    <td class="py-3 px-4 text-center text-text-secondary">{item.quantity}</td>
                    <td class="py-3 px-4 text-right text-text-secondary">{(item.unit_price || 0).toLocaleString('id-ID')}</td>
                    <td class="py-3 px-4 text-right font-medium text-text-primary">{(item.unit_price * item.quantity).toLocaleString('id-ID')}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
          <div class="bg-surface-subtle/50 border-t border-border">
            {#if selectedTransaction.tax && selectedTransaction.tax > 0}
              <div class="flex justify-between items-center py-2 px-4 text-sm">
                <span class="text-text-muted">Subtotal (DPP)</span>
                <span class="text-text-secondary">{((selectedTransaction.total_amount || 0) - selectedTransaction.tax).toLocaleString('id-ID')}</span>
              </div>
              <div class="flex justify-between items-center py-2 px-4 text-sm border-t border-border/50">
                <span class="text-text-muted">PPN 11%</span>
                <span class="text-text-secondary">{(selectedTransaction.tax || 0).toLocaleString('id-ID')}</span>
              </div>
            {/if}
            <div class="flex justify-between items-center py-3 px-4 border-t border-border/50">
              <span class="font-bold text-text-primary">TOTAL</span>
              <span class="font-bold text-lg text-text-primary">Rp {(selectedTransaction.total_amount || 0).toLocaleString('id-ID')}</span>
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
