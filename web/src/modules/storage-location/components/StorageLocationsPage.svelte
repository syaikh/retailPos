<script lang="ts">
  import { onMount } from 'svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { getStorageLocations, createStorageLocation, updateStorageLocation, deleteStorageLocation, bulkUpdateStorageLocations, bulkDeleteStorageLocations } from '../services/storage-location-service';
  import type { StorageLocation } from '../types';
  import { getWarehouses } from '$modules/product/services/product-service';
  import { getActiveStores } from '$modules/stores';
  import { Pagination } from '$shared/ui';
  import { debounce } from '$shared/utils/debounce';
  import { useSortable } from '$shared/composables/useSortable.svelte';
  import { labels } from '$shared/i18n';
  import StorageLocationsToolbar from './StorageLocationsToolbar.svelte';
  import StorageLocationsTable from './StorageLocationsTable.svelte';
  import CreateStorageLocationModal from './CreateStorageLocationModal.svelte';
  import EditStorageLocationModal from './EditStorageLocationModal.svelte';
  import DeleteStorageLocationModal from './DeleteStorageLocationModal.svelte';

  const authStore = useAuthStore();

  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('storage_location.create'));
  const canUpdate = $derived(userPermissions.includes('storage_location.update'));
  const canDelete = $derived(userPermissions.includes('storage_location.delete'));

  let loading = $state(true);
  let locations = $state<StorageLocation[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let statusFilter = $state('all');
  const { sortState, handleSort } = useSortable('code', 'asc', load);

  let warehouseOptions = $state<{ value: number; label: string }[]>([]);
  let storeOptions = $state<{ value: number; label: string }[]>([]);
  let warehouseMap = $state<Record<number, string>>({});
  let storeMap = $state<Record<number, string>>({});

  let showCreateModal = $state(false);
  let creating = $state(false);
  let showEditModal = $state(false);
  let selectedLocation = $state<StorageLocation | null>(null);
  let saving = $state(false);
  let showDeleteModal = $state(false);
  let deleteTargetName = $state('');
  let deleting = $state(false);

  async function loadScopeOptions() {
    try {
      const [warehouses, stores] = await Promise.all([getWarehouses(), getActiveStores()]);
      warehouseOptions = warehouses.map((w) => ({ value: w.id, label: w.code ? `${w.name} (${w.code})` : w.name }));
      storeOptions = stores.map((s) => ({ value: s.id, label: s.name }));
      warehouseMap = Object.fromEntries(warehouses.map((w) => [w.id, w.code ? `${w.name} (${w.code})` : w.name]));
      storeMap = Object.fromEntries(stores.map((s) => [s.id, s.name]));
    } catch {
      warehouseOptions = [];
      storeOptions = [];
    }
  }

  async function load() {
    loading = true;
    try {
      const filters: any = { limit, offset };
      if (searchQuery.trim()) filters.search = searchQuery.trim();
      if (statusFilter !== 'all') filters.is_active = statusFilter === 'active';

      const result = await getStorageLocations(filters);

      if (sortState.sortBy === 'code') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? a.code.localeCompare(b.code) : b.code.localeCompare(a.code));
      } else if (sortState.sortBy === 'name') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name));
      } else if (sortState.sortBy === 'status') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? (a.is_active ? 1 : 0) - (b.is_active ? 1 : 0) : (b.is_active ? 1 : 0) - (a.is_active ? 1 : 0));
      } else if (sortState.sortBy === 'updated_at') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? (a.updated_at || '').localeCompare(b.updated_at || '') : (b.updated_at || '').localeCompare(a.updated_at || ''));
      }

      locations = result.data;
      total = result.total;
    } catch (e) {
      toast.error(labels.toastFailedLoadStorageLocations);
    } finally {
      loading = false;
    }
  }

  const debouncedSearch = debounce(() => { offset = 0; load(); }, 300);

  function handleSearch() { debouncedSearch(); }
  function handleStatusChange() { offset = 0; load(); }
  function handlePageChange(newOffset: number, newLimit: number) {
    limit = newLimit;
    offset = newOffset;
    load();
  }

  async function handleCreate(data: { code: string; name: string; warehouse_id?: number | null; store_id?: number | null; notes?: string }) {
    creating = true;
    try {
      await createStorageLocation(data);
      toast.success(labels.toastStorageLocationCreated);
      showCreateModal = false;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedCreateLocation);
    } finally {
      creating = false;
    }
  }

  function openEdit(g: StorageLocation) {
    selectedLocation = g;
    showEditModal = true;
  }

  async function handleEditSave(data: any) {
    saving = true;
    try {
      await updateStorageLocation(data.id, data);
      toast.success(labels.toastStorageLocationUpdated);
      showEditModal = false;
      selectedLocation = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedUpdateLocation);
    } finally {
      saving = false;
    }
  }

  function openDelete(g: StorageLocation) {
    selectedLocation = g;
    deleteTargetName = g.name;
    showDeleteModal = true;
  }

  async function handleDeleteConfirm() {
    if (!selectedLocation) return;
    deleting = true;
    try {
      await deleteStorageLocation(selectedLocation.id);
      toast.success(labels.toastStorageLocationDeleted);
      showDeleteModal = false;
      selectedLocation = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedDeleteLocation);
    } finally {
      deleting = false;
    }
  }

  async function handleBulkActivate(ids: number[]) {
    try {
      const updated = await bulkUpdateStorageLocations(ids, true);
      toast.success(labels.toastLocationsActivated.replace('{count}', String(updated)));
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedActivateLocations);
    }
  }

  async function handleBulkDeactivate(ids: number[]) {
    try {
      const updated = await bulkUpdateStorageLocations(ids, false);
      toast.success(labels.toastLocationsDeactivated.replace('{count}', String(updated)));
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedDeactivateLocations);
    }
  }

  async function handleBulkDelete(ids: number[]) {
    try {
      const deleted = await bulkDeleteStorageLocations(ids);
      toast.success(labels.toastLocationsDeleted.replace('{count}', String(deleted)));
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedDeleteLocation);
    }
  }

  onMount(() => {
    loadScopeOptions();
    load();
  });
</script>

<div class="space-y-5">
  <StorageLocationsToolbar
    bind:searchQuery
    bind:statusFilter
    {canCreate}
    onsearch={handleSearch}
    onstatuschange={handleStatusChange}
    oncreate={() => showCreateModal = true}
  />

  <div class="card overflow-x-auto">
    <StorageLocationsTable
      {locations}
      {loading}
      {searchQuery}
      {warehouseMap}
      {storeMap}
      {canUpdate}
      {canDelete}
      {canCreate}
      sortBy={sortState.sortBy}
      sortDir={sortState.sortDir}
      onsort={handleSort}
      onedit={openEdit}
      ondelete={openDelete}
      onbulkactivate={handleBulkActivate}
      onbulkdeactivate={handleBulkDeactivate}
      onbulkdelete={handleBulkDelete}
    />

    {#if !loading && locations.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
      </div>
    {/if}
  </div>
</div>

<CreateStorageLocationModal bind:open={showCreateModal} bind:creating {warehouseOptions} {storeOptions} oncreate={handleCreate} />
<EditStorageLocationModal bind:open={showEditModal} bind:location={selectedLocation} bind:saving {warehouseOptions} {storeOptions} onsave={handleEditSave} oncancel={() => { showEditModal = false; selectedLocation = null; }} />
<DeleteStorageLocationModal bind:open={showDeleteModal} bind:deleting targetName={deleteTargetName} oncancel={() => { showDeleteModal = false; selectedLocation = null; }} onconfirm={handleDeleteConfirm} />
