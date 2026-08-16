<script lang="ts">
  import { Drawer, Badge, Button, Skeleton } from '$shared/ui';
  import { getPurchaseOrderById, getReceipts } from '../services/po-service';
  import type { PurchaseOrder, GoodsReceipt } from '../types';
  import { labels, t, formatLocaleDate } from '$shared/i18n';
  import { Package, Printer, Pencil, Check, XCircle, Copy, Truck } from 'lucide-svelte';

  let {
    poId = $bindable(),
    open = $bindable(false),
    reloadKey = 0,
    onedit,
    onconfirm,
    oncancel,
    onreceive,
    canEdit = false,
    canConfirm = false,
    canCancel = false,
    canReceive = false,
  }: {
    poId?: number | null;
    open?: boolean;
    reloadKey?: number;
    onedit?: (po: PurchaseOrder) => void;
    onconfirm?: (po: PurchaseOrder) => void;
    oncancel?: (po: PurchaseOrder) => void;
    onreceive?: (po: PurchaseOrder) => void;
    canEdit?: boolean;
    canConfirm?: boolean;
    canCancel?: boolean;
    canReceive?: boolean;
  } = $props();

  let po = $state<PurchaseOrder | null>(null);
  let receipts = $state<GoodsReceipt[]>([]);
  let loading = $state(false);
  let poCopied = $state(false);

  $effect(() => {
    if (open && poId) {
      loadPO(poId);
    } else if (!open) {
      poCopied = false;
    }
    reloadKey; // tracked dependency
  });

  async function loadPO(id: number) {
    loading = true;
    try {
      po = await getPurchaseOrderById(id);
      receipts = await getReceipts(id);
    } catch {
      po = null;
    } finally {
      loading = false;
    }
  }

  function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'danger' | 'primary' | 'muted' {
    switch (status) {
      case 'draft': return 'muted';
      case 'confirmed': return 'primary';
      case 'partial_received': return 'warning';
      case 'fully_received': return 'success';
      case 'cancelled':
      case 'rejected': return 'danger';
      case 'waiting_approval': return 'default';
      default: return 'muted';
    }
  }

  function titleCase(str: string): string {
    return str.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
  }

  function formatDate(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    return formatLocaleDate(new Date(dateStr), { day: 'numeric', month: 'short', year: 'numeric' });
  }

  function formatCurrency(value: number): string {
    return (value || 0).toLocaleString('id-ID');
  }
</script>

  <Drawer bind:open title={po ? po.po_number : labels.purchaseOrder} width={580}>
  {#if loading}
    <div class="space-y-3 px-1" aria-busy="true" aria-label={labels.loading}>
      <Skeleton class="h-6 w-48" />
      <Skeleton class="h-4 w-72" />
      <div class="space-y-2 mt-6">
        <Skeleton class="h-4 w-full" />
        <Skeleton class="h-4 w-full" />
        <Skeleton class="h-4 w-full" />
        <Skeleton class="h-4 w-2/3" />
      </div>
      <div class="space-y-2 mt-6">
        <Skeleton class="h-10 w-full" />
        <Skeleton class="h-10 w-full" />
        <Skeleton class="h-10 w-full" />
        <Skeleton class="h-10 w-3/4" />
      </div>
    </div>
  {:else if !po}
    <div class="flex flex-col items-center justify-center py-16 text-text-muted">
      <Package class="w-12 h-12 mb-3" />
      <p class="text-sm">{labels.poNotFound}</p>
    </div>
  {:else}
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <span class="flex items-center gap-3">
          <span class="flex items-center gap-2">
            <span class="text-base font-semibold">{po!.po_number}</span>
            <button type="button" class="p-0.5 hover:text-primary transition-colors w-5 h-5 flex items-center justify-center shrink-0" title={labels.copyPoNumber} aria-label={labels.copyPoNumber} onclick={() => { navigator.clipboard.writeText(po!.po_number!); poCopied = true; setTimeout(() => poCopied = false, 1500); }}>
              {#if poCopied}
                <span class="text-xs text-primary font-bold leading-none">✓</span>
              {:else}
                <Copy size={14} class="text-text-muted hover:text-primary" />
              {/if}
            </button>
          </span>
          {#if receipts.length > 0}
            <span class="flex items-center gap-1.5 text-sm text-text-muted border-l border-border pl-3">
              <Truck size={14} />
              <span class="font-medium text-text-primary">DO#</span>
              {#each receipts as r, i}
                <span>{r.delivery_order_number || '-'}{i < receipts.length - 1 ? ',' : ''}</span>
              {/each}
            </span>
          {/if}
        </span>
        <Badge variant={getStatusVariant(po.status)} size="sm">{titleCase(po.status)}</Badge>
      </div>

      <div class="grid grid-cols-2 gap-4 text-sm">
        <div>
          <p class="text-text-muted mb-0.5">{labels.supplier}</p>
          <p class="font-medium text-text-primary">{(po as any).supplier_name || t('supplierWithId', { id: po.supplier_id })}</p>
        </div>
        <div>
          <p class="text-text-muted mb-0.5">{labels.expectedDate}</p>
          <p class="font-medium text-text-primary">{formatDate(po.expected_date)}</p>
        </div>
        <div>
          <p class="text-text-muted mb-0.5">{labels.paymentTerm}</p>
          <p class="font-medium text-text-primary">{po.payment_term || '-'}</p>
        </div>
        <div>
          <p class="text-text-muted mb-0.5">{labels.supplierRef}</p>
          <p class="font-medium text-text-primary">{po.supplier_reference_number || '-'}</p>
        </div>
        <div class="col-span-2">
          <p class="text-text-muted mb-0.5">{labels.deliveryAddress}</p>
          <p class="font-medium text-text-primary">{po.delivery_address || '-'}</p>
        </div>
        <div class="col-span-2">
          <p class="text-text-muted mb-0.5">{labels.notes}</p>
          <p class="font-medium text-text-primary">{po.notes || '-'}</p>
        </div>
      </div>

      <div>
        <h3 class="text-sm font-semibold text-text-primary mb-2">{labels.items}</h3>
        <div class="border border-border rounded-xl overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-muted/50">
              <tr class="border-b text-left text-xs text-text-muted">
                <th class="px-3 py-2 font-semibold">{labels.product}</th>
                <th class="px-3 py-2 font-semibold text-right">{labels.qty}</th>
                <th class="px-3 py-2 font-semibold text-right">{labels.unitCost}</th>
                <th class="px-3 py-2 font-semibold text-right">{labels.discount}</th>
                <th class="px-3 py-2 font-semibold text-right">{labels.subtotal}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each po.items as item}
                <tr>
                  <td class="px-3 py-2.5 text-text-primary">{item.product_name || t('productWithId', { id: item.product_id })}</td>
                  <td class="px-3 py-2.5 text-text-secondary text-right tabular-nums">{item.qty_ordered}</td>
                  <td class="px-3 py-2.5 text-text-secondary text-right tabular-nums">{formatCurrency(item.unit_cost)}</td>
                  <td class="px-3 py-2.5 text-text-secondary text-right tabular-nums">{formatCurrency(item.discount_amount)}</td>
                  <td class="px-3 py-2.5 text-text-secondary text-right tabular-nums font-medium">{formatCurrency(item.subtotal)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

      <div class="border-t border-border pt-4 space-y-1.5 text-sm">
        <div class="flex justify-between text-text-secondary">
          <span>{labels.subtotal}</span>
          <span class="tabular-nums">{formatCurrency(po.subtotal)}</span>
        </div>
        {#if po.discount_amount}
          <div class="flex justify-between text-text-secondary">
            <span>{labels.discount}</span>
            <span class="tabular-nums">-{formatCurrency(po.discount_amount)}</span>
          </div>
        {/if}
        {#if po.tax_amount}
          <div class="flex justify-between text-text-secondary">
            <span>{labels.pajak}</span>
            <span class="tabular-nums">{formatCurrency(po.tax_amount)}</span>
          </div>
        {/if}
        <div class="flex justify-between font-semibold text-text-primary text-base pt-1.5 border-t border-border">
          <span>{labels.grandTotal}</span>
          <span class="tabular-nums">{formatCurrency(po.grand_total)}</span>
        </div>
      </div>

      <div class="border-t border-border pt-4 grid grid-cols-2 gap-4 text-xs text-text-muted">
        <div>
          <p>{labels.createdAt}: {formatDate(po.created_at)}</p>
          {#if po.confirmed_at}<p>{labels.confirmed}: {formatDate(po.confirmed_at)}</p>{/if}
          {#if po.cancelled_at}<p>{labels.statusCancelled}: {formatDate(po.cancelled_at)}</p>{/if}
        </div>
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex items-center justify-between w-full gap-2">
      <div class="flex items-center gap-2">
        {#if po}
          {#if canEdit && po.status === 'draft'}
            <Button variant="secondary" size="sm" onclick={() => onedit?.(po!)}>
              <Pencil size={14} />
              {labels.edit}
            </Button>
          {/if}
          {#if canConfirm && po.status === 'draft'}
            <Button variant="primary" size="sm" onclick={() => onconfirm?.(po!)}>
              <Check size={14} />
              {labels.confirm}
            </Button>
          {/if}
          {#if canReceive && (po.status === 'confirmed' || po.status === 'partial_received')}
            <Button variant="primary" size="sm" onclick={() => onreceive?.(po!)}>
              <Package size={14} />
              {labels.receiveGoods}
            </Button>
          {/if}
          {#if canCancel && (po.status === 'draft' || po.status === 'confirmed')}
            <Button variant="secondary" size="sm" onclick={() => oncancel?.(po!)}>
              <XCircle size={14} />
              {labels.cancelPo}
            </Button>
          {/if}
        {/if}
      </div>
      <Button variant="secondary" size="sm" onclick={() => window.print()} disabled={!po}>
        <Printer size={14} />
        {labels.print}
      </Button>
    </div>
  {/snippet}
</Drawer>
