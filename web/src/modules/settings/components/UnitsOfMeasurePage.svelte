<script>
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { debounce } from '$shared/utils/debounce';
  import { useAuthStore } from '$modules/auth';
  import { formatDateInJakarta } from '$shared/utils/jakartaTime';
  import { getUnitsOfMeasure, createUnitOfMeasure, updateUnitOfMeasure, deleteUnitOfMeasure } from '$modules/settings/services/settings-service';

  const authStore = useAuthStore();

  import { Button, Input, Modal, Skeleton, BulkActionDropdown, ImportWizard, SearchBar } from '$shared/ui';
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

  let userRole = $derived(
    authStore.user?.role?.name ||
    (authStore.user?.role && typeof authStore.user?.role === 'object' ? authStore.user.role.name : authStore.user?.role) ||
    ''
  );
  let canCreate = $derived(['superadmin', 'admin'].includes(userRole));
  let canEdit = $derived(['superadmin', 'admin'].includes(userRole));
  let canDelete = $derived(['superadmin', 'admin'].includes(userRole));
  let canView = $derived(authStore.user != null);

  function formatDate(dateStr) {
    if (!dateStr) return '—';
    return formatDateInJakarta(dateStr);
  }

  let filteredUoms = $derived(
    uoms.filter(u =>
      !searchQuery ||
      u.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      u.code.toLowerCase().includes(searchQuery.toLowerCase())
    )
  );

  async function fetchUoms() {
    try {
      loading = true;
      uoms = await getUnitsOfMeasure();
    } catch {
      toast.error('Gagal memuat unit');
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    await fetchUoms();
  });

  function handleSearchInput() {
    debouncedSearchFetch();
  }

  const debouncedSearchFetch = debounce(() => {
    fetchUoms();
  }, 400);

  let showImportWizard = $state(false);

  function handleImportComplete() {
    fetchUoms();
    toast.success('UOM import completed');
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
      toast.error('Kode unit wajib diisi');
      return;
    }
    if (!form.name.trim()) {
      toast.error('Nama unit wajib diisi');
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
        toast.success(modalMode === 'add' ? 'Unit berhasil ditambahkan' : 'Unit berhasil diperbarui');
        showModal = false;
        await fetchUoms();
      } else {
        toast.error('Gagal menyimpan unit');
      }
    } catch {
      toast.error('Kesalahan jaringan');
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedUom) return;
    try {
      const ok = await deleteUnitOfMeasure(selectedUom.id);
      if (ok) {
        toast.success(`Unit "${selectedUom.name}" berhasil dihapus`);
        await fetchUoms();
      } else {
        toast.error('Gagal menghapus unit');
      }
    } catch {
      toast.error('Gagal menghapus unit');
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
        <SearchBar bind:value={searchQuery} placeholder="Search by name or code..." oninput={handleSearchInput} inputClass="h-10" />
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
            Tambah Unit
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
            <th class="text-left p-4 font-semibold" style="width: 15%;">CODE</th>
            <th class="text-left p-4 font-semibold" style="width: 30%;">UNIT NAME</th>
            <th class="text-left p-4 font-semibold w-48">DESCRIPTION</th>
            <th class="text-left p-4 font-semibold w-36">CREATED</th>
            <th class="text-center p-4 font-semibold w-20">ACTIONS</th>
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
    {:else if filteredUoms.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Ruler size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No units found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? `No results for "${searchQuery}"` : 'Start by adding your first unit of measure'}
        </p>
      </div>
    {:else}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 15%;">CODE</th>
            <th class="text-left p-4 font-semibold" style="width: 30%;">UNIT NAME</th>
            <th class="text-left p-4 font-semibold w-48">DESCRIPTION</th>
            <th class="text-left p-4 font-semibold w-36">CREATED</th>
            <th class="text-center p-4 font-semibold w-20">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredUoms as uom (uom.id)}
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
                      title="Edit"
                      aria-label="Edit"
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
                      title="Hapus"
                      aria-label="Hapus"
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
  </div>
</div>

<Modal bind:open={showModal} title={modalMode === 'add' ? 'Tambah Unit' : 'Edit Unit'} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveUom(); }} class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="uom-code" class="block text-sm font-medium text-text-secondary mb-2">Kode <span class="text-danger">*</span></label>
        <Input id="uom-code" type="text" placeholder="Contoh: pcs, kg, box" bind:value={form.code} required />
      </div>
      <div>
        <label for="uom-name" class="block text-sm font-medium text-text-secondary mb-2">Nama Unit <span class="text-danger">*</span></label>
        <Input id="uom-name" type="text" placeholder="Contoh: Pieces, Kilogram" bind:value={form.name} required />
      </div>
    </div>
    <div>
      <label for="uom-desc" class="block text-sm font-medium text-text-secondary mb-2">Deskripsi <span class="text-text-muted text-xs">(opsional)</span></label>
      <Input tag="textarea" id="uom-desc" placeholder="Deskripsi singkat unit…" class="min-h-[80px] resize-y" bind:value={form.description} />
    </div>
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-3">
        <label class="flex items-center gap-3 cursor-pointer select-none group">
          <div class="relative">
            <input type="checkbox" class="sr-only peer" bind:checked={form.is_active} />
            <div class="w-10 h-5 bg-surface-default border border-border rounded-full peer peer-checked:bg-primary-subtle peer-checked:border-primary/50 transition-colors"></div>
            <div class="absolute left-1 top-1 w-3 h-3 bg-text-muted rounded-full peer-checked:translate-x-5 peer-checked:bg-primary-light transition-transform shadow-sm"></div>
          </div>
          <span class="text-sm font-medium text-text-secondary group-hover:text-text-primary transition-colors">
            {form.is_active ? 'Aktif' : 'Tidak Aktif'}
          </span>
        </label>
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false} disabled={saving}>Batal</Button>
    <Button variant="primary" class="min-w-32" onclick={saveUom} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Menyimpan...
      {:else}
        {modalMode === 'add' ? 'Tambah Unit' : 'Simpan Perubahan'}
      {/if}
    </Button>
  {/snippet}
</Modal>

<ImportWizard
  bind:open={showImportWizard}
  module="uoms"
  displayName="Units of Measure"
  onComplete={handleImportComplete}
/>

<Modal bind:open={showDeleteModal} title="Hapus Unit" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Hapus "{selectedUom?.name}"?</p>
    <p class="text-text-muted text-sm">Unit akan dihapus secara permanen dan tidak dapat dikembalikan.</p>
  </div>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showDeleteModal = false}>Batal</Button>
    <Button variant="danger" onclick={confirmDelete}>Hapus</Button>
  {/snippet}
</Modal>
