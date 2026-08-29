<script lang="ts">
  import { Button, Pagination, Skeleton, ActionBadge } from '$shared/ui';
  import { Search } from 'lucide-svelte';
  import { formatDateInJakarta, formatTimeInJakarta } from '$shared/utils/jakartaTime';
  import { labels } from '$shared/i18n';

  let {
    items = [],
    loading = false,
    total = 0,
    limit = 20,
    offset = 0,
    onpagechange = (newOffset: number, newLimit: number) => {},
    onrowclick = (log: any) => {},
  }: {
    items?: any[];
    loading?: boolean;
    total?: number;
    limit?: number;
    offset?: number;
    onpagechange?: (newOffset: number, newLimit: number) => void;
    onrowclick?: (log: any) => void;
  } = $props();

  function formatTimestamp(d: string | null | undefined) {
    if (!d) return { date: '—', time: '', full: '—' };
    const dateStr = formatDateInJakarta(d);
    const timeStr = formatTimeInJakarta(d);
    return { date: dateStr, time: timeStr, full: `${dateStr} ${timeStr}` };
  }
</script>

<div>
  {#if loading}
    <div class="card p-0 overflow-hidden">
      <div class="divide-y divide-border/70">
        {#each { length: 8 } as _}
          <div class="flex items-center h-9 px-4">
            <div class="w-[180px] shrink-0">
              <Skeleton width="w-32" height="h-3" />
            </div>
            <div class="w-[180px] shrink-0 flex items-center gap-2">
              <Skeleton width="w-6" height="h-6" rounded="rounded-full" />
              <Skeleton width="w-24" height="h-3" />
            </div>
            <div class="w-[120px] shrink-0">
              <Skeleton width="w-20" height="h-3" />
            </div>
            <div class="w-[120px] shrink-0">
              <Skeleton width="w-16" height="h-3.5" rounded="rounded-full" />
            </div>
            <div class="flex-1">
              <Skeleton width="w-full max-w-sm" height="h-3" />
            </div>
            <div class="w-[150px] shrink-0">
              <Skeleton width="w-24" height="h-3" />
            </div>
          </div>
        {/each}
      </div>
    </div>
  {:else if items.length === 0}
    <div class="card px-4 py-24 flex flex-col items-center justify-center text-center">
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
        <Search size={32} class="text-text-muted" />
      </div>
      <p class="text-text-primary font-semibold mt-4">{labels.noAuditLogsFound}</p>
      <p class="text-text-muted text-sm mt-1 max-w-sm">
        {labels.tryAdjustingYourSearch}
      </p>
      <Button variant="secondary" class="mt-6" onclick={() => onpagechange(0, limit)}> {labels.clearFilters} </Button>
    </div>
  {:else}
    <div class="card p-0 overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left whitespace-nowrap">
          <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
            <tr class="text-[11px] font-semibold text-text-muted uppercase tracking-wider h-10">
              <th class="px-4 py-3 w-[180px]">{labels.timestamp}</th>
              <th class="px-4 py-3 w-[180px]">{labels.actor}</th>
              <th class="px-4 py-3 w-[120px]">{labels.resource}</th>
              <th class="px-4 py-3 w-[120px]">{labels.action}</th>
              <th class="px-4 py-3">{labels.description}</th>
              <th class="px-4 py-3 w-[150px]">{labels.ipAddress}</th>
              <th class="px-4 py-3 w-[140px]">{labels.store}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border/70">
            {#each items as log (log.id)}
              {@const ts = formatTimestamp(log.created_at)}
              <tr
                class="h-10 px-4 leading-none border-t border-border/70 hover:bg-surface-hover/50 transition-colors cursor-pointer"
                onclick={() => onrowclick(log)}
                tabindex="0"
                role="button"
                onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onrowclick(log); } }}
              >
                 <td class="py-2 align-middle">
                   <span class="text-text-primary font-medium text-sm leading-snug">{ts.date}</span>
                   <span class="block text-text-muted text-[10px]">{ts.time}</span>
                 </td>
                 <td class="py-2 align-middle">
                  {#if log.username && log.username !== '—'}
                    <div class="flex items-center gap-2">
                      <div class="w-6 h-6 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
                        <span class="text-[10px] font-bold text-white">{log.username.charAt(0).toUpperCase()}</span>
                      </div>
                      <span class="font-medium text-text-primary text-sm truncate max-w-[130px]">{log.username}</span>
                    </div>
                  {:else}
                    <span class="font-medium text-text-muted text-sm">—</span>
                  {/if}
                 </td>
                 <td class="py-2 align-middle">
                  <span
                    class="font-mono text-sm text-text-secondary bg-surface-hover px-2 py-1 rounded border border-border/50 capitalize"
                  >
                    {log.entity_type || '—'}
                  </span>
                 </td>
                 <td class="py-2 align-middle">
                  <ActionBadge action={log.action} />
                 </td>
                 <td class="py-2 align-middle text-sm text-text-secondary truncate max-w-xs leading-snug" title={log.description}>
                  {log.description || '—'}
                 </td>
                  <td class="py-2 align-middle font-mono text-[10px] text-text-muted leading-none">
                   {log.ip_address || '—'}
                  </td>
                  <td class="py-2 align-middle text-sm text-text-secondary max-w-[140px] truncate">
                   {log.store_name || (log.store_id ? String(log.store_id) : '—')}
                  </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="p-4 bg-surface-subtle/30 border-t border-border/50">
        <Pagination {total} {limit} {offset} onPageChange={onpagechange} />
      </div>
    </div>
  {/if}
</div>
