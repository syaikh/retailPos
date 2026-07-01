<script>
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { debounce } from '$shared/utils/debounce';
  import { useAuthStore } from '$modules/auth';
  import { formatDateInJakarta } from '$shared/utils/jakartaTime';
  import { getBrands, createBrand, updateBrand, deleteBrand } from '$modules/settings/services/settings-service';

  const authStore = useAuthStore();

  import { Button, Input, Modal, Skeleton, BulkActionDropdown, ImportWizard, SearchBar } from '$shared/ui';
  import { Plus, Pencil, Trash2, Tag, Loader2 } from 'lucide-svelte';

  let loading = $state(true);
  let brands = $state([]);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedBrand = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);

  let form = $state({
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

  let filteredBrands = $derived(
    brands.filter(b =>
      !searchQuery || b.name.toLowerCase().includes(searchQuery.toLowerCase())
    )
  );

  async function fetchBrands() {
    try {
      loading = true;
      brands = await getBrands();
    } catch {
      toast.error('Gagal memuat brand');
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    await fetchBrands();
  });

  function handleSearchInput() {
    debouncedSearchFetch();
  }

  const debouncedSearchFetch = debounce(() => {
    fetchBrands();
  }, 400);

  let showImportWizard = $state(false);

  function openAdd() {
    modalMode = 'add';
    form = { name: '', description: '', is_active: true };
    showModal = true;
  }

  function openEdit(brand) {
    modalMode = 'edit';
    selectedBrand = brand;
    form = {
      name: brand.name,
      description: brand.description || '',
      is_active: brand.is_active !== false
    };
    showModal = true;
  }

  function openDelete(brand) {
    selectedBrand = brand;
    showDeleteModal = true;
  }

  async function saveBrand() {
    if (!form.name.trim()) {
      toast.error('Nama brand wajib diisi');
      return;
    }
    try {
      saving = true;
      let ok;
      if (modalMode === 'add') {
        ok = await createBrand({ name: form.name.trim(), description: form.description.trim() || undefined });
      } else {
        ok = await updateBrand(selectedBrand.id, {
          name: form.name.trim(),
          description: form.description.trim() || undefined,
          is_active: form.is_active
        });
      }
      if (ok) {
        toast.success(modalMode === 'add' ? 'Brand berhasil ditambahkan' : 'Brand berhasil diperbarui');
        showModal = false;
        await fetchBrands();
      } else {
        toast.error('Gagal menyimpan brand');
      }
    } catch {
      toast.error('Kesalahan jaringan');
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedBrand) return;
    try {
      const ok = await deleteBrand(selectedBrand.id);
      if (ok) {
        toast.success(`Brand "${selectedBrand.name}" berhasil dihapus`);
        await fetchBrands();
      } else {
        toast.error('Gagal menghapus brand');
      }
    } catch {
      toast.error('Gagal menghapus brand');
    } finally {
      showDeleteModal = false;
      selectedBrand = null;
    }
  }

  function handleImportComplete() {
    fetchBrands();
    toast.success('Brand import completed');
  }
</script>

<div class="space-y-5">
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="flex-2">
        <SearchBar bind:value={searchQuery} placeholder="Search by name..." oninput={handleSearchInput} inputClass="h-10" />
      </div>
      {#if canCreate}
        <div class="flex items-center gap-2">
          <BulkActionDropdown
            module="brands"
            canExport={true}
            canImport={true}
            onImport={() => showImportWizard = true}
          />
          <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={openAdd}>
            <Plus size={18} />
            Tambah Brand
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
            <th class="text-left p-4 font-semibold" style="width: 40%;">BRAND NAME</th>
            <th class="text-left p-4 font-semibold w-48">DESCRIPTION</th>
            <th class="text-left p-4 font-semibold w-36">CREATED</th>
            <th class="text-center p-4 font-semibold w-20">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each Array(5) as _}
            <tr class="border-t border-border">
              <td class="p-4 min-w-0"><Skeleton class="h-4 w-full" /></td>
              <td class="p-4 w-48"><Skeleton class="h-4 w-3/4" /></td>
              <td class="p-4 w-36"><Skeleton class="h-4 w-2/3" /></td>
              <td class="p-4 w-20"><Skeleton class="h-4 w-8" /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if filteredBrands.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Tag size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No brands found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? `No results for "${searchQuery}"` : 'Start by adding your first brand'}
        </p>
      </div>
    {:else}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 40%;">BRAND NAME</th>
            <th class="text-left p-4 font-semibold w-48">DESCRIPTION</th>
            <th class="text-left p-4 font-semibold w-36">CREATED</th>
            <th class="text-center p-4 font-semibold w-20">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredBrands as brand (brand.id)}
            <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
              <td class="p-4 pr-6" style="width: 40%;">
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-full bg-primary-subtle flex items-center justify-center shrink-0">
                    <Tag size={14} class="text-primary-light" />
                  </div>
                  <div class="min-w-0">
                    <p class="font-medium truncate" title={brand.name}>{brand.name}</p>
                  </div>
                </div>
              </td>
              <td class="p-4 w-40 text-text-secondary text-sm">
                {brand.description || '—'}
              </td>
              <td class="p-4 w-36 text-text-secondary text-sm">
                {formatDate(brand.created_at)}
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
                      onclick={() => openEdit(brand)}
                    >
                      <Pencil size={14} />
                    </Button>
                  {/if}
                  {#if canDelete}
                    <Button
                      variant="ghost"
                      size="icon"
                      class="text-text-muted hover:text-danger hover:bg-danger-subtle"
                      onclick={() => openDelete(brand)}
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

<Modal bind:open={showModal} title={modalMode === 'add' ? 'Tambah Brand' : 'Edit Brand'} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveBrand(); }} class="space-y-4">
    <div>
      <label for="brand-name" class="block text-sm font-medium text-text-secondary mb-2">Nama Brand <span class="text-danger">*</span></label>
      <Input id="brand-name" type="text" placeholder="Contoh: Indofood" bind:value={form.name} required />
    </div>
    <div>
      <label for="brand-desc" class="block text-sm font-medium text-text-secondary mb-2">Deskripsi <span class="text-text-muted text-xs">(opsional)</span></label>
      <Input tag="textarea" id="brand-desc" placeholder="Deskripsi singkat brand…" class="min-h-[80px] resize-y" bind:value={form.description} />
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
    <Button variant="primary" class="min-w-32" onclick={saveBrand} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Menyimpan...
      {:else}
        {modalMode === 'add' ? 'Tambah Brand' : 'Simpan Perubahan'}
      {/if}
    </Button>
  {/snippet}
</Modal>

<ImportWizard
  bind:open={showImportWizard}
  module="brands"
  displayName="Brands"
  onComplete={handleImportComplete}
/>

<Modal bind:open={showDeleteModal} title="Hapus Brand" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Hapus "{selectedBrand?.name}"?</p>
    <p class="text-text-muted text-sm">Brand akan dihapus secara permanen dan tidak dapat dikembalikan.</p>
  </div>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showDeleteModal = false}>Batal</Button>
    <Button variant="danger" onclick={confirmDelete}>Hapus</Button>
  {/snippet}
</Modal>
