<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/router';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { getCustomerGroups, createCustomerGroup, updateCustomerGroup, deleteCustomerGroup, bulkUpdateCustomerGroups, bulkDeleteCustomerGroups } from '../services/customer-group-service';
  import type { CustomerGroup } from '../types';
  import { Pagination } from '$shared/ui';
  import { debounce } from '$shared/utils/debounce';
  import CustomerGroupsToolbar from './CustomerGroupsToolbar.svelte';
  import CustomerGroupsTable from './CustomerGroupsTable.svelte';
  import CreateCustomerGroupModal from './CreateCustomerGroupModal.svelte';
  import EditCustomerGroupModal from './EditCustomerGroupModal.svelte';
  import DeleteCustomerGroupModal from './DeleteCustomerGroupModal.svelte';
  import CustomerGroupDetailDrawer from './CustomerGroupDetailDrawer.svelte';
  import ImportWizard from '$shared/ui/ImportWizard.svelte';

  const authStore = useAuthStore();

  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('customer_group:create'));
  const canUpdate = $derived(userPermissions.includes('customer_group:update'));
  const canDelete = $derived(userPermissions.includes('customer_group:delete'));

  let loading = $state(true);
  let groups = $state<CustomerGroup[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let statusFilter = $state('all');
  let hasCustomersFilter = $state('all');
  let sortBy = $state('name');
  let sortDir = $state<'asc' | 'desc'>('asc');

  let showCreateModal = $state(false);
  let creating = $state(false);
  let showEditModal = $state(false);
  let selectedGroup = $state<CustomerGroup | null>(null);
  let saving = $state(false);
  let showDeleteModal = $state(false);
  let deleteTargetName = $state('');
  let deleting = $state(false);
  let showImportWizard = $state(false);
  let showDetailDrawer = $state(false);
  let detailGroup = $state<CustomerGroup | null>(null);

  const stats = $derived.by(() => {
    const activeCount = groups.filter(g => g.is_active).length;
    const inactiveCount = groups.filter(g => !g.is_active).length;
    const customerCount = groups.reduce((sum, g) => sum + (g.customer_count || 0), 0);
    return { total, activeCount, inactiveCount, customerCount };
  });

  async function load() {
    loading = true;
    try {
      const filters: any = { limit, offset };
      if (searchQuery.trim()) filters.search = searchQuery.trim();
      if (statusFilter !== 'all') filters.is_active = statusFilter === 'active';
      if (hasCustomersFilter !== 'all') filters.has_customers = hasCustomersFilter === 'yes';

      const result = await getCustomerGroups(filters);

      if (sortBy === 'name') {
        result.data.sort((a, b) => sortDir === 'asc' ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name));
      } else if (sortBy === 'status') {
        result.data.sort((a, b) => sortDir === 'asc' ? (a.is_active ? 1 : 0) - (b.is_active ? 1 : 0) : (b.is_active ? 1 : 0) - (a.is_active ? 1 : 0));
      } else if (sortBy === 'created_at') {
        result.data.sort((a, b) => sortDir === 'asc' ? (a.created_at || '').localeCompare(b.created_at || '') : (b.created_at || '').localeCompare(a.created_at || ''));
      } else if (sortBy === 'customer_count') {
        result.data.sort((a, b) => sortDir === 'asc' ? (a.customer_count || 0) - (b.customer_count || 0) : (b.customer_count || 0) - (a.customer_count || 0));
      } else if (sortBy === 'updated_at') {
        result.data.sort((a, b) => sortDir === 'asc' ? (a.updated_at || '').localeCompare(b.updated_at || '') : (b.updated_at || '').localeCompare(a.updated_at || ''));
      }

      groups = result.data;
      total = result.total;
    } catch (e) {
      toast.error('Gagal memuat customer groups');
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

  async function handleCreate(data: { name: string; description?: string }) {
    creating = true;
    try {
      await createCustomerGroup(data);
      toast.success('Customer group berhasil dibuat');
      showCreateModal = false;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Gagal membuat group');
    } finally {
      creating = false;
    }
  }

  function viewMembers(g: CustomerGroup) {
    sessionStorage.setItem('customerGroupFilter', String(g.id));
    goto('/customers');
  }

  function duplicateGroup(g: CustomerGroup) {
    selectedGroup = { ...g, name: `${g.name} (Salinan)`, id: 0 } as CustomerGroup;
    showCreateModal = true;
  }

  function openEdit(g: CustomerGroup) {
    selectedGroup = g;
    showEditModal = true;
  }

  async function handleEditSave(data: { id: number; name: string; description?: string; is_active: boolean }) {
    saving = true;
    try {
      await updateCustomerGroup(data.id, data);
      toast.success('Customer group berhasil diupdate');
      showEditModal = false;
      selectedGroup = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Gagal update group');
    } finally {
      saving = false;
    }
  }

  function openDelete(g: CustomerGroup) {
    selectedGroup = g;
    deleteTargetName = g.name;
    showDeleteModal = true;
  }

  async function handleDeleteConfirm() {
    if (!selectedGroup) return;
    deleting = true;
    try {
      await deleteCustomerGroup(selectedGroup.id);
      toast.success('Customer group berhasil dihapus');
      showDeleteModal = false;
      selectedGroup = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Gagal menghapus group');
    } finally {
      deleting = false;
    }
  }

  async function handleBulkActivate(ids: number[]) {
    try {
      const updated = await bulkUpdateCustomerGroups(ids, true);
      toast.success(`${updated} group berhasil diaktifkan`);
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Gagal mengaktifkan group');
    }
  }

  async function handleBulkDeactivate(ids: number[]) {
    try {
      const updated = await bulkUpdateCustomerGroups(ids, false);
      toast.success(`${updated} group berhasil dinonaktifkan`);
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Gagal menonaktifkan group');
    }
  }

  async function handleBulkDelete(ids: number[]) {
    try {
      const deleted = await bulkDeleteCustomerGroups(ids);
      toast.success(`${deleted} group berhasil dihapus`);
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Gagal menghapus group');
    }
  }

  function handleImport() {
    showImportWizard = true;
  }

  function openDetail(g: CustomerGroup) {
    detailGroup = g;
    showDetailDrawer = true;
  }

  onMount(() => { load(); });
</script>

<div class="space-y-5">
  <div>
    <h1 class="text-xl font-bold text-text-primary">Customer Groups</h1>
    <p class="text-sm text-text-muted mt-1">Kelompokkan customer untuk segmentasi dan pricing</p>
  </div>

  <CustomerGroupsToolbar
    bind:searchQuery
    bind:statusFilter
    bind:hasCustomersFilter
    {canCreate}
    {stats}
    onsearch={handleSearch}
    onstatuschange={handleStatusChange}
    oncreate={() => showCreateModal = true}
    onimport={handleImport}
  />

  <div class="card overflow-x-auto">
    <CustomerGroupsTable
      {groups}
      {loading}
      {searchQuery}
      {canUpdate}
      {canDelete}
      {canCreate}
      bind:sortBy
      bind:sortDir
      onsort={handleSort}
      onviewmembers={viewMembers}
      onduplicate={duplicateGroup}
      onedit={openEdit}
      ondelete={openDelete}
      onbulkactivate={handleBulkActivate}
      onbulkdeactivate={handleBulkDeactivate}
      onbulkdelete={handleBulkDelete}
      onrowclick={openDetail}
    />

    {#if !loading && groups.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
      </div>
    {/if}
  </div>
</div>

<CreateCustomerGroupModal bind:open={showCreateModal} bind:creating oncreate={handleCreate} />
<EditCustomerGroupModal bind:open={showEditModal} bind:group={selectedGroup} bind:saving onsave={handleEditSave} oncancel={() => { showEditModal = false; selectedGroup = null; }} />
<DeleteCustomerGroupModal bind:open={showDeleteModal} bind:deleting targetName={deleteTargetName} oncancel={() => { showDeleteModal = false; selectedGroup = null; }} onconfirm={handleDeleteConfirm} />
<ImportWizard bind:open={showImportWizard} module="customer_groups" displayName="Customer Groups" onComplete={() => load()} />
<CustomerGroupDetailDrawer bind:open={showDetailDrawer} group={detailGroup} onclose={() => { showDetailDrawer = false; detailGroup = null; }} />
