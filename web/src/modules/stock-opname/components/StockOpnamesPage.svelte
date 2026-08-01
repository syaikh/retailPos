<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/router';
  import { useStockOpnameStore } from '../stores/stock-opname-store.svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { Badge, Button, Card, Input, Modal, PageHeader, Pagination, SearchBar, EmptyState, Skeleton } from '$shared/ui';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { Plus } from 'lucide-svelte';
  import { STOCK_OPNAME_STATUS_LABELS } from '../types';
  import type { StockOpnameSession } from '../types';

  const store = useStockOpnameStore();
  const authStore = useAuthStore();
  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('stock_opname.create'));
  const canView = $derived(userPermissions.includes('stock_opname.view'));
  const canExport = $derived(userPermissions.includes('stock_opname.export'));

  let showCreateModal = $state(false);
  let creating = $state(false);
  let createScopeType = $state('store');
  let createScopeID = $state(1);
  let createBlind = $state(false);

  let firstLoad = true;
  let loadTimer: ReturnType<typeof setTimeout>;
  $effect(() => {
    store.currentFilters;
    if (firstLoad) {
      firstLoad = false;
      store.loadSessions(store.currentFilters);
      return;
    }
    clearTimeout(loadTimer);
    loadTimer = setTimeout(() => {
      store.page = 0;
      store.loadSessions(store.currentFilters);
    }, 300);
    return () => clearTimeout(loadTimer);
  });

  function handlePageChange(newOffset: number, newLimit: number) {
    store.pageSize = newLimit;
    store.page = Math.floor(newOffset / newLimit);
    store.loadSessions(store.currentFilters);
  }

  function handleCreate() {
    createScopeType = 'store';
    createScopeID = 1;
    createBlind = false;
    showCreateModal = true;
  }

  async function submitCreate() {
    creating = true;
    try {
      const session = await store.createSession({
        scope_type: createScopeType,
        scope_id: createScopeID,
        blind_count: createBlind,
      });
      toast.success(`Stock opname ${session.session_number} created`);
      showCreateModal = false;
      store.loadSessions(store.currentFilters);
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to create stock opname');
    } finally {
      creating = false;
    }
  }

  function openDetail(id: number) {
    store.clearCurrent();
    goto(`/stock-opnames/${id}`);
  }

  function statusBadge(status: string) {
    switch (status) {
      case 'draft': return 'muted';
      case 'counting': return 'primary';
      case 'pending_approval': return 'warning';
      case 'needs_recount': return 'warning';
      case 'approved': return 'success';
      case 'cancelled': return 'danger';
      default: return 'default';
    }
  }

  async function doExport(session: StockOpnameSession) {
    try {
      const blob = await store.exportCSV(session.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `stock-opname-${session.session_number}.csv`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (e) {
      toast.error('Failed to export stock opname');
    }
  }
</script>

<div class="space-y-5">
  <PageHeader title="Stock Opname" subtitle="Physical stock count sessions">
    {#snippet actions()}
      {#if canCreate}
        <Button onclick={handleCreate}>
          <Plus class="w-4 h-4" /> New Stock Opname
        </Button>
      {/if}
    {/snippet}
  </PageHeader>

  <div class="flex flex-wrap items-center gap-3">
    <div class="w-72 max-w-full">
      <SearchBar bind:query={store.searchFilter} placeholder="Search session number..." />
    </div>
    <div class="w-48">
      <Input tag="select" bind:value={store.statusFilter}>
        {#snippet children()}
          <option value="">All Status</option>
          {#each Object.entries(STOCK_OPNAME_STATUS_LABELS) as [value, label]}
            <option value={value}>{label}</option>
          {/each}
        {/snippet}
      </Input>
    </div>
  </div>

  <Card class="overflow-hidden">
    {#if store.loading && store.sessions.length === 0}
      <div class="space-y-2 p-4">
        <Skeleton class="h-10 w-full" />
        <Skeleton class="h-10 w-full" />
        <Skeleton class="h-10 w-full" />
      </div>
    {:else if store.sessions.length === 0}
      <EmptyState
        title="No stock opname sessions"
        description="Create a new stock opname session to start counting physical stock."
      />
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left whitespace-nowrap">
          <thead class="bg-surface-subtle border-b border-border">
            <tr class="text-[11px] font-semibold text-text-muted uppercase tracking-wider h-10">
              <th class="px-4">Session</th>
              <th class="px-4">Scope</th>
              <th class="px-4">Status</th>
              <th class="px-4">Blind</th>
              <th class="px-4">Created</th>
              <th class="px-4">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border/60">
            {#each store.sessions as session}
              <tr class="hover:bg-surface-subtle cursor-pointer" onclick={() => openDetail(session.id)}>
                <td class="px-4 py-3 font-medium text-text-primary">{session.session_number}</td>
                <td class="px-4 py-3 text-text-secondary">{session.scope_type} #{session.scope_id}</td>
                <td class="px-4 py-3">
                  <Badge variant={statusBadge(session.status)}>{STOCK_OPNAME_STATUS_LABELS[session.status] || session.status}</Badge>
                </td>
                <td class="px-4 py-3 text-text-secondary">{session.blind_count ? 'Yes' : 'No'}</td>
                <td class="px-4 py-3 text-text-secondary">{formatDateTimeInJakarta(session.created_at)}</td>
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2" onclick={(e) => e.stopPropagation()}>
                    {#if canExport}
                      <Button variant="ghost" size="sm" onclick={() => doExport(session)}>Export</Button>
                    {/if}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <div class="p-4">
        <Pagination total={store.total} limit={store.pageSize} offset={store.offset} onPageChange={handlePageChange} />
      </div>
    {/if}
  </Card>
</div>

<Modal bind:open={showCreateModal} title="New Stock Opname" size="md">
  {#snippet children()}
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1">Scope Type</label>
        <Input tag="select" bind:value={createScopeType}>
          {#snippet children()}
            <option value="store">Store</option>
            <option value="warehouse">Warehouse</option>
            <option value="category">Category</option>
            <option value="product">Product</option>
          {/snippet}
        </Input>
      </div>
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1">Scope ID</label>
        <Input type="number" bind:value={createScopeID} min={1} />
      </div>
      <label class="flex items-center gap-2 text-sm text-text-secondary cursor-pointer">
        <input type="checkbox" bind:checked={createBlind} class="accent-primary" />
        Blind count (hide system quantities from counters)
      </label>
      <p class="text-xs text-text-muted">All active products with general stock will be included in the session.</p>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showCreateModal = false)}>Cancel</Button>
      <Button onclick={submitCreate} disabled={creating}>
        {#if creating}Creating...{:else}Create{/if}
      </Button>
    </div>
  {/snippet}
</Modal>
