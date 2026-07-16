<script>
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte.ts';
  import { useAuthStore } from '$modules/auth';
  import { getSuppliers, createSupplier, updateSupplier, deleteSupplier } from '../services/supplier-service.ts';
  import { Button, Input, Modal, Skeleton, SearchBar, Pagination, ConfirmDeleteModal } from '$shared/ui';
  import { Plus, Pencil, Trash2, Truck, Loader2 } from 'lucide-svelte';

  const authStore = useAuthStore();

  let loading = $state(true);
  let suppliers = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedSupplier = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);

  let form = $state({
    name: '',
    code: '',
    contact_name: '',
    phone: '',
    email: '',
    address: '',
    notes: '',
    is_active: true
  });

  let canCreate = $derived((authStore.user?.permissions || []).includes('pricing:create'));
  let canEdit = $derived((authStore.user?.permissions || []).includes('pricing:update'));
  let canDelete = $derived((authStore.user?.permissions || []).includes('pricing:delete'));

  async function fetchSuppliers() {
    loading = true;
    const result = await getSuppliers({ limit, offset, search: searchQuery });
    suppliers = result.data;
    total = result.total;
    loading = false;
  }

  function openAdd() {
    modalMode = 'add';
    form = { name: '', code: '', contact_name: '', phone: '', email: '', address: '', notes: '', is_active: true };
    showModal = true;
  }

  function openEdit(supplier) {
    modalMode = 'edit';
    selectedSupplier = supplier;
    form = {
      name: supplier.name,
      code: supplier.code,
      contact_name: supplier.contact_name || '',
      phone: supplier.phone || '',
      email: supplier.email || '',
      address: supplier.address || '',
      notes: supplier.notes || '',
      is_active: supplier.is_active
    };
    showModal = true;
  }

  function openDelete(supplier) {
    selectedSupplier = supplier;
    showDeleteModal = true;
  }

  async function saveSupplier(e) {
    e.preventDefault();
    if (!form.name || !form.code) {
      toast.error('Name and Code are required');
      return;
    }
    saving = true;
    let ok;
    if (modalMode === 'add') {
      ok = await createSupplier(form);
    } else {
      ok = await updateSupplier(selectedSupplier.id, form);
    }
    saving = false;

    if (ok) {
      toast.success(modalMode === 'add' ? 'Supplier created' : 'Supplier updated');
      showModal = false;
      fetchSuppliers();
    } else {
      toast.error('Failed to save supplier');
    }
  }

  async function confirmDelete() {
    if (!selectedSupplier) return;
    const ok = await deleteSupplier(selectedSupplier.id);
    if (ok) {
      toast.success('Supplier deleted');
      showDeleteModal = false;
      fetchSuppliers();
    } else {
      toast.error('Failed to delete supplier');
    }
  }

  function handlePageChange(e) {
    offset = e.detail.offset;
    fetchSuppliers();
  }

  let searchTimeout;
  function handleSearch() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => { offset = 0; fetchSuppliers(); }, 300);
  }

  onMount(() => fetchSuppliers());
</script>

<div class="space-y-5">
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="flex-2">
        <SearchBar bind:value={searchQuery} placeholder="Search suppliers..." oninput={handleSearch} inputClass="h-10" />
      </div>
      {#if canCreate}
        <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={openAdd}>
          <Plus size={18} /> Add Supplier
        </Button>
      {/if}
    </div>
  </div>

  <div class="card">
    {#if loading}
      <table class="w-full">
        <thead><tr><th>Name</th><th>Code</th><th>Contact</th><th>Phone</th><th>Status</th></tr></thead>
        <tbody>{#each Array(5) as _}<tr>{#each Array(5) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
      </table>
    {:else if suppliers.length === 0}
      <div class="flex flex-col items-center justify-center py-12 text-gray-400">
        <Truck class="w-12 h-12 mb-3" />
        <p>No suppliers found</p>
      </div>
    {:else}
      <table class="w-full">
        <thead>
          <tr class="border-b text-left text-sm text-gray-500">
            <th class="px-4 py-3">Name</th>
            <th class="px-4 py-3">Code</th>
            <th class="px-4 py-3">Contact</th>
            <th class="px-4 py-3">Phone</th>
            <th class="px-4 py-3">Email</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each suppliers as supplier (supplier.id)}
            <tr class="border-b hover:bg-gray-50">
              <td class="px-4 py-3 font-medium">{supplier.name}</td>
              <td class="px-4 py-3"><code class="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{supplier.code}</code></td>
              <td class="px-4 py-3">{supplier.contact_name || '-'}</td>
              <td class="px-4 py-3">{supplier.phone || '-'}</td>
              <td class="px-4 py-3">{supplier.email || '-'}</td>
              <td class="px-4 py-3">
                {#if supplier.is_active}
                  <span class="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Active</span>
                {:else}
                  <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600">Inactive</span>
                {/if}
              </td>
              <td class="px-4 py-3 text-right">
                {#if canEdit}
                  <Button variant="ghost" size="icon" onclick={() => openEdit(supplier)}><Pencil class="w-4 h-4" /></Button>
                {/if}
                {#if canDelete}
                  <Button variant="ghost" size="icon" onclick={() => openDelete(supplier)}><Trash2 class="w-4 h-4 text-red-500" /></Button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  {#if total > limit}
    <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
  {/if}
</div>

<Modal bind:open={showModal} title={modalMode === 'add' ? 'Add Supplier' : 'Edit Supplier'} size="md">
  <form onsubmit={saveSupplier} class="space-y-4">
    <Input label="Name" bind:value={form.name} required />
    <Input label="Code" bind:value={form.code} required />
    <Input label="Contact Name" bind:value={form.contact_name} />
    <Input label="Phone" bind:value={form.phone} />
    <Input label="Email" bind:value={form.email} type="email" />
    <Input label="Address" bind:value={form.address} tag="textarea" />
    <Input label="Notes" bind:value={form.notes} tag="textarea" />
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-2">
        <input type="checkbox" bind:checked={form.is_active} id="is_active" class="rounded" />
        <label for="is_active" class="text-sm">Active</label>
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false}>Cancel</Button>
    <Button variant="primary" onclick={saveSupplier} disabled={saving}>
      {#if saving}<Loader2 class="w-4 h-4 mr-2 animate-spin" />{/if}
      {modalMode === 'add' ? 'Create' : 'Update'}
    </Button>
  {/snippet}
</Modal>

<ConfirmDeleteModal bind:open={showDeleteModal} onconfirm={confirmDelete} />
