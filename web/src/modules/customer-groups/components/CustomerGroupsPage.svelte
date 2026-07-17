<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/router';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { getCustomerGroups, createCustomerGroup, updateCustomerGroup, deleteCustomerGroup } from '../services/customer-group-service';
  import type { CustomerGroup } from '../types';
  import { Pagination } from '$shared/ui';
  import { debounce } from '$shared/utils/debounce';
  import CustomerGroupsToolbar from './CustomerGroupsToolbar.svelte';
  import CustomerGroupsTable from './CustomerGroupsTable.svelte';
  import CreateCustomerGroupModal from './CreateCustomerGroupModal.svelte';
  import EditCustomerGroupModal from './EditCustomerGroupModal.svelte';
  import DeleteCustomerGroupModal from './DeleteCustomerGroupModal.svelte';

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

  async function load() {
    loading = true;
    try {
      const filters: any = { limit, offset };
      if (searchQuery.trim()) filters.search = searchQuery.trim();
      if (statusFilter !== 'all') filters.is_active = statusFilter === 'active';

      const result = await getCustomerGroups(filters);

      if (sortBy === 'name') {
        result.data.sort((a, b) => sortDir === 'asc' ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name));
      } else if (sortBy === 'status') {
        result.data.sort((a, b) => sortDir === 'asc' ? (a.is_active ? 1 : 0) - (b.is_active ? 1 : 0) : (b.is_active ? 1 : 0) - (a.is_active ? 1 : 0));
      } else if (sortBy === 'created_at') {
        result.data.sort((a, b) => sortDir === 'asc' ? (a.created_at || '').localeCompare(b.created_at || '') : (b.created_at || '').localeCompare(a.created_at || ''));
      }

      groups = result.data;
      total = result.total;
    } catch (e) {
      toast.error('Failed to load customer groups');
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
      toast.success('Customer group created');
      showCreateModal = false;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Failed to create group');
    } finally {
      creating = false;
    }
  }

  function viewMembers(g: CustomerGroup) {
    sessionStorage.setItem('customerGroupFilter', String(g.id));
    goto('/customers');
  }

  function openEdit(g: CustomerGroup) {
    selectedGroup = g;
    showEditModal = true;
  }

  async function handleEditSave(data: { id: number; name: string; description?: string; is_active: boolean }) {
    saving = true;
    try {
      await updateCustomerGroup(data.id, data);
      toast.success('Customer group updated');
      showEditModal = false;
      selectedGroup = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Failed to update group');
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
      toast.success('Customer group deleted');
      showDeleteModal = false;
      selectedGroup = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Failed to delete group');
    } finally {
      deleting = false;
    }
  }

  onMount(() => { load(); });
</script>

<div class="space-y-5">
  <CustomerGroupsToolbar
    bind:searchQuery
    bind:statusFilter
    {canCreate}
    onsearch={handleSearch}
    onstatuschange={handleStatusChange}
    oncreate={() => showCreateModal = true}
  />

  <div class="card overflow-hidden">
    <CustomerGroupsTable
      {groups}
      {loading}
      {searchQuery}
      {canUpdate}
      {canDelete}
      bind:sortBy
      bind:sortDir
      onsort={handleSort}
      onviewmembers={viewMembers}
      onedit={openEdit}
      ondelete={openDelete}
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
