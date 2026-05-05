<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Search, ScrollText, RefreshCw, CalendarDays } from 'lucide-svelte';

  let loading = $state(true);
  let items = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let selectedAction = $state('all');

  const actionTypes = ['all', 'CREATE', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT'];

  const actionVariant = (a) => {
    if (a?.toUpperCase() === 'CREATE') return 'success';
    if (a?.toUpperCase() === 'DELETE') return 'danger';
    if (a?.toUpperCase() === 'UPDATE') return 'warning';
    if (a?.toUpperCase() === 'LOGIN' || a?.toUpperCase() === 'LOGOUT') return 'primary';
    return 'muted';
  };

  async function fetchLogs() {
    try {
      loading = true;
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery
      });
      if (selectedAction !== 'all') params.append('action', selectedAction);

      const r = await apiFetch(`/api/audit-logs?${params.toString()}`);
      if (r.ok) {
        const data = await r.json();
        items = data.data || [];
        total = data.total || 0;
      }
    } catch {
      toast.error('Failed to load audit logs');
    } finally {
      loading = false;
    }
  }

  // Debounced search
  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchLogs();
  }, 400);

  $effect(() => {
    searchQuery;
    debouncedSearch();
  });

  $effect(() => {
    selectedAction;
    offset;
    limit;
    untrackedFetch();
  });

  function untrackedFetch() {
    fetchLogs();
  }

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
  }

  onMount(fetchLogs);
</script>

<div class="space-y-5">
  <PageHeader title="Audit Logs" subtitle="System activity and security trail">
    {#snippet actions()}
      <button class="btn btn-secondary" onclick={fetchLogs} disabled={loading}>
        <RefreshCw size={14} class={loading ? 'animate-spin' : ''} />
        Refresh
      </button>
    {/snippet}
  </PageHeader>

  <!-- Filters -->
  <div class="card p-4 flex flex-col sm:flex-row gap-3">
    <div class="relative flex-1">
      <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
      <input type="text" placeholder="Search by actor, resource…" class="input pl-9" bind:value={searchQuery} />
    </div>

    <!-- Action type pill filter -->
    <div class="flex items-center gap-2 overflow-x-auto no-scrollbar">
      {#each actionTypes as action}
        <button
          class={selectedAction === action ? 'pill-tab-active' : 'pill-tab'}
          onclick={() => selectedAction = action}
        >
          {action === 'all' ? 'All' : action}
        </button>
      {/each}
    </div>
  </div>

  <!-- Table -->
  <div class="card p-0 overflow-hidden">
    <div class="px-4 py-3 border-b border-border flex items-center justify-between">
      <p class="text-sm font-semibold text-text-primary">Activity Log</p>
      {#if !loading}
        <span class="badge badge-muted">{total} entries</span>
      {/if}
    </div>

    {#if loading}
      <div class="divide-y divide-border">
        {#each { length: 8 } as _}
          <div class="flex items-center gap-4 px-4 py-3.5">
            <Skeleton width="w-20" height="h-3" />
            <Skeleton width="w-16" height="h-6" rounded="rounded-full" />
            <Skeleton width="w-24" height="h-3" />
            <Skeleton width="w-32" height="h-3" />
            <Skeleton width="w-24" height="h-3" class="ml-auto" />
          </div>
        {/each}
      </div>
    {:else if items.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
          <ScrollText size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No log entries found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? `No results for "${searchQuery}"` : 'System activity will appear here'}
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table>
          <thead>
            <tr>
              <th>Timestamp</th>
              <th>Actor</th>
              <th>Action</th>
              <th>Resource</th>
              <th>Description</th>
              <th>IP Address</th>
            </tr>
          </thead>
          <tbody>
            {#each items as log (log.id)}
              <tr>
                <td class="text-text-muted text-xs whitespace-nowrap">
                  <div class="flex items-center gap-1.5">
                    <CalendarDays size={12} class="opacity-50 shrink-0" />
                    {new Date(log.created_at || log.timestamp).toLocaleString('id-ID')}
                  </div>
                </td>
                <td>
                  <span class="font-medium text-text-primary text-sm">{log.actor || log.username || '—'}</span>
                </td>
                <td>
                  <Badge variant={actionVariant(log.action)}>
                    {log.action || '—'}
                  </Badge>
                </td>
                <td>
                  <span class="font-mono text-xs text-text-secondary bg-surface px-2 py-0.5 rounded-md">
                    {log.resource || '—'}
                  </span>
                </td>
                <td class="text-text-secondary text-sm max-w-xs truncate">
                  {log.description || '—'}
                </td>
                <td class="font-mono text-xs text-text-muted">
                  {log.ip_address || '—'}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="p-4 bg-surface-subtle/30">
        <Pagination 
          {total} 
          {limit} 
          {offset} 
          onPageChange={handlePageChange} 
        />
      </div>
    {/if}
  </div>
</div>