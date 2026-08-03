<script lang="ts">
  import { goto } from '$app/router';
  import { useStockOpnameStore } from '../stores/stock-opname-store.svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { Button, Input, Modal, Pagination, SelectSearch } from '$shared/ui';
  import { Plus, Trash2 } from 'lucide-svelte';
  import { getActiveStores } from '$modules/stores';
  import {
    getBrands,
    getCategories,
    getProductOptions,
    getWarehouses,
  } from '$modules/product/services/product-service';
  import { getSuppliers } from '$modules/supplier/services/supplier-service';
  import { getStorageLocations } from '$modules/storage-location/services/storage-location-service';
  import type { StockOpnameSession, StockOpnameScopeType } from '../types';
  import { STOCK_OPNAME_SCOPE_LABELS, STOCK_OPNAME_SCOPE_TYPES } from '../types';
  import StockOpnamesToolbar from './StockOpnamesToolbar.svelte';
  import StockOpnamesTable from './StockOpnamesTable.svelte';

  const store = useStockOpnameStore();
  const authStore = useAuthStore();
  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('stock_opname.create'));
  const canExport = $derived(userPermissions.includes('stock_opname.export'));
  const canReport = $derived(userPermissions.includes('stock_opname.report'));

  interface CreateScopeRow {
    scope_type: StockOpnameScopeType;
    scope_id: number | undefined;
  }

  let showCreateModal = $state(false);
  let creating = $state(false);
  let createTitle = $state('');
  let createNotes = $state('');
  let createBlind = $state(false);
  let createRows = $state<CreateScopeRow[]>([{ scope_type: 'store', scope_id: undefined }]);

  let optionCache = $state<Partial<Record<StockOpnameScopeType, { value: number; label: string }[]>>>({});
  let optionsLoading = $state(false);

  function scopeOptionsFor(type: StockOpnameScopeType): { value: number; label: string }[] {
    return optionCache[type] ?? [];
  }

  async function loadOptions(type: StockOpnameScopeType) {
    if (optionCache[type] || type === 'manual') return;
    optionsLoading = true;
    try {
      let loaded: { value: number; label: string }[] = [];
      if (type === 'store') {
        const stores = await getActiveStores();
        loaded = stores.map((s) => ({ value: s.id, label: s.name }));
      } else if (type === 'warehouse') {
        const warehouses = await getWarehouses();
        loaded = warehouses.map((w) => ({
          value: w.id,
          label: w.code ? `${w.name} (${w.code})` : w.name,
        }));
      } else if (type === 'category') {
        const categories = await getCategories();
        loaded = categories.map((c) => ({ value: c.id, label: c.name }));
      } else if (type === 'brand') {
        const brands = await getBrands();
        loaded = brands.map((b) => ({ value: b.id, label: b.name }));
      } else if (type === 'supplier') {
        const res = await getSuppliers({ limit: 500, offset: 0, is_active: true });
        loaded = res.data.map((s) => ({ value: s.id, label: s.name }));
      } else if (type === 'product') {
        const options = await getProductOptions();
        loaded = options.map((p) => ({
          value: p.id,
          label: p.sku ? `${p.name} (${p.sku})` : p.name,
        }));
      } else if (type === 'location') {
        const res = await getStorageLocations({ is_active: true, limit: 500, offset: 0 });
        loaded = res.data.map((l) => ({
          value: l.id,
          label: l.code ? `${l.name} (${l.code})` : l.name,
        }));
      }
      optionCache = { ...optionCache, [type]: loaded };
    } catch {
      optionCache = { ...optionCache, [type]: [] };
    } finally {
      optionsLoading = false;
    }
  }

  function addRow() {
    createRows = [...createRows, { scope_type: 'store', scope_id: undefined }];
  }

  function removeRow(index: number) {
    createRows = createRows.filter((_, i) => i !== index);
  }

  function onRowTypeChange(row: CreateScopeRow) {
    row.scope_id = undefined;
    loadOptions(row.scope_type);
  }

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

  $effect(() => {
    const unsubWS = store.subscribeToWS();
    return () => unsubWS();
  });

  function handlePageChange(newOffset: number, newLimit: number) {
    store.pageSize = newLimit;
    store.page = Math.floor(newOffset / newLimit);
    store.loadSessions(store.currentFilters);
  }

  function handleCreate() {
    createTitle = '';
    createNotes = '';
    createBlind = false;
    createRows = [{ scope_type: 'store', scope_id: undefined }];
    showCreateModal = true;
    loadOptions('store');
  }

  async function submitCreate() {
    const scopes = createRows.map((row) => ({
      scope_type: row.scope_type,
      scope_id: row.scope_type === 'manual' ? 0 : (row.scope_id ?? 0),
    }));
    if (scopes.length === 0) {
      toast.error('Add at least one scope');
      return;
    }
    if (scopes.some((s) => s.scope_type !== 'manual' && s.scope_id <= 0)) {
      toast.error('Select a scope for every non-manual row');
      return;
    }
    creating = true;
    try {
      const session = await store.createSession({
        title: createTitle || undefined,
        scopes,
        blind_count: createBlind,
        notes: createNotes || undefined,
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
    {canReport}
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
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>Title</span>
          <Input type="text" bind:value={createTitle} placeholder="Optional title" />
        </label>
        <label class="flex items-center gap-2 text-sm text-text-secondary cursor-pointer self-end pb-2.5">
          <input type="checkbox" bind:checked={createBlind} class="accent-primary" />
          Blind count (hide system quantities from counters)
        </label>
      </div>

      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-sm font-medium text-text-secondary">Scopes</span>
          <Button variant="secondary" size="sm" onclick={addRow}>
            <Plus class="w-4 h-4" /> Add Scope
          </Button>
        </div>
        {#each createRows as row, i (i)}
          <div class="grid grid-cols-1 md:grid-cols-[200px_1fr_auto] gap-3 items-start">
            <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
              <span>Type</span>
              <Input tag="select" bind:value={row.scope_type} onchange={() => onRowTypeChange(row)}>
                {#snippet children()}
                  {#each STOCK_OPNAME_SCOPE_TYPES as t}
                    <option value={t}>{STOCK_OPNAME_SCOPE_LABELS[t]}</option>
                  {/each}
                {/snippet}
              </Input>
            </label>
            <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
              <span>Scope</span>
              {#if row.scope_type === 'manual'}
                <div class="rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-muted">
                  All active products
                </div>
              {:else if optionsLoading && !optionCache[row.scope_type]}
                <div class="rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-muted">Loading...</div>
              {:else}
                <SelectSearch
                  bind:value={row.scope_id}
                  options={scopeOptionsFor(row.scope_type)}
                  placeholder="Select scope..."
                  searchPlaceholder="Search..."
                  disabled={scopeOptionsFor(row.scope_type).length === 0}
                  notFoundText="No matching scope found"
                />
              {/if}
            </label>
            {#if createRows.length > 1}
              <Button variant="ghost" size="sm" class="self-end mb-1" onclick={() => removeRow(i)} aria-label="Remove scope">
                <Trash2 class="w-4 h-4" />
              </Button>
            {/if}
          </div>
        {/each}
        {#if createRows.some((r) => r.scope_type === 'location') && createRows.length > 1}
          <p class="text-xs text-amber-600">Storage Location scope must be the only scope — remove the other rows.</p>
        {/if}
      </div>

      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>Notes</span>
        <Input tag="textarea" bind:value={createNotes} placeholder="Optional notes" rows={2} />
      </label>
      <p class="text-xs text-text-muted">The session covers the union of all selected scopes. Sessions may run in parallel as long as they never count the same SKU.</p>
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
