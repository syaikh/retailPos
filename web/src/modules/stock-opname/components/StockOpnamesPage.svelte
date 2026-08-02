<script lang="ts">
  import { goto } from '$app/router';
  import { useStockOpnameStore } from '../stores/stock-opname-store.svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { Button, Input, Modal, Pagination } from '$shared/ui';
  import type { StockOpnameSession } from '../types';
  import StockOpnamesToolbar from './StockOpnamesToolbar.svelte';
  import StockOpnamesTable from './StockOpnamesTable.svelte';

  const store = useStockOpnameStore();
  const authStore = useAuthStore();
  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('stock_opname.create'));
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
  <StockOpnamesToolbar
    bind:searchQuery={store.searchFilter}
    bind:statusFilter={store.statusFilter}
    {canCreate}
    oncreate={handleCreate}
  />

  <div class="card overflow-hidden">
    <StockOpnamesTable
      sessions={store.sessions}
      loading={store.loading}
      searchQuery={store.searchFilter}
      {canExport}
      onview={(session) => openDetail(session.id)}
      onexport={doExport}
    />

    {#if !store.loading && store.sessions.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination total={store.total} limit={store.pageSize} offset={store.offset} onPageChange={handlePageChange} />
      </div>
    {/if}
  </div>
</div>

<Modal bind:open={showCreateModal} title="New Stock Opname" size="md">
  {#snippet children()}
    <div class="space-y-4">
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>Scope Type</span>
        <Input tag="select" bind:value={createScopeType}>
          {#snippet children()}
            <option value="store">Store</option>
            <option value="warehouse">Warehouse</option>
            <option value="category">Category</option>
            <option value="product">Product</option>
          {/snippet}
        </Input>
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>Scope ID</span>
        <Input type="number" bind:value={createScopeID} min={1} />
      </label>
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
