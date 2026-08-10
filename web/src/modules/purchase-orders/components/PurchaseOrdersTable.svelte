<script lang="ts">
  import { Button, Badge, Skeleton, SortableHeader, Dropdown } from '$shared/ui';
  import { labels, t } from '$shared/i18n';
  import { MoreVertical, Eye, Pencil, Package, Check, XCircle, Copy } from 'lucide-svelte';
  import type { PurchaseOrder } from '../types';

  let {
    purchaseOrders = [],
    loading = false,
    searchQuery = '',
    canView = false,
    canEdit = false,
    canConfirm = false,
    canReceive = false,
    canCancel = false,
    sortBy = 'created_at',
    sortDir = 'desc',
    onsort = () => {},
    onview = () => {},
    onedit = () => {},
    onconfirm = () => {},
    onreceive = () => {},
    oncancel = () => {},
  }: {
    purchaseOrders?: PurchaseOrder[];
    loading?: boolean;
    searchQuery?: string;
    canView?: boolean;
    canEdit?: boolean;
    canConfirm?: boolean;
    canReceive?: boolean;
    canCancel?: boolean;
    sortBy?: string;
    sortDir?: 'asc' | 'desc';
    onsort?: (col: string) => void;
    onview?: (po: PurchaseOrder) => void;
    onedit?: (po: PurchaseOrder) => void;
    onconfirm?: (po: PurchaseOrder) => void;
    onreceive?: (po: PurchaseOrder) => void;
    oncancel?: (po: PurchaseOrder) => void;
  } = $props();

  function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'danger' | 'primary' | 'muted' {
    switch (status) {
      case 'draft':
        return 'muted';
      case 'confirmed':
        return 'primary';
      case 'partial_received':
        return 'warning';
      case 'fully_received':
        return 'success';
      case 'cancelled':
      case 'rejected':
        return 'danger';
      case 'waiting_approval':
        return 'default';
      default:
        return 'muted';
    }
  }

  function formatCurrency(value: number): string {
    return value.toLocaleString('id-ID');
  }

  function formatDate(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    });
  }

  function titleCase(str: string): string {
    return str.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
  }

  let copiedPOs = $state(new Set<number>());

  function handleCopyPO(poId: number, poNumber: string) {
    navigator.clipboard.writeText(poNumber);
    copiedPOs = new Set([...copiedPOs, poId]);
    setTimeout(() => {
      const next = new Set(copiedPOs);
      next.delete(poId);
      copiedPOs = next;
    }, 1500);
  }
</script>

{#if loading}
  <div class="overflow-x-auto">
    <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label={labels.loadingPurchaseOrders}>
      <colgroup>
        <col style="width: 15%;" />
        <col style="width: 17%;" />
        <col style="width: 18%;" />
        <col style="width: 15%;" />
        <col style="width: 15%;" />
        <col style="width: 12%;" />
        <col style="width: 8%;" />
      </colgroup>
      <thead><tr><th>{labels.poNumber}</th><th>{labels.supplierLabel}</th><th>{labels.statusLabel}</th><th>{labels.expectedDateLabel}</th><th>{labels.grandTotal}</th><th>{labels.createdAtLabel}</th><th></th></tr></thead>
      <tbody>{#each Array(5) as _}<tr>{#each Array(7) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
    </table>
  </div>
{:else if purchaseOrders.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-text-muted" role="status">
    <Package class="w-12 h-12 mb-3" aria-hidden="true" />
    <p class="text-text-primary font-medium">{labels.noPurchaseOrdersFound}</p>
    {#if searchQuery}
      <p class="text-sm text-text-muted mt-1">{labels.tryAdjustingYourSearch}</p>
    {/if}
  </div>
{:else}
  <div class="overflow-x-auto">
    <table class="w-full min-w-[1000px]" style="table-layout: fixed;" role="grid" aria-label={labels.purchaseOrders}>
      <colgroup>
        <col style="width: 15%;" />
        <col style="width: 17%;" />
        <col style="width: 18%;" />
        <col style="width: 15%;" />
        <col style="width: 15%;" />
        <col style="width: 12%;" />
        <col style="width: 8%;" />
      </colgroup>
      <thead class="bg-muted/50">
        <tr class="border-b text-left text-sm text-text-muted">
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">
            <SortableHeader label={labels.poNumber} column="po_number" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">
            <SortableHeader label={labels.supplierLabel} column="supplier_name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">
            <SortableHeader label={labels.statusLabel} column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">
            <SortableHeader label={labels.expectedDateLabel} column="expected_date" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap text-right" scope="col">
            <SortableHeader label={labels.grandTotal} column="grand_total" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
          </th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">
            <SortableHeader label={labels.createdAtLabel} column="created_at" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap text-right" scope="col">{labels.actionsLabel}</th>
        </tr>
      </thead>
      <tbody>
          {#each purchaseOrders as po (po.id)}
            <tr class="border-b border-border transition-colors hover:bg-muted/50 cursor-pointer" onclick={() => onview(po)}>
              <td class="px-4 py-3 text-sm font-medium max-w-0">
                <span class="flex items-center gap-1.5">
                  <span class="truncate">{po.po_number}</span>
                  <button type="button" class="p-0.5 hover:text-primary transition-colors w-5 h-5 flex items-center justify-center shrink-0" title={labels.copyPoNumber} aria-label={labels.copyPoNumber} onclick={(e) => { e.stopPropagation(); handleCopyPO(po.id, po.po_number); }}>
                    {#if copiedPOs.has(po.id)}
                      <span class="text-xs text-primary font-bold leading-none">✓</span>
                    {:else}
                      <Copy size={14} class="text-text-muted hover:text-primary" />
                    {/if}
                  </button>
                </span>
              </td>
              <td class="px-4 py-3 text-sm text-text-secondary max-w-0"><span class="truncate block">{(po as any).supplier_name || 'N/A'}</span></td>
              <td class="px-4 py-3 whitespace-nowrap">
                <Badge variant={getStatusVariant(po.status)} size="sm">{titleCase(po.status)}</Badge>
              </td>
              <td class="px-4 py-3 text-sm text-text-secondary tabular-nums whitespace-nowrap">{formatDate(po.expected_date)}</td>
              <td class="px-4 py-3 text-sm text-text-secondary tabular-nums text-right whitespace-nowrap">{formatCurrency(po.grand_total)}</td>
              <td class="px-4 py-3 text-sm text-text-secondary tabular-nums whitespace-nowrap">{formatDate(po.created_at)}</td>
              <td class="px-4 py-3 text-center">
                <Dropdown items={[
                  ...(canView ? [{ label: labels.detail, icon: Eye, onclick: () => onview(po) }] : []),
                  ...(canEdit && po.status === 'draft' ? [{ label: labels.edit, icon: Pencil, onclick: () => onedit(po) }] : []),
                  ...(canConfirm && po.status === 'draft' ? [{ label: labels.confirm, icon: Check, onclick: () => onconfirm(po) }] : []),
                  ...(canReceive && (po.status === 'confirmed' || po.status === 'partial_received') ? [{ label: labels.receiveGoods, icon: Package, onclick: () => onreceive(po) }] : []),
                  ...(canCancel && (po.status === 'draft' || po.status === 'confirmed') ? [{ label: labels.cancel, icon: XCircle, onclick: () => oncancel(po) }] : []),
                ]}>
                  {#snippet trigger({ toggle })}
                    <button type="button" onclick={(e) => { e.stopPropagation(); toggle(); }} class="p-1 rounded-lg hover:bg-muted transition-colors" aria-label={t('actionsFor', { name: po.po_number })}>
                      <MoreVertical size={16} class="text-text-muted" />
                    </button>
                  {/snippet}
                </Dropdown>
              </td>
            </tr>
          {/each}
      </tbody>
    </table>
  </div>
{/if}
