<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { useAuthStore } from '$modules/auth';
  import { getSuppliers, createSupplier, updateSupplier, deleteSupplier } from '../services/supplier-service';
  import type { Supplier } from '../types';
  import { debounce } from '$shared/utils/debounce';
  import { Button, Input, Modal, Skeleton, SearchBar, Pagination, ConfirmDeleteModal, SortableHeader } from '$shared/ui';
  import { Plus, Pencil, Trash2, Truck, Loader2 } from 'lucide-svelte';

  const authStore = useAuthStore();

  let loading = $state(true);
  let suppliers = $state<Supplier[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedSupplier = $state<Supplier | null>(null);
  let modalMode = $state<'add' | 'edit'>('add');
  let saving = $state(false);
  let sortBy = $state('name');
  let sortDir = $state<'asc' | 'desc'>('asc');
  let statusFilter = $state<'all' | 'active' | 'inactive'>('all');

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
    const params: any = { limit, offset, search: searchQuery, sort_by: sortBy, sort_dir: sortDir };
    if (statusFilter === 'active') params.is_active = true;
    else if (statusFilter === 'inactive') params.is_active = false;
    const result = await getSuppliers(params);
    suppliers = result.data;
    total = result.total;
    loading = false;
  }

  function handleSort(col: string) {
    if (sortBy === col) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = col;
      sortDir = 'asc';
    }
    fetchSuppliers();
  }

  function openAdd() {
    modalMode = 'add';
    form = { name: '', code: '', contact_name: '', phone: '', email: '', address: '', notes: '', is_active: true };
    showModal = true;
  }

  function openEdit(supplier: Supplier) {
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

  function openDelete(supplier: Supplier) {
    selectedSupplier = supplier;
    showDeleteModal = true;
  }

  async function saveSupplier(e: Event) {
    e.preventDefault();
    if (!form.name || !form.code) {
      toast.error('Name and Code are required');
      return;
    }
    saving = true;
    let ok: boolean;
    if (modalMode === 'add') {
      ok = await createSupplier(form);
    } else {
      ok = await updateSupplier(selectedSupplier!.id, form);
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

  function handlePageChange(newOffset: number, newLimit: number) {
    offset = newOffset;
    limit = newLimit;
    fetchSuppliers();
  }

  const debouncedSearch = debounce(() => { offset = 0; fetchSuppliers(); }, 300);
  function handleSearch() { debouncedSearch(); }

  function handleFilterChange() {
    offset = 0;
    fetchSuppliers();
  }

  onMount(() => fetchSuppliers());
</script>

<div class="space-y-5">
  <div class="card p-4">
    <div class="flex items-center gap-3">
      <div class="flex-1">
        <SearchBar bind:value={searchQuery} placeholder="Search suppliers by name or contact..." oninput={handleSearch} inputClass="h-10" />
      </div>
      <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default" role="group" aria-label="Status filter">
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'all'; handleFilterChange(); }}
          aria-pressed={statusFilter === 'all'}
        >All</button>
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'active'; handleFilterChange(); }}
          aria-pressed={statusFilter === 'active'}
        >Active</button>
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'inactive'; handleFilterChange(); }}
          aria-pressed={statusFilter === 'inactive'}
        >Inactive</button>
      </div>
      {#if canCreate}
        <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={openAdd}>
          <Plus size={18} /> Add Supplier
        </Button>
      {/if}
    </div>
  </div>

  <div class="card overflow-hidden">
    {#if loading}
      <div class="overflow-x-auto">
      <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label="Loading suppliers">
        <colgroup>
          <col style="width: 25%;" />
          <col style="width: 20%;" />
          <col style="width: 18%;" />
          <col style="width: 22%;" />
          <col style="width: 10%;" />
          <col style="width: 5%;" />
        </colgroup>
        <thead><tr><th>Name</th><th>Contact</th><th>Phone</th><th>Email</th><th>Status</th><th></th></tr></thead>
        <tbody>{#each Array(5) as _}<tr>{#each Array(6) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
      </table>
      </div>
    {:else if suppliers.length === 0}
      <div class="flex flex-col items-center justify-center py-12 text-gray-400" role="status">
        <Truck class="w-12 h-12 mb-3" aria-hidden="true" />
        <p>No suppliers found</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
      <table class="w-full min-w-[700px]" style="table-layout: fixed;" role="grid" aria-label="Suppliers">
        <colgroup>
          <col style="width: 25%;" />
          <col style="width: 20%;" />
          <col style="width: 18%;" />
          <col style="width: 22%;" />
          <col style="width: 10%;" />
          <col style="width: 5%;" />
        </colgroup>
        <thead class="bg-muted/50">
          <tr class="border-b text-left text-sm text-text-muted">
            <th class="px-4 py-3 font-semibold" scope="col">
              <SortableHeader label="NAME" column="name" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold" scope="col">
              <SortableHeader label="CONTACT" column="contact_name" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold" scope="col">
              <SortableHeader label="PHONE" column="phone" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold" scope="col">
              <SortableHeader label="EMAIL" column="email" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold" scope="col">
              <SortableHeader label="STATUS" column="is_active" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold text-right" scope="col">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each suppliers as supplier (supplier.id)}
            <tr class="border-b border-border hover:bg-surface-hover/50 transition-colors">
              <td class="px-4 py-3 font-medium truncate">{supplier.name}</td>
              <td class="px-4 py-3 text-text-secondary truncate">{supplier.contact_name || '-'}</td>
              <td class="px-4 py-3 text-text-secondary tabular-nums">{supplier.phone || '-'}</td>
              <td class="px-4 py-3 text-text-secondary text-sm truncate">{supplier.email || '-'}</td>
              <td class="px-4 py-3">
                {#if supplier.is_active}
                  <span class="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Active</span>
                {:else}
                  <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600">Inactive</span>
                {/if}
              </td>
              <td class="px-4 py-3 text-right">
                <div class="flex items-center justify-end gap-1" role="group" aria-label="Actions for {supplier.name}">
                  {#if canEdit}
                    <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light" onclick={() => openEdit(supplier)} aria-label="Edit {supplier.name}"><Pencil class="w-4 h-4" /></Button>
                  {/if}
                  {#if canDelete}
                    <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger hover:bg-danger-subtle" onclick={() => openDelete(supplier)} aria-label="Delete {supplier.name}"><Trash2 class="w-4 h-4" /></Button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
      </div>

      {#if !loading && suppliers.length > 0}
        <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
          <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
        </div>
      {/if}
    {/if}
  </div>
</div>

<Modal bind:open={showModal} title={modalMode === 'add' ? 'Add Supplier' : 'Edit Supplier'} size="md">
  <form onsubmit={saveSupplier} class="space-y-3">
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="sup_name" class="block text-xs font-medium text-text-secondary mb-1">Supplier Name <span class="text-danger">*</span></label>
        <Input id="sup_name" bind:value={form.name} required placeholder="PT Sumber Makmur" class="h-9 text-sm" />
      </div>
      <div>
        <label for="sup_code" class="block text-xs font-medium text-text-secondary mb-1">Supplier Code <span class="text-danger">*</span></label>
        <Input id="sup_code" bind:value={form.code} required placeholder="SUP-001" class="h-9 text-sm" />
      </div>
    </div>
    <div class="grid grid-cols-3 gap-3">
      <div>
        <label for="sup_contact" class="block text-xs font-medium text-text-secondary mb-1">Contact Person</label>
        <Input id="sup_contact" bind:value={form.contact_name} placeholder="Budi Santoso" class="h-9 text-sm" />
      </div>
      <div>
        <label for="sup_phone" class="block text-xs font-medium text-text-secondary mb-1">Phone</label>
        <Input id="sup_phone" bind:value={form.phone} placeholder="021-12345678" class="h-9 text-sm" />
      </div>
      <div>
        <label for="sup_email" class="block text-xs font-medium text-text-secondary mb-1">Email</label>
        <Input id="sup_email" type="email" bind:value={form.email} placeholder="info@supplier.co.id" class="h-9 text-sm" />
      </div>
    </div>
    <div>
      <label for="sup_address" class="block text-xs font-medium text-text-secondary mb-1">Address</label>
      <Input id="sup_address" tag="textarea" bind:value={form.address} placeholder="Full address..." class="min-h-[40px] resize-y text-sm" />
    </div>
    <div>
      <label for="sup_notes" class="block text-xs font-medium text-text-secondary mb-1">Notes</label>
      <Input id="sup_notes" tag="textarea" bind:value={form.notes} placeholder="Additional notes..." class="min-h-[40px] resize-y text-sm" />
    </div>
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-2">
        <input type="checkbox" bind:checked={form.is_active} id="is_active" class="rounded" />
        <label for="is_active" class="text-sm text-text-secondary">Active</label>
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false} disabled={saving}>Cancel</Button>
    <Button variant="primary" class="min-w-32" onclick={saveSupplier} disabled={saving}>
      {#if saving}<Loader2 class="w-4 h-4 mr-2 animate-spin" />{/if}
      {modalMode === 'add' ? 'Create' : 'Update'}
    </Button>
  {/snippet}
</Modal>

<ConfirmDeleteModal bind:open={showDeleteModal} onconfirm={confirmDelete} />
