<script lang="ts">
  import { Drawer, Badge, Button } from '$shared/ui';
  import { getPurchaseOrderById } from '../services/po-service';
  import type { PurchaseOrder } from '../types';
  import { Loader2, Package, Printer } from 'lucide-svelte';

  let {
    poId = $bindable(),
    open = $bindable(false),
  }: {
    poId?: number | null;
    open?: boolean;
  } = $props();

  let po = $state<PurchaseOrder | null>(null);
  let loading = $state(false);

  $effect(() => {
    if (open && poId) {
      loadPO(poId);
    } else if (!open) {
      po = null;
    }
  });

  async function loadPO(id: number) {
    loading = true;
    try {
      po = await getPurchaseOrderById(id);
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
    return new Date(dateStr).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
  }

  function formatCurrency(value: number): string {
    return (value || 0).toLocaleString('id-ID');
  }
</script>

  <Drawer bind:open title={po ? po.po_number : 'Purchase Order'} width={580}>
  {#if loading}
    <div class="flex items-center justify-center py-16">
      <Loader2 size={28} class="animate-spin text-text-muted" />
    </div>
  {:else if !po}
    <div class="flex flex-col items-center justify-center py-16 text-text-muted">
      <Package class="w-12 h-12 mb-3" />
      <p class="text-sm">Purchase order not found</p>
    </div>
  {:else}
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <Badge variant={getStatusVariant(po.status)} size="sm">{titleCase(po.status)}</Badge>
      </div>

      <div class="grid grid-cols-2 gap-4 text-sm">
        <div>
          <p class="text-text-muted mb-0.5">Supplier</p>
          <p class="font-medium text-text-primary">{(po as any).supplier_name || `Supplier #${po.supplier_id}`}</p>
        </div>
        <div>
          <p class="text-text-muted mb-0.5">Expected Date</p>
          <p class="font-medium text-text-primary">{formatDate(po.expected_date)}</p>
        </div>
        <div>
          <p class="text-text-muted mb-0.5">Payment Term</p>
          <p class="font-medium text-text-primary">{po.payment_term || '-'}</p>
        </div>
        <div>
          <p class="text-text-muted mb-0.5">Supplier Ref</p>
          <p class="font-medium text-text-primary">{po.supplier_reference_number || '-'}</p>
        </div>
        <div class="col-span-2">
          <p class="text-text-muted mb-0.5">Delivery Address</p>
          <p class="font-medium text-text-primary">{po.delivery_address || '-'}</p>
        </div>
        <div class="col-span-2">
          <p class="text-text-muted mb-0.5">Notes</p>
          <p class="font-medium text-text-primary">{po.notes || '-'}</p>
        </div>
      </div>

      <div>
        <h3 class="text-sm font-semibold text-text-primary mb-2">Items</h3>
        <div class="border border-border rounded-xl overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-muted/50">
              <tr class="border-b text-left text-xs text-text-muted">
                <th class="px-3 py-2 font-semibold">Product</th>
                <th class="px-3 py-2 font-semibold text-right">Qty</th>
                <th class="px-3 py-2 font-semibold text-right">Unit Cost</th>
                <th class="px-3 py-2 font-semibold text-right">Discount</th>
                <th class="px-3 py-2 font-semibold text-right">Subtotal</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each po.items as item}
                <tr>
                  <td class="px-3 py-2.5 text-text-primary">{item.product_name || `Product #${item.product_id}`}</td>
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
          <span>Subtotal</span>
          <span class="tabular-nums">{formatCurrency(po.subtotal)}</span>
        </div>
        {#if po.discount_amount}
          <div class="flex justify-between text-text-secondary">
            <span>Discount</span>
            <span class="tabular-nums">-{formatCurrency(po.discount_amount)}</span>
          </div>
        {/if}
        {#if po.tax_amount}
          <div class="flex justify-between text-text-secondary">
            <span>Tax</span>
            <span class="tabular-nums">{formatCurrency(po.tax_amount)}</span>
          </div>
        {/if}
        <div class="flex justify-between font-semibold text-text-primary text-base pt-1.5 border-t border-border">
          <span>Grand Total</span>
          <span class="tabular-nums">{formatCurrency(po.grand_total)}</span>
        </div>
      </div>

      <div class="border-t border-border pt-4 grid grid-cols-2 gap-4 text-xs text-text-muted">
        <div>
          <p>Created: {formatDate(po.created_at)}</p>
          {#if po.confirmed_at}<p>Confirmed: {formatDate(po.confirmed_at)}</p>{/if}
          {#if po.cancelled_at}<p>Cancelled: {formatDate(po.cancelled_at)}</p>{/if}
        </div>
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      <div></div>
      <Button variant="secondary" size="sm" onclick={() => window.print()} disabled={!po}>
        <Printer size={14} />
        Print
      </Button>
    </div>
  {/snippet}
</Drawer>
