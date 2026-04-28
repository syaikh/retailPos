<script lang="ts">
  import { onMount } from 'svelte';
  import { writable } from 'svelte/store';
  import { Card } from '$lib/components/ui';
  import { Badge } from '$lib/components/ui';
  import { Button } from '$lib/components/ui';

  let roles = [];
  let permissions = [];
  let loading = true;
  let showModal = false;
  let editingRole = null;
  let newRoleName = '';
  let selectedPerms = [];

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

  function openModal(role = null) {
    editingRole = role;
    newRoleName = role ? role.name : '';
    selectedPerms = role ? role.permissions?.map(p => p.code) : [];
    showModal = true;
  }

  async function saveRole() {
    try {
      const url = editingRole ? \`/api/admin/roles/\${editingRole.id}\` : '/api/admin/roles';
      const method = editingRole ? 'PUT' : 'POST';
      const body = JSON.stringify({ name: newRoleName, description: '', permission_ids: selectedPerms });
      await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body });
      showModal = false;
      onMount();
    } catch(e) { alert('Gagal menyimpan'); }
  }

  async function deleteRole(id) {
    if (confirm('Yakin hapus role?')) {
      try {
        await fetch(\`/api/admin/roles/\${id}\`, { method: 'DELETE' });
        onMount();
      } catch(e) { alert('Gagal menghapus'); }
    }
  }

  async function updatePermissions(roleId, permCodes) {
    try {
      await fetch(\`/api/admin/roles/\${roleId}/permissions\`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ permission_ids: permCodes })
      });
      onMount();
    } catch(e) { alert('Gagal memperbarui'); }
  }
</script>

<div class="p-6 bg-gray-100 min-h-screen">
  <div class="flex justify-between items-center mb-6">
    <div><h1 class="text-2xl font-bold text-gray-800">Manajemen Role</h1><p class="text-gray-600">Kelola role dan permissions</p></div>
    <Button on:click={() => openModal()}>+ Tambah Role</Button>
  </div>
  {#if loading}
    <div class="flex justify-center py-12"><div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div></div>
  {:else}
    <div class="grid gap-4">
      {#each roles as role}
      <Card class="p-5">
        <div class="flex items-start justify-between mb-3">
          <div>
            <h3 class="text-lg font-semibold">{role.name}</h3>
            <p class="text-sm text-gray-500">{role.description || 'Tidak ada deskripsi'}</p>
          </div>
          <div class="flex gap-2">
            <Button size="sm" variant="outline" on:click={() => openModal(role)}>Edit</Button>
            {#if !role.is_system}
              <Button size="sm" variant="danger" on:click={() => deleteRole(role.id)}>Hapus</Button>
            {/if}
          </div>
        </div>
        <div class="border-t pt-3">
          <p class="text-sm font-medium text-gray-500 mb-2">Permissions</p>
          <div class="flex flex-wrap gap-2">
            {#each permissions as perm}
              <label class="flex items-center gap-1 cursor-pointer">
                <input type="checkbox"
                  checked={role.permissions?.some(p => p.code === perm.code)}
                  on:change={e => {
                    const codes = role.permissions?.map(p => p.code) || [];
                    if (e.target.checked) codes.push(perm.code);
                    else codes = codes.filter(c => c !== perm.code);
                    updatePermissions(role.id, codes);
                  }}
                  class="w-3 h-3"
                />
                <span class="text-xs">{perm.name}</span>
              </label>
            {/each}
          </div>
        </div>
      </Card>
      {/each}
    </div>
  {/if}

  {#if showModal}
    <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4" on:click={() => showModal = false}>
      <div class="bg-white rounded-lg p-6 w-full max-w-md" on:click|stopPropagation>
        <h2 class="text-xl font-bold mb-4">{editingRole ? 'Edit Role' : 'Tambah Role'}</h2>
        <input bind:value={newRoleName} placeholder="Nama role" class="w-full px-4 py-2 border rounded mb-4" />
        <p class="text-sm text-gray-500 mb-2">Permissions:</p>
        <div class="max-h-48 overflow-y-auto border rounded p-2 mb-4">
          {#each permissions as perm}
            <label class="flex items-center gap-2 p-1 cursor-pointer">
              <input type="checkbox" bind:group={selectedPerms} value={perm.code} />
              <span class="text-sm">{perm.name}</span>
            </label>
          {/each}
        </div>
        <div class="flex gap-2 justify-end">
          <Button variant="outline" on:click={() => showModal = false}>Batal</Button>
          <Button on:click={saveRole}>Simpan</Button>
        </div>
      </div>
    </div>
  {/if}
</div>
