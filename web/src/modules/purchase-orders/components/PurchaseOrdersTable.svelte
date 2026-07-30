<script lang="ts">
  import { Button, Badge, Skeleton, SortableHeader, Dropdown } from '$shared/ui';
  import { MoreVertical, Eye, Pencil, Package, Check, XCircle } from 'lucide-svelte';
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
</script>

{#if loading}
  <div class="overflow-x-auto">
    <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label="Loading purchase orders">
      <colgroup>
        <col style="width: 15%;" />
        <col style="width: 22%;" />
        <col style="width: 12%;" />
        <col style="width: 13%;" />
        <col style="width: 13%;" />
        <col style="width: 13%;" />
        <col style="width: 12%;" />
      </colgroup>
      <thead><tr><th>PO NUMBER</th><th>SUPPLIER</th><th>STATUS</th><th>EXPECTED DATE</th><th>GRAND TOTAL</th><th>CREATED AT</th><th></th></tr></thead>
      <tbody>{#each Array(5) as _}<tr>{#each Array(7) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
    </table>
  </div>
{:else if purchaseOrders.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-text-muted" role="status">
    <Package class="w-12 h-12 mb-3" aria-hidden="true" />
    <p class="text-text-primary font-medium">No purchase orders found</p>
    {#if searchQuery}
      <p class="text-sm text-text-muted mt-1">Try adjusting your search or filters</p>
    {/if}
  </div>
{:else}
  <div class="overflow-x-auto">
    <table class="w-full min-w-[900px]" style="table-layout: fixed;" role="grid" aria-label="Purchase orders">
      <colgroup>
        <col style="width: 15%;" />
        <col style="width: 22%;" />
        <col style="width: 12%;" />
        <col style="width: 13%;" />
        <col style="width: 13%;" />
        <col style="width: 13%;" />
        <col style="width: 12%;" />
      </colgroup>
      <thead class="bg-muted/50">
        <tr class="border-b text-left text-sm text-text-muted">
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="PO NUMBER" column="po_number" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="SUPPLIER" column="supplier_name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="STATUS" column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="EXPECTED DATE" column="expected_date" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold text-right" scope="col">
            <SortableHeader label="GRAND TOTAL" column="grand_total" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="CREATED AT" column="created_at" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold text-right" scope="col">ACTIONS</th>
        </tr>
      </thead>
      <tbody>
          {#each purchaseOrders as po (po.id)}
            <tr class="border-b border-border transition-colors hover:bg-muted/50 cursor-pointer" onclick={() => onview(po)}>
              <td class="px-4 py-3 text-sm font-medium truncate">{po.po_number}</td>
              <td class="px-4 py-3 text-sm text-text-secondary truncate">{(po as any).supplier_name || 'N/A'}</td>
              <td class="px-4 py-3">
                <Badge variant={getStatusVariant(po.status)} size="sm">{titleCase(po.status)}</Badge>
              </td>
              <td class="px-4 py-3 text-sm text-text-secondary tabular-nums">{formatDate(po.expected_date)}</td>
              <td class="px-4 py-3 text-sm text-text-secondary tabular-nums text-right">{formatCurrency(po.grand_total)}</td>
              <td class="px-4 py-3 text-sm text-text-secondary tabular-nums">{formatDate(po.created_at)}</td>
              <td class="px-4 py-3 text-center">
                <Dropdown items={[
                  ...(canView ? [{ label: 'View', icon: Eye, onclick: () => onview(po) }] : []),
                  ...(canEdit && po.status === 'draft' ? [{ label: 'Edit', icon: Pencil, onclick: () => onedit(po) }] : []),
                  ...(canConfirm && po.status === 'draft' ? [{ label: 'Confirm', icon: Check, onclick: () => onconfirm(po) }] : []),
                  ...(canReceive && (po.status === 'confirmed' || po.status === 'partial_received') ? [{ label: 'Receive', icon: Package, onclick: () => onreceive(po) }] : []),
                  ...(canCancel && (po.status === 'draft' || po.status === 'confirmed') ? [{ label: 'Cancel', icon: XCircle, onclick: () => oncancel(po) }] : []),
                ]}>
                  {#snippet trigger({ toggle })}
                    <button type="button" onclick={(e) => { e.stopPropagation(); toggle(); }} class="p-1 rounded-lg hover:bg-muted transition-colors" aria-label="Actions for {po.po_number}">
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
