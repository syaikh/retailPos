<script lang="ts">
  import { Drawer } from '$shared/ui';
  import { Clock, Users, Palette, Shield } from 'lucide-svelte';
  import type { CustomerGroup } from '../types';
  import apiClient from '$shared/api/http-client';
  import { formatDateInJakarta, formatTimeInJakarta, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime';

  let {
    group = null,
    open = $bindable(false),
    onclose = () => {},
  }: {
    group?: CustomerGroup | null;
    open?: boolean;
    onclose?: () => void;
  } = $props();

  let auditLogs = $state<any[]>([]);
  let auditLoading = $state(false);

  $effect(() => {
    if (open && group) {
      loadAuditTrail(group.id);
    }
  });

  async function loadAuditTrail(groupId: number) {
    auditLoading = true;
    auditLogs = [];
    try {
      const r = await apiClient.get('/audit-logs', { params: { entity_type: 'customer_group', entity_id: groupId, limit: 50, offset: 0 } });
      auditLogs = r.data.data || [];
    } catch {
      auditLogs = [];
    } finally {
      auditLoading = false;
    }
  }

  function timeAgo(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    const dateObj = new Date(dateStr);
    const nowMs = Date.now() + JAKARTA_OFFSET_MS;
    const shiftedDate = new Date(dateObj.getTime() + JAKARTA_OFFSET_MS);
    const diffMs = nowMs - shiftedDate.getTime();
    const mins = Math.floor(diffMs / 60000);
    if (mins < 1) return 'Baru saja';
    if (mins < 60) return `${mins}m lalu`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}j lalu`;
    const days = Math.floor(hrs / 24);
    if (days < 30) return `${days}h lalu`;
    return formatDateInJakarta(dateStr);
  }

  function formatTimestamp(d: string | null | undefined): string {
    if (!d) return '—';
    return `${formatDateInJakarta(d)} ${formatTimeInJakarta(d)}`;
  }

  function getActionVerb(action: string): string {
    const v = (action || '').toUpperCase();
    if (v === 'CREATE') return 'Dibuat';
    if (v === 'UPDATE') return 'Diupdate';
    if (v === 'DELETE') return 'Dihapus';
    return action;
  }

  function getActionVariant(action: string): string {
    const v = (action || '').toUpperCase();
    if (v === 'CREATE') return 'success';
    if (v === 'UPDATE') return 'warning';
    if (v === 'DELETE') return 'danger';
    return 'muted';
  }
</script>

<Drawer bind:open={open} width={480} ariaLabel="Detail Customer Group" {onclose}>
  {#if group}
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex items-center gap-4">
        <div
          class="w-12 h-12 rounded-full flex items-center justify-center text-lg font-bold shrink-0"
          style={group.color ? `background-color: ${group.color}20; color: ${group.color}` : ''}
          class:bg-primary-subtle={!group.color}
          class:text-primary-light={!group.color}
        >
          {group.name?.charAt(0)?.toUpperCase() || '?'}
        </div>
        <div class="min-w-0">
          <h3 class="text-base font-semibold text-text-primary truncate">{group.name}</h3>
          {#if group.description}
            <p class="text-sm text-text-muted truncate">{group.description}</p>
          {/if}
        </div>
      </div>

      <!-- Details grid -->
      <div class="grid grid-cols-2 gap-3">
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1 flex items-center gap-1.5">
            <Users size={10} /> Customers
          </p>
          <p class="text-lg font-semibold text-text-primary tabular-nums">{group.customer_count?.toLocaleString('id-ID') ?? '0'}</p>
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1">Status</p>
          <span class="inline-flex items-center gap-1.5 text-sm font-medium {group.is_active ? 'text-success-light' : 'text-danger-light'}">
            <span class="w-2 h-2 rounded-full {group.is_active ? 'bg-success-light' : 'bg-danger-light'}"></span>
            {group.is_active ? 'Aktif' : 'Nonaktif'}
          </span>
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1 flex items-center gap-1.5">
            <Palette size={10} /> Warna
          </p>
          <div class="flex items-center gap-2">
            <span class="w-4 h-4 rounded-full border border-border/50" style={group.color ? `background-color: ${group.color}` : 'background-color: #636E72'}></span>
            <span class="text-sm text-text-secondary font-mono">{group.color || '#636E72'}</span>
          </div>
        </div>
        <div class="bg-surface-default rounded-lg p-3 border border-border/50">
          <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-1 flex items-center gap-1.5">
            <Clock size={10} /> Diperbarui
          </p>
          <p class="text-sm text-text-primary">{timeAgo(group.updated_at || group.created_at)}</p>
          <p class="text-xs text-text-muted mt-0.5 font-mono">{formatTimestamp(group.updated_at || group.created_at)}</p>
        </div>
      </div>

      <!-- Audit Trail -->
      <div>
        <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mb-3 flex items-center gap-1.5">
          <Shield size={10} /> Riwayat Aktivitas
        </p>
        {#if auditLoading}
          <div class="space-y-2">
            {#each { length: 3 } as _}
              <div class="flex items-center gap-3 p-3 bg-surface-default rounded-lg border border-border/50 animate-pulse">
                <div class="w-8 h-8 rounded-full bg-muted/50"></div>
                <div class="flex-1 space-y-1.5">
                  <div class="h-3 w-3/4 bg-muted/50 rounded"></div>
                  <div class="h-2.5 w-1/3 bg-muted/50 rounded"></div>
                </div>
              </div>
            {/each}
          </div>
        {:else if auditLogs.length === 0}
          <div class="p-4 text-center bg-surface-default/50 rounded-lg border border-dashed border-border/40">
            <p class="text-sm text-text-muted">Belum ada riwayat aktivitas.</p>
          </div>
        {:else}
          <div class="space-y-2">
            {#each auditLogs as log}
              <div class="flex items-start gap-3 p-3 bg-surface-default rounded-lg border border-border/50">
                <div class="w-8 h-8 rounded-full flex items-center justify-center shrink-0 {getActionVariant(log.action) === 'success' ? 'bg-success-subtle' : getActionVariant(log.action) === 'danger' ? 'bg-danger-subtle' : 'bg-warning-subtle'}">
                  <span class="text-xs font-bold {getActionVariant(log.action) === 'success' ? 'text-success-light' : getActionVariant(log.action) === 'danger' ? 'text-danger-light' : 'text-warning-light'}">
                    {getActionVerb(log.action).charAt(0)}
                  </span>
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-sm text-text-primary">
                    <span class="font-medium">{log.username || 'System'}</span>
                    <span class="text-text-muted"> {getActionVerb(log.action).toLowerCase()} </span>
                    {#if log.description}
                      <span class="text-text-muted text-xs">— {log.description}</span>
                    {/if}
                  </p>
                  <p class="text-xs text-text-muted mt-0.5 flex items-center gap-1.5">
                    <Clock size={10} />
                    {timeAgo(log.created_at)} · {formatTimestamp(log.created_at)}
                  </p>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</Drawer>
