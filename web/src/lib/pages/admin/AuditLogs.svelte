<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import { Search, ScrollText, RefreshCw, CalendarDays } from 'lucide-svelte';

  let loading = $state(true);
  let items = $state([]);
  let searchQuery = $state('');
  let selectedAction = $state('all');

  const actionTypes = ['all', 'CREATE', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT'];

  const filtered = $derived(
    items.filter(log => {
      const matchesSearch = !searchQuery ||
        (log.actor || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
        (log.resource || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
        (log.description || '').toLowerCase().includes(searchQuery.toLowerCase());
      const matchesAction = selectedAction === 'all' || log.action === selectedAction;
      return matchesSearch && matchesAction;
    })
  );

  const actionVariant = (a) => {
    if (a === 'CREATE') return 'success';
    if (a === 'DELETE') return 'danger';
    if (a === 'UPDATE') return 'warning';
    if (a === 'LOGIN' || a === 'LOGOUT') return 'primary';
    return 'muted';
  };

  async function fetchLogs() {
    try {
      loading = true;
      const r = await apiFetch('/api/admin/audit-logs');
      if (r.ok) {
        const data = await r.json();
        items = data.data || data || [];
      }
    } catch {
      toast.error('Failed to load audit logs');
    } finally {
      loading = false;
    }
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
        <span class="badge badge-muted">{filtered.length} entries</span>
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
    {:else if filtered.length === 0}
      <div class="empty-state py-16">
        <div class="empty-state-icon bg-surface w-20 h-20">
          <ScrollText size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold">No log entries found</p>
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
            {#each filtered as log (log.id)}
              <tr>
                <td class="text-text-muted text-xs whitespace-nowrap">
                  <div class="flex items-center gap-1.5">
                    <CalendarDays size={12} class="opacity-50 flex-shrink-0" />
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
    {/if}
  </div>
</div>