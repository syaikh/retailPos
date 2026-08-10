<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$shared/api/http-client';
  import { toast } from '$shared/stores/toast.svelte';
  import { debounce } from '$shared/utils/debounce';
  import { useAuthStore } from '$modules/auth';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';
  import { formatDateInJakarta } from '$shared/utils/jakartaTime';
  import { labels, t } from '$shared/i18n';

  const authStore = useAuthStore();
  const rbac = useRBAC();

  import { Button, Input, Modal, Pagination, SearchBar, Skeleton, BulkActionDropdown, ImportWizard, ToggleSwitch, ConfirmDeleteModal, SortableHeader } from '$shared/ui';
  import { Plus, Pencil, Trash2, Tag, Loader2, X } from 'lucide-svelte';

  let loading = $state(true);
  let categories = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedCategory = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);
  let sortBy = $state('name');
  let sortDir = $state('asc');

  let form = $state({
    name: '',
    description: '',
    is_active: true
  });

// RBAC derived from auth store
let canCreate = $derived(rbac.can(Permissions.category.create));
let canEdit = $derived(rbac.can(Permissions.category.update));
let canDelete = $derived(rbac.can(Permissions.category.delete));
// Show content if user loaded (API will enforce 403 for cashier)
let canView = $derived(authStore.user != null);

  function formatDate(dateStr) {
    if (!dateStr) return '—';
    return formatDateInJakarta(dateStr);
  }

  function handleSort(column) {
    if (sortBy === column) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = column;
      sortDir = 'asc';
    }
  }

  let sortedCategories = $derived(
    [...categories].sort((a, b) => {
      let aVal, bVal;
      switch (sortBy) {
        case 'name':
          aVal = a.name.toLowerCase();
          bVal = b.name.toLowerCase();
          break;
        case 'slug':
          aVal = a.slug.toLowerCase();
          bVal = b.slug.toLowerCase();
          break;
        case 'product_count':
          aVal = a.product_count || 0;
          bVal = b.product_count || 0;
          break;
        case 'created_at':
          aVal = new Date(a.created_at).getTime();
          bVal = new Date(b.created_at).getTime();
          break;
        default:
          return 0;
      }
      if (sortDir === 'asc') {
        return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
      } else {
        return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
      }
    })
  );

  let showImportWizard = $state(false);

  function handleImportComplete() {
    fetchCategories();
    toast.success(labels.toastCategoryImportCompleted);
  }

  async function fetchCategories(isSearch = false) {
    // Note: Authorization checked via API response (403), not frontend
    try {
      if (!isSearch) loading = true;
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery
      });
      const res = await apiFetch(`/api/categories/manage?${params.toString()}`);
      if (res.ok) {
        const data = await res.json();
        categories = data.data || [];
        total = data.total || 0;
      } else if (res.status === 403) {
        toast.error(labels.accessDenied);
        categories = [];
        total = 0;
      }
    } catch {
      toast.error(labels.toastFailedToLoadCategories);
    } finally {
      if (!isSearch) loading = false;
    }
  }

  onMount(async () => {
    // Fetch immediately - API will handle 401/403 if not authenticated
    await fetchCategories(false);
  });

  // Search: event-driven, NOT $effect
  function handleSearchInput() {
    offset = 0;
    if (searchQuery === '') {
      fetchCategories(false);
    } else {
      debouncedSearchFetch();
    }
  }

  const debouncedSearchFetch = debounce(() => {
    fetchCategories(true);
  }, 400);


  // Pagination
  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
    fetchCategories(false);
  }

  function openAdd() {
    modalMode = 'add';
    form = { name: '', description: '', is_active: true };
    showModal = true;
  }

  function openEdit(cat) {
    modalMode = 'edit';
    selectedCategory = cat;
    form = {
      name: cat.name,
      description: cat.description || '',
      is_active: cat.is_active !== false
    };
    showModal = true;
  }

  function openDelete(cat) {
    selectedCategory = cat;
    showDeleteModal = true;
  }

  async function saveCategory() {
    if (!form.name.trim()) {
      toast.error(labels.errorCategoryNameRequired);
      return;
    }
    try {
      saving = true;
      const method = modalMode === 'add' ? 'POST' : 'PUT';
      const url = modalMode === 'add' ? '/api/categories' : `/api/categories/${selectedCategory.id}`;
      const payload = {
        name: form.name.trim(),
        description: form.description.trim()
      };
      if (modalMode === 'edit') {
        payload.is_active = form.is_active;
      }
      const r = await apiFetch(url, { method, body: JSON.stringify(payload) });
      if (r.ok) {
        toast.success(modalMode === 'add' ? labels.toastCategoryAdded : labels.toastCategoryUpdated);
        showModal = false;
        await fetchCategories();
      } else {
        const err = await r.json();
        toast.error(err.error || labels.toastFailedSaveCategory);
      }
    } catch {
      toast.error(labels.networkError);
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedCategory) return;
    try {
      const r = await apiFetch(`/api/categories/${selectedCategory.id}`, { method: 'DELETE' });
      if (r.ok) {
        toast.success(t('toastCategoryDeleted', { name: selectedCategory.name }));
        await fetchCategories();
      } else {
        const err = await r.json();
        toast.error(err.error || labels.toastFailedDeleteCategory);
      }
    } catch {
      toast.error(labels.toastFailedDeleteCategory);
    } finally {
      showDeleteModal = false;
      selectedCategory = null;
    }
  }
</script>

<div class="space-y-5">
  <!-- Search -->
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="flex-2">
        <SearchBar bind:value={searchQuery} placeholder={labels.searchByNameOrSlug} oninput={handleSearchInput} inputClass="h-10" />
      </div>
      {#if canCreate}
        <div class="flex items-center gap-2">
          <BulkActionDropdown
            module="categories"
            canExport={true}
            canImport={true}
            onImport={() => showImportWizard = true}
          />
          <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={openAdd}>
            <Plus size={18} />
            {labels.addCategory}
          </Button>
        </div>
      {/if}
    </div>
  </div>

  <!-- Table -->
  <div class="card overflow-hidden">

    {#if loading}
      <table class="w-full table-fixed" aria-busy="true" aria-label={labels.loadingCategories}>
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 50%;">{labels.categoryName}</th>
            <th class="text-left p-4 font-semibold w-32">{labels.slugLabel}</th>
            <th class="text-right p-4 font-semibold w-24">{labels.products}</th>
            <th class="text-left p-4 font-semibold w-28">{labels.createdAt}</th>
            <th class="text-center p-4 font-semibold w-20">{labels.actions}</th>
          </tr>
        </thead>
        <tbody>
          {#each Array(5) as _}
            <tr class="border-t border-border">
              <td class="p-4 min-w-0"><Skeleton class="h-4 w-full" /></td>
              <td class="p-4 w-32"><Skeleton class="h-4 w-3/4" /></td>
              <td class="p-4 text-right w-24"><Skeleton class="h-4 w-1/2 ml-auto" /></td>
              <td class="p-4 w-28"><Skeleton class="h-4 w-2/3" /></td>
              <td class="p-4 w-20"><Skeleton class="h-4 w-8" /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if categories.length === 0}
      <div class="px-4 py-12 text-center" role="status">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Tag size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">{labels.noCategoriesFound}</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? t('noResultsFor', { query: searchQuery }) : labels.addFirstCategory}
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
      <table class="w-full table-fixed min-w-[700px]">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 50%;">
              <SortableHeader label={labels.categoryName} column="name" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="text-left p-4 font-semibold w-32">
              <SortableHeader label={labels.slugLabel} column="slug" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="text-right p-4 font-semibold w-24">
              <SortableHeader label={labels.products} column="product_count" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} align="right" />
            </th>
            <th class="text-left p-4 font-semibold w-28">
              <SortableHeader label={labels.createdAt} column="created_at" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="text-center p-4 font-semibold w-20">{labels.actions}</th>
          </tr>
        </thead>
        <tbody>
          {#each sortedCategories as cat (cat.id)}
            <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
              <td class="p-4 pr-6" style="width: 50%;">
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-full bg-primary-subtle flex items-center justify-center shrink-0">
                    <Tag size={14} class="text-primary-light" />
                  </div>
                  <div class="min-w-0">
                    <p class="font-medium truncate" title={cat.name}>{cat.name}</p>
                    {#if cat.description}
                      <p class="text-xs text-text-muted truncate">{cat.description}</p>
                    {/if}
                  </div>
                </div>
              </td>
              <td class="p-4 w-32 text-text-secondary text-sm">{cat.slug}</td>
              <td class="p-4 text-right w-24">
                <span class="inline-flex items-center justify-center min-w-[28px] px-2 py-0.5 rounded-full text-xs font-semibold
                  {cat.product_count > 0 ? 'bg-primary-subtle text-primary-light' : 'bg-surface-default text-text-muted'}">
                  {cat.product_count ?? 0}
                </span>
              </td>
              <td class="p-4 w-28 text-text-secondary text-sm">
                {formatDate(cat.created_at)}
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
                      onclick={() => openEdit(cat)}
                    >
                      <Pencil size={14} />
                    </Button>
                  {/if}
                  {#if canDelete}
                    <Button
                      variant="ghost"
                      size="icon"
                      class={cat.product_count > 0 ? 'text-text-muted/30 cursor-not-allowed' : 'text-text-muted hover:text-danger hover:bg-danger-subtle'}
                      onclick={() => openDelete(cat)}
                      disabled={cat.product_count > 0}
                      title={cat.product_count > 0 ? labels.categoryHasProducts : labels.delete}
                      aria-label={cat.product_count > 0 ? labels.categoryHasProducts : labels.delete}
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

      {#if !loading && categories.length > 0}
        <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
          <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
        </div>
      {/if}
    {/if}
  </div>
</div>

<!-- Add/Edit Modal -->
<Modal bind:open={showModal} title={modalMode === 'add' ? labels.addCategory : labels.editCategory} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveCategory(); }} class="space-y-4">
    <div>
      <label for="cat-name" class="block text-sm font-medium text-text-secondary mb-2">{labels.categoryName} <span class="text-danger">*</span></label>
      <Input id="cat-name" type="text" placeholder={labels.contohMakananBayi} bind:value={form.name} required />
    </div>
    <div>
      <label for="cat-desc" class="block text-sm font-medium text-text-secondary mb-2">{labels.description} <span class="text-text-muted text-xs">{labels.optionalShort}</span></label>
      <Input tag="textarea" id="cat-desc" placeholder={labels.deskripsiSingkatKategori} class="min-h-[80px] resize-y" bind:value={form.description} />
    </div>
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-3">
        <ToggleSwitch bind:checked={form.is_active} label={form.is_active ? labels.active : labels.inactive} />
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false} disabled={saving}>{labels.cancel}</Button>
    <Button variant="primary" class="min-w-32" onclick={saveCategory} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> {labels.saving}
      {:else}
        {modalMode === 'add' ? labels.addCategory : labels.simpanPerubahan}
      {/if}
    </Button>
  {/snippet}
</Modal>

<ImportWizard
  bind:open={showImportWizard}
  module="categories"
  displayName={labels.categories}
  onComplete={handleImportComplete}
/>


<ConfirmDeleteModal bind:open={showDeleteModal} title={labels.deleteCategory} itemName={selectedCategory?.name} confirmLabel={labels.delete} cancelLabel={labels.cancel} description={labels.deleteCategoryDescription} loading={false} onconfirm={confirmDelete} oncancel={() => showDeleteModal = false} />