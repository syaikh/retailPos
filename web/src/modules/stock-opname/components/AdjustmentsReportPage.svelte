<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/router';
  import { useStockOpnameStore } from '../stores/stock-opname-store.svelte';
  import { Badge, Card, Dropdown, EmptyState, Input, PageHeader, Pagination, Skeleton } from '$shared/ui';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { ChevronDown, PackageSearch } from 'lucide-svelte';

  const store = useStockOpnameStore();

  let searchQuery = $state('');
  let statusFilter = $state('');
  let pageSize = $state(20);
  let page = $state(0);

  const statusOptions = [
    { value: 'posted', label: 'Posted' },
    { value: 'reversed', label: 'Reversed' },
  ];

  const statusLabel = $derived(
    statusOptions.find(s => s.value === statusFilter)?.label || 'All Status'
  );

  const statusItems = $derived([
    { label: 'All Status', checked: statusFilter === '', onclick: () => { statusFilter = ''; load(); } },
    ...statusOptions.map(opt => ({
      label: opt.label,
      checked: statusFilter === opt.value,
      onclick: () => { statusFilter = opt.value; load(); },
    })),
  ]);

  let searchTimer: ReturnType<typeof setTimeout>;
  function load() {
    page = 0;
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      store.loadAdjustments({ status: statusFilter || undefined, search: searchQuery || undefined, limit: pageSize, offset: page * pageSize });
    }, 300);
  }

  onMount(() => {
    store.loadAdjustments({ status: undefined, search: undefined, limit: pageSize, offset: 0 });
  });

  function handlePageChange(newOffset: number, newLimit: number) {
    if (newLimit && newLimit !== pageSize) pageSize = newLimit;
    page = Math.floor(newOffset / newLimit);
    store.loadAdjustments({ status: statusFilter || undefined, search: searchQuery || undefined, limit: pageSize, offset: page * pageSize });
  }

  function statusVariant(status: string): 'default' | 'success' | 'warning' | 'danger' | 'primary' | 'muted' {
    switch (status) {
      case 'posted':
        return 'success';
      case 'reversed':
        return 'warning';
      case 'draft':
        return 'muted';
      default:
        return 'default';
    }
  }
</script>

<div class="space-y-5">
  <PageHeader title="Stock Opname Adjustments" subtitle="Adjustment documents (IA-...) created when a verified stock opname is posted to inventory." />

  <div class="card p-3">
    <div class="flex flex-wrap items-center gap-3">
      <div class="min-w-0 flex-[2_1_200px]">
        <Input type="text" placeholder="Search adjustment / session number..." bind:value={searchQuery} oninput={load} />
      </div>
      <Dropdown placement="bottom-start" items={statusItems}>
        {#snippet trigger({ toggle })}
          <button
            type="button"
            class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {statusFilter !== '' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
            onclick={toggle}
          >
            <span>{statusLabel}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
        {/snippet}
      </Dropdown>
    </div>
  </div>

  <Card class="overflow-hidden">
    {#if store.adjustmentsLoading && store.adjustments.length === 0}
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left whitespace-nowrap">
          <thead class="bg-surface-subtle border-b border-border">
            <tr class="text-[11px] font-semibold text-text-muted uppercase tracking-wider h-10">
              <th class="px-4">Adjustment</th>
              <th class="px-4">Session</th>
              <th class="px-4">Status</th>
              <th class="px-4 text-right">Total Diff</th>
              <th class="px-4 text-right">Total Value</th>
              <th class="px-4">Created By</th>
              <th class="px-4">Created At</th>
            </tr>
          </thead>
          <tbody>
            {#each Array(5) as _}
              <tr><td colspan="7" class="px-4 py-2"><Skeleton class="h-5 w-full" /></td></tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else if store.adjustments.length === 0}
      <div class="flex flex-col items-center justify-center py-14 text-text-muted" role="status">
        <PackageSearch class="w-12 h-12 mb-3" aria-hidden="true" />
        <p class="text-text-primary font-medium">No adjustments found</p>
        <p class="text-sm text-text-muted mt-1">Adjustments appear here once a stock opname is posted.</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left whitespace-nowrap">
          <thead class="bg-surface-subtle border-b border-border">
            <tr class="text-[11px] font-semibold text-text-muted uppercase tracking-wider h-10">
              <th class="px-4">Adjustment</th>
              <th class="px-4">Session</th>
              <th class="px-4">Status</th>
              <th class="px-4 text-right">Total Diff</th>
              <th class="px-4 text-right">Total Value</th>
              <th class="px-4">Created By</th>
              <th class="px-4">Created At</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border/60">
            {#each store.adjustments as adj}
              <tr
                class="cursor-pointer hover:bg-surface-subtle transition-colors"
                onclick={() => goto(`/stock-opnames/${adj.session_id}`)}
                title="Open source stock opname"
              >
                <td class="px-4 py-3 font-medium text-text-primary">{adj.adjustment_number}</td>
                <td class="px-4 py-3 text-text-secondary">{adj.session_number}</td>
                <td class="px-4 py-3">
                  <Badge variant={statusVariant(adj.status)} size="sm">{adj.status}</Badge>
                </td>
                <td class="px-4 py-3 text-right tabular-nums {adj.total_difference === 0 ? 'text-text-secondary' : adj.total_difference > 0 ? 'text-success-light' : 'text-danger-light'}">
                  {adj.total_difference.toLocaleString('id-ID')}
                </td>
                <td class="px-4 py-3 text-right tabular-nums text-text-primary">{adj.total_adjustment.toLocaleString('id-ID')}</td>
                <td class="px-4 py-3 text-text-secondary">{adj.created_by_name}</td>
                <td class="px-4 py-3 text-text-secondary tabular-nums">{formatDateTimeInJakarta(adj.created_at)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      {#if store.adjustmentsTotal > 0}
        <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
          <Pagination total={store.adjustmentsTotal} limit={pageSize} offset={page * pageSize} onPageChange={handlePageChange} />
        </div>
      {/if}
    {/if}
  </Card>
</div>
