<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Card, Button, Badge } from '$lib/components/ui';
  import { Plus, Edit, Trash2, X } from 'lucide-svelte';
  import { ui } from '$lib/stores/ui';

  let users: any[] = [];
  let roles: any[] = [];
  let loading = true;
  let search = '';

  // Modal state
  let showModal = false;
  let isEditing = false;
  let editingUserId: number | null = null;
  let form = {
    username: '',
    email: '',
    password: '',
    role_id: null as number | null,
    store_id: null as number | null,
    is_active: true
  };
  let formErrors: Record<string, string> = {};

  onMount(async () => {
    await Promise.all([
      fetchUsers(),
      fetchRoles()
    ]);
    loading = false;
  });

  async function fetchUsers() {
    try {
      const res = await fetch('/api/admin/users');
      const data = await res.json();
      users = data.data || [];
    } catch (e) { console.error(e); }
  }

  async function fetchRoles() {
    try {
      const res = await fetch('/api/admin/roles');
      const data = await res.json();
      roles = data.data || [];
    } catch (e) { console.error(e); }
  }

  $: filtered = users.filter(u =>
    u.username.toLowerCase().includes(search.toLowerCase()) ||
    u.email.toLowerCase().includes(search.toLowerCase())
  );

  function openCreate() {
    isEditing = false;
    editingUserId = null;
    form = { username: '', email: '', password: '', role_id: null, store_id: null, is_active: true };
    formErrors = {};
    showModal = true;
  }

  function openEdit(user: any) {
    isEditing = true;
    editingUserId = user.id;
    form = {
      username: user.username,
      email: user.email,
      password: '',
      role_id: user.role?.id || user.role_id || null,
      store_id: user.store_id || null,
      is_active: user.is_active
    };
    formErrors = {};
    showModal = true;
  }

  async function saveUser() {
    formErrors = {};
    // Basic validation
    if (!form.username) formErrors.username = 'Username wajib diisi';
    if (!form.email) formErrors.email = 'Email wajib diisi';
    if (!form.role_id) formErrors.role_id = 'Role wajib dipilih';
    if (!isEditing && !form.password) formErrors.password = 'Password wajib diisi';

    if (Object.keys(formErrors).length > 0) return;

    const payload: any = {
      username: form.username,
      email: form.email,
      role_id: form.role_id,
      store_id: form.store_id,
      is_active: form.is_active
    };
    if (form.password) payload.password = form.password;

    try {
      const url = isEditing ? `/api/admin/users/${editingUserId}` : '/api/admin/users';
      const method = isEditing ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Gagal menyimpan user');
      }
      showModal = false;
      await fetchUsers();
      ui.success(`User berhasil ${isEditing ? 'diperbarui' : 'dibuat'}`);
    } catch (e: any) {
      ui.error(e.message);
    }
  }

  async function deleteUser(user: any) {
    if (!confirm(`Yakin ingin menghapus user ${user.username}?`)) return;
    try {
      const res = await fetch(`/api/admin/users/${user.id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Gagal menghapus user');
      await fetchUsers();
      ui.success('User berhasil dihapus');
    } catch (e: any) {
      ui.error(e.message);
    }
  }

  async function toggleActive(user: any) {
    try {
      const payload = {
        username: user.username,
        email: user.email,
        role_id: user.role?.id || user.role_id,
        store_id: user.store_id,
        is_active: !user.is_active
      };
      const res = await fetch(`/api/admin/users/${user.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (!res.ok) throw new Error('Gagal memperbarui status');
      user.is_active = !user.is_active;
      users = [...users];
    } catch (e: any) {
      ui.error(e.message);
    }
  }
</script>

<div class="p-6 bg-gray-100 min-h-screen">
  <div class="flex justify-between items-center mb-6">
    <div>
      <h1 class="text-2xl font-bold text-gray-800">Manajemen Pengguna</h1>
      <p class="text-gray-600">Kelola user dan akses</p>
    </div>
    <Button on:click={openCreate}><Plus size={18} /> Tambah User</Button>
  </div>

  <Card class="p-4 mb-4">
    <input type="text" bind:value={search} placeholder="Cari user..." class="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500" />
  </Card>

  {#if loading}
    <div class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
    </div>
  {:else}
    <Card class="overflow-hidden">
      <table class="w-full">
        <thead class="bg-gray-50">
          <tr class="border-b">
            <th class="text-left p-4 font-semibold">User</th>
            <th class="text-left p-4 font-semibold">Email</th>
            <th class="text-left p-4 font-semibold">Role</th>
            <th class="text-left p-4 font-semibold">Status</th>
            <th class="text-left p-4 font-semibold">Aksi</th>
          </tr>
        </thead>
        <tbody>
          {#each filtered as user (user.id)}
            <tr class="border-b hover:bg-gray-50">
              <td class="p-4">
                <div class="font-medium">{user.username}</div>
              </td>
              <td class="p-4 text-sm text-gray-600">{user.email}</td>
              <td class="p-4">
                <Badge variant={user.role?.name === 'superadmin' ? 'danger' : 'secondary'}>
                  {user.role?.name || user.role || '-'}
                </Badge>
              </td>
              <td class="p-4">
                <Badge variant={user.is_active ? 'success' : 'warning'}>
                  {user.is_active ? 'Aktif' : 'Nonaktif'}
                </Badge>
              </td>
              <td class="p-4">
                <div class="flex gap-2">
                  <button on:click={() => openEdit(user)} class="p-1 text-blue-600 hover:bg-blue-50 rounded" title="Edit">
                    <Edit size={16} />
                  </button>
                  <button on:click={() => deleteUser(user)} class="p-1 text-red-600 hover:bg-red-50 rounded" title="Hapus">
                    <Trash2 size={16} />
                  </button>
                  <button on:click={() => toggleActive(user)} class="px-2 py-1 text-xs rounded {user.is_active ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}" title="Toggle Aktif">
                    {user.is_active ? 'Nonaktifkan' : 'Aktifkan'}
                  </button>
                </div>
              </td>
            </tr>
          {/each}
          {#if filtered.length === 0}
            <tr><td colspan="5" class="text-center py-8 text-gray-400">Tidak ada user ditemukan</td></tr>
          {/if}
        </tbody>
      </table>
    </Card>
  {/if}
</div>

{#if showModal}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={() => showModal = false}>
    <div class="bg-white rounded-lg p-6 w-full max-w-md" on:click|stopPropagation>
      <div class="flex justify-between items-center mb-4">
        <h2 class="text-xl font-bold">{isEditing ? 'Edit User' : 'Tambah User Baru'}</h2>
        <button on:click={() => showModal = false} class="text-gray-500 hover:text-gray-700"><X size={20} /></button>
      </div>

      <form on:submit|preventDefault={saveUser}>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Username</label>
            <input type="text" bind:value={form.username} class="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-blue-500 focus:border-blue-500" class:border-red-500={formErrors.username} />
            {#if formErrors.username}<p class="text-red-600 text-xs mt-1">{formErrors.username}</p>{/if}
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input type="email" bind:value={form.email} class="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-blue-500 focus:border-blue-500" class:border-red-500={formErrors.email} />
            {#if formErrors.email}<p class="text-red-600 text-xs mt-1">{formErrors.email}</p>{/if}
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Password {isEditing ? '(kosongkan jika tidak diubah)' : ''}
            </label>
            <input type="password" bind:value={form.password} class="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-blue-500 focus:border-blue-500" class:border-red-500={formErrors.password} />
            {#if formErrors.password}<p class="text-red-600 text-xs mt-1">{formErrors.password}</p>{/if}
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Role</label>
            <select bind:value={form.role_id} class="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-blue-500" class:border-red-500={formErrors.role_id}>
              <option value={null}>Pilih role...</option>
              {#each roles as role}
                <option value={role.id}>{role.name} {role.description ? '- ' + role.description : ''}</option>
              {/each}
            </select>
            {#if formErrors.role_id}<p class="text-red-600 text-xs mt-1">{formErrors.role_id}</p>{/if}
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Store ID (opsional)</label>
            <input type="number" bind:value={form.store_id} class="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-blue-500" placeholder="Kosongkan jika tidak ada" />
          </div>

          <div class="flex items-center gap-2">
            <input type="checkbox" id="is_active" bind:checked={form.is_active} class="rounded" />
            <label for="is_active" class="text-sm font-medium text-gray-700">Aktif</label>
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <Button type="button" variant="outline" on:click={() => showModal = false}>Batal</Button>
          <Button type="submit">Simpan</Button>
        </div>
      </form>
    </div>
  </div>
{/if}