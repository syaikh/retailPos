<script lang="ts">
  import { onMount } from 'svelte';
  import { Card, Button, Badge } from '$lib/components/ui';
  import { Search } from 'lucide-svelte';
  import { ui } from '$lib/stores/ui';

  interface AuditLog {
    id: number;
    user_id: number | null;
    username: string;
    role: string;
    action: string;
    entity_type: string;
    entity_id: number | null;
    ip_address: string;
    created_at: string;
    old_values: any;
    new_values: any;
  }

  let logs: AuditLog[] = [];
  let loading = true;
  let search = '';
  let limit = 50;
  let offset = 0;
  let total = 0;

  onMount(async () => {
    await fetchLogs();
  });

  async function fetchLogs() {
    loading = true;
    try {
      const params = new URLSearchParams();
      params.set('limit', String(limit));
      params.set('offset', String(offset));
      if (search) params.set('user_id', search); // simple search by user_id for now
      const res = await fetch(`/api/admin/audit-logs?${params.toString()}`);
      if (!res.ok) throw new Error('Failed to fetch');
      const data = await res.json();
      logs = data.data || [];
      total = data.total || 0;
    } catch (e) {
      console.error(e);
      ui.error('Gagal memuat audit log');
    } finally {
      loading = false;
    }
  }

  function handleSearch() {
    offset = 0;
    fetchLogs();
  }

  function getActionBadgeVariant(action: string): 'success' | 'danger' | 'warning' | 'info' | 'secondary' {
    switch (action?.toLowerCase()) {
      case 'create': return 'success';
      case 'delete': return 'danger';
      case 'update': return 'warning';
      default: return 'info';
    }
  }

  function formatDate(dateStr: string): string {
    if (!dateStr) return '-';
    const d = new Date(dateStr);
    return d.toLocaleString('id-ID');
  }

  function jsonPreview(data: any): string {
    if (!data) return '-';
    try {
      return JSON.stringify(data, null, 2);
    } catch {
      return String(data);
    }
  }
</script>

<div class="p-6 bg-gray-100 min-h-screen">
  <div class="mb-6">
    <h1 class="text-2xl font-bold text-gray-800">Audit Logs</h1>
    <p class="text-gray-600">Riwayat aktivitas sistem</p>
  </div>

  <Card class="p-4 mb-4">
    <div class="flex gap-4">
      <div class="flex-1 relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"><Search size={18} /></span>
        <input
          type="text"
          bind:value={search}
          on:input={handleSearch}
          placeholder="Filter by User ID..."
          class="w-full pl-10 pr-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500"
        />
      </div>
    </div>
  </Card>

  {#if loading}
    <div class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
    </div>
  {:else}
    <Card class="overflow-hidden p-0">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 border-b">
          <tr>
            <th class="text-left p-3 font-semibold">Waktu</th>
            <th class="text-left p-3 font-semibold">User</th>
            <th class="text-left p-3 font-semibold">Role</th>
            <th class="text-left p-3 font-semibold">Aksi</th>
            <th class="text-left p-3 font-semibold">Entity</th>
            <th class="text-left p-3 font-semibold">IP</th>
          </tr>
        </thead>
        <tbody>
          {#each logs as log}
            <tr class="border-b hover:bg-gray-50">
              <td class="p-3 whitespace-nowrap text-xs text-gray-500">{formatDate(log.created_at)}</td>
              <td class="p-3">
                <div class="font-medium">{log.username}</div>
                <div class="text-xs text-gray-400">ID: {log.user_id ?? '-'}</div>
              </td>
              <td class="p-3"><Badge variant="secondary">{log.role}</Badge></td>
              <td class="p-3"><Badge variant={getActionBadgeVariant(log.action)}>{log.action}</Badge></td>
              <td class="p-3 text-xs">
                <div>{log.entity_type}</div>
                {log.entity_id ? <div class="text-gray-400">ID: {log.entity_id}</div> : ''}
              </td>
              <td class="p-3 text-xs text-gray-500">{log.ip_address}</td>
            </tr>
          {/each}
          {#if logs.length === 0}
            <tr><td colspan="6" class="text-center py-8 text-gray-400">Tidak ada audit log</td></tr>
          {/if}
        </tbody>
      </table>
    </Card>
    {#if total > limit}
      <div class="mt-4 text-sm text-gray-600">
        Menampilkan {logs.length} dari {total} entri
      </div>
    {/if}
  {/if}
</div>