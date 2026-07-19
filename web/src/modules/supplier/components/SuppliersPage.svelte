<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { useAuthStore } from '$modules/auth';
  import { goto } from '$app/router';
  import { getSuppliers, createSupplier, updateSupplier, deleteSupplier, bulkUpdateSuppliers, bulkDeleteSuppliers } from '../services/supplier-service';
  import type { Supplier } from '../types';
  import { Pagination } from '$shared/ui';
  import { debounce } from '$shared/utils/debounce';
  import SuppliersToolbar from './SuppliersToolbar.svelte';
  import SuppliersTable from './SuppliersTable.svelte';
  import SupplierFormModal from './SupplierFormModal.svelte';
  import SupplierDetailDrawer from './SupplierDetailDrawer.svelte';
  import ConfirmDeleteModal from '$shared/ui/ConfirmDeleteModal.svelte';
  import ImportWizard from '$shared/ui/ImportWizard.svelte';

  const authStore = useAuthStore();

  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('pricing:create'));
  const canUpdate = $derived(userPermissions.includes('pricing:update'));
  const canDelete = $derived(userPermissions.includes('pricing:delete'));
  const canExport = $derived(userPermissions.includes('pricing:read'));
  const canImport = $derived(userPermissions.includes('pricing:create'));

  let loading = $state(true);
  let suppliers = $state<Supplier[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let statusFilter = $state('all');
  let sortBy = $state('name');
  let sortDir = $state<'asc' | 'desc'>('asc');

  let showFormModal = $state(false);
  let formMode = $state<'add' | 'edit'>('add');
  let selectedSupplier = $state<Supplier | null>(null);
  let saving = $state(false);

  let showDeleteModal = $state(false);
  let deleteTargetName = $state('');
  let deleting = $state(false);

  let showImportWizard = $state(false);
  let showDetailDrawer = $state(false);
  let detailSupplier = $state<Supplier | null>(null);

  async function load() {
    loading = true;
    try {
      const params: any = { limit, offset, search: searchQuery, sort_by: sortBy, sort_dir: sortDir };
      if (statusFilter === 'active') params.is_active = true;
      else if (statusFilter === 'inactive') params.is_active = false;

      const result = await getSuppliers(params);
      suppliers = result.data;
      total = result.total;
    } catch {
      toast.error('Failed to load suppliers');
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
  function handleSort(col: string) {
    if (sortBy === col) { sortDir = sortDir === 'asc' ? 'desc' : 'asc'; }
    else { sortBy = col; sortDir = 'asc'; }
    load();
  }

  function openAdd() {
    formMode = 'add';
    selectedSupplier = null;
    showFormModal = true;
  }

  function openEdit(supplier: Supplier) {
    formMode = 'edit';
    selectedSupplier = supplier;
    showFormModal = true;
  }

  async function handleFormSave(data: any) {
    saving = true;
    try {
      if (formMode === 'add') {
        const ok = await createSupplier(data);
        if (ok) {
          toast.success('Supplier created');
          showFormModal = false;
          await load();
        } else {
          toast.error('Failed to create supplier');
        }
      } else {
        const ok = await updateSupplier(selectedSupplier!.id, data);
        if (ok) {
          toast.success('Supplier updated');
          showFormModal = false;
          await load();
        } else {
          toast.error('Failed to update supplier');
        }
      }
    } catch {
      toast.error('Failed to save supplier');
    } finally {
      saving = false;
    }
  }

  function openDelete(supplier: Supplier) {
    selectedSupplier = supplier;
    deleteTargetName = supplier.name;
    showDeleteModal = true;
  }

  async function handleDeleteConfirm() {
    if (!selectedSupplier) return;
    deleting = true;
    try {
      const ok = await deleteSupplier(selectedSupplier.id);
      if (ok) {
        toast.success('Supplier deleted');
        showDeleteModal = false;
        selectedSupplier = null;
        await load();
      } else {
        toast.error('Failed to delete supplier');
      }
    } catch {
      toast.error('Failed to delete supplier');
    } finally {
      deleting = false;
    }
  }

  async function handleBulkActivate(ids: number[]) {
    try {
      const updated = await bulkUpdateSuppliers(ids, true);
      toast.success(`${updated} suppliers activated`);
      await load();
    } catch {
      toast.error('Failed to activate suppliers');
    }
  }

  async function handleBulkDeactivate(ids: number[]) {
    try {
      const updated = await bulkUpdateSuppliers(ids, false);
      toast.success(`${updated} suppliers deactivated`);
      await load();
    } catch {
      toast.error('Failed to deactivate suppliers');
    }
  }

  async function handleBulkDelete(ids: number[]) {
    try {
      const deleted = await bulkDeleteSuppliers(ids);
      toast.success(`${deleted} suppliers deleted`);
      await load();
    } catch {
      toast.error('Failed to delete suppliers');
    }
  }

  function handleImport() {
    showImportWizard = true;
  }

  function openDetail(supplier: Supplier) {
    detailSupplier = supplier;
    showDetailDrawer = true;
  }

  function duplicateSupplier(supplier: Supplier) {
    selectedSupplier = { ...supplier, name: `${supplier.name} (Copy)`, id: 0 } as Supplier;
    formMode = 'add';
    showFormModal = true;
  }

  function viewSupplierProducts(supplier: Supplier) {
    const params = new URLSearchParams({ supplier_id: supplier.id.toString(), supplier_name: supplier.name });
    goto(`/inventory/products?${params.toString()}`);
  }

  onMount(() => { load(); });
</script>

<div class="space-y-5">
  <SuppliersToolbar
    bind:searchQuery
    bind:statusFilter
    {canCreate}
    {canExport}
    {canImport}
    onsearch={handleSearch}
    onstatuschange={handleStatusChange}
    oncreate={openAdd}
    onimport={handleImport}
  />

  <div class="card overflow-x-auto">
    <SuppliersTable
      {suppliers}
      {loading}
      {searchQuery}
      canEdit={canUpdate}
      {canDelete}
      {canCreate}
      {sortBy}
      {sortDir}
      onsort={handleSort}
      onedit={openEdit}
      ondelete={openDelete}
      onduplicate={duplicateSupplier}
      onviewproducts={viewSupplierProducts}
      onrowclick={openDetail}
      onbulkactivate={handleBulkActivate}
      onbulkdeactivate={handleBulkDeactivate}
      onbulkdelete={handleBulkDelete}
    />

    {#if !loading && suppliers.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
      </div>
    {/if}
  </div>
</div>

<SupplierFormModal
  bind:open={showFormModal}
  mode={formMode}
  supplier={selectedSupplier}
  {saving}
  onsave={handleFormSave}
  oncancel={() => { showFormModal = false; selectedSupplier = null; }}
/>

<ConfirmDeleteModal
  bind:open={showDeleteModal}
  onconfirm={handleDeleteConfirm}
  loading={deleting}
  itemName={deleteTargetName}
/>

<ImportWizard bind:open={showImportWizard} module="suppliers" displayName="Suppliers" onComplete={() => load()} />

<SupplierDetailDrawer
  bind:open={showDetailDrawer}
  supplier={detailSupplier}
  canEdit={canUpdate}
  {canDelete}
  onclose={() => { showDetailDrawer = false; detailSupplier = null; }}
  onedit={(s) => { showDetailDrawer = false; openEdit(s); }}
  ondelete={(s) => { showDetailDrawer = false; openDelete(s); }}
  onviewproducts={(s) => { showDetailDrawer = false; viewSupplierProducts(s); }}
/>