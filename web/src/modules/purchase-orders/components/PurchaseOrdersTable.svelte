<script lang="ts">
  import { Button, Badge, Skeleton, SortableHeader } from '$shared/ui';
  import { Pencil, Eye, Package } from 'lucide-svelte';
  import type { PurchaseOrder } from '../types';

  let {
    purchaseOrders = [],
    loading = false,
    searchQuery = '',
    canView = false,
    canEdit = false,
    canReceive = false,
    sortBy = 'created_at',
    sortDir = 'desc',
    onsort = () => {},
    onview = () => {},
    onedit = () => {},
    onreceive = () => {},
  }: {
    purchaseOrders?: PurchaseOrder[];
    loading?: boolean;
    searchQuery?: string;
    canView?: boolean;
    canEdit?: boolean;
    canReceive?: boolean;
    sortBy?: string;
    sortDir?: 'asc' | 'desc';
    onsort?: (col: string) => void;
    onview?: (po: PurchaseOrder) => void;
    onedit?: (po: PurchaseOrder) => void;
    onreceive?: (po: PurchaseOrder) => void;
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
</script>

{#if loading}
  <div class="overflow-x-auto">
    <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label="Loading purchase orders">
      <colgroup>
        <col style="width: 12%;" />
        <col style="width: 18%;" />
        <col style="width: 14%;" />
        <col style="width: 12%;" />
        <col style="width: 14%;" />
        <col style="width: 12%;" />
        <col style="width: 18%;" />
      </colgroup>
      <thead><tr><th>PO Number</th><th>Supplier</th><th>Status</th><th>Expected Date</th><th>Grand Total</th><th>Created At</th><th></th></tr></thead>
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
        <col style="width: 12%;" />
        <col style="width: 18%;" />
        <col style="width: 14%;" />
        <col style="width: 12%;" />
        <col style="width: 14%;" />
        <col style="width: 12%;" />
        <col style="width: 18%;" />
      </colgroup>
      <thead class="bg-muted/50">
        <tr class="border-b text-left text-sm text-text-muted">
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="PO Number" column="po_number" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="Supplier" column="supplier_name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="Status" column="status" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="Expected Date" column="expected_date" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold text-right" scope="col">
            <SortableHeader label="Grand Total" column="grand_total" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
          </th>
          <th class="px-4 py-3 font-semibold" scope="col">
            <SortableHeader label="Created At" column="created_at" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
          </th>
          <th class="px-4 py-3 font-semibold text-right" scope="col">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each purchaseOrders as po (po.id)}
          <tr class="border-b border-border transition-colors hover:bg-muted/50">
            <td class="px-4 py-3 text-sm font-medium truncate">{po.po_number}</td>
            <td class="px-4 py-3 text-sm text-text-secondary truncate">{(po as any).supplier_name || 'N/A'}</td>
            <td class="px-4 py-3">
              <Badge variant={getStatusVariant(po.status)} size="sm">{po.status.replace(/_/g, ' ')}</Badge>
            </td>
            <td class="px-4 py-3 text-sm text-text-secondary tabular-nums">{formatDate(po.expected_date)}</td>
            <td class="px-4 py-3 text-sm text-text-secondary tabular-nums text-right">{formatCurrency(po.grand_total)}</td>
            <td class="px-4 py-3 text-sm text-text-secondary tabular-nums">{formatDate(po.created_at)}</td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-1">
                {#if canView}
                  <Button variant="ghost" size="sm" onclick={() => onview(po)} aria-label="View {po.po_number}">
                    <Eye size={14} /> View
                  </Button>
                {/if}
                {#if canEdit && po.status === 'draft'}
                  <Button variant="ghost" size="sm" onclick={() => onedit(po)} aria-label="Edit {po.po_number}">
                    <Pencil size={14} /> Edit
                  </Button>
                {/if}
                {#if canReceive && (po.status === 'confirmed' || po.status === 'partial_received')}
                  <Button variant="ghost" size="sm" onclick={() => onreceive(po)} aria-label="Receive {po.po_number}">
                    <Package size={14} /> Receive
                  </Button>
                {/if}
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
