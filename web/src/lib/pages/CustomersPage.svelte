<script lang="ts">
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { Pencil, Trash2, Check, X, Plus, ArrowUpDown, Search, UserPlus, Loader2 } from 'lucide-svelte';
  import { auth } from '$lib/stores/auth';
  import { toast } from '$lib/stores/toast';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import { debounce } from '$lib/utils/debounce';

  const userPermissions = $derived($auth.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('customer:create'));
  const canUpdate = $derived(userPermissions.includes('customer:update'));
  const canDelete = $derived(userPermissions.includes('customer:delete'));

  let customers = $state<any[]>([]);
  let loading = $state(false);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let statusFilter = $state('all');

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

  let skipNextSearch = false;

  function clearSearch() {
    skipNextSearch = true;
    searchQuery = '';
    load(0, limit);
  }

  function handleSearchInput() {
    if (skipNextSearch) {
      skipNextSearch = false;
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
    if (!confirm(`Deactivate customer "${c.name}"?`)) return;
    try {
      await apiClient.delete(`/customers/${c.id}`);
      if (editingId === c.id) editingId = null;
      toast.success(`Customer "${c.name}" deactivated`);
      await load();
    } catch (e: any) {
      toast.error(e?.response?.data?.error || 'Failed to deactivate customer');
    }
  }

  onMount(load);
</script>

<div class="space-y-4">
  <div class="border border-border rounded-xl p-4 space-y-3 bg-bg-card">
    <div class="flex items-center gap-3">
      <div class="relative flex-1">
        <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
        <input
          type="text"
          placeholder="Search customers..."
          bind:value={searchQuery}
          oninput={handleSearchInput}
          class="input pl-10 pr-8 w-full"
        />
        {#if searchQuery}
          <button
            onclick={clearSearch}
            class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
            title="Clear search"
          >
            <X size={14} />
          </button>
        {/if}
      </div>
      <select
        bind:value={statusFilter}
        onchange={handleStatusFilterChange}
        class="input h-10 px-3 w-40"
      >
        <option value="all">All Status</option>
        <option value="active">Active</option>
        <option value="inactive">Inactive</option>
      </select>
      {#if canCreate}
        <button
          onclick={() => { resetForm(); showCreateModal = true; }}
          class="btn btn-primary rounded-full shrink-0 shadow-glow-primary-sm px-5"
        >
          <Plus size={18} />
          Add Customer
        </button>
      {/if}
    </div>
  </div>

  <div class="border border-border rounded-xl overflow-hidden bg-bg-card">
    <table class="min-w-full text-sm table-fixed">
      <thead class="bg-surface-hover text-text-secondary">
        <tr>
          <th class="text-left px-4 py-2 w-[30%]">
            <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('name')}>
              NAME <ArrowUpDown size={14} class="text-text-muted" />
            </button>
          </th>
          <th class="text-left px-4 py-2 w-[18%]">
            <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('phone')}>
              PHONE <ArrowUpDown size={14} class="text-text-muted" />
            </button>
          </th>
          <th class="text-left px-4 py-2 w-[26%]">
            <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('email')}>
              EMAIL <ArrowUpDown size={14} class="text-text-muted" />
            </button>
          </th>
          <th class="text-left px-4 py-2 w-[14%]">
            <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('status')}>
              STATUS <ArrowUpDown size={14} class="text-text-muted" />
            </button>
          </th>
          <th class="text-left px-4 py-2 w-[12%]">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td class="px-4 py-3" colspan={5}>Loading...</td></tr>
        {:else if customers.length === 0}
          <tr class="border-t border-border">
            <td colspan={5}>
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
                    <input class="flex-1 h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editName} />
                  </div>
                </td>
                <td class="px-4 py-1.5 h-12 overflow-hidden"><input class="w-full h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editPhone} /></td>
                <td class="px-4 py-1.5 h-12 overflow-hidden"><input class="w-full h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editEmail} /></td>
                <td class="px-4 py-1.5 h-12 overflow-hidden">
                  <label class="flex items-center gap-2 text-xs">
                    <input type="checkbox" bind:checked={editActive} />
                    {editActive ? 'Active' : 'Inactive'}
                  </label>
                </td>
                <td class="px-4 py-1.5 h-12 overflow-hidden">
                  <div class="flex items-center gap-1">
                    <button class="btn-icon btn-ghost text-text-muted hover:text-primary-light transition-all active:scale-90" onclick={() => saveEdit(c.id)} title="Save">
                      <Check size={14} />
                    </button>
                    <button class="btn-icon btn-ghost text-text-muted hover:text-danger transition-all active:scale-90" onclick={cancelEdit} title="Cancel">
                      <X size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            {:else}
              <tr class="border-t border-border">
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
                    <span class="badge badge-success text-xs">Active</span>
                  {:else}
                    <span class="badge badge-danger text-xs">Inactive</span>
                  {/if}
                </td>
                <td class="px-4 py-1.5 h-12 overflow-hidden">
                  <div class="flex items-center gap-1">
                    {#if canUpdate}
                      <button class="btn-icon btn-ghost text-text-muted hover:text-primary-light transition-all active:scale-90" onclick={() => startEdit(c)} title="Edit">
                        <Pencil size={14} />
                      </button>
                    {/if}
                    {#if canDelete && c.is_active !== false}
                      <button class="btn-icon btn-ghost text-text-muted hover:text-danger hover:bg-danger-subtle transition-all active:scale-90" onclick={() => deactivateCustomer(c)} title="Deactivate">
                        <Trash2 size={14} />
                      </button>
                    {/if}
                  </div>
                </td>
              </tr>
            {/if}
          {/each}
        {/if}
      </tbody>
    </table>

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
      <label class="text-xs font-semibold text-text-secondary">Name <span class="text-danger">*</span></label>
      <input
        class="input"
        class:border-danger={fieldErrors.name}
        placeholder="e.g. John Doe"
        bind:value={formName}
      />
      {#if fieldErrors.name}
        <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
      {/if}
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="space-y-1">
        <label class="text-xs font-semibold text-text-secondary">Phone <span class="text-danger">*</span></label>
        <input
          class="input"
          class:border-danger={fieldErrors.phone}
          placeholder="e.g. 08123456789"
          bind:value={formPhone}
        />
        {#if fieldErrors.phone}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.phone}</p>
        {/if}
      </div>
      <div class="space-y-1">
        <label class="text-xs font-semibold text-text-secondary">Email <span class="text-danger">*</span></label>
        <input
          class="input"
          class:border-danger={fieldErrors.email}
          placeholder="e.g. john@example.com"
          bind:value={formEmail}
        />
        {#if fieldErrors.email}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.email}</p>
        {/if}
      </div>
    </div>
    <div class="space-y-1">
      <label class="text-xs font-semibold text-text-secondary">Note</label>
      <textarea
        class="input min-h-[60px] resize-none"
        placeholder="Optional notes about this customer"
        bind:value={formNote}
      ></textarea>
    </div>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary rounded-full px-5" onclick={() => showCreateModal = false}>Cancel</button>
    <button class="btn btn-primary rounded-full px-5" disabled={creating} onclick={createCustomer}>
      {#if creating}
        <Loader2 size={14} class="animate-spin mr-1" /> Creating...
      {:else}
        <UserPlus size={14} class="mr-1" /> Create Customer
      {/if}
    </button>
  {/snippet}
</Modal>
