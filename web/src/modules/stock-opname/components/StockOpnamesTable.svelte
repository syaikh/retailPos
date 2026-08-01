<script lang="ts">
  import { Button, Badge, Skeleton } from '$shared/ui';
  import { ClipboardList } from 'lucide-svelte';
  import type { StockOpnameSession } from '../types';
  import { STOCK_OPNAME_STATUS_LABELS } from '../types';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';

  let {
    sessions = [],
    loading = false,
    searchQuery = '',
    canExport = false,
    onview = () => {},
    onexport = () => {},
  }: {
    sessions?: StockOpnameSession[];
    loading?: boolean;
    searchQuery?: string;
    canExport?: boolean;
    onview?: (session: StockOpnameSession) => void;
    onexport?: (session: StockOpnameSession) => void;
  } = $props();

  function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'danger' | 'primary' | 'muted' {
    switch (status) {
      case 'draft':
        return 'muted';
      case 'counting':
        return 'primary';
      case 'pending_approval':
      case 'needs_recount':
        return 'warning';
      case 'approved':
        return 'success';
      case 'cancelled':
        return 'danger';
      default:
        return 'default';
    }
  }
</script>

{#if loading}
  <div class="overflow-x-auto">
    <table class="w-full" style="table-layout: fixed;" aria-busy="true" aria-label="Loading stock opname sessions">
      <colgroup>
        <col style="width: 20%;" />
        <col style="width: 18%;" />
        <col style="width: 18%;" />
        <col style="width: 10%;" />
        <col style="width: 22%;" />
        <col style="width: 12%;" />
      </colgroup>
      <thead><tr><th>SESSION</th><th>SCOPE</th><th>STATUS</th><th>BLIND</th><th>CREATED</th><th>ACTIONS</th></tr></thead>
      <tbody>{#each Array(5) as _}<tr>{#each Array(6) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
    </table>
  </div>
{:else if sessions.length === 0}
  <div class="flex flex-col items-center justify-center py-12 text-text-muted" role="status">
    <ClipboardList class="w-12 h-12 mb-3" aria-hidden="true" />
    <p class="text-text-primary font-medium">No stock opname sessions found</p>
    {#if searchQuery}
      <p class="text-sm text-text-muted mt-1">Try adjusting your search or filters</p>
    {/if}
  </div>
{:else}
  <div class="overflow-x-auto">
    <table class="w-full min-w-[800px]" style="table-layout: fixed;" role="grid" aria-label="Stock opname sessions">
      <colgroup>
        <col style="width: 20%;" />
        <col style="width: 18%;" />
        <col style="width: 18%;" />
        <col style="width: 10%;" />
        <col style="width: 22%;" />
        <col style="width: 12%;" />
      </colgroup>
      <thead class="bg-muted/50">
        <tr class="border-b text-left text-sm text-text-muted">
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">SESSION</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">SCOPE</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">STATUS</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">BLIND</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap" scope="col">CREATED</th>
          <th class="px-4 py-3 font-semibold whitespace-nowrap text-right" scope="col">ACTIONS</th>
        </tr>
      </thead>
      <tbody>
          {#each sessions as session (session.id)}
            <tr class="border-b border-border transition-colors hover:bg-muted/50 cursor-pointer" onclick={() => onview(session)}>
              <td class="px-4 py-3 text-sm font-medium max-w-0"><span class="truncate block">{session.session_number}</span></td>
              <td class="px-4 py-3 text-sm text-text-secondary whitespace-nowrap">{session.scope_type} #{session.scope_id}</td>
              <td class="px-4 py-3 whitespace-nowrap">
                <Badge variant={getStatusVariant(session.status)} size="sm">{STOCK_OPNAME_STATUS_LABELS[session.status] || session.status}</Badge>
              </td>
              <td class="px-4 py-3 text-sm text-text-secondary whitespace-nowrap">{session.blind_count ? 'Yes' : 'No'}</td>
              <td class="px-4 py-3 text-sm text-text-secondary tabular-nums whitespace-nowrap">{formatDateTimeInJakarta(session.created_at)}</td>
              <td class="px-4 py-3 text-right whitespace-nowrap">
                {#if canExport}
                  <Button variant="ghost" size="sm" onclick={(e) => { e.stopPropagation(); onexport(session); }}>Export</Button>
                {/if}
              </td>
            </tr>
          {/each}
      </tbody>
    </table>
  </div>
{/if}
