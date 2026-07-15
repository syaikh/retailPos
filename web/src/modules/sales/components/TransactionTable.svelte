<script lang="ts">
  import { Badge, Pagination, Skeleton, SortableHeader } from '$shared/ui';
  import { Banknote } from 'lucide-svelte';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';

  let {
    salesData = [],
    loading = false,
    total = 0,
    limit = 20,
    offset = 0,
    sortBy = $bindable('created_at'),
    sortDir = $bindable('desc'),
    ontogglesort = () => {},
    onpagechange = () => {},
    onrowclick = () => {},
  }: {
    salesData?: any[];
    loading?: boolean;
    total?: number;
    limit?: number;
    offset?: number;
    sortBy?: string;
    sortDir?: 'asc' | 'desc';
    ontogglesort?: (col: string) => void;
    onpagechange?: (offset: number, limit: number) => void;
    onrowclick?: (sale: any) => void;
  } = $props();

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

  function handleSort(col: string) {
    ontogglesort(col);
  }

  function handlePageChange(newOffset: number, newLimit: number) {
    onpagechange(newOffset, newLimit);
  }

  function handleRowClick(sale: any) {
    onrowclick(sale);
  }

  function handleRowKeydown(e: KeyboardEvent, sale: any) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleRowClick(sale);
    }
  }
</script>

<div class="card p-0 overflow-hidden">
  {#if loading}
    <div aria-busy="true" aria-label="Loading transactions">
      <div class="divide-y divide-border">
      {#each { length: 5 } as _}
        <div class="flex items-center gap-4 px-4 py-3.5">
          <Skeleton width="w-32" height="h-4" />
          <Skeleton width="w-24" height="h-4" />
          <Skeleton width="w-20" height="h-6" rounded="rounded-full" class="ml-auto" />
          <Skeleton width="w-28" height="h-4" />
        </div>
      {/each}
      </div>
    </div>
  {:else if salesData.length === 0}
    <div class="px-4 py-12 text-center" role="status">
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
        <Banknote size={32} class="text-text-muted" />
      </div>
      <p class="text-text-primary font-semibold mt-4">No transactions found</p>
      <p class="text-text-muted text-sm mt-1">Try adjusting the search or date range</p>
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full min-w-[860px]">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold">
              <SortableHeader label="INVOICE" column="invoice_number" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="text-left p-4 font-semibold">
              <SortableHeader label="DATE" column="created_at" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="text-left p-4 font-semibold w-[30%]">CUSTOMER</th>
            <th class="text-left p-4 font-semibold">ITEMS</th>
            <th class="text-left p-4 font-semibold">
              <SortableHeader label="PAYMENT" column="payment_method" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="text-right p-4 font-semibold">
              <SortableHeader label="TOTAL (RP)" column="total_amount" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} align="right" />
            </th>
          </tr>
        </thead>
        <tbody>
          {#each salesData as sale (sale.id)}
            <tr
              class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer"
              onclick={() => handleRowClick(sale)}
              onkeydown={(e) => handleRowKeydown(e, sale)}
              tabindex="0"
              role="button"
            >
              <td class="p-4">
                <span class="text-sm font-medium text-text-primary">
                  {sale.invoice_number}
                </span>
              </td>
              <td class="p-4 text-sm text-text-secondary">
                {formatDateTime(new Date(sale.created_at))}
              </td>
              <td class="p-4 text-sm text-text-secondary">
                {sale.customer_name || '—'}
              </td>
              <td class="p-4 text-sm text-text-secondary">
                {sale.items?.length || 0} items
              </td>
              <td class="p-4">
                <Badge variant={getPaymentMethodVariant(sale.payment_method)} class="text-xs px-2.5 py-0.5">
                  {sale.payment_method || '—'}
                </Badge>
              </td>
              <td class="p-4 text-right text-sm font-semibold text-text-primary">
                {(sale.total_amount || 0).toLocaleString('id-ID')}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <div class="p-4 bg-surface-subtle/30 border-t border-border/50">
      <Pagination
        {total}
        {limit}
        {offset}
        onPageChange={handlePageChange}
      />
    </div>
  {/if}
</div>
