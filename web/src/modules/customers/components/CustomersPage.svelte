<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/router';
  import apiClient from '$shared/api/http-client';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { Pagination, ImportWizard } from '$shared/ui';
  import { debounce } from '$shared/utils/debounce';
  import { getCustomerGroups } from '$modules/customer-groups';
  import { ArrowLeft } from 'lucide-svelte';
  import CreateCustomerModal from './CreateCustomerModal.svelte';
  import EditCustomerModal from './EditCustomerModal.svelte';
  import DeactivateCustomerModal from './DeactivateCustomerModal.svelte';
  import BulkStatusModal from './BulkStatusModal.svelte';
  import BulkDeleteModal from './BulkDeleteModal.svelte';
  import CustomerToolbar from './CustomerToolbar.svelte';
  import CustomerTable from './CustomerTable.svelte';
  import BulkActionBar from './BulkActionBar.svelte';

  const authStore = useAuthStore();

  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('customer:create'));
  const canUpdate = $derived(userPermissions.includes('customer:update'));
  const canDelete = $derived(userPermissions.includes('customer:delete'));
  const canRead = $derived(userPermissions.includes('customer:read'));

  let customers = $state<any[]>([]);
  let loading = $state(false);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let statusFilter = $state('all');
  let groupFilter = $state('all');
  let availableGroups = $state<{ id: number; name: string }[]>([]);

  let selectedIds = $state(new Set<number>());
  let showBulkStatusModal = $state(false);
  let bulkStatusTargetIsActive = $state(true);
  let isBulkUpdating = $state(false);
  let showBulkDeleteModal = $state(false);
  let isBulkDeleting = $state(false);

  function clearSelection() {
    selectedIds = new Set();
  }

  async function handleBulkStatusUpdate() {
    isBulkUpdating = true;
    const eligibleIds = customers.filter(c => selectedIds.has(c.id) && (c.is_active !== false) !== bulkStatusTargetIsActive).map(c => c.id);
    const skippedCount = selectedIds.size - eligibleIds.length;
    if (eligibleIds.length === 0) {
      toast.warning(`All selected customer(s) already ${bulkStatusTargetIsActive ? 'Active' : 'Deactivated'}`);
      isBulkUpdating = false;
      showBulkStatusModal = false;
      return;
    }
    try {
      await apiClient.post('/customers/bulk/status', { ids: eligibleIds, is_active: bulkStatusTargetIsActive });
      toast.success(`${bulkStatusTargetIsActive ? 'Activated' : 'Deactivated'} ${eligibleIds.length} customer(s)`);
      if (skippedCount > 0) {
        toast.warning(`${skippedCount} customer(s) already ${bulkStatusTargetIsActive ? 'Active' : 'Deactivated'}`);
      }
      selectedIds = new Set();
      await load();
    } catch (err: any) {
      toast.error(err?.response?.data?.error || 'Failed to update customer status');
    } finally {
      isBulkUpdating = false;
      showBulkStatusModal = false;
    }
  }

  async function handleBulkDelete() {
    isBulkDeleting = true;
    try {
      const ids = Array.from(selectedIds);
      await apiClient.post('/customers/bulk/delete', { ids });
      toast.success(`Deleted ${ids.length} customer(s)`);
      selectedIds = new Set();
      await load();
    } catch (err: any) {
      toast.error(err?.response?.data?.error || 'Failed to delete customers');
    } finally {
      isBulkDeleting = false;
      showBulkDeleteModal = false;
    }
  }

  let showEditModal = $state(false);
  let editTarget = $state<any>(null);
  let isSaving = $state(false);

  let sortBy = $state('name');
  let sortDir = $state<'asc' | 'desc'>('asc');

  let showCreateModal = $state(false);
  let creating = $state(false);
  let formName = $state('');
  let formPhone = $state('');
  let formEmail = $state('');
  let formAddress = $state('');
  let formNote = $state('');
  let formGroupId = $state<number | null>(null);
  let fieldErrors = $state({ name: '', phone: '', email: '', address: '', note: '' });

  let deactivateTarget = $state<any>(null);
  let showDeactivateModal = $state(false);
  let deactivating = $state(false);

  function validateEmail(email: string): boolean {
    if (!email) return true;
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  }

  function validatePhone(phone: string): boolean {
    if (!phone) return true;
    return /^[0-9+\-() ]{7,20}$/.test(phone);
  }

  function validateForm(): boolean {
    const errors = { name: '', phone: '', email: '', address: '', note: '' };
    let valid = true;

    if (!formName.trim()) {
      errors.name = 'Name is required';
      valid = false;
    } else if (formName.trim().length > 200) {
      errors.name = 'Name must be at most 200 characters';
      valid = false;
    }

    if (!formPhone.trim()) {
      errors.phone = 'Phone is required';
      valid = false;
    } else if (!validatePhone(formPhone.trim())) {
      errors.phone = 'Invalid phone format';
      valid = false;
    }

    if (!formEmail.trim()) {
      errors.email = 'Email is required';
      valid = false;
    } else if (!validateEmail(formEmail.trim())) {
      errors.email = 'Invalid email format';
      valid = false;
    }

    fieldErrors = errors;
    return valid;
  }

  function resetForm() {
    formName = '';
    formPhone = '';
    formEmail = '';
    formAddress = '';
    formNote = '';
    formGroupId = null;
    fieldErrors = { name: '', phone: '', email: '', address: '', note: '' };
  }

  function getStatusFilterParams(): string | undefined {
    if (statusFilter === 'active') return 'true';
    if (statusFilter === 'inactive') return 'false';
    return undefined;
  }

  async function load(newOffset = offset, newLimit = limit) {
    selectedIds = new Set();
    loading = true;
    try {
      offset = newOffset;
      limit = newLimit;
      const params: any = { limit, offset, search: searchQuery || undefined };
      const activeParam = getStatusFilterParams();
      if (activeParam !== undefined) params.is_active = activeParam;
      if (groupFilter !== 'all') params.customer_group_id = groupFilter;
      const r = await apiClient.get('/customers', { params });
      customers = r.data.data || [];
      total = r.data.total || 0;
      sortCustomers();
    } catch (e: any) {
      console.error(e);
      toast.error(e?.response?.data?.error || e?.message || 'Failed to load customers');
    } finally {
      loading = false;
    }
  }

  const debouncedSearch = debounce(() => load(0, limit), 400);

  function handleSearchInput() {
    if (!searchQuery) {
      load(0, limit);
      return;
    }
    debouncedSearch();
  }

  function handleStatusFilterChange() {
    offset = 0;
    load(0, limit);
  }

  function handleGroupFilterChange() {
    offset = 0;
    load(0, limit);
  }

  function handleSort(column: string) {
    if (sortBy === column) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = column;
      sortDir = 'asc';
    }
    sortCustomers();
  }

  function sortCustomers() {
    customers.sort((a, b) => {
      let aVal: any, bVal: any;
      switch (sortBy) {
        case 'name': aVal = (a.name || '').toLowerCase(); bVal = (b.name || '').toLowerCase(); break;
        case 'phone': aVal = (a.phone || '').toLowerCase(); bVal = (b.phone || '').toLowerCase(); break;
        case 'email': aVal = (a.email || '').toLowerCase(); bVal = (b.email || '').toLowerCase(); break;
        case 'group': aVal = (a.customer_group_name || '').toLowerCase(); bVal = (b.customer_group_name || '').toLowerCase(); break;
        case 'status': aVal = a.is_active !== false ? 1 : 0; bVal = b.is_active !== false ? 1 : 0; break;
        default: return 0;
      }
      if (sortDir === 'asc') {
        return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
      } else {
        return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
      }
    });
  }

  async function createCustomer() {
    if (!validateForm()) return;
    creating = true;
    try {
      await apiClient.post('/customers', {
        name: formName.trim(),
        phone: formPhone.trim(),
        email: formEmail.trim(),
        address: formAddress.trim() || null,
        note: formNote.trim() || null,
        customer_group_id: formGroupId,
      });
      toast.success(`Customer "${formName.trim()}" created successfully`);
      resetForm();
      showCreateModal = false;
      await load();
    } catch (e: any) {
      const msg = e?.response?.data?.error || 'Failed to create customer';
      toast.error(msg);
    } finally {
      creating = false;
    }
  }

  function startEdit(c: any) {
    editTarget = c;
    showEditModal = true;
  }

  async function handleEditSave(data: any) {
    isSaving = true;
    try {
      await apiClient.put(`/customers/${data.id}`, {
        name: data.name,
        phone: data.phone,
        email: data.email,
        address: data.address,
        note: data.note,
        is_active: data.is_active,
        customer_group_id: data.customer_group_id,
      });
      toast.success('Customer updated successfully');
      showEditModal = false;
      editTarget = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Failed to update customer');
    } finally {
      isSaving = false;
    }
  }

  function handleEditCancel() {
    showEditModal = false;
    editTarget = null;
  }

  async function deactivateCustomer(c: any) {
    deactivateTarget = c;
    showDeactivateModal = true;
  }

  async function confirmDeactivate() {
    if (!deactivateTarget) return;
    deactivating = true;
    try {
      await apiClient.delete(`/customers/${deactivateTarget.id}`);
      toast.success(`Customer "${deactivateTarget.name}" deactivated`);
      showDeactivateModal = false;
      deactivateTarget = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Failed to deactivate customer');
    } finally {
      deactivating = false;
    }
  }

  let showImportWizard = $state(false);
  let backToGroups = $state<{ offset: number; limit: number; groupName: string } | null>(null);

  function handleBackToGroups() {
    if (backToGroups) {
      sessionStorage.setItem('customerGroupsReturnPage', JSON.stringify({ offset: backToGroups.offset, limit: backToGroups.limit }));
    }
    goto('/customer-groups');
  }

  function handleImportComplete() {
    load();
    toast.success('Customer import completed');
  }

  onMount(async () => {
    const urlGroup = sessionStorage.getItem('customerGroupFilter');
    const backPage = sessionStorage.getItem('customerGroupsBackPage');
    if (urlGroup) {
      sessionStorage.removeItem('customerGroupFilter');
      groupFilter = urlGroup;
    }
    if (backPage) {
      sessionStorage.removeItem('customerGroupsBackPage');
      try {
        const parsed = JSON.parse(backPage);
        backToGroups = { offset: parsed.offset || 0, limit: parsed.limit || 20, groupName: parsed.groupName || '' };
      } catch { /* ignore */ }
    }
    load();
    try {
      const result = await getCustomerGroups({ limit: 100, offset: 0 });
      availableGroups = result.data.filter(g => g.is_active);
    } catch { /* ignore */ }
  });
</script>

<div class="space-y-5">
  {#if backToGroups}
    <div class="card px-4 py-3 flex items-center gap-3">
      <button
        type="button"
        class="inline-flex items-center gap-1.5 text-sm font-medium text-text-secondary hover:text-text-primary transition-colors"
        onclick={handleBackToGroups}
      >
        <ArrowLeft size={16} />
        Kembali
      </button>
      <span class="text-border-default">|</span>
      <span class="text-sm text-text-muted">
        Menampilkan anggota dari: <span class="font-medium text-text-secondary">{backToGroups.groupName}</span>
      </span>
    </div>
  {/if}

  <CustomerToolbar
    bind:searchQuery
    bind:statusFilter
    bind:groupFilter
    {canCreate}
    groups={availableGroups}
    onsearch={handleSearchInput}
    onstatuschange={handleStatusFilterChange}
    ongroupchange={handleGroupFilterChange}
    oncreate={() => { resetForm(); showCreateModal = true; }}
    onImport={() => showImportWizard = true}
  />

  <div class="card overflow-hidden">
    <CustomerTable
      {customers}
      {loading}
      {searchQuery}
      {canUpdate}
      {canDelete}
      bind:selectedIds
      bind:sortBy
      bind:sortDir
      onsort={handleSort}
      onedit={startEdit}
      ondeactivate={deactivateCustomer}
    />

    <BulkActionBar
      selectedCount={selectedIds.size}
      {canUpdate}
      {canDelete}
      onstatus={() => { bulkStatusTargetIsActive = customers.some(c => selectedIds.has(c.id) && c.is_active === false); showBulkStatusModal = true; }}
      ondelete={() => showBulkDeleteModal = true}
      onclear={clearSelection}
    />

    {#if !loading && customers.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination {total} {limit} {offset} onPageChange={load} />
      </div>
    {/if}
  </div>
</div>

<CreateCustomerModal
  bind:open={showCreateModal}
  bind:formName
  bind:formPhone
  bind:formEmail
  bind:formAddress
  bind:formNote
  bind:formGroupId
  bind:fieldErrors
  bind:creating
  groups={availableGroups}
  oncreate={createCustomer}
/>

<EditCustomerModal
  bind:open={showEditModal}
  customer={editTarget}
  bind:saving={isSaving}
  groups={availableGroups}
  onsave={handleEditSave}
  oncancel={handleEditCancel}
/>

<DeactivateCustomerModal
  bind:open={showDeactivateModal}
  targetName={deactivateTarget?.name ?? ''}
  bind:deactivating
  oncancel={() => { showDeactivateModal = false; deactivateTarget = null; }}
  onconfirm={confirmDeactivate}
/>

<BulkStatusModal
  bind:open={showBulkStatusModal}
  selectedCount={selectedIds.size}
  affectedCount={customers.filter(c => selectedIds.has(c.id) && (c.is_active !== false) !== bulkStatusTargetIsActive).length}
  bind:isActive={bulkStatusTargetIsActive}
  bind:updating={isBulkUpdating}
  oncancel={() => showBulkStatusModal = false}
  onconfirm={handleBulkStatusUpdate}
/>

<BulkDeleteModal
  bind:open={showBulkDeleteModal}
  count={selectedIds.size}
  bind:deleting={isBulkDeleting}
  oncancel={() => showBulkDeleteModal = false}
  onconfirm={handleBulkDelete}
/>

<ImportWizard
  bind:open={showImportWizard}
  module="customers"
  displayName="Customers"
  onComplete={handleImportComplete}
/>


