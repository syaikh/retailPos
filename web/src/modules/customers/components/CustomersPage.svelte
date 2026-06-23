<script lang="ts">
  import { onMount } from 'svelte';
  import apiClient from '$shared/api/http-client';
  import { Badge, Button, SearchBar, Skeleton } from '$shared/ui';
  import { Pencil, Trash2, Check, X, Plus, Search, UserPlus, Loader2 } from 'lucide-svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { Input, Modal, Pagination } from '$shared/ui';
  import { debounce } from '$shared/utils/debounce';

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

  let selectedIds = $state(new Set<number>());
  let showBulkStatusModal = $state(false);
  let bulkStatusTargetIsActive = $state(true);
  let isBulkUpdating = $state(false);
  let showBulkDeleteModal = $state(false);
  let isBulkDeleting = $state(false);

  let allSelected = $derived(customers.length > 0 && customers.every(c => selectedIds.has(c.id)));
  let someSelected = $derived(selectedIds.size > 0 && !allSelected);

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(customers.map(c => c.id));
    }
  }

  function toggleSelect(id: number) {
    const next = new Set(selectedIds);
    if (next.has(id)) { next.delete(id); } else { next.add(id); }
    selectedIds = next;
  }

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

  let showStatusDropdown = $state(false);
  let editingId = $state<number | null>(null);
  let editName = $state('');
  let editPhone = $state('');
  let editEmail = $state('');
  let editNote = $state('');
  let editActive = $state(true);

  let sortBy = $state('name');
  let sortDir = $state('asc');

  let showCreateModal = $state(false);
  let creating = $state(false);
  let formName = $state('');
  let formPhone = $state('');
  let formEmail = $state('');
  let formNote = $state('');
  let fieldErrors = $state({ name: '', phone: '', email: '', note: '' });

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
    const errors = { name: '', phone: '', email: '', note: '' };
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
    formNote = '';
    fieldErrors = { name: '', phone: '', email: '', note: '' };
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
      if (activeParam !== undefined) params.isActive = activeParam;
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

  function getInitials(name: string): string {
    if (!name) return '?';
    const parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0].charAt(0) + parts[1].charAt(0)).toUpperCase();
    }
    return parts[0].charAt(0).toUpperCase();
  }

  async function createCustomer() {
    if (!validateForm()) return;
    creating = true;
    try {
      await apiClient.post('/customers', {
        name: formName.trim(),
        phone: formPhone.trim(),
        email: formEmail.trim(),
        note: formNote.trim() || undefined,
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
    editingId = c.id;
    editName = c.name || '';
    editPhone = c.phone || '';
    editEmail = c.email || '';
    editNote = c.note || '';
    editActive = c.is_active !== false;
  }

  function cancelEdit() {
    editingId = null;
  }

  async function saveEdit(id: number) {
    if (!editName.trim()) { toast.error('Name is required'); return; }
    if (!editPhone.trim()) { toast.error('Phone is required'); return; }
    if (!validatePhone(editPhone.trim())) { toast.error('Invalid phone format'); return; }
    if (!editEmail.trim()) { toast.error('Email is required'); return; }
    if (!validateEmail(editEmail.trim())) { toast.error('Invalid email format'); return; }
    try {
      await apiClient.put(`/customers/${id}`, {
        name: editName.trim(),
        phone: editPhone.trim() || undefined,
        email: editEmail.trim() || undefined,
        note: editNote.trim() || undefined,
        is_active: editActive,
      });
      toast.success('Customer updated successfully');
      editingId = null;
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Failed to update customer');
    }
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
      if (editingId === deactivateTarget.id) editingId = null;
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

  const handleClickOutside = (e) => {
    if (showStatusDropdown && !e.target.closest('.status-filter-container')) showStatusDropdown = false;
  };
  const handleEsc = (e) => {
    if (e.key === 'Escape') showStatusDropdown = false;
  };

  onMount(() => {
    load();
    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleEsc);
    return () => {
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleEsc);
    };
  });
</script>

<div class="space-y-4">
  <div class="border border-border rounded-xl p-4 space-y-3 bg-bg-card">
    <div class="flex items-center gap-3">
      <div class="flex-1">
        <SearchBar bind:value={searchQuery} placeholder="Search by name, phone, or email..." oninput={handleSearchInput} />
      </div>
      <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default">
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'all'; handleStatusFilterChange(); }}
        >
          All
        </button>
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'active'; handleStatusFilterChange(); }}
        >
          Active
        </button>
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'inactive'; handleStatusFilterChange(); }}
        >
          Inactive
        </button>
      </div>
      {#if canCreate}
        <Button onclick={() => { resetForm(); showCreateModal = true; }} variant="primary" class="shrink-0 shadow-glow-primary-sm px-5">
          <Plus size={18} />
          Add Customer
        </Button>
      {/if}
    </div>
  </div>

  <div class="card overflow-hidden">
    <table class="min-w-full text-sm table-fixed">
      <thead class="bg-muted/50">
        <tr>
          <th class="p-4 font-semibold w-12">
            <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={allSelected} bind:indeterminate={someSelected} onchange={toggleSelectAll} aria-label="Select all customers" />
          </th>
          <th class="text-left p-4 font-semibold w-[26%]">
            <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('name')}>
              NAME {#if sortBy === 'name'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </button>
          </th>
          <th class="text-left p-4 font-semibold w-[18%]">
            <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('phone')}>
              PHONE {#if sortBy === 'phone'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </button>
          </th>
          <th class="text-left p-4 font-semibold w-[26%]">
            <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('email')}>
              EMAIL {#if sortBy === 'email'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </button>
          </th>
          <th class="text-left p-4 font-semibold w-[14%]">
            <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('status')}>
              STATUS {#if sortBy === 'status'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </button>
          </th>
          <th class="text-center p-4 font-semibold w-20">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          {#each { length: 5 } as _, i}
            <tr class="border-t border-border">
              <td class="px-4 py-3" colspan={6}>
                <div class="flex items-center gap-3">
                  <Skeleton width="w-8" height="h-8" rounded="rounded-full" />
                  <div class="flex-1 space-y-2">
                    <Skeleton width="w-3/5" height="h-4" />
                    <Skeleton width="w-2/5" height="h-4" />
                  </div>
                </div>
              </td>
            </tr>
          {/each}
        {:else if customers.length === 0}
          <tr class="border-t border-border">
            <td colspan={6}>
              <div class="px-4 py-16 text-center">
                <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
                  <Search size={32} class="text-text-muted" />
                </div>
                <p class="text-text-primary font-semibold mt-4">
                  {searchQuery ? 'No customers found' : 'No customers yet'}
                </p>
                <p class="text-text-muted text-sm mt-1">
                  {searchQuery ? `No customers matching "${searchQuery}"` : 'Start by adding your first customer'}
                </p>
              </div>
            </td>
          </tr>
            {:else}
              {#each customers as c}
                {#if editingId === c.id}
                  <tr class="border-t border-border bg-primary-subtle/10">
                    <td class="px-4 py-1.5 h-12 overflow-hidden">
                      <div class="flex items-center gap-2">
                        <div class="w-7 h-7 rounded-full bg-primary-subtle text-primary-light flex items-center justify-center text-[10px] font-bold shrink-0">
                          {getInitials(editName)}
                        </div>
                        <input class="flex-1 h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editName} aria-label="Edit name" />
                      </div>
                    </td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden"><input class="w-full h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editPhone} aria-label="Edit phone" /></td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden"><input class="w-full h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editEmail} aria-label="Edit email" /></td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden">
                      <label class="flex items-center gap-2 text-xs">
                        <input type="checkbox" bind:checked={editActive} />
                        {editActive ? 'Active' : 'Inactive'}
                      </label>
                    </td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden">
                      <div class="flex items-center gap-1">
                        <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light transition-all active:scale-90" onclick={() => saveEdit(c.id)} title="Save" aria-label="Save">
                          <Check size={14} />
                        </Button>
                        <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger transition-all active:scale-90" onclick={cancelEdit} title="Cancel" aria-label="Cancel">
                          <X size={14} />
                        </Button>
                      </div>
                    </td>
                  </tr>
                {:else}
                  <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
                    <td class="px-4 py-1.5 h-12 w-12" onclick={(e) => e.stopPropagation()}>
                      <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={selectedIds.has(c.id)} onchange={() => toggleSelect(c.id)} aria-label="Select {c.name}" />
                    </td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden">
                      <div class="flex items-center gap-3">
                        <div class="w-8 h-8 rounded-full bg-primary-subtle text-primary-light flex items-center justify-center text-xs font-bold shrink-0">
                          {getInitials(c.name)}
                        </div>
                        <span class="truncate">{c.name}</span>
                      </div>
                    </td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden">{c.phone || '—'}</td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden">{c.email || '—'}</td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden">
                      {#if c.is_active !== false}
                        <Badge variant="success" size="sm">Active</Badge>
                      {:else}
                        <Badge variant="danger" size="sm">Inactive</Badge>
                      {/if}
                    </td>
                    <td class="px-4 py-1.5 h-12 overflow-hidden text-center">
                      <div class="flex items-center justify-center gap-1">
                        {#if canUpdate}
                          <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light transition-all active:scale-90" onclick={() => startEdit(c)} title="Edit" aria-label="Edit">
                            <Pencil size={14} />
                          </Button>
                        {/if}
                        {#if canDelete && c.is_active !== false}
                          <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger hover:bg-danger-subtle transition-all active:scale-90" onclick={() => deactivateCustomer(c)} title="Deactivate" aria-label="Deactivate">
                            <Trash2 size={14} />
                          </Button>
                        {/if}
                      </div>
                    </td>
                  </tr>
                {/if}
              {/each}
            {/if}
          </tbody>
        </table>

        {#if selectedIds.size > 0}
          <div class="px-4 py-2.5 bg-primary/5 border-t border-primary/20 flex items-center gap-3">
            <span class="text-sm font-semibold text-text-primary">{selectedIds.size} selected</span>
            <div class="flex items-center gap-2 ml-auto">
              {#if canUpdate}
                <Button variant="secondary" class="text-xs px-3 py-1.5 h-auto" onclick={() => { bulkStatusTargetIsActive = customers.some(c => selectedIds.has(c.id) && c.is_active === false); showBulkStatusModal = true; }}>
                  Change Status
                </Button>
              {/if}
              {#if canDelete}
                <Button variant="danger" class="text-xs px-3 py-1.5 h-auto" onclick={() => showBulkDeleteModal = true}>Delete</Button>
              {/if}
              <button type="button" class="text-xs px-3 py-1.5 h-auto text-text-muted hover:text-text-secondary transition-colors font-medium" onclick={clearSelection}>Clear</button>
            </div>
          </div>
        {/if}

        {#if !loading && customers.length > 0}
          <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
            <Pagination {total} {limit} {offset} onPageChange={load} />
          </div>
        {/if}
      </div>
    </div>

<Modal bind:open={showCreateModal} title="Add Customer" size="md">
  <div class="space-y-4">
    <div class="space-y-1">
      <label for="customer-name" class="text-xs font-semibold text-text-secondary">Name <span class="text-danger">*</span></label>
      <Input
        id="customer-name"
        class={fieldErrors.name ? 'border-danger' : ''}
        placeholder="e.g. John Doe"
        bind:value={formName}
      />
      {#if fieldErrors.name}
        <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
      {/if}
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="space-y-1">
        <label for="customer-phone" class="text-xs font-semibold text-text-secondary">Phone <span class="text-danger">*</span></label>
        <Input
          id="customer-phone"
          class={fieldErrors.phone ? 'border-danger' : ''}
          placeholder="e.g. 08123456789"
          bind:value={formPhone}
        />
        {#if fieldErrors.phone}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.phone}</p>
        {/if}
      </div>
      <div class="space-y-1">
        <label for="customer-email" class="text-xs font-semibold text-text-secondary">Email <span class="text-danger">*</span></label>
        <Input
          id="customer-email"
          class={fieldErrors.email ? 'border-danger' : ''}
          placeholder="e.g. john@example.com"
          bind:value={formEmail}
        />
        {#if fieldErrors.email}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.email}</p>
        {/if}
      </div>
    </div>
    <div class="space-y-1">
      <label for="customer-note" class="text-xs font-semibold text-text-secondary">Note</label>
      <Input
        tag="textarea"
        id="customer-note"
        class="min-h-[60px] resize-none"
        placeholder="Optional notes about this customer"
        bind:value={formNote}
      />
    </div>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={() => showCreateModal = false}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={creating} onclick={createCustomer}>
      {#if creating}
        <Loader2 size={14} class="animate-spin mr-1" /> Creating...
      {:else}
        <UserPlus size={14} class="mr-1" /> Create Customer
      {/if}
    </Button>
  {/snippet}
</Modal>

  <Modal bind:open={showDeactivateModal} title="Deactivate Customer" size="sm">
  <p class="text-sm text-text-secondary">
    Are you sure you want to deactivate <strong class="text-text-primary">{deactivateTarget?.name}</strong>? This will hide them from active listings but preserve their history.
  </p>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={() => { showDeactivateModal = false; deactivateTarget = null; }}>Cancel</Button>
    <Button variant="danger" class="px-5" disabled={deactivating} onclick={confirmDeactivate}>
      {#if deactivating}
        <Loader2 size={14} class="animate-spin mr-1" /> Deactivating...
      {:else}
        <Trash2 size={14} class="mr-1" /> Deactivate
      {/if}
    </Button>
  {/snippet}
</Modal>

<Modal bind:open={showBulkStatusModal} title="Bulk Update Status" size="sm">
  <div class="py-2">
    <p class="text-text-primary font-semibold mb-3">
      Set selected customers to <span class="text-primary-light">{bulkStatusTargetIsActive ? 'Active' : 'Inactive'}</span>
    </p>
    <p class="text-sm text-text-secondary mb-4">
      {customers.filter(c => selectedIds.has(c.id) && (c.is_active !== false) !== bulkStatusTargetIsActive).length} of {selectedIds.size} customer(s) will be updated.
    </p>
    <div class="flex flex-wrap gap-2 justify-center">
      <button
        class="px-4 py-2 rounded-lg text-sm font-medium border transition-all {bulkStatusTargetIsActive ? 'bg-success-subtle border-success text-success-light' : 'bg-surface-default border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
        onclick={() => bulkStatusTargetIsActive = true}
      >Activate</button>
      <button
        class="px-4 py-2 rounded-lg text-sm font-medium border transition-all {!bulkStatusTargetIsActive ? 'bg-danger-subtle border-danger text-danger-light' : 'bg-surface-default border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
        onclick={() => bulkStatusTargetIsActive = false}
      >Deactivate</button>
    </div>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isBulkUpdating} onclick={() => showBulkStatusModal = false}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={isBulkUpdating} onclick={handleBulkStatusUpdate}>
      {#if isBulkUpdating}
        <Loader2 size={14} class="animate-spin mr-1" /> Updating...
      {:else}
        Update
      {/if}
    </Button>
  {/snippet}
</Modal>

<Modal bind:open={showBulkDeleteModal} title="Delete Customers" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Delete {selectedIds.size} customer(s)?</p>
    <p class="text-text-muted text-sm">This will permanently remove them from the active customer list. Their transaction history will be preserved.</p>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isBulkDeleting} onclick={() => showBulkDeleteModal = false}>Cancel</Button>
    <Button variant="danger" class="px-5" disabled={isBulkDeleting} onclick={handleBulkDelete}>
      {#if isBulkDeleting}
        <Loader2 size={14} class="animate-spin mr-1" /> Deleting...
      {:else}
        Delete
      {/if}
    </Button>
  {/snippet}
</Modal>
