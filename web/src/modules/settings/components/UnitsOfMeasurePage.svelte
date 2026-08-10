<script>
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { debounce } from '$shared/utils/debounce';
  import { useAuthStore } from '$modules/auth';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';
  import { formatDateInJakarta } from '$shared/utils/jakartaTime';
  import { labels, t } from '$shared/i18n';
  import { getUnitsOfMeasure, createUnitOfMeasure, updateUnitOfMeasure, deleteUnitOfMeasure } from '$modules/settings/services/settings-service';

  const rbac = useRBAC();

  import { Button, Input, Modal, Skeleton, BulkActionDropdown, ImportWizard, SearchBar, ToggleSwitch, ConfirmDeleteModal, Pagination } from '$shared/ui';
  import { Plus, Pencil, Trash2, Ruler, Loader2 } from 'lucide-svelte';

  let loading = $state(true);
  let uoms = $state([]);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedUom = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);

  let form = $state({
    code: '',
    name: '',
    description: '',
    is_active: true
  });

  let canCreate = $derived(rbac.can(Permissions.product.create));
  let canEdit = $derived(rbac.can(Permissions.product.update));
  let canDelete = $derived(rbac.can(Permissions.product.delete));

  function formatDate(dateStr) {
    if (!dateStr) return '—';
    return formatDateInJakarta(dateStr);
  }

  let pageSize = $state(20);
  let page = $state(0);
  let total = $state(0);

  let offset = $derived(page * pageSize);

  async function fetchUoms(offset = 0, limit = 20) {
    try {
      loading = true;
      const result = await getUnitsOfMeasure({ limit, offset, search: searchQuery || undefined });
      uoms = result.data;
      total = result.total;
      page = Math.floor(offset / limit);
      pageSize = limit;
    } catch {
      toast.error(labels.toastFailedToLoadUnitsOfMeasure);
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    await fetchUoms(0, pageSize);
  });

  function handleSearchInput() {
    page = 0;
    debouncedSearchFetch();
  }

  const debouncedSearchFetch = debounce(() => {
    fetchUoms();
  }, 400);

  let showImportWizard = $state(false);

  function handleImportComplete() {
    fetchUoms();
    toast.success(labels.toastUomImportCompleted);
  }

  function openAdd() {
    modalMode = 'add';
    form = { code: '', name: '', description: '', is_active: true };
    showModal = true;
  }

  function openEdit(uom) {
    modalMode = 'edit';
    selectedUom = uom;
    form = {
      code: uom.code,
      name: uom.name,
      description: uom.description || '',
      is_active: uom.is_active !== false
    };
    showModal = true;
  }

  function openDelete(uom) {
    selectedUom = uom;
    showDeleteModal = true;
  }

  async function saveUom() {
    if (!form.code.trim()) {
      toast.error(labels.errorUomCodeRequired);
      return;
    }
    if (!form.name.trim()) {
      toast.error(labels.errorUomNameRequired);
      return;
    }
    try {
      saving = true;
      let ok;
      if (modalMode === 'add') {
        ok = await createUnitOfMeasure({
          code: form.code.trim(),
          name: form.name.trim(),
          description: form.description.trim() || undefined
        });
      } else {
        ok = await updateUnitOfMeasure(selectedUom.id, {
          code: form.code.trim(),
          name: form.name.trim(),
          description: form.description.trim() || undefined,
          is_active: form.is_active
        });
      }
      if (ok) {
        toast.success(modalMode === 'add' ? labels.toastUnitAdded : labels.toastUnitUpdated);
        showModal = false;
        await fetchUoms();
      } else {
        toast.error(labels.toastFailedSaveUnit);
      }
    } catch {
      toast.error(labels.networkError);
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedUom) return;
    try {
      const ok = await deleteUnitOfMeasure(selectedUom.id);
      if (ok) {
        toast.success(t('toastUnitDeleted', { name: selectedUom.name }));
        await fetchUoms();
      } else {
        toast.error(labels.toastFailedDeleteUnit);
      }
    } catch {
      toast.error(labels.toastFailedDeleteUnit);
    } finally {
      showDeleteModal = false;
      selectedUom = null;
    }
  }
</script>

<div class="space-y-5">
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="flex-2">
        <SearchBar bind:value={searchQuery} placeholder={labels.searchByNameOrCode} oninput={handleSearchInput} inputClass="h-10" />
      </div>
      {#if canCreate}
        <div class="flex items-center gap-2">
          <BulkActionDropdown
            module="uoms"
            canExport={true}
            canImport={true}
            onImport={() => showImportWizard = true}
          />
          <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={openAdd}>
            <Plus size={18} />
            {labels.addUnit}
          </Button>
        </div>
      {/if}
    </div>
  </div>

  <div class="card overflow-hidden">
    {#if loading}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 15%;">{labels.code}</th>
            <th class="text-left p-4 font-semibold" style="width: 30%;">{labels.unitName}</th>
            <th class="text-left p-4 font-semibold w-48">{labels.description}</th>
            <th class="text-left p-4 font-semibold w-36">{labels.createdAt}</th>
            <th class="text-center p-4 font-semibold w-20">{labels.actions}</th>
          </tr>
        </thead>
        <tbody>
          {#each Array(5) as _}
            <tr class="border-t border-border">
              <td class="p-4 w-20"><Skeleton class="h-4 w-16" /></td>
              <td class="p-4 min-w-0"><Skeleton class="h-4 w-full" /></td>
              <td class="p-4 w-48"><Skeleton class="h-4 w-3/4" /></td>
              <td class="p-4 w-36"><Skeleton class="h-4 w-2/3" /></td>
              <td class="p-4 w-20"><Skeleton class="h-4 w-8" /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if uoms.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Ruler size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">{labels.noUnitsFound}</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? t('noResultsFor', { query: searchQuery }) : labels.addFirstUnit}
        </p>
      </div>
    {:else}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 15%;">{labels.code}</th>
            <th class="text-left p-4 font-semibold" style="width: 30%;">{labels.unitName}</th>
            <th class="text-left p-4 font-semibold w-48">{labels.description}</th>
            <th class="text-left p-4 font-semibold w-36">{labels.createdAt}</th>
            <th class="text-center p-4 font-semibold w-20">{labels.actions}</th>
          </tr>
        </thead>
        <tbody>
           {#each uoms as uom (uom.id)}
            <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
              <td class="p-4 w-20">
                <span class="font-mono text-sm font-semibold text-primary-light bg-primary-subtle/30 px-2 py-0.5 rounded">{uom.code}</span>
              </td>
              <td class="p-4 pr-6" style="width: 30%;">
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-full bg-primary-subtle flex items-center justify-center shrink-0">
                    <Ruler size={14} class="text-primary-light" />
                  </div>
                  <div class="min-w-0">
                    <p class="font-medium truncate" title={uom.name}>{uom.name}</p>
                  </div>
                </div>
              </td>
              <td class="p-4 w-40 text-text-secondary text-sm">
                {uom.description || '—'}
              </td>
              <td class="p-4 w-36 text-text-secondary text-sm">
                {formatDate(uom.created_at)}
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
                      onclick={() => openEdit(uom)}
                    >
                      <Pencil size={14} />
                    </Button>
                  {/if}
                  {#if canDelete}
                    <Button
                      variant="ghost"
                      size="icon"
                      class="text-text-muted hover:text-danger hover:bg-danger-subtle"
                      onclick={() => openDelete(uom)}
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
    {/if}
    {#if !loading && total > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination
          total={total}
          limit={pageSize}
          offset={offset}
          onPageChange={(newOffset, newLimit) => {
            fetchUoms(newOffset, newLimit);
          }}
        />
      </div>
    {/if}
  </div>
</div>

<Modal bind:open={showModal} title={modalMode === 'add' ? labels.addUnit : labels.editUnit} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveUom(); }} class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="uom-code" class="block text-sm font-medium text-text-secondary mb-2">{labels.code} <span class="text-danger">*</span></label>
        <Input id="uom-code" type="text" placeholder={labels.contohPcsKgBox} bind:value={form.code} required />
      </div>
      <div>
        <label for="uom-name" class="block text-sm font-medium text-text-secondary mb-2">{labels.unitName} <span class="text-danger">*</span></label>
        <Input id="uom-name" type="text" placeholder={labels.contohPiecesKilogram} bind:value={form.name} required />
      </div>
    </div>
    <div>
      <label for="uom-desc" class="block text-sm font-medium text-text-secondary mb-2">{labels.description} <span class="text-text-muted text-xs">{labels.optionalShort}</span></label>
      <Input tag="textarea" id="uom-desc" placeholder={labels.deskripsiSingkatUnit} class="min-h-[80px] resize-y" bind:value={form.description} />
    </div>
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-3">
        <ToggleSwitch bind:checked={form.is_active} label={form.is_active ? labels.active : labels.inactive} />
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false} disabled={saving}>{labels.cancel}</Button>
    <Button variant="primary" class="min-w-32" onclick={saveUom} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> {labels.saving}
      {:else}
        {modalMode === 'add' ? labels.addUnit : labels.simpanPerubahan}
      {/if}
    </Button>
  {/snippet}
</Modal>

<ImportWizard
  bind:open={showImportWizard}
  module="uoms"
  displayName={labels.unitsOfMeasure}
  onComplete={handleImportComplete}
/>

<ConfirmDeleteModal bind:open={showDeleteModal} title={labels.deleteUnit} itemName={selectedUom?.name} confirmLabel={labels.delete} cancelLabel={labels.cancel} description={labels.deleteUnitDescription} loading={false} onconfirm={confirmDelete} oncancel={() => showDeleteModal = false} />
