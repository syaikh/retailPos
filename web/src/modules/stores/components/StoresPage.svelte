<script>
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { debounce } from '$shared/utils/debounce';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';
  import { formatDateInJakarta } from '$shared/utils/jakartaTime';
  import { labels, t } from '$shared/i18n';
  import { useSortable } from '$shared/composables/useSortable.svelte';
  import { getStores, createStore, updateStore, deleteStore } from '../services/stores-service';

  const rbac = useRBAC();

  import { Button, Input, Modal, Skeleton, BulkActionDropdown, ImportWizard, SearchBar, ToggleSwitch, ConfirmDeleteModal, Pagination, SortableHeader } from '$shared/ui';
  import { Plus, Pencil, Trash2, Store, Loader2 } from 'lucide-svelte';

  let loading = $state(true);
  let stores = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let statusFilter = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedStore = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);
  const { sortState, handleSort } = useSortable('name', 'asc');
  let showImportWizard = $state(false);

  let form = $state({
    name: '',
    address: '',
    phone: '',
    is_active: true
  });

  let canCreate = $derived(rbac.can(Permissions.store.create));
  let canEdit = $derived(rbac.can(Permissions.store.update));
  let canDelete = $derived(rbac.can(Permissions.store.delete));

  function formatDate(dateStr) {
    if (!dateStr) return '—';
    return formatDateInJakarta(dateStr);
  }

  let sortedStores = $derived(
    [...stores].sort((a, b) => {
      let aVal, bVal;
      switch (sortState.sortBy) {
        case 'name':
          aVal = a.name.toLowerCase();
          bVal = b.name.toLowerCase();
          break;
        case 'created_at':
          aVal = new Date(a.created_at).getTime();
          bVal = new Date(b.created_at).getTime();
          break;
        default:
          return 0;
      }
      if (sortState.sortDir === 'asc') {
        return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
      } else {
        return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
      }
    })
  );

  function setStatusFilter(value) {
    statusFilter = value;
    offset = 0;
    fetchStores(false);
  }

  async function fetchStores(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const res = await getStores({ limit, offset, search: searchQuery || undefined, is_active: statusFilter ? statusFilter === 'active' : undefined });
      stores = res.data;
      total = res.total;
    } catch {
      toast.error(labels.toastFailedLoadStores);
    } finally {
      if (!isSearch) loading = false;
    }
  }

  onMount(async () => {
    await fetchStores(false);
  });

  function handleSearchInput() {
    offset = 0;
    if (searchQuery === '') {
      fetchStores(false);
    } else {
      debouncedSearchFetch();
    }
  }

  const debouncedSearchFetch = debounce(() => {
    fetchStores(true);
  }, 400);

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
    fetchStores(false);
  }

  function openAdd() {
    modalMode = 'add';
    form = { name: '', address: '', phone: '', is_active: true };
    showModal = true;
  }

  function openEdit(store) {
    modalMode = 'edit';
    selectedStore = store;
    form = {
      name: store.name,
      address: store.address || '',
      phone: store.phone || '',
      is_active: store.is_active !== false
    };
    showModal = true;
  }

  function openDelete(store) {
    selectedStore = store;
    showDeleteModal = true;
  }

  async function saveStore() {
    if (!form.name.trim()) {
      toast.error(labels.errorStoreNameRequired);
      return;
    }
    try {
      saving = true;
      let ok;
      if (modalMode === 'add') {
        ok = await createStore({
          name: form.name.trim(),
          address: form.address.trim() || undefined,
          phone: form.phone.trim() || undefined
        });
      } else {
        ok = await updateStore(selectedStore.id, {
          name: form.name.trim(),
          address: form.address.trim() || undefined,
          phone: form.phone.trim() || undefined,
          is_active: form.is_active
        });
      }
      if (ok) {
        toast.success(modalMode === 'add' ? labels.toastStoreAdded : labels.toastStoreUpdated);
        showModal = false;
        await fetchStores();
      } else {
        toast.error(labels.toastFailedSaveStore);
      }
    } catch {
      toast.error(labels.networkError);
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedStore) return;
    try {
      const ok = await deleteStore(selectedStore.id);
      if (ok) {
        toast.success(t('toastStoreDeleted', { name: selectedStore.name }));
        await fetchStores();
      } else {
        toast.error(labels.toastStoreDeleteInUse);
      }
    } catch {
      toast.error(labels.toastFailedDeleteStore);
    } finally {
      showDeleteModal = false;
      selectedStore = null;
    }
  }

  function handleImportComplete() {
    fetchStores();
    toast.success(labels.toastStoreImportCompleted);
  }
</script>

<div class="space-y-5">
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="flex-2">
        <SearchBar bind:value={searchQuery} placeholder={labels.searchStoresBy} oninput={handleSearchInput} inputClass="h-10" />
      </div>
      {#if canCreate}
        <div class="flex items-center gap-2">
          <BulkActionDropdown
            module="stores"
            canExport={true}
            canImport={true}
            onImport={() => showImportWizard = true}
          />
          <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={openAdd}>
            <Plus size={18} />
            {labels.addStore}
          </Button>
        </div>
      {/if}
    </div>
    <div class="flex items-center gap-2 mt-3">
      {#each [
        { label: labels.all, value: '' },
        { label: labels.active, value: 'active' },
        { label: labels.inactive, value: 'inactive' }
      ] as chip}
        <button
          type="button"
          onclick={() => setStatusFilter(chip.value)}
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === chip.value ? 'bg-primary/10 border border-primary/30 text-primary-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover border border-transparent'}"
        >
          {chip.label}
        </button>
      {/each}
    </div>
  </div>

  <div class="card overflow-hidden">
    {#if loading}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 30%;">{labels.storeName}</th>
            <th class="text-left p-4 font-semibold w-40">{labels.address}</th>
            <th class="text-left p-4 font-semibold w-36">{labels.phone}</th>
            <th class="text-left p-4 font-semibold w-28">{labels.status}</th>
            <th class="text-left p-4 font-semibold w-28">{labels.createdAt}</th>
            <th class="text-center p-4 font-semibold w-20">{labels.actions}</th>
          </tr>
        </thead>
        <tbody>
          {#each Array(5) as _}
            <tr class="border-t border-border">
              <td class="p-4 min-w-0"><Skeleton class="h-4 w-full" /></td>
              <td class="p-4 w-40"><Skeleton class="h-4 w-3/4" /></td>
              <td class="p-4 w-36"><Skeleton class="h-4 w-2/3" /></td>
              <td class="p-4 w-28"><Skeleton class="h-4 w-2/3" /></td>
              <td class="p-4 w-28"><Skeleton class="h-4 w-2/3" /></td>
              <td class="p-4 w-20"><Skeleton class="h-4 w-8" /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if stores.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Store size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">{labels.noStoresFound}</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? t('noResultsFor', { query: searchQuery }) : labels.addFirstStore}
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full table-fixed min-w-[720px]">
          <thead class="bg-muted/50">
            <tr>
              <th class="text-left p-4 font-semibold" style="width: 30%;">
                <SortableHeader label={labels.storeName} column="name" sortColumn={sortState.sortBy} sortDirection={sortState.sortDir} onsort={handleSort} />
              </th>
              <th class="text-left p-4 font-semibold w-40">{labels.address}</th>
              <th class="text-left p-4 font-semibold w-36">{labels.phone}</th>
              <th class="text-left p-4 font-semibold w-28">{labels.status}</th>
              <th class="text-left p-4 font-semibold w-28">
                <SortableHeader label={labels.createdAt} column="created_at" sortColumn={sortState.sortBy} sortDirection={sortState.sortDir} onsort={handleSort} />
              </th>
              <th class="text-center p-4 font-semibold w-20">{labels.actions}</th>
            </tr>
          </thead>
          <tbody>
            {#each sortedStores as store (store.id)}
              <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
                <td class="p-4 pr-6" style="width: 30%;">
                  <div class="flex items-center gap-3">
                    <div class="w-8 h-8 rounded-full bg-primary-subtle flex items-center justify-center shrink-0">
                      <Store size={14} class="text-primary-light" />
                    </div>
                    <div class="min-w-0">
                      <p class="font-medium truncate" title={store.name}>{store.name}</p>
                    </div>
                  </div>
                </td>
                <td class="p-4 w-40 text-text-secondary text-sm">
                  <span class="block truncate" title={store.address || ''}>{store.address || '—'}</span>
                </td>
                <td class="p-4 w-36 text-text-secondary text-sm">{store.phone || '—'}</td>
                <td class="p-4 w-28">
                  {#if store.is_active}
                    <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-success/10 text-success-light border border-success/20">
                      <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
                      {labels.active}
                    </span>
                  {:else}
                    <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-surface-default text-text-muted border border-border">
                      {labels.inactive}
                    </span>
                  {/if}
                </td>
                <td class="p-4 w-28 text-text-secondary text-sm">
                  {formatDate(store.created_at)}
                </td>
                <td class="p-4 w-20">
                  <div class="flex items-center justify-center gap-2">
                    {#if canEdit}
                      <Button
                        variant="ghost"
                        size="icon"
                        class="text-text-muted hover:text-primary-light"
                        title={labels.edit}
                        aria-label={labels.edit}
                        onclick={() => openEdit(store)}
                      >
                        <Pencil size={14} />
                      </Button>
                    {/if}
                    {#if canDelete}
                      <Button
                        variant="ghost"
                        size="icon"
                        class="text-text-muted hover:text-danger hover:bg-danger-subtle"
                        onclick={() => openDelete(store)}
                        title={labels.delete}
                        aria-label={labels.delete}
                      >
                        <Trash2 size={14} />
                      </Button>
                    {/if}
                    {#if !canEdit && !canDelete}
                      <span class="text-xs text-text-muted">—</span>
                    {/if}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
    {#if !loading && total > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
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

<Modal bind:open={showModal} title={modalMode === 'add' ? labels.addStore : labels.editStore} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveStore(); }} class="space-y-4">
    <div>
      <label for="store-name" class="block text-sm font-medium text-text-secondary mb-2">{labels.storeName} <span class="text-danger">*</span></label>
      <Input id="store-name" type="text" placeholder={labels.contohNamaToko} bind:value={form.name} required maxlength="100" />
    </div>
    <div>
      <label for="store-address" class="block text-sm font-medium text-text-secondary mb-2">{labels.address} <span class="text-text-muted text-xs">{labels.optionalShort}</span></label>
      <Input id="store-address" type="text" placeholder={labels.contohAlamat} bind:value={form.address} />
    </div>
    <div>
      <label for="store-phone" class="block text-sm font-medium text-text-secondary mb-2">{labels.phone} <span class="text-text-muted text-xs">{labels.optionalShort}</span></label>
      <Input id="store-phone" type="text" placeholder={labels.contohTelepon} bind:value={form.phone} />
    </div>
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-3">
        <ToggleSwitch bind:checked={form.is_active} label={form.is_active ? labels.active : labels.inactive} />
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false} disabled={saving}>{labels.cancel}</Button>
    <Button variant="primary" class="min-w-32" onclick={saveStore} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> {labels.saving}
      {:else}
        {modalMode === 'add' ? labels.addStore : labels.simpanPerubahan}
      {/if}
    </Button>
  {/snippet}
</Modal>

<ImportWizard
  bind:open={showImportWizard}
  module="stores"
  displayName={labels.stores}
  onComplete={handleImportComplete}
 />

<ConfirmDeleteModal bind:open={showDeleteModal} title={labels.deleteStore} itemName={selectedStore?.name} confirmLabel={labels.delete} cancelLabel={labels.cancel} description={labels.deleteStoreDescription} loading={false} onconfirm={confirmDelete} oncancel={() => showDeleteModal = false} />
