<script lang="ts">
  import { onMount } from 'svelte';
  import { Card } from '$lib/components/ui';
  import { Badge } from '$lib/components/ui';
  import { Button } from '$lib/components/ui';

  interface Role {
    id: number;
    name: string;
    description?: string;
    permissions?: { code: string }[];
  }

  let roles: Role[] = [];
  let permissions: any[] = [];
  let loading = true;
  let showModal = false;
  let editingRole: Role | null = null;
  let newRoleName = '';
  let selectedPerms: string[] = [];

  onMount(async () => {
    try {
      const [r, p] = await Promise.all([
        fetch('/api/admin/roles').then(r => r.ok ? r.json() : { data: [] }),
        fetch('/api/admin/permissions').then(r => r.ok ? r.json() : { data: [] })
      ]);
      roles = r.data || [];
      permissions = p.data || [];
    } catch(e) { console.error(e); } finally { loading = false; }
  });

  function openModal(role: Role | null = null) {
    editingRole = role;
    newRoleName = role ? role.name : '';
    selectedPerms = role ? role.permissions?.map(p => p.code) || [] : [];
    showModal = true;
  }

  async function saveRole() {
    try {
      const url = editingRole ? `/api/admin/roles/${editingRole.id}` : '/api/admin/roles';
      const method = editingRole ? 'PUT' : 'POST';
      const body = JSON.stringify({ 
        name: newRoleName, 
        description: '', 
        permission_ids: selectedPerms 
      });
      await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body });
      showModal = false;
      // Refresh list
      const r = await fetch('/api/admin/roles');
      const d = await r.json();
      roles = d.data || [];
    } catch(e) { console.error('Failed to save role'); }
  }

  async function deleteRole(id: number) {
    if (confirm('Yakin hapus role?')) {
      try {
        await fetch(`/api/admin/roles/${id}`, { method: 'DELETE' });
        roles = roles.filter(r => r.id !== id);
      } catch(e) { console.error('Failed to delete role'); }
    }
  }

  async function updatePermissions(roleId: number, permCodes: string[]) {
    try {
      await fetch(`/api/admin/roles/${roleId}/permissions`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ permission_ids: permCodes })
      });
    } catch(e) { console.error('Failed to update perms'); }
  }
</script>

{#if loading}
  <div class="flex justify-center py-12"><div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div></div>
{:else}
  <div class="mb-6 flex justify-between items-center">
    <div><h1 class="text-2xl font-bold text-gray-800">Manajemen Role</h1><p class="text-gray-600">Kelola akses & perizinan</p></div>
    <Button onclick={() => openModal(null)}>+ Tambah Role</Button>
  </div>

  <Card class="overflow-hidden">
    <table class="w-full">
      <thead class="bg-gray-50"><tr class="border-b"><th class="text-left p-4 font-semibold">Nama</th><th class="text-left p-4 font-semibold">Deskripsi</th><th class="text-left p-4 font-semibold">Permissions</th><th class="text-left p-4 font-semibold">Aksi</th></tr></thead>
      <tbody>
        {#each roles as role}
          <tr class="border-b hover:bg-gray-50">
            <td class="p-4 font-medium">{role.name}</td>
            <td class="p-4 text-sm text-gray-600">{role.description || '-'}</td>
            <td class="p-4">
              <div class="flex flex-wrap gap-1">
                {#each role.permissions || [] as perm}
                  <Badge variant="secondary" class="text-xs">{perm.code}</Badge>
                {/each}
              </div>
            </td>
            <td class="p-4">
               <button onclick={() => openModal(role)} class="text-blue-600 hover:underline text-sm mr-2">Edit</button>
              {#if !role.description?.includes('system')}
                 <button onclick={() => deleteRole(role.id)} class="text-red-600 hover:underline text-sm">Hapus</button>
              {/if}
            </td>
          </tr>
        {/each}
        {#if roles.length === 0}<tr><td colspan="4" class="text-center py-8 text-gray-400">Tidak ada role</td></tr>{/if}
      </tbody>
    </table>
  </Card>
{/if}

{#if showModal}
   <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => showModal = false}>
     <div class="bg-white rounded-lg p-6 w-full max-w-md" onclick={(e) => { e.stopPropagation(); }}>
      <div class="flex justify-between items-center mb-4">
        <h2 class="text-xl font-bold">{editingRole ? 'Edit Role' : 'Tambah Role Baru'}</h2>
         <button onclick={() => showModal = false} class="text-gray-500 hover:text-gray-700"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg></button>
      </div>
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-1">Nama Role</label>
          <input type="text" bind:value={newRoleName} class="w-full px-3 py-2 border rounded" placeholder="Contoh: editor" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Permissions</label>
          <div class="flex flex-wrap gap-2 max-h-48 overflow-y-auto border rounded p-2">
            {#each permissions as perm}
              <label class="flex items-center gap-1 text-sm">
                <input type="checkbox" value={perm.code} bind:group={selectedPerms} />
                {perm.code}
              </label>
            {/each}
          </div>
        </div>
        <div class="flex justify-end gap-2">
           <Button variant="outline" onclick={() => showModal = false}>Batal</Button>
           <Button onclick={saveRole}>Simpan</Button>
        </div>
      </div>
    </div>
  </div>
{/if}
