<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import { auth } from '$lib/stores/auth';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import { Search, ScrollText, RefreshCw, CalendarDays, X, List, Plus, Edit, Trash, LogIn, LogOut, Info, ArrowRight, Monitor, Globe, ChevronRight, Package, ShoppingCart, Users, Tag, Shield, Store } from 'lucide-svelte';

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
  let showDetailsModal = $state(false);
  let selectedLog = $state(null);

  let userRole = $derived(
    $auth.user?.role?.name ||
    ($auth.user?.role && typeof $auth.user?.role === 'object' ? $auth.user.role.name : $auth.user?.role) ||
    ''
  );
  let canView = $derived(userRole === 'superadmin');

  const actionFilters = [
    { id: 'all', label: 'All', icon: List, color: 'text-text-muted', activeBg: 'bg-surface-default', activeText: 'text-text-primary' },
    { id: 'CREATE', label: 'Create', icon: Plus, color: 'text-success-light', activeBg: 'bg-success-subtle/50', activeText: 'text-success-light' },
    { id: 'UPDATE', label: 'Update', icon: Edit, color: 'text-warning-light', activeBg: 'bg-warning-subtle/50', activeText: 'text-warning-light' },
    { id: 'DELETE', label: 'Delete', icon: Trash, color: 'text-danger-light', activeBg: 'bg-danger-subtle/50', activeText: 'text-danger-light' },
    { id: 'LOGIN', label: 'Login', icon: LogIn, color: 'text-primary-light', activeBg: 'bg-primary-subtle/50', activeText: 'text-primary-light' },
    { id: 'LOGOUT', label: 'Logout', icon: LogOut, color: 'text-primary-light', activeBg: 'bg-primary-subtle/50', activeText: 'text-primary-light' }
  ];

  const entityFilters = [
    { id: 'all', label: 'All', icon: List },
    { id: 'auth', label: 'Auth', icon: LogIn },
    { id: 'user', label: 'User', icon: Users },
    { id: 'role', label: 'Role', icon: Shield },
    { id: 'product', label: 'Product', icon: Package },
    { id: 'sale', label: 'Sale', icon: ShoppingCart },
    { id: 'category', label: 'Category', icon: Tag },
    { id: 'brand', label: 'Brand', icon: Store }
  ];

  let selectedEntity = $state('all');

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
      if (selectedEntity !== 'all') params.append('entity_type', selectedEntity);

      const r = await apiFetch(`/api/audit-logs?${params.toString()}`, {
        signal: abortController.signal
      });

      // Check if this request is still the current one
      if (requestId !== currentRequestId) {
        return; // Request was cancelled, ignore result
      }

      if (r.ok) {
        const data = await r.json();
        console.log('Audit logs loaded:', data);
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
  let prevSelectedEntity = $state('all');
  let prevOffset = $state(0);
  let prevLimit = $state(20);

  // Debounced search function
  const debouncedSearchFetch = debounce(() => {
    offset = 0;
    fetchLogs();
  }, 400);

  // Single effect that handles all state changes
  $effect(() => {
    const currentSearchQuery = searchQuery;
    const currentSelectedAction = selectedAction;
    const currentSelectedEntity = selectedEntity;
    const currentOffset = offset;
    const currentLimit = limit;

    if (!hasInitialized) {
      hasInitialized = true;
      fetchLogs();
      prevSearchQuery = currentSearchQuery;
      prevSelectedAction = currentSelectedAction;
      prevSelectedEntity = currentSelectedEntity;
      prevOffset = currentOffset;
      prevLimit = currentLimit;
      return;
    }

    const searchChanged = currentSearchQuery !== prevSearchQuery;
    const filterChanged = currentSelectedAction !== prevSelectedAction || currentSelectedEntity !== prevSelectedEntity;
    const paginationChanged = currentOffset !== prevOffset || currentLimit !== prevLimit;

    if (searchChanged) {
      debouncedSearchFetch();
    } else if (filterChanged || paginationChanged) {
      fetchLogs();
    }

    prevSearchQuery = currentSearchQuery;
    prevSelectedAction = currentSelectedAction;
    prevSelectedEntity = currentSelectedEntity;
    prevOffset = currentOffset;
    prevLimit = currentLimit;
  });

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
  }

  function getChanges(log) {
    const oldVal = log?.old_values || {};
    const newVal = log?.new_values || {};
    
    const allKeys = new Set([...Object.keys(oldVal), ...Object.keys(newVal)]);
    const changes = [];
    
    for (const key of allKeys) {
      const ov = oldVal[key];
      const nv = newVal[key];
      
      // Ignore keys where values are identical
      if (JSON.stringify(ov) === JSON.stringify(nv)) continue;
      
      // Ignore sensitive or internal keys if any (passwords should be hashed anyway, but just in case)
      if (['password', 'password_hash', 'token', 'token_hash'].includes(key.toLowerCase())) continue;
      
      changes.push({
        key,
        old: ov,
        new: nv
      });
    }
    
    return changes;
  }

  let changes = $derived(selectedLog ? getChanges(selectedLog) : []);
</script>

<div class="space-y-5">
  {#if !canView}
    <div class="card px-4 py-12 text-center">
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
        <ScrollText size={32} class="text-text-muted" />
      </div>
      <p class="text-text-primary font-semibold mt-4">Access Denied</p>
      <p class="text-text-muted text-sm mt-1">Audit logs are restricted to superadmin only</p>
    </div>
  {:else}
    <!-- Filters -->
    <div class="card p-4">
      <!-- Search Section -->
      <div class="mb-4">
        <div class="relative">
          <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <input type="text" placeholder="Search by actor, entity type..." class="input w-full pl-9 pr-10" bind:value={searchQuery} />
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

      <!-- Entity Type Filter -->
      <div class="flex flex-wrap p-1 bg-surface-subtle/40 rounded-xl border border-border/40 backdrop-blur-md">
        {#each entityFilters as entity}
          {@const Icon = entity.icon}
          <button
            class="flex items-center gap-2 px-3 py-1.5 rounded-lg font-medium transition-all duration-200 group text-sm
              {selectedEntity === entity.id
                ? 'bg-surface-default text-text-primary shadow-sm scale-105 z-10'
                : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover/30'}"
            onclick={() => selectedEntity = entity.id}
          >
            <Icon size={14} />
            <span>{entity.label}</span>
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
                <th class="w-10"></th>
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
                    <span class="font-medium text-text-primary text-sm">{log.username || '—'}</span>
                  </td>
                  <td>
                    <Badge variant={actionVariant(log.action)}>
                      {log.action || '—'}
                    </Badge>
                  </td>
                  <td>
                    <span class="font-mono text-xs text-text-secondary bg-surface px-2 py-0.5 rounded-md">
                      {log.entity_type || '—'}
                    </span>
                  </td>
                  <td class="text-text-secondary text-sm max-w-xs truncate">
                    {log.description || '—'}
                  </td>
                  <td class="font-mono text-xs text-text-muted">
                    {log.ip_address || '—'}
                  </td>
                  <td class="text-right">
                    <button 
                      class="btn-icon w-8 h-8 rounded-lg hover:bg-primary-subtle/30 text-text-muted hover:text-primary-light transition-all"
                      onclick={() => { selectedLog = log; showDetailsModal = true; }}
                      title="View Details"
                    >
                      <Info size={15} />
                    </button>
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
  {/if}
</div>

<!-- Log Details Modal - only rendered when canView -->
{#if canView}
<Modal bind:open={showDetailsModal} title="Audit Log Details" size="lg">
  {#if selectedLog}
    <div class="space-y-6">
      <!-- Header Info -->
      <div class="flex flex-col md:flex-row gap-4 p-4 bg-surface-subtle/30 rounded-2xl border border-border/40">
        <div class="flex-1 space-y-3">
          <div class="flex items-center gap-2 text-xs text-text-muted">
            <CalendarDays size={14} />
            {new Date(selectedLog.created_at).toLocaleString('en-US', { dateStyle: 'long', timeStyle: 'medium' })}
          </div>
          <div class="flex items-center gap-3">
            <Badge variant={actionVariant(selectedLog.action)} class="uppercase text-[10px] tracking-widest px-3">
              {selectedLog.action}
            </Badge>
            <span class="font-mono text-xs px-2 py-1 bg-surface rounded-md border border-border/50 text-text-secondary">
              {selectedLog.entity_type}
            </span>
          </div>
          <p class="text-lg font-bold text-text-primary leading-tight">
            {selectedLog.description}
          </p>
        </div>

        <div class="flex flex-col gap-2 min-w-[200px]">
          <div class="flex items-center gap-2 p-2 bg-surface rounded-xl border border-border/30">
            <div class="w-8 h-8 rounded-lg bg-primary-subtle/30 flex items-center justify-center text-primary-light shrink-0">
              <List size={16} />
            </div>
            <div class="truncate">
              <p class="text-[10px] uppercase tracking-wider text-text-muted font-bold">Actor</p>
              <p class="text-sm font-semibold text-text-primary truncate">{selectedLog.username}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 p-2 bg-surface rounded-xl border border-border/30">
            <div class="w-8 h-8 rounded-lg bg-info-subtle/30 flex items-center justify-center text-info-light shrink-0">
              <Globe size={16} />
            </div>
            <div class="truncate">
              <p class="text-[10px] uppercase tracking-wider text-text-muted font-bold">IP Address</p>
              <p class="text-sm font-semibold text-text-primary">{selectedLog.ip_address}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Browser/Device Info -->
      {#if selectedLog.user_agent}
        <div class="p-4 bg-surface-subtle/20 rounded-2xl border border-border/20">
          <div class="flex items-start gap-3">
            <Monitor size={16} class="mt-0.5 text-text-muted shrink-0" />
            <div>
              <p class="text-[10px] uppercase tracking-wider text-text-muted font-bold mb-1">User Agent / Device Information</p>
              <p class="text-xs text-text-secondary leading-relaxed font-mono bg-surface/50 p-2 rounded-lg border border-border/30">
                {selectedLog.user_agent}
              </p>
            </div>
          </div>
        </div>
      {/if}

      <!-- Changes/Diff Section -->
      <div class="space-y-3">
        <h4 class="text-sm font-bold text-text-primary flex items-center gap-2">
          <Edit size={16} class="text-primary-light" />
          Data Changes
        </h4>
        
        {#if changes.length > 0}
          <div class="grid gap-3">
            {#each changes as change}
              <div class="group flex flex-col border border-border/40 rounded-xl overflow-hidden transition-all hover:border-primary/20">
                <div class="bg-surface px-3 py-2 border-b border-border/40 flex items-center justify-between">
                  <span class="text-xs font-bold text-primary-light uppercase tracking-wider">{change.key.replace(/_/g, ' ')}</span>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-[1fr,auto,1fr] items-center gap-3 p-3 bg-surface-subtle/10">
                  <div class="space-y-1">
                    <p class="text-[9px] uppercase tracking-widest text-text-muted font-bold">Previous</p>
                    <div class="text-xs font-mono p-2 bg-danger-subtle/10 text-danger-light rounded-lg border border-danger/10 break-all">
                      {change.old !== null && change.old !== undefined ? JSON.stringify(change.old) : '—'}
                    </div>
                  </div>
                  
                  <div class="flex justify-center">
                    <div class="w-8 h-8 rounded-full bg-surface flex items-center justify-center border border-border/50 text-text-muted shadow-sm">
                      <ChevronRight size={14} />
                    </div>
                  </div>

                  <div class="space-y-1">
                    <p class="text-[9px] uppercase tracking-widest text-text-muted font-bold">New Value</p>
                    <div class="text-xs font-mono p-2 bg-success-subtle/10 text-success-light rounded-lg border border-success/10 break-all">
                      {change.new !== null && change.new !== undefined ? JSON.stringify(change.new) : '—'}
                    </div>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {:else if selectedLog.action === 'CREATE' || selectedLog.action === 'UPDATE' || selectedLog.action === 'DELETE'}
          <div class="p-8 text-center bg-surface-subtle/20 rounded-2xl border border-dashed border-border/40">
            <ScrollText size={32} class="mx-auto text-text-muted opacity-30 mb-3" />
            <p class="text-sm text-text-muted italic">No specific data changes were captured for this action.</p>
          </div>
        {:else}
          <div class="p-8 text-center bg-surface-subtle/20 rounded-2xl border border-dashed border-border/40">
            <Info size={32} class="mx-auto text-text-muted opacity-30 mb-3" />
            <p class="text-sm text-text-muted italic">System event log (no data changes).</p>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <button class="btn btn-primary px-8" onclick={() => showDetailsModal = false}>Close Details</button>
  {/snippet}
</Modal>
{/if}
