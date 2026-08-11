<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/router';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { labels, t } from '$shared/i18n';
  import { useSortable } from '$shared/composables/useSortable.svelte';
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
  const canCreate = $derived(userPermissions.includes('customer_group.create'));
  const canUpdate = $derived(userPermissions.includes('customer_group.update'));
  const canDelete = $derived(userPermissions.includes('customer_group.delete'));

  let loading = $state(true);
  let groups = $state<CustomerGroup[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let statusFilter = $state('all');
  let hasCustomersFilter = $state('all');
  const { sortState, handleSort } = useSortable('name', 'asc', load);

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

  async function load() {
    loading = true;
    try {
      const filters: any = { limit, offset };
      if (searchQuery.trim()) filters.search = searchQuery.trim();
      if (statusFilter !== 'all') filters.is_active = statusFilter === 'active';
      if (hasCustomersFilter !== 'all') filters.has_customers = hasCustomersFilter === 'yes';

      const result = await getCustomerGroups(filters);

      if (sortState.sortBy === 'name') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name));
      } else if (sortState.sortBy === 'status') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? (a.is_active ? 1 : 0) - (b.is_active ? 1 : 0) : (b.is_active ? 1 : 0) - (a.is_active ? 1 : 0));
      } else if (sortState.sortBy === 'created_at') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? (a.created_at || '').localeCompare(b.created_at || '') : (b.created_at || '').localeCompare(a.created_at || ''));
      } else if (sortState.sortBy === 'customer_count') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? (a.customer_count || 0) - (b.customer_count || 0) : (b.customer_count || 0) - (a.customer_count || 0));
      } else if (sortState.sortBy === 'updated_at') {
        result.data.sort((a, b) => sortState.sortDir === 'asc' ? (a.updated_at || '').localeCompare(b.updated_at || '') : (b.updated_at || '').localeCompare(a.updated_at || ''));
      }

      groups = result.data;
      total = result.total;
    } catch (e) {
      toast.error(labels.toastFailedLoadCustomerGroups);
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

  async function handleCreate(data: { name: string; description?: string }) {
    creating = true;
    try {
      await createCustomerGroup(data);
      toast.success(labels.toastCustomerGroupCreated);
      showCreateModal = false;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedCreateCustomerGroup);
    } finally {
      creating = false;
    }
  }

  function viewMembers(g: CustomerGroup) {
    sessionStorage.setItem('customerGroupFilter', String(g.id));
    sessionStorage.setItem('customerGroupsBackPage', JSON.stringify({ offset, limit, groupName: g.name }));
    goto('/customers');
  }

  function duplicateGroup(g: CustomerGroup) {
    selectedGroup = { ...g, name: t('copySuffix', { name: g.name }), id: 0 } as CustomerGroup;
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
      toast.success(labels.toastCustomerGroupUpdated);
      showEditModal = false;
      selectedGroup = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedUpdateCustomerGroup);
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
      toast.success(labels.toastCustomerGroupDeleted);
      showDeleteModal = false;
      selectedGroup = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedDeleteCustomerGroup);
    } finally {
      deleting = false;
    }
  }

  async function handleBulkActivate(ids: number[]) {
    try {
      const updated = await bulkUpdateCustomerGroups(ids, true);
      toast.success(t('toastBulkActivatedGroups', { count: updated }));
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedBulkActivateGroups);
    }
  }

  async function handleBulkDeactivate(ids: number[]) {
    try {
      const updated = await bulkUpdateCustomerGroups(ids, false);
      toast.success(t('toastBulkDeactivatedGroups', { count: updated }));
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedBulkDeactivateGroups);
    }
  }

  async function handleBulkDelete(ids: number[]) {
    try {
      const deleted = await bulkDeleteCustomerGroups(ids);
      toast.success(t('toastBulkDeletedGroups', { count: deleted }));
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || labels.toastFailedBulkDeleteGroups);
    }
  }

  function handleImport() {
    showImportWizard = true;
  }

  function openDetail(g: CustomerGroup) {
    detailGroup = g;
    showDetailDrawer = true;
  }

  onMount(() => {
    const returnPage = sessionStorage.getItem('customerGroupsReturnPage');
    if (returnPage) {
      sessionStorage.removeItem('customerGroupsReturnPage');
      try {
        const parsed = JSON.parse(returnPage);
        offset = parsed.offset || 0;
        limit = parsed.limit || 20;
      } catch { /* ignore */ }
    }
    load();
  });
</script>

<div class="space-y-5">
  <CustomerGroupsToolbar
    bind:searchQuery
    bind:statusFilter
    bind:hasCustomersFilter
    {canCreate}
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
      sortBy={sortState.sortBy}
      sortDir={sortState.sortDir}
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
