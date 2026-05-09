<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Search, ScrollText, RefreshCw, CalendarDays, X, List, Plus, Edit, Trash, LogIn, LogOut } from 'lucide-svelte';

  let loading = $state(true);
  let items = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let selectedAction = $state('all');

  // Request tracking to prevent duplicate requests
  let currentRequestId = $state(0);
  let abortController = $state(null);
  let hasInitialized = $state(false);

  const actionFilters = [
    { id: 'all', label: 'All', icon: List, color: 'text-text-muted', activeBg: 'bg-surface-default', activeText: 'text-text-primary' },
    { id: 'CREATE', label: 'Create', icon: Plus, color: 'text-success-light', activeBg: 'bg-success-subtle/50', activeText: 'text-success-light' },
    { id: 'UPDATE', label: 'Update', icon: Edit, color: 'text-warning-light', activeBg: 'bg-warning-subtle/50', activeText: 'text-warning-light' },
    { id: 'DELETE', label: 'Delete', icon: Trash, color: 'text-danger-light', activeBg: 'bg-danger-subtle/50', activeText: 'text-danger-light' },
    { id: 'LOGIN', label: 'Login', icon: LogIn, color: 'text-primary-light', activeBg: 'bg-primary-subtle/50', activeText: 'text-primary-light' },
    { id: 'LOGOUT', label: 'Logout', icon: LogOut, color: 'text-primary-light', activeBg: 'bg-primary-subtle/50', activeText: 'text-primary-light' }
  ];

  const actionVariant = (a) => {
    if (a?.toUpperCase() === 'CREATE') return 'success';
    if (a?.toUpperCase() === 'DELETE') return 'danger';
    if (a?.toUpperCase() === 'UPDATE') return 'warning';
    if (a?.toUpperCase() === 'LOGIN' || a?.toUpperCase() === 'LOGOUT') return 'primary';
    return 'muted';
  };

  async function fetchLogs() {
    // Cancel any ongoing request
    if (abortController) {
      abortController.abort();
    }

    const requestId = ++currentRequestId;
    abortController = new AbortController();

    try {
      loading = true;
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery
      });
      if (selectedAction !== 'all') params.append('action', selectedAction);

      const r = await apiFetch(`/api/audit-logs?${params.toString()}`, {
        signal: abortController.signal
      });

      // Check if this request is still the current one
      if (requestId !== currentRequestId) {
        return; // Request was cancelled, ignore result
      }

      if (r.ok) {
        const data = await r.json();
        items = data.data || [];
        total = data.total || 0;
      }
    } catch (error) {
      // Don't show error for aborted requests
      if (error.name !== 'AbortError') {
        toast.error('Failed to load audit logs');
      }
    } finally {
      // Only clear loading if this is the current request
      if (requestId === currentRequestId) {
        loading = false;
        abortController = null;
      }
    }
  }

  // Track previous values to detect what changed
  let prevSearchQuery = $state('');
  let prevSelectedAction = $state('all');
  let prevOffset = $state(0);
  let prevLimit = $state(20);

  // Debounced search function
  const debouncedSearchFetch = debounce(() => {
    offset = 0; // Reset to first page when searching
    fetchLogs();
  }, 400);

  // Single effect that handles all state changes
  $effect(() => {
    const currentSearchQuery = searchQuery;
    const currentSelectedAction = selectedAction;
    const currentOffset = offset;
    const currentLimit = limit;

    // Skip if component hasn't initialized yet
    if (!hasInitialized) {
      hasInitialized = true;
      fetchLogs(); // Initial load
      // Update previous values
      prevSearchQuery = currentSearchQuery;
      prevSelectedAction = currentSelectedAction;
      prevOffset = currentOffset;
      prevLimit = currentLimit;
      return;
    }

    // Check what changed
    const searchChanged = currentSearchQuery !== prevSearchQuery;
    const filterChanged = currentSelectedAction !== prevSelectedAction;
    const paginationChanged = currentOffset !== prevOffset || currentLimit !== prevLimit;

    if (searchChanged) {
      // Search changed - use debounced fetch
      debouncedSearchFetch();
    } else if (filterChanged || paginationChanged) {
      // Filter or pagination changed - immediate fetch
      fetchLogs();
    }

    // Update previous values
    prevSearchQuery = currentSearchQuery;
    prevSelectedAction = currentSelectedAction;
    prevOffset = currentOffset;
    prevLimit = currentLimit;
  });

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
  }
</script>

<div class="space-y-5">


  <!-- Filters -->
  <div class="card p-4">
    <!-- Search Section -->
    <div class="mb-4">
      <div class="relative">
        <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
        <input type="text" placeholder="Search by actor, resource…" class="input w-full pl-9 pr-10" bind:value={searchQuery} />
        {#if searchQuery}
          <button
            onclick={() => searchQuery = ''}
            class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
            title="Clear search"
          >
            <X size={14} />
          </button>
        {/if}
      </div>
    </div>

    <!-- Filter and Action Section -->
    <div class="flex flex-wrap items-center gap-3">
      <div class="flex flex-wrap p-1 bg-surface-subtle/40 rounded-xl border border-border/40 backdrop-blur-md">
        {#each actionFilters as action}
          {@const Icon = action.icon}
          <button
            class="flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-all duration-200 group
              {selectedAction === action.id
                ? `${action.activeBg} ${action.activeText} shadow-sm scale-105 z-10`
                : `text-text-muted hover:text-text-secondary hover:bg-surface-hover/30`}"
            onclick={() => selectedAction = action.id}
          >
            <Icon size={16} class={selectedAction === action.id ? '' : 'opacity-60 group-hover:opacity-100 transition-opacity'} />
            <span class="text-sm">{action.label}</span>
          </button>
        {/each}
      </div>

      <!-- Refresh Button -->
      <button
        onclick={fetchLogs}
        disabled={loading}
        class="ml-auto btn btn-outline btn-primary px-5 py-2.5 rounded-xl shadow-glow-primary-sm hover:shadow-glow-primary transition-all active:scale-95"
      >
        {#if loading}
          <RefreshCw size={16} class="animate-spin mr-2" />
          Refreshing...
        {:else}
          <RefreshCw size={16} class="mr-2" />
          Refresh
        {/if}
      </button>
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
          <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
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
              <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
                <td class="text-text-muted text-xs whitespace-nowrap">
                  <div class="flex items-center gap-1.5">
                    <CalendarDays size={12} class="opacity-50 shrink-0" />
                    {new Date(log.created_at || log.timestamp).toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'medium' })}
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