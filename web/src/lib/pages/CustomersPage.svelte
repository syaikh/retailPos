<script lang="ts">
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { Pencil, Trash2, Check, X, Plus } from 'lucide-svelte';
  import { auth } from '$lib/stores/auth';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Search } from 'lucide-svelte';
  import { debounce } from '$lib/utils/debounce';

  const userPermissions = $derived($auth.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('customer:create'));
  const canUpdate = $derived(userPermissions.includes('customer:update'));
  const canDelete = $derived(userPermissions.includes('customer:delete'));

  let customers = $state<any[]>([]);
  let loading = $state(false);
  let errorMsg = $state('');
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let name = $state('');
  let phone = $state('');
  let email = $state('');
  let note = $state('');

  let editingId = $state<number | null>(null);
  let editName = $state('');
  let editPhone = $state('');
  let editEmail = $state('');
  let editNote = $state('');
  let editActive = $state(true);

  function validateEmail(email: string): boolean {
    if (!email) return true;
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  }

  function validatePhone(phone: string): boolean {
    if (!phone) return true;
    return /^[0-9+\-() ]{7,20}$/.test(phone);
  }

  async function load(newOffset = offset, newLimit = limit) {
    loading = true;
    errorMsg = '';
    try {
      offset = newOffset;
      limit = newLimit;
      const r = await apiClient.get('/customers', { params: { limit, offset, search: searchQuery || undefined } });
      customers = r.data.data || [];
      total = r.data.total || 0;
    } catch (e: any) {
      console.error(e);
      errorMsg = e?.response?.data?.error || e?.message || 'Failed to load customers';
    } finally {
      loading = false;
    }
  }

  // Debounced search - only triggers on user input
  const debouncedSearch = debounce(() => load(0, limit), 400);

  // Track when we're programmatically clearing to avoid double trigger
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

  async function createCustomer() {
    if (!name.trim()) { errorMsg = 'Name is required'; return; }
    if (!phone.trim()) { errorMsg = 'Phone is required'; return; }
    if (!validatePhone(phone.trim())) { errorMsg = 'Invalid phone format'; return; }
    if (!email.trim()) { errorMsg = 'Email is required'; return; }
    if (!validateEmail(email.trim())) { errorMsg = 'Invalid email format'; return; }
    errorMsg = '';
    try {
      await apiClient.post('/customers', {
        name: name.trim(),
        phone: phone.trim(),
        email: email.trim(),
        note: note.trim() || undefined,
      });
      name = ''; phone = ''; email = ''; note = '';
      await load();
    } catch (e: any) {
      errorMsg = e?.response?.data?.error || 'Failed to create customer';
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
    if (!editName.trim()) { errorMsg = 'Name is required'; return; }
    if (!editPhone.trim()) { errorMsg = 'Phone is required'; return; }
    if (!validatePhone(editPhone.trim())) { errorMsg = 'Invalid phone format'; return; }
    if (!editEmail.trim()) { errorMsg = 'Email is required'; return; }
    if (!validateEmail(editEmail.trim())) { errorMsg = 'Invalid email format'; return; }
    errorMsg = '';
    try {
      await apiClient.put(`/customers/${id}`, {
        name: editName.trim(),
        phone: editPhone.trim() || undefined,
        email: editEmail.trim() || undefined,
        note: editNote.trim() || undefined,
        is_active: editActive,
      });
      editingId = null;
      await load();
    } catch (e: any) {
      errorMsg = e?.response?.data?.error || 'Failed to update customer';
    }
  }

  async function deactivateCustomer(c: any) {
    if (!confirm(`Deactivate customer "${c.name}"?`)) return;
    try {
      await apiClient.delete(`/customers/${c.id}`);
      if (editingId === c.id) editingId = null;
      await load();
    } catch (e: any) {
      errorMsg = e?.response?.data?.error || 'Failed to deactivate customer';
    }
  }

  onMount(load);
</script>

<div class="space-y-4">
  <h1 class="text-2xl font-bold text-text-primary">Customers</h1>

  <!-- Search -->
  <div class="relative max-w-xs">
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

  {#if canCreate}
    <div class="border border-border rounded-xl p-4 space-y-3 bg-bg-card">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
        <input class="input" placeholder="Name *" bind:value={name} required />
        <input class="input" placeholder="Phone *" bind:value={phone} required />
        <input class="input" placeholder="Email *" bind:value={email} required />
        <button class="btn btn-primary" onclick={createCustomer}>
          <Plus size={14} /> Create
        </button>
      </div>
    </div>
  {/if}

  {#if errorMsg}
    <div class="text-danger text-sm">{errorMsg}</div>
  {/if}

  <div class="border border-border rounded-xl overflow-hidden bg-bg-card">
    <table class="min-w-full text-sm table-fixed">
      <thead class="bg-surface-hover text-text-secondary">
        <tr>
          <th class="text-left px-4 py-2 w-[30%]">Name</th>
          <th class="text-left px-4 py-2 w-[18%]">Phone</th>
          <th class="text-left px-4 py-2 w-[26%]">Email</th>
          <th class="text-left px-4 py-2 w-[14%]">Status</th>
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
                <td class="px-4 py-1.5 h-12 overflow-hidden"><input class="w-full h-6 px-2.5 py-1 text-xs leading-none rounded-lg bg-bg-secondary text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary-default/20 border-0" bind:value={editName} /></td>
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
                <td class="px-4 py-1.5 h-12 overflow-hidden">{c.name}</td>
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
  </div>

  <!-- Pagination -->
  <Pagination {total} {limit} {offset} onPageChange={load} />
</div>